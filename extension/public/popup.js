/**
 * Popup UI JavaScript
 */

// Check backend health on popup open
document.addEventListener('DOMContentLoaded', async () => {
    await checkHealth();
    loadSettings();
});

/**
 * Check backend health
 */
async function checkHealth() {
    const statusDot = document.getElementById('status-dot');
    const statusText = document.getElementById('status-text');

    try {
        const response = await fetch('http://127.0.0.1:8080/health');
        const data = await response.json();

        if (data.status === 'healthy') {
            statusDot.classList.remove('offline');
            statusText.textContent = `Backend running (v${data.version})`;
        } else {
            statusDot.classList.add('offline');
            statusText.textContent = 'Backend unhealthy';
        }
    } catch (error) {
        statusDot.classList.add('offline');
        statusText.textContent = 'Backend offline - Please start server';
    }
}

/**
 * Load settings from storage
 */
async function loadSettings() {
    const { enabled = true } = await chrome.storage.local.get(['enabled']);
    document.getElementById('enable-toggle').checked = enabled;
}

/**
 * Save settings
 */
document.getElementById('enable-toggle').addEventListener('change', async (e) => {
    await chrome.storage.local.set({ enabled: e.target.checked });
    console.log('Settings saved:', e.target.checked);
});

/**
 * Open settings (placeholder)
 */
document.getElementById('open-settings').addEventListener('click', (e) => {
    e.preventDefault();
    alert('Settings page coming soon!');
});
