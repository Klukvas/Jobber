// Background service worker: handles token refresh

const API_BASE = "https://jobber-app.com";
const REFRESH_ALARM = "jobber-token-refresh";
const REFRESH_INTERVAL_MINUTES = 10;

chrome.runtime.onInstalled.addListener(() => {
  chrome.alarms.create(REFRESH_ALARM, {
    periodInMinutes: REFRESH_INTERVAL_MINUTES,
  });
});

chrome.alarms.onAlarm.addListener(async (alarm) => {
  if (alarm.name !== REFRESH_ALARM) return;

  const { refreshToken, apiBase } = await chrome.storage.local.get([
    "refreshToken",
    "apiBase",
  ]);

  if (!refreshToken) return;

  const base = apiBase || API_BASE;

  try {
    const response = await fetch(`${base}/api/v1/auth/refresh`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ refresh_token: refreshToken }),
    });

    if (!response.ok) {
      await chrome.storage.local.remove(["accessToken", "refreshToken"]);
      return;
    }

    const data = await response.json();
    await chrome.storage.local.set({
      accessToken: data.access_token,
      refreshToken: data.refresh_token,
    });
  } catch {
    // Network error — will retry on next alarm
  }
});
