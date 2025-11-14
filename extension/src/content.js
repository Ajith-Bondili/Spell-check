/**
 * Content Script - Monitors typing on all web pages
 *
 * This script runs on EVERY webpage and monitors:
 * - <input type="text">
 * - <textarea>
 * - <div contenteditable>
 *
 * On SPACE: Calls /spell for fast correction
 * On PUNCTUATION: Calls /rescore for context-aware correction
 */

console.log('🎯 Local Autocorrect: Content script loaded');

// Configuration
const CONFIG = {
    backendUrl: 'http://127.0.0.1:8080',
    autoCorrectThreshold: 0.9,
    suggestionThreshold: 0.5,
    debounceMs: 150, // Wait before calling /rescore
};

// Track current suggestion
let currentSuggestion = null;
let suggestionElement = null;

/**
 * Initialize autocorrect on page load
 */
function init() {
    console.log('🚀 Initializing Local Autocorrect');

    // Listen for input events on all text fields
    document.addEventListener('input', handleInput, true);
    document.addEventListener('keydown', handleKeyDown, true);

    // Create suggestion UI element
    createSuggestionElement();
}

/**
 * Handle input events (typing)
 */
function handleInput(event) {
    const target = event.target;

    // Only process text inputs
    if (!isTextInput(target)) {
        return;
    }

    // Get the last character typed
    const value = getInputValue(target);
    const lastChar = value.slice(-1);

    // On space: fast spell check
    if (lastChar === ' ') {
        const word = getLastWord(value);
        if (word && word.length > 1) {
            checkSpelling(word, target);
        }
    }

    // On punctuation: context-aware check
    if (isPunctuation(lastChar)) {
        const context = value;
        const word = getLastWord(value.slice(0, -1)); // Exclude punctuation
        if (word && word.length > 1) {
            checkSpellingWithContext(word, context, target);
        }
    }
}

/**
 * Handle keyboard shortcuts
 */
function handleKeyDown(event) {
    // Tab: Accept suggestion
    if (event.key === 'Tab' && currentSuggestion) {
        event.preventDefault();
        applySuggestion(event.target, currentSuggestion);
    }

    // Escape: Dismiss suggestion
    if (event.key === 'Escape' && currentSuggestion) {
        event.preventDefault();
        dismissSuggestion();
    }
}

/**
 * Check if element is a text input
 */
function isTextInput(element) {
    if (!element) return false;

    // Regular inputs
    if (element.tagName === 'INPUT') {
        const type = element.type.toLowerCase();

        // SECURITY: Never autocorrect password fields!
        if (type === 'password') {
            return false;
        }

        return type === 'text' || type === 'search' || type === 'email' || type === '';
    }

    // Textareas
    if (element.tagName === 'TEXTAREA') {
        return true;
    }

    // Contenteditable divs (Gmail, Notion, etc.)
    if (element.isContentEditable) {
        return true;
    }

    return false;
}

/**
 * Get input value (works for regular inputs and contenteditable)
 */
function getInputValue(element) {
    if (element.isContentEditable) {
        return element.innerText || element.textContent || '';
    }
    return element.value || '';
}

/**
 * Set input value
 */
function setInputValue(element, value) {
    if (element.isContentEditable) {
        element.innerText = value;
    } else {
        element.value = value;
    }
}

/**
 * Get the last word from text
 */
function getLastWord(text) {
    // Remove trailing spaces/punctuation
    text = text.trim();

    // Split by whitespace and get last word
    const words = text.split(/\s+/);
    const lastWord = words[words.length - 1];

    // Remove punctuation from end
    return lastWord.replace(/[.,!?;:]$/, '');
}

/**
 * Check if character is punctuation
 */
function isPunctuation(char) {
    return /[.,!?;:]/.test(char);
}

/**
 * Call /spell endpoint for fast spelling check
 */
async function checkSpelling(word, target) {
    try {
        const response = await fetch(`${CONFIG.backendUrl}/spell`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ text: word }),
        });

        if (!response.ok) {
            console.error('❌ Spell check failed:', response.statusText);
            return;
        }

        const data = await response.json();
        console.log('📝 Spell check result:', data);

        // Handle correction
        if (data.should_auto_correct && data.best_candidate) {
            // Auto-correct high confidence
            autoCorrect(target, word, data.best_candidate.word);
        } else if (data.best_candidate && data.best_candidate.confidence >= CONFIG.suggestionThreshold) {
            // Show suggestion for medium confidence
            showSuggestion(target, data.best_candidate.word);
        }
    } catch (error) {
        console.error('❌ Error checking spelling:', error);
    }
}

/**
 * Call /rescore endpoint for context-aware correction
 */
async function checkSpellingWithContext(word, context, target) {
    try {
        const response = await fetch(`${CONFIG.backendUrl}/rescore`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ text: word, context: context }),
        });

        if (!response.ok) {
            console.error('❌ Rescore failed:', response.statusText);
            return;
        }

        const data = await response.json();
        console.log('🧠 Context-aware result:', data);

        // Handle correction (same logic as fast check for now)
        if (data.should_auto_correct && data.best_candidate) {
            autoCorrect(target, word, data.best_candidate.word);
        } else if (data.best_candidate && data.best_candidate.confidence >= CONFIG.suggestionThreshold) {
            showSuggestion(target, data.best_candidate.word);
        }
    } catch (error) {
        console.error('❌ Error checking with context:', error);
    }
}

/**
 * Auto-correct: Replace word immediately
 */
function autoCorrect(target, oldWord, newWord) {
    console.log(`✅ Auto-correcting: ${oldWord} → ${newWord}`);

    const currentValue = getInputValue(target);
    const newValue = currentValue.replace(new RegExp(oldWord + '(?!\\w)', 'g'), newWord);

    setInputValue(target, newValue);

    // Dispatch input event so other scripts know the value changed
    target.dispatchEvent(new Event('input', { bubbles: true }));
}

/**
 * Show suggestion bubble
 */
function showSuggestion(target, suggestion) {
    console.log(`💡 Showing suggestion: ${suggestion}`);

    currentSuggestion = suggestion;

    // Position suggestion near cursor
    const rect = target.getBoundingClientRect();
    suggestionElement.textContent = `Did you mean: ${suggestion}? (Tab to accept, Esc to dismiss)`;
    suggestionElement.style.display = 'block';
    suggestionElement.style.top = `${rect.bottom + window.scrollY + 5}px`;
    suggestionElement.style.left = `${rect.left + window.scrollX}px`;
}

/**
 * Dismiss suggestion
 */
function dismissSuggestion() {
    currentSuggestion = null;
    suggestionElement.style.display = 'none';
}

/**
 * Apply suggestion when user presses Tab
 */
function applySuggestion(target, suggestion) {
    const currentValue = getInputValue(target);
    const lastWord = getLastWord(currentValue);

    const newValue = currentValue.replace(new RegExp(lastWord + '(?!\\w)', 'g'), suggestion);
    setInputValue(target, newValue);

    dismissSuggestion();

    // Dispatch input event
    target.dispatchEvent(new Event('input', { bubbles: true }));
}

/**
 * Create suggestion UI element
 */
function createSuggestionElement() {
    suggestionElement = document.createElement('div');
    suggestionElement.id = 'local-autocorrect-suggestion';
    suggestionElement.style.cssText = `
        position: absolute;
        z-index: 999999;
        background: #2d3748;
        color: white;
        padding: 8px 12px;
        border-radius: 6px;
        font-size: 14px;
        font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
        box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
        display: none;
        max-width: 300px;
        pointer-events: none;
    `;
    document.body.appendChild(suggestionElement);
}

// Initialize when DOM is ready
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
} else {
    init();
}
