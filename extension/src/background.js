/**
 * Background Service Worker
 *
 * Handles:
 * - Extension lifecycle
 * - Settings storage
 * - Health monitoring
 */

console.log('🔧 Local Autocorrect: Background service worker loaded');

// Check backend health on install
chrome.runtime.onInstalled.addListener(async () => {
    console.log('📦 Extension installed');

    // Set default settings
    await chrome.storage.local.set({
        enabled: true,
        backendUrl: 'http://127.0.0.1:8080',
        autoCorrectThreshold: 0.9,
        suggestionThreshold: 0.5,
    });

    // Check if backend is running
    checkBackendHealth();
});

/**
 * Check if backend server is healthy
 */
async function checkBackendHealth() {
    try {
        const response = await fetch('http://127.0.0.1:8080/health');
        const data = await response.json();

        if (data.status === 'healthy') {
            console.log('✅ Backend is healthy:', data);
            await chrome.storage.local.set({ backendHealthy: true });
        } else {
            console.warn('⚠️ Backend responded but not healthy');
            await chrome.storage.local.set({ backendHealthy: false });
        }
    } catch (error) {
        console.error('❌ Backend is not running:', error);
        await chrome.storage.local.set({ backendHealthy: false });
    }
}

// Check health every 30 seconds
setInterval(checkBackendHealth, 30000);

// Listen for messages from content scripts
chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
    if (message.type === 'CHECK_HEALTH') {
        checkBackendHealth().then(() => {
            chrome.storage.local.get(['backendHealthy'], (result) => {
                sendResponse({ healthy: result.backendHealthy });
            });
        });
        return true; // Keep channel open for async response
    }
});
