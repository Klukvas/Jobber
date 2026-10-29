// Background service worker
importScripts("shared/auth-worker.js");
const REFRESH_ALARM = "jobber-token-refresh";
const REFRESH_INTERVAL_MINUTES = 10;

// Clicking the toolbar icon opens the side panel (the single primary UI).
// Registered at the top level so it applies on every service-worker spin-up.
chrome.sidePanel
  .setPanelBehavior({ openPanelOnActionClick: true })
  .catch(() => {});

// ── Setup ──────────────────────────────────────────────

chrome.runtime.onInstalled.addListener(async () => {
  await chrome.contextMenus.removeAll();

  chrome.contextMenus.create({
    id: "save-to-jobber",
    title: "Save to Cursopus",
    contexts: ["page", "link"],
  });

  chrome.contextMenus.create({
    id: "autofill-with-jobber",
    title: "Autofill with Cursopus",
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
      // Trigger via runtime messaging (not a page-dispatchable window event) so
      // a page script can't silently trigger autofill of the user's PII.
      await chrome.tabs.sendMessage(tab.id, { action: "jobber-autofill" });
    } catch (err) {
      console.warn("[Cursopus] Autofill injection failed:", err.message);
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
    console.error("[Cursopus] Cannot open side panel:", err.message);
    return;
  }

  const originalSession = await JobberAuth.snapshot();
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
    console.warn("[Cursopus] Cannot extract page text:", err.message);
  }

  // Publication and consumption share the auth queue, so an older consumer
  // cannot remove a newer command while it is publishing.
  await JobberAuth.publishPending(originalSession, {
    tabId: tab.id, pageData, timestamp: Date.now(),
  });
}

// ── Message Handling ───────────────────────────────────

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  // Only our own content scripts / extension pages can reach us
  // (externally_connectable is absent) — assert it as defence-in-depth.
  if (sender.id !== chrome.runtime.id) return false;

  if (typeof message.action === "string" && message.action.startsWith("auth")) {
    // Credentials and profile writes are only available to extension pages.
    if (sender.tab || !sender.url?.startsWith(chrome.runtime.getURL(""))) return false;
    JobberAuth.handle(message, sender.documentId || sender.url).then(
      (data) => sendResponse({ data }),
      (error) => sendResponse({ error: error.message }),
    );
    return true;
  }

  if (message.action === "getAutofillProfile") {
    // Content scripts receive only a validated profile, never credentials.
    JobberAuth.getAutofillProfile().then(sendResponse, () => sendResponse({ profile: null }));
    return true;
  }

  if (message.action === "updateBadge") {
    if (Number.isFinite(message.count) && message.count >= 0) {
      updateBadge(message.count);
    }
    sendResponse({ ok: true });
    return false;
  }

  if (message.action === "extractPageText") {
    // A content-script sender may only read its own tab; the side panel (an
    // extension page, so no sender.tab) passes the target tabId explicitly.
    const tabId = sender.tab ? sender.tab.id : message.tabId;
    extractPageText(tabId).then(sendResponse);
    return true;
  }

  if (message.action === "quickSave") {
    handleQuickSave(message, sendResponse).then((result) => { if (result) sendResponse(result); });
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

async function handleQuickSave({ title, company, url, source }, onSuccess) {
  const session = await JobberAuth.snapshot();
  if (!session.accessToken) return { error: "Not logged in. Open Cursopus extension to sign in." };
  const request = (path, options) => JobberAuth.authedFetch(path, options, session);

  try {
    // Create company if name provided (best-effort — a failure just omits it)
    let companyId;
    if (company) {
      const compRes = await request("/api/v1/companies", {
        method: "POST",
        body: JSON.stringify({ name: company }),
      });
      if (compRes.status === 401) {
        return {
          error: "Session expired. Open the Cursopus extension to sign in again.",
        };
      }
      if (compRes.ok) {
        const c = await compRes.json();
        companyId = c.id;
      }
    }

    // Only store a real http(s) job URL — never a javascript:/data: scheme.
    const safeUrl = url && /^https?:\/\//i.test(url) ? url : undefined;

    // Create job
    const jobRes = await request("/api/v1/jobs", {
      method: "POST",
      body: JSON.stringify({
        title,
        company_id: companyId || undefined,
        source: source || undefined,
        url: safeUrl,
      }),
    });

    if (jobRes.status === 401) {
      return {
        error: "Session expired. Open the Cursopus extension to sign in again.",
      };
    }
    if (jobRes.status === 403) {
      return {
        error: "Jobs limit reached. Upgrade your plan at cursopus.com.",
      };
    }
    if (!jobRes.ok) {
      const err = await jobRes.json().catch(() => null);
      return { error: err?.error_message || "Failed to save" };
    }

    const job = await jobRes.json();

    return await JobberAuth.finishQuickSave(session, job, updateBadge, onSuccess);
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
  const session = await JobberAuth.snapshot();
  if (session.accessToken) await JobberAuth.refresh(session).catch(() => {});
});
