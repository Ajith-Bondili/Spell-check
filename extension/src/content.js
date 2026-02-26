/**
 * Content Script - monitors typing and requests corrections from background.
 */

console.log('🎯 Local Autocorrect: Content script loaded');

let runtimeState = {
    enabled: true,
    suggestionThreshold: 0.5,
};

let currentSuggestion = null;
let suggestionElement = null;

function init() {
    document.addEventListener('input', handleInput, true);
    document.addEventListener('keydown', handleKeyDown, true);
    chrome.storage.onChanged.addListener(handleStorageChange);
    createSuggestionElement();
    loadRuntimeState();
}

async function loadRuntimeState() {
    const response = await sendMessage({ type: 'GET_STATE' });
    if (!response?.error && response) {
        runtimeState = { ...runtimeState, ...response };
    }
}

function handleStorageChange(changes, areaName) {
    if (areaName !== 'local') {
        return;
    }
    if (changes.enabled) {
        runtimeState.enabled = changes.enabled.newValue;
    }
    if (changes.suggestionThreshold) {
        runtimeState.suggestionThreshold = changes.suggestionThreshold.newValue;
    }
}

async function handleInput(event) {
    const target = event.target;
    if (!runtimeState.enabled || !isTextInput(target)) {
        return;
    }

    const value = getInputValue(target);
    if (!value) {
        return;
    }

    const lastChar = value.slice(-1);
    if (lastChar === ' ') {
        const word = getLastWord(value);
        if (word && word.length > 1) {
            await requestCorrection({ target, word, context: value, useContext: false });
        }
    }

    if (isPunctuation(lastChar)) {
        const context = value;
        const word = getLastWord(value.slice(0, -1));
        if (word && word.length > 1) {
            await requestCorrection({ target, word, context, useContext: true });
        }
    }
}

function handleKeyDown(event) {
    if (event.key === 'Tab' && currentSuggestion) {
        event.preventDefault();
        applySuggestion(currentSuggestion.target, currentSuggestion);
        return;
    }

    if (event.key === 'Escape' && currentSuggestion) {
        event.preventDefault();
        sendFeedback(currentSuggestion.originalWord, currentSuggestion.suggestionWord, false);
        dismissSuggestion();
    }
}

async function requestCorrection({ target, word, context, useContext }) {
    const response = await sendMessage({
        type: 'CHECK_TEXT',
        payload: { word, context, useContext },
    });

    if (!response || response.error || !response.result) {
        return;
    }

    const result = response.result;
    if (result.should_auto_correct && result.best_candidate) {
        autoCorrect(target, word, result.best_candidate.word);
    } else if (
        result.best_candidate &&
        result.best_candidate.confidence >= runtimeState.suggestionThreshold
    ) {
        showSuggestion(target, word, result.best_candidate.word);
    } else {
        dismissSuggestion();
    }
}

function isTextInput(element) {
    if (!element) return false;

    if (element.tagName === 'INPUT') {
        const type = (element.type || '').toLowerCase();
        if (type === 'password') {
            return false;
        }
        return ['text', 'search', 'email', ''].includes(type);
    }

    if (element.tagName === 'TEXTAREA') {
        return true;
    }

    if (element.isContentEditable) {
        return true;
    }

    return false;
}

function getInputValue(element) {
    if (element.isContentEditable) {
        return element.innerText || element.textContent || '';
    }
    return element.value || '';
}

function setInputValue(element, value) {
    if (element.isContentEditable) {
        element.innerText = value;
    } else {
        element.value = value;
    }
}

function getLastWord(text) {
    text = text.trim();
    if (!text) {
        return '';
    }
    const words = text.split(/\s+/);
    return words[words.length - 1].replace(/[.,!?;:]$/, '');
}

function isPunctuation(char) {
    return /[.,!?;:]/.test(char);
}

function autoCorrect(target, oldWord, newWord) {
    const currentValue = getInputValue(target);
    const updated = replaceLastWholeWord(currentValue, oldWord, newWord);
    setInputValue(target, updated);
    target.dispatchEvent(new Event('input', { bubbles: true }));
}

function showSuggestion(target, originalWord, suggestionWord) {
    currentSuggestion = {
        target,
        originalWord,
        suggestionWord,
    };

    const rect = target.getBoundingClientRect();
    suggestionElement.textContent = `Did you mean "${suggestionWord}"? (Tab accept / Esc dismiss)`;
    suggestionElement.style.display = 'block';
    suggestionElement.style.top = `${rect.bottom + window.scrollY + 5}px`;
    suggestionElement.style.left = `${rect.left + window.scrollX}px`;
}

function dismissSuggestion() {
    currentSuggestion = null;
    suggestionElement.style.display = 'none';
}

function applySuggestion(target, suggestion) {
    const currentValue = getInputValue(target);
    const updated = replaceLastWholeWord(currentValue, suggestion.originalWord, suggestion.suggestionWord);
    setInputValue(target, updated);
    target.dispatchEvent(new Event('input', { bubbles: true }));
    sendFeedback(suggestion.originalWord, suggestion.suggestionWord, true);
    dismissSuggestion();
}

function replaceLastWholeWord(text, fromWord, toWord) {
    const escaped = escapeRegex(fromWord);
    const pattern = new RegExp(`\\b${escaped}\\b(?![\\s\\S]*\\b${escaped}\\b)`, 'i');
    return text.replace(pattern, toWord);
}

function escapeRegex(value) {
    return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

function createSuggestionElement() {
    suggestionElement = document.createElement('div');
    suggestionElement.id = 'local-autocorrect-suggestion';
    suggestionElement.style.cssText = `
        position: absolute;
        z-index: 999999;
        background: #1f2937;
        color: #f8fafc;
        padding: 8px 12px;
        border-radius: 6px;
        font-size: 13px;
        font-family: ui-sans-serif, system-ui, -apple-system, 'Segoe UI', sans-serif;
        box-shadow: 0 10px 30px rgba(15, 23, 42, 0.18);
        border: 1px solid #334155;
        display: none;
        max-width: 320px;
        pointer-events: none;
    `;
    document.body.appendChild(suggestionElement);
}

function sendMessage(message) {
    return new Promise((resolve) => {
        chrome.runtime.sendMessage(message, (response) => {
            resolve(response);
        });
    });
}

function sendFeedback(original, suggestion, accepted) {
    sendMessage({
        type: 'SEND_FEEDBACK',
        payload: {
            original,
            suggestion,
            accepted,
        },
    }).catch(() => {
        // ignore feedback failures in content script
    });
}

if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
} else {
    init();
}
