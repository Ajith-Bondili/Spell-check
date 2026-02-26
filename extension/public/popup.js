const $ = (id) => document.getElementById(id);

document.addEventListener('DOMContentLoaded', async () => {
    bindEvents();
    await refreshAll();
});

function bindEvents() {
    $('refresh-health').addEventListener('click', refreshHealth);
    $('save-settings').addEventListener('click', saveSettings);
    $('sync-settings').addEventListener('click', syncSettingsFromBackend);
    $('add-word-btn').addEventListener('click', addCustomWord);
    $('ignore-word-btn').addEventListener('click', addIgnoredWord);
    $('ignore-pair-btn').addEventListener('click', addIgnoredPair);
    $('reset-stats').addEventListener('click', resetStats);
    $('reload-state').addEventListener('click', reloadBackendState);

    $('auto-threshold').addEventListener('input', () => {
        $('auto-threshold-value').textContent = Number($('auto-threshold').value).toFixed(2);
    });
    $('suggestion-threshold').addEventListener('input', () => {
        $('suggestion-threshold-value').textContent = Number($('suggestion-threshold').value).toFixed(2);
    });
}

async function refreshAll() {
    await Promise.all([
        refreshHealth(),
        loadSettings(),
        loadDictionary(),
        loadStats(),
    ]);
}

async function refreshHealth() {
    const result = await sendMessage({ type: 'CHECK_HEALTH' });
    const dot = $('status-dot');
    const text = $('status-text');
    if (result?.healthy) {
        dot.classList.add('ok');
        text.textContent = `Backend online (${result.details?.version || 'unknown'})`;
    } else {
        dot.classList.remove('ok');
        text.textContent = `Backend offline${result?.error ? `: ${result.error}` : ''}`;
    }
}

async function loadSettings() {
    const state = await sendMessage({ type: 'GET_STATE' });
    if (!state || state.error) {
        setMessage(state?.error || 'Unable to load settings', true);
        return;
    }

    $('enable-toggle').checked = Boolean(state.enabled);
    $('mode-select').value = state.mode || 'conservative';
    $('auto-threshold').value = state.autoCorrectThreshold ?? 0.75;
    $('suggestion-threshold').value = state.suggestionThreshold ?? 0.5;
    $('max-suggestions').value = state.maxSuggestions ?? 5;
    $('auto-threshold-value').textContent = Number($('auto-threshold').value).toFixed(2);
    $('suggestion-threshold-value').textContent = Number($('suggestion-threshold').value).toFixed(2);
}

async function saveSettings() {
    const payload = {
        enabled: $('enable-toggle').checked,
        mode: $('mode-select').value,
        autoCorrectThreshold: Number($('auto-threshold').value),
        suggestionThreshold: Number($('suggestion-threshold').value),
        maxSuggestions: Number($('max-suggestions').value),
    };

    const response = await sendMessage({ type: 'UPDATE_SETTINGS', payload });
    if (response?.error) {
        setMessage(`Save failed: ${response.error}`, true);
        return;
    }
    setMessage('Settings saved');
}

async function syncSettingsFromBackend() {
    const response = await sendMessage({ type: 'SYNC_SETTINGS_FROM_BACKEND' });
    if (response?.error) {
        setMessage(`Sync failed: ${response.error}`, true);
        return;
    }
    await loadSettings();
    setMessage('Settings synced from backend');
}

async function loadDictionary() {
    const response = await sendMessage({ type: 'GET_DICTIONARY' });
    if (response?.error) {
        setMessage(`Dictionary load failed: ${response.error}`, true);
        return;
    }
    const words = response?.data?.words || [];
    renderDictionary(words);
}

function renderDictionary(words) {
    const container = $('dictionary-list');
    container.innerHTML = '';
    if (words.length === 0) {
        container.innerHTML = '<div class="list-item"><span>No custom words yet</span></div>';
        return;
    }

    words.forEach((entry) => {
        const row = document.createElement('div');
        row.className = 'list-item';

        const label = document.createElement('span');
        label.textContent = `${entry.word} (${entry.frequency})`;
        row.appendChild(label);

        const button = document.createElement('button');
        button.textContent = 'Remove';
        button.className = 'btn secondary';
        button.style.padding = '4px 6px';
        button.style.fontSize = '11px';
        button.addEventListener('click', async () => {
            const result = await sendMessage({
                type: 'REMOVE_CUSTOM_WORD',
                payload: { word: entry.word },
            });
            if (result?.error) {
                setMessage(`Remove failed: ${result.error}`, true);
                return;
            }
            setMessage(`Removed "${entry.word}"`);
            loadDictionary();
        });

        row.appendChild(button);
        container.appendChild(row);
    });
}

async function addCustomWord() {
    const word = $('add-word-input').value.trim();
    if (!word) {
        setMessage('Enter a custom word first', true);
        return;
    }

    const response = await sendMessage({
        type: 'ADD_CUSTOM_WORD',
        payload: { word },
    });
    if (response?.error) {
        setMessage(`Add word failed: ${response.error}`, true);
        return;
    }
    $('add-word-input').value = '';
    setMessage(`Added "${word}"`);
    await loadDictionary();
}

async function addIgnoredWord() {
    const word = $('ignore-word-input').value.trim();
    if (!word) {
        setMessage('Enter a word to ignore', true);
        return;
    }
    const response = await sendMessage({
        type: 'ADD_IGNORE_RULE',
        payload: { word },
    });
    if (response?.error) {
        setMessage(`Ignore failed: ${response.error}`, true);
        return;
    }
    $('ignore-word-input').value = '';
    setMessage(`Now ignoring "${word}"`);
}

async function addIgnoredPair() {
    const original = $('ignore-original-input').value.trim();
    const suggestion = $('ignore-suggestion-input').value.trim();
    if (!original || !suggestion) {
        setMessage('Provide both original and suggestion', true);
        return;
    }

    const response = await sendMessage({
        type: 'ADD_IGNORE_RULE',
        payload: { original, suggestion },
    });
    if (response?.error) {
        setMessage(`Ignore pair failed: ${response.error}`, true);
        return;
    }
    $('ignore-original-input').value = '';
    $('ignore-suggestion-input').value = '';
    setMessage(`Blocked ${original} → ${suggestion}`);
}

async function loadStats() {
    const response = await sendMessage({ type: 'GET_STATS' });
    if (response?.error) {
        setMessage(`Stats failed: ${response.error}`, true);
        return;
    }

    const stats = response?.data?.stats || {};
    $('stat-total').textContent = String(stats.total_requests || 0);
    $('stat-auto').textContent = String(stats.auto_corrected || 0);
    $('stat-suggest').textContent = String(stats.suggestions || 0);
    $('stat-skip').textContent = String(stats.skipped || 0);
}

async function resetStats() {
    const response = await sendMessage({ type: 'RESET_STATS' });
    if (response?.error) {
        setMessage(`Reset failed: ${response.error}`, true);
        return;
    }
    setMessage('Stats reset');
    await loadStats();
}

async function reloadBackendState() {
    const response = await sendMessage({ type: 'RELOAD_BACKEND_STATE' });
    if (response?.error) {
        setMessage(`Reload failed: ${response.error}`, true);
        return;
    }
    setMessage('Backend state reloaded');
    await refreshAll();
}

function setMessage(text, isError = false) {
    const node = $('popup-message');
    node.textContent = text;
    node.style.color = isError ? '#fecaca' : '#bfdbfe';
}

function sendMessage(message) {
    return new Promise((resolve) => {
        chrome.runtime.sendMessage(message, (response) => {
            resolve(response);
        });
    });
}
