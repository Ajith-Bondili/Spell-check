/**
 * Background service worker
 *
 * Centralizes:
 * - Runtime settings sync
 * - Backend health checks
 * - API calls for content/popup scripts
 */

const DEFAULT_STATE = {
    enabled: true,
    backendUrl: 'http://127.0.0.1:8080',
    mode: 'conservative',
    autoCorrectThreshold: 0.75,
    suggestionThreshold: 0.5,
    maxSuggestions: 5,
    backendHealthy: false,
};

console.log('🔧 Local Autocorrect: Background service worker loaded');

chrome.runtime.onInstalled.addListener(async () => {
    await ensureDefaults();
    await syncSettingsFromBackend();
    await checkBackendHealth();
});

async function ensureDefaults() {
    const current = await chrome.storage.local.get(Object.keys(DEFAULT_STATE));
    const updates = {};
    for (const [key, value] of Object.entries(DEFAULT_STATE)) {
        if (typeof current[key] === 'undefined') {
            updates[key] = value;
        }
    }
    if (Object.keys(updates).length > 0) {
        await chrome.storage.local.set(updates);
    }
}

async function getRuntimeState() {
    return chrome.storage.local.get(Object.keys(DEFAULT_STATE));
}

async function getBackendUrl() {
    const { backendUrl = DEFAULT_STATE.backendUrl } = await chrome.storage.local.get(['backendUrl']);
    return backendUrl;
}

async function backendRequest(path, options = {}) {
    const backendUrl = await getBackendUrl();
    const url = `${backendUrl}${path}`;
    const response = await fetch(url, options);
    const data = await response.json().catch(() => ({}));
    if (!response.ok) {
        const errorMessage = data.error || `backend request failed: ${response.status}`;
        throw new Error(errorMessage);
    }
    return data;
}

async function checkBackendHealth() {
    try {
        const data = await backendRequest('/health');
        const healthy = data.status === 'healthy';
        await chrome.storage.local.set({ backendHealthy: healthy });
        return { healthy, details: data };
    } catch (error) {
        await chrome.storage.local.set({ backendHealthy: false });
        return { healthy: false, error: error.message };
    }
}

async function syncSettingsFromBackend() {
    try {
        const settings = await backendRequest('/settings');
        await chrome.storage.local.set({
            enabled: settings.enabled,
            mode: settings.mode,
            autoCorrectThreshold: settings.auto_correct_threshold,
            suggestionThreshold: settings.suggestion_threshold,
            maxSuggestions: settings.max_suggestions,
        });
        return settings;
    } catch (error) {
        console.warn('⚠️ Failed to sync settings from backend:', error.message);
        return null;
    }
}

async function pushSettingsToBackend(partialSettings) {
    const current = await getRuntimeState();
    const payload = {
        enabled: typeof partialSettings.enabled === 'boolean' ? partialSettings.enabled : current.enabled,
        mode: partialSettings.mode || current.mode,
        auto_correct_threshold: typeof partialSettings.autoCorrectThreshold === 'number'
            ? partialSettings.autoCorrectThreshold
            : current.autoCorrectThreshold,
        suggestion_threshold: typeof partialSettings.suggestionThreshold === 'number'
            ? partialSettings.suggestionThreshold
            : current.suggestionThreshold,
        max_suggestions: typeof partialSettings.maxSuggestions === 'number'
            ? partialSettings.maxSuggestions
            : current.maxSuggestions,
    };

    const updated = await backendRequest('/settings', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
    });

    await chrome.storage.local.set({
        enabled: updated.enabled,
        mode: updated.mode,
        autoCorrectThreshold: updated.auto_correct_threshold,
        suggestionThreshold: updated.suggestion_threshold,
        maxSuggestions: updated.max_suggestions,
    });
    return updated;
}

async function runCorrection(payload) {
    const state = await getRuntimeState();
    if (!state.enabled) {
        return {
            skipped: true,
            reason: 'disabled',
            candidates: [],
            original: payload.word,
        };
    }

    const endpoint = payload.useContext ? '/rescore' : '/spell';
    const body = payload.useContext
        ? { text: payload.word, context: payload.context || '' }
        : { text: payload.word, context: payload.context || '' };

    return backendRequest(endpoint, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
    });
}

setInterval(() => {
    checkBackendHealth().catch((error) => {
        console.error('health check failed:', error);
    });
}, 30000);

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
    (async () => {
        try {
            switch (message.type) {
                case 'CHECK_HEALTH': {
                    const health = await checkBackendHealth();
                    sendResponse(health);
                    return;
                }
                case 'GET_STATE': {
                    const state = await getRuntimeState();
                    sendResponse(state);
                    return;
                }
                case 'SYNC_SETTINGS_FROM_BACKEND': {
                    const settings = await syncSettingsFromBackend();
                    sendResponse({ settings });
                    return;
                }
                case 'UPDATE_SETTINGS': {
                    const updated = await pushSettingsToBackend(message.payload || {});
                    sendResponse({ settings: updated });
                    return;
                }
                case 'CHECK_TEXT': {
                    const result = await runCorrection(message.payload || {});
                    sendResponse({ result });
                    return;
                }
                case 'GET_DICTIONARY': {
                    const data = await backendRequest('/dictionary');
                    sendResponse({ data });
                    return;
                }
                case 'ADD_CUSTOM_WORD': {
                    const data = await backendRequest('/dictionary/words', {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify(message.payload || {}),
                    });
                    sendResponse({ data });
                    return;
                }
                case 'REMOVE_CUSTOM_WORD': {
                    const word = encodeURIComponent(message.payload?.word || '');
                    const data = await backendRequest(`/dictionary/words/${word}`, {
                        method: 'DELETE',
                    });
                    sendResponse({ data });
                    return;
                }
                case 'ADD_IGNORE_RULE': {
                    const data = await backendRequest('/dictionary/ignore', {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify(message.payload || {}),
                    });
                    sendResponse({ data });
                    return;
                }
                case 'GET_STATS': {
                    const data = await backendRequest('/stats');
                    sendResponse({ data });
                    return;
                }
                case 'RESET_STATS': {
                    const data = await backendRequest('/stats/reset', {
                        method: 'POST',
                    });
                    sendResponse({ data });
                    return;
                }
                case 'SEND_FEEDBACK': {
                    const data = await backendRequest('/feedback', {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify(message.payload || {}),
                    });
                    sendResponse({ data });
                    return;
                }
                case 'RELOAD_BACKEND_STATE': {
                    const data = await backendRequest('/reload', {
                        method: 'POST',
                    });
                    sendResponse({ data });
                    return;
                }
                default:
                    sendResponse({ error: `unknown message type: ${message.type}` });
            }
        } catch (error) {
            console.error('background handler error:', error);
            sendResponse({ error: error.message });
        }
    })();

    return true;
});
