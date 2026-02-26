/**
 * Content Script - monitors typing and requests corrections from background.
 */

console.log('🎯 Local Autocorrect: Content script loaded');

const DEFAULT_UNDO_TTL_MS = 6000;
const SESSION_ID = `sess_${Date.now()}_${Math.floor(Math.random() * 1_000_000)}`;

let runtimeState = {
    enabled: true,
    suggestionThreshold: 0.5,
};

let currentSuggestion = null;
let currentUndo = null;
let suggestionElement = null;
let undoElement = null;
let undoTimer = null;

const requestCounters = new WeakMap();

function init() {
    document.addEventListener('input', handleInput, true);
    document.addEventListener('keydown', handleKeyDown, true);
    chrome.storage.onChanged.addListener(handleStorageChange);
    createSuggestionElement();
    createUndoElement();
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
        return;
    }

    if ((event.ctrlKey || event.metaKey) && event.shiftKey && event.key === 'Backspace') {
        if (currentUndo) {
            event.preventDefault();
            undoLastAutoCorrection('hotkey');
        }
    }
}

function nextRequestToken(target) {
    const current = requestCounters.get(target) || 0;
    const next = current + 1;
    requestCounters.set(target, next);
    return next;
}

function isStaleRequest(target, token) {
    return (requestCounters.get(target) || 0) !== token;
}

async function requestCorrection({ target, word, context, useContext }) {
    const token = nextRequestToken(target);
    const response = await sendMessage({
        type: 'CHECK_TEXT',
        payload: {
            word,
            context,
            useContext,
            domain: window.location.hostname,
            sessionId: SESSION_ID,
            cursorToken: getCursorToken(target),
        },
    });

    if (isStaleRequest(target, token)) {
        return;
    }
    if (!response || response.error || !response.result) {
        return;
    }

    const result = response.result;
    if (result.should_auto_correct && result.best_candidate) {
        applyAutoCorrection({ target, word, result });
        dismissSuggestion();
        return;
    }

    if (
        result.best_candidate &&
        result.best_candidate.confidence >= runtimeState.suggestionThreshold
    ) {
        showSuggestion(target, word, result.best_candidate.word, result.explanation);
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

function applyAutoCorrection({ target, word, result }) {
    const originalValue = getInputValue(target);
    const correctedValue = replaceLastWholeWord(originalValue, word, result.best_candidate.word);
    if (correctedValue === originalValue) {
        return;
    }

    setInputValue(target, correctedValue);
    target.dispatchEvent(new Event('input', { bubbles: true }));

    const undoEntry = {
        target,
        originalWord: word,
        suggestionWord: result.best_candidate.word,
        beforeText: originalValue,
        afterText: correctedValue,
        correctionId: result.correction_id || '',
        reason: result.reason || '',
        explanation: result.explanation || '',
        source: result.source || '',
        mode: result.decision_mode || '',
        confidence: result.best_candidate.confidence || 0,
        domain: window.location.hostname,
        expiresAt: Date.now() + (result.undo_ttl_ms || DEFAULT_UNDO_TTL_MS),
    };

    currentUndo = undoEntry;
    showUndoChip(undoEntry);
    recordAppliedCorrection(undoEntry).catch(() => {
        // Best effort journaling only.
    });
}

function showSuggestion(target, originalWord, suggestionWord, explanation) {
    currentSuggestion = {
        target,
        originalWord,
        suggestionWord,
    };

    const rect = target.getBoundingClientRect();
    const helper = explanation ? ` • ${explanation}` : '';
    suggestionElement.textContent = `Did you mean "${suggestionWord}"? (Tab accept / Esc dismiss)${helper}`;
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

function showUndoChip(entry) {
    if (undoTimer) {
        clearTimeout(undoTimer);
    }

    const rect = entry.target.getBoundingClientRect();
    undoElement.style.top = `${rect.bottom + window.scrollY + 6}px`;
    undoElement.style.left = `${rect.left + window.scrollX}px`;

    const explanation = entry.explanation || 'Auto-correct applied.';
    undoElement.querySelector('[data-role="message"]').textContent =
        `${entry.originalWord} -> ${entry.suggestionWord} • ${explanation}`;
    undoElement.style.display = 'block';

    undoTimer = setTimeout(() => {
        dismissUndoChip();
    }, Math.max(200, entry.expiresAt - Date.now()));
}

function dismissUndoChip() {
    currentUndo = null;
    if (undoTimer) {
        clearTimeout(undoTimer);
        undoTimer = null;
    }
    undoElement.style.display = 'none';
}

async function undoLastAutoCorrection(source) {
    if (!currentUndo) {
        return;
    }
    if (Date.now() > currentUndo.expiresAt) {
        dismissUndoChip();
        return;
    }

    const { target, beforeText, originalWord, suggestionWord, correctionId } = currentUndo;
    setInputValue(target, beforeText);
    target.dispatchEvent(new Event('input', { bubbles: true }));

    if (correctionId) {
        await sendMessage({
            type: 'UNDO_CORRECTION',
            payload: { correction_id: correctionId, source },
        });
    }
    sendFeedback(originalWord, suggestionWord, false);
    dismissUndoChip();
}

async function keepOriginalWord() {
    if (!currentUndo) {
        return;
    }
    await sendMessage({
        type: 'ADD_IGNORE_RULE',
        payload: { word: currentUndo.originalWord },
    });
    dismissUndoChip();
}

async function blockCorrectionPair() {
    if (!currentUndo) {
        return;
    }
    await sendMessage({
        type: 'ADD_IGNORE_RULE',
        payload: {
            original: currentUndo.originalWord,
            suggestion: currentUndo.suggestionWord,
        },
    });
    dismissUndoChip();
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
        z-index: 999998;
        background: #1f2937;
        color: #f8fafc;
        padding: 8px 12px;
        border-radius: 6px;
        font-size: 13px;
        font-family: ui-sans-serif, system-ui, -apple-system, 'Segoe UI', sans-serif;
        box-shadow: 0 10px 30px rgba(15, 23, 42, 0.18);
        border: 1px solid #334155;
        display: none;
        max-width: 420px;
        pointer-events: none;
    `;
    document.body.appendChild(suggestionElement);
}

function createUndoElement() {
    undoElement = document.createElement('div');
    undoElement.id = 'local-autocorrect-undo';
    undoElement.style.cssText = `
        position: absolute;
        z-index: 999999;
        background: #0f172a;
        color: #e2e8f0;
        border: 1px solid #334155;
        border-radius: 8px;
        box-shadow: 0 14px 36px rgba(15, 23, 42, 0.22);
        padding: 10px;
        width: min(520px, calc(100vw - 20px));
        display: none;
        pointer-events: auto;
        font-family: ui-sans-serif, system-ui, -apple-system, 'Segoe UI', sans-serif;
        font-size: 12px;
    `;

    undoElement.innerHTML = `
        <div data-role="message" style="margin-bottom:8px; line-height:1.4;"></div>
        <div style="display:flex; gap:6px; flex-wrap:wrap;">
            <button data-role="undo" style="border:none; border-radius:6px; padding:6px 9px; cursor:pointer; font-weight:700; background:#38bdf8; color:#082f49;">Undo</button>
            <button data-role="keep" style="border:none; border-radius:6px; padding:6px 9px; cursor:pointer; font-weight:700; background:#334155; color:#e2e8f0;">Always Keep Word</button>
            <button data-role="block" style="border:none; border-radius:6px; padding:6px 9px; cursor:pointer; font-weight:700; background:#7f1d1d; color:#fee2e2;">Never Replace Pair</button>
        </div>
    `;

    undoElement.querySelector('[data-role="undo"]').addEventListener('click', () => {
        undoLastAutoCorrection('chip');
    });
    undoElement.querySelector('[data-role="keep"]').addEventListener('click', () => {
        keepOriginalWord();
    });
    undoElement.querySelector('[data-role="block"]').addEventListener('click', () => {
        blockCorrectionPair();
    });

    document.body.appendChild(undoElement);
}

function getCursorToken(target) {
    if (target.isContentEditable) {
        return `${target.tagName || 'editable'}:${(target.innerText || '').length}`;
    }
    return `${target.tagName || 'input'}:${target.selectionStart ?? (target.value || '').length}`;
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
        // Best effort feedback only.
    });
}

async function recordAppliedCorrection(entry) {
    const response = await sendMessage({
        type: 'RECORD_APPLIED_CORRECTION',
        payload: {
            correction_id: entry.correctionId,
            original: entry.originalWord,
            suggestion: entry.suggestionWord,
            domain: entry.domain,
            source: entry.source,
            mode: entry.mode,
            reason: entry.reason,
            explanation: entry.explanation,
            confidence: entry.confidence,
            session_id: SESSION_ID,
            before_text: entry.beforeText,
            after_text: entry.afterText,
        },
    });

    const correctionID = response?.data?.record?.correction_id;
    if (correctionID && currentUndo) {
        currentUndo.correctionId = correctionID;
    }
}

if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
} else {
    init();
}
