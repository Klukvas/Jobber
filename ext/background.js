// Background service worker
const API_BASE = "https://jobber-app.com";
const ALLOWED_ORIGINS = ["https://jobber-app.com", "http://localhost:8080"];
const REFRESH_ALARM = "jobber-token-refresh";
const REFRESH_INTERVAL_MINUTES = 10;

function isAllowedOrigin(url) {
  return ALLOWED_ORIGINS.some(
    (origin) => url === origin || url.startsWith(origin),
  );
}

function getSafeApiBase(storedBase) {
  return storedBase && isAllowedOrigin(storedBase) ? storedBase : API_BASE;
}

// ── Setup ──────────────────────────────────────────────

chrome.runtime.onInstalled.addListener(async () => {
  await chrome.contextMenus.removeAll();

  chrome.contextMenus.create({
    id: "save-to-jobber",
    title: "Save to Jobber",
    contexts: ["page", "link"],
  });

  chrome.contextMenus.create({
    id: "autofill-with-jobber",
    title: "Autofill with Jobber",
    contexts: ["page"],
  });

  const existing = await chrome.alarms.get(REFRESH_ALARM);
  if (!existing) {
    chrome.alarms.create(REFRESH_ALARM, {
      periodInMinutes: REFRESH_INTERVAL_MINUTES,
    });
  }
});

// ── Context Menu ───────────────────────────────────────

chrome.contextMenus.onClicked.addListener(async (info, tab) => {
  if (!tab) return;

  if (info.menuItemId === "save-to-jobber") {
    await openSidePanelWithPageData(tab);
    return;
  }

  if (info.menuItemId === "autofill-with-jobber") {
    try {
      // Inject autofill script (noop if already injected via manifest)
      await chrome.scripting.insertCSS({
        target: { tabId: tab.id },
        files: ["content/autofill.css"],
      });
      await chrome.scripting.executeScript({
        target: { tabId: tab.id },
        files: ["content/autofill.js"],
      });
      // Trigger autofill immediately (don't wait for button click)
      await chrome.scripting.executeScript({
        target: { tabId: tab.id },
        func: () =>
          window.dispatchEvent(new CustomEvent("jobber-autofill-trigger")),
      });
    } catch (err) {
      console.warn("[Jobber] Autofill injection failed:", err.message);
    }
    return;
  }
});

// ── Keyboard Shortcut ──────────────────────────────────

chrome.commands.onCommand.addListener(async (command) => {
  if (command !== "save-job") return;
  const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });
  if (tab) await openSidePanelWithPageData(tab);
});

// ── Side Panel Opener ──────────────────────────────────

async function openSidePanelWithPageData(tab) {
  // Open side panel FIRST — must happen synchronously in user gesture context
  try {
    await chrome.sidePanel.open({ windowId: tab.windowId });
  } catch (err) {
    console.error("[Jobber] Cannot open side panel:", err.message);
    return;
  }

  // Extract page text AFTER panel is open (panel listens for storage changes)
  let pageData = null;
  try {
    const [result] = await chrome.scripting.executeScript({
      target: { tabId: tab.id },
      func: () => ({
        text: document.body.innerText.trim().substring(0, 50000),
        url: location.href,
      }),
    });
    pageData = result.result;
  } catch (err) {
    console.warn("[Jobber] Cannot extract page text:", err.message);
  }

  // Store data — side panel picks it up via storage listener
  await chrome.storage.session.set({
    pendingParse: { tabId: tab.id, pageData, timestamp: Date.now() },
  });
}

// ── Message Handling ───────────────────────────────────

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  if (message.action === "updateBadge") {
    updateBadge(message.count);
    sendResponse({ ok: true });
    return false;
  }

  if (message.action === "extractPageText") {
    extractPageText(message.tabId).then(sendResponse);
    return true;
  }

  if (message.action === "quickSave") {
    handleQuickSave(message).then(sendResponse);
    return true;
  }

  if (message.action === "openSidePanel") {
    (async () => {
      const [tab] = await chrome.tabs.query({
        active: true,
        currentWindow: true,
      });
      if (tab) await openSidePanelWithPageData(tab);
      sendResponse({ ok: true });
    })();
    return true;
  }

  return false;
});

// ── Page Text Extraction ───────────────────────────────

async function extractPageText(tabId) {
  try {
    const [result] = await chrome.scripting.executeScript({
      target: { tabId },
      func: () => ({
        text: document.body.innerText.trim().substring(0, 50000),
        url: location.href,
      }),
    });
    return result.result;
  } catch {
    return { error: "Cannot read this page. Try a job posting page." };
  }
}

// ── Quick Save (from content scripts) ──────────────────

async function handleQuickSave({ title, company, url, source }) {
  const { accessToken, apiBase: storedBase } = await chrome.storage.local.get([
    "accessToken",
    "apiBase",
  ]);

  if (!accessToken) {
    return { error: "Not logged in. Open Jobber extension to sign in." };
  }

  const apiBase = getSafeApiBase(storedBase);
  const headers = {
    "Content-Type": "application/json",
    Authorization: `Bearer ${accessToken}`,
  };

  try {
    // Create company if name provided
    let companyId;
    if (company) {
      const compRes = await fetch(`${apiBase}/api/v1/companies`, {
        method: "POST",
        headers,
        body: JSON.stringify({ name: company }),
      });
      if (compRes.ok) {
        const c = await compRes.json();
        companyId = c.id;
      }
    }

    // Create job
    const jobRes = await fetch(`${apiBase}/api/v1/jobs`, {
      method: "POST",
      headers,
      body: JSON.stringify({
        title,
        company_id: companyId || undefined,
        source: source || undefined,
        url: url || undefined,
      }),
    });

    if (jobRes.status === 403) {
      return {
        error: "Jobs limit reached. Upgrade your plan at jobber-app.com.",
      };
    }
    if (!jobRes.ok) {
      const err = await jobRes.json().catch(() => null);
      return { error: err?.error_message || "Failed to save" };
    }

    const job = await jobRes.json();

    // Track URL
    const { importedUrls = [] } =
      await chrome.storage.local.get("importedUrls");
    if (job.url && !importedUrls.includes(job.url)) {
      await chrome.storage.local.set({
        importedUrls: [...importedUrls, job.url].slice(-500),
      });
    }

    // Update badge
    const { savedCount = 0 } = await chrome.storage.session.get("savedCount");
    const newCount = savedCount + 1;
    await chrome.storage.session.set({ savedCount: newCount });
    await updateBadge(newCount);

    return { ok: true, job };
  } catch {
    return { error: "Network error. Check your connection." };
  }
}

// ── Badge ──────────────────────────────────────────────

async function updateBadge(count) {
  const text = count > 0 ? String(count) : "";
  await chrome.action.setBadgeText({ text });
  await chrome.action.setBadgeBackgroundColor({ color: "#4361ee" });
}

// Restore badge on startup
chrome.runtime.onStartup.addListener(async () => {
  const { savedCount = 0 } = await chrome.storage.session.get("savedCount");
  await updateBadge(savedCount);
});

// ── Token Refresh ──────────────────────────────────────

chrome.alarms.onAlarm.addListener(async (alarm) => {
  if (alarm.name !== REFRESH_ALARM) return;

  const { refreshToken, apiBase } = await chrome.storage.local.get([
    "refreshToken",
    "apiBase",
  ]);
  if (!refreshToken) return;

  const base = getSafeApiBase(apiBase);

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
