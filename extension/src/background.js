/**
 * Background service worker
 *
 * Centralizes:
 * - Runtime settings sync
 * - Backend health checks
 * - API calls for content/popup scripts
 * - Domain profile cache
 */

const PROFILE_CACHE_TTL_MS = 5 * 60 * 1000;

const DEFAULT_STATE = {
    enabled: true,
    backendUrl: 'http://127.0.0.1:8080',
    mode: 'conservative',
    autoCorrectThreshold: 0.75,
    suggestionThreshold: 0.5,
    maxSuggestions: 5,
    respectSlang: false,
    backendHealthy: false,
};

const domainProfileCache = new Map();

console.log('🔧 Local Autocorrect: Background service worker loaded');

chrome.runtime.onInstalled.addListener(async () => {
    await ensureDefaults();
    await syncSettingsFromBackend();
    await checkBackendHealth();
});

setInterval(() => {
    checkBackendHealth().catch((error) => {
        console.error('health check failed:', error);
    });
}, 30000);

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

function encodePath(value) {
    return encodeURIComponent((value || '').trim().toLowerCase());
}

function extractDomainFromUrl(url) {
    try {
        const parsed = new URL(url);
        return parsed.hostname.toLowerCase();
    } catch {
        return '';
    }
}

async function getActiveTabDomain() {
    const tabs = await chrome.tabs.query({ active: true, currentWindow: true });
    const tab = tabs?.[0];
    if (!tab?.url) {
        return '';
    }
    return extractDomainFromUrl(tab.url);
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
            respectSlang: Boolean(settings.respect_slang),
        });
        return settings;
    } catch (error) {
        console.warn('Failed to sync settings from backend:', error.message);
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
        respect_slang: typeof partialSettings.respectSlang === 'boolean'
            ? partialSettings.respectSlang
            : Boolean(current.respectSlang),
    };

    const updated = await backendRequest('/settings', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
    });

    domainProfileCache.clear();
    await chrome.storage.local.set({
        enabled: updated.enabled,
        mode: updated.mode,
        autoCorrectThreshold: updated.auto_correct_threshold,
        suggestionThreshold: updated.suggestion_threshold,
        maxSuggestions: updated.max_suggestions,
        respectSlang: Boolean(updated.respect_slang),
    });
    return updated;
}

function fromProfilePayload(profile) {
    return {
        enabled: profile.enabled,
        mode: profile.mode,
        autoCorrectThreshold: profile.auto_correct_threshold,
        suggestionThreshold: profile.suggestion_threshold,
        maxSuggestions: profile.max_suggestions,
        respectSlang: Boolean(profile.respect_slang),
    };
}

async function getDomainProfile(domain, forceRefresh = false) {
    const normalizedDomain = (domain || '').trim().toLowerCase();
    if (!normalizedDomain) {
        const fallback = await backendRequest('/profiles/default');
        return { domain: '', profile: fallback, source: 'default' };
    }

    const cached = domainProfileCache.get(normalizedDomain);
    if (!forceRefresh && cached && cached.expiresAt > Date.now()) {
        return cached.value;
    }

    try {
        const data = await backendRequest(`/profiles/domain/${encodePath(normalizedDomain)}`);
        const value = { domain: data.domain || normalizedDomain, profile: data.profile, source: 'domain' };
        domainProfileCache.set(normalizedDomain, {
            value,
            expiresAt: Date.now() + PROFILE_CACHE_TTL_MS,
        });
        return value;
    } catch (error) {
        const fallback = await backendRequest('/profiles/default');
        const value = { domain: normalizedDomain, profile: fallback, source: 'default' };
        domainProfileCache.set(normalizedDomain, {
            value,
            expiresAt: Date.now() + PROFILE_CACHE_TTL_MS,
        });
        return value;
    }
}

async function saveDomainProfile(domain, partialSettings) {
    const profileData = await getDomainProfile(domain, true);
    const base = fromProfilePayload(profileData.profile);
    const payload = {
        enabled: typeof partialSettings.enabled === 'boolean' ? partialSettings.enabled : base.enabled,
        mode: partialSettings.mode || base.mode,
        auto_correct_threshold: typeof partialSettings.autoCorrectThreshold === 'number'
            ? partialSettings.autoCorrectThreshold
            : base.autoCorrectThreshold,
        suggestion_threshold: typeof partialSettings.suggestionThreshold === 'number'
            ? partialSettings.suggestionThreshold
            : base.suggestionThreshold,
        max_suggestions: typeof partialSettings.maxSuggestions === 'number'
            ? partialSettings.maxSuggestions
            : base.maxSuggestions,
        respect_slang: typeof partialSettings.respectSlang === 'boolean'
            ? partialSettings.respectSlang
            : base.respectSlang,
    };

    const result = await backendRequest(`/profiles/domain/${encodePath(domain)}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
    });
    domainProfileCache.delete((domain || '').toLowerCase());
    return result;
}

async function deleteDomainProfile(domain) {
    const result = await backendRequest(`/profiles/domain/${encodePath(domain)}`, {
        method: 'DELETE',
    });
    domainProfileCache.delete((domain || '').toLowerCase());
    return result;
}

async function runCorrection(payload) {
    const state = await getRuntimeState();
    if (!state.enabled) {
        return {
            skipped: true,
            reason: 'disabled',
            explanation: 'Autocorrect disabled in extension settings.',
            candidates: [],
            original: payload.word,
        };
    }

    const endpoint = payload.useContext ? '/rescore' : '/spell';
    const body = {
        text: payload.word,
        context: payload.context || '',
        domain: payload.domain || '',
        session_id: payload.sessionId || '',
        cursor_token: payload.cursorToken || '',
    };

    return backendRequest(endpoint, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
    });
}

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
    (async () => {
        try {
            switch (message.type) {
                case 'CHECK_HEALTH': {
                    sendResponse(await checkBackendHealth());
                    return;
                }
                case 'GET_STATE': {
                    sendResponse(await getRuntimeState());
                    return;
                }
                case 'SYNC_SETTINGS_FROM_BACKEND': {
                    sendResponse({ settings: await syncSettingsFromBackend() });
                    return;
                }
                case 'UPDATE_SETTINGS': {
                    sendResponse({ settings: await pushSettingsToBackend(message.payload || {}) });
                    return;
                }
                case 'CHECK_TEXT': {
                    sendResponse({ result: await runCorrection(message.payload || {}) });
                    return;
                }
                case 'GET_DICTIONARY': {
                    sendResponse({ data: await backendRequest('/dictionary') });
                    return;
                }
                case 'ADD_CUSTOM_WORD': {
                    sendResponse({
                        data: await backendRequest('/dictionary/words', {
                            method: 'POST',
                            headers: { 'Content-Type': 'application/json' },
                            body: JSON.stringify(message.payload || {}),
                        }),
                    });
                    return;
                }
                case 'REMOVE_CUSTOM_WORD': {
                    const word = encodeURIComponent(message.payload?.word || '');
                    sendResponse({ data: await backendRequest(`/dictionary/words/${word}`, { method: 'DELETE' }) });
                    return;
                }
                case 'ADD_IGNORE_RULE': {
                    sendResponse({
                        data: await backendRequest('/dictionary/ignore', {
                            method: 'POST',
                            headers: { 'Content-Type': 'application/json' },
                            body: JSON.stringify(message.payload || {}),
                        }),
                    });
                    return;
                }
                case 'GET_STATS': {
                    sendResponse({ data: await backendRequest('/stats') });
                    return;
                }
                case 'RESET_STATS': {
                    sendResponse({ data: await backendRequest('/stats/reset', { method: 'POST' }) });
                    return;
                }
                case 'SEND_FEEDBACK': {
                    sendResponse({
                        data: await backendRequest('/feedback', {
                            method: 'POST',
                            headers: { 'Content-Type': 'application/json' },
                            body: JSON.stringify(message.payload || {}),
                        }),
                    });
                    return;
                }
                case 'RECORD_APPLIED_CORRECTION': {
                    sendResponse({
                        data: await backendRequest('/corrections/applied', {
                            method: 'POST',
                            headers: { 'Content-Type': 'application/json' },
                            body: JSON.stringify(message.payload || {}),
                        }),
                    });
                    return;
                }
                case 'UNDO_CORRECTION': {
                    sendResponse({
                        data: await backendRequest('/undo', {
                            method: 'POST',
                            headers: { 'Content-Type': 'application/json' },
                            body: JSON.stringify(message.payload || {}),
                        }),
                    });
                    return;
                }
                case 'GET_PAIN_POINTS': {
                    sendResponse({ data: await backendRequest('/insights/pain-points') });
                    return;
                }
                case 'GET_PROFILES': {
                    sendResponse({ data: await backendRequest('/profiles') });
                    return;
                }
                case 'GET_DOMAIN_PROFILE': {
                    const domain = message.payload?.domain || '';
                    sendResponse({ data: await getDomainProfile(domain, Boolean(message.payload?.forceRefresh)) });
                    return;
                }
                case 'UPDATE_DOMAIN_PROFILE': {
                    const domain = message.payload?.domain || '';
                    const profile = message.payload?.profile || {};
                    sendResponse({ data: await saveDomainProfile(domain, profile) });
                    return;
                }
                case 'DELETE_DOMAIN_PROFILE': {
                    const domain = message.payload?.domain || '';
                    sendResponse({ data: await deleteDomainProfile(domain) });
                    return;
                }
                case 'GET_ACTIVE_DOMAIN': {
                    sendResponse({ domain: await getActiveTabDomain() });
                    return;
                }
                case 'RELOAD_BACKEND_STATE': {
                    const data = await backendRequest('/reload', { method: 'POST' });
                    domainProfileCache.clear();
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
