// DOM elements
const views = {
  login: document.getElementById("view-login"),
  idle: document.getElementById("view-idle"),
  loading: document.getElementById("view-loading"),
  preview: document.getElementById("view-preview"),
  success: document.getElementById("view-success"),
};

const loginForm = document.getElementById("login-form");
const loginEmail = document.getElementById("login-email");
const loginPassword = document.getElementById("login-password");
const loginError = document.getElementById("login-error");
const btnLogin = document.getElementById("btn-login");

const currentUrl = document.getElementById("current-url");
const btnParse = document.getElementById("btn-parse");
const btnLogout = document.getElementById("btn-logout");

const saveForm = document.getElementById("save-form");
const previewTitle = document.getElementById("preview-title");
const previewCompany = document.getElementById("preview-company");
const previewSource = document.getElementById("preview-source");
const previewUrl = document.getElementById("preview-url");
const previewNotes = document.getElementById("preview-notes");
const btnSave = document.getElementById("btn-save");
const btnBack = document.getElementById("btn-back");
const saveError = document.getElementById("save-error");

const idleError = document.getElementById("idle-error");
const btnDone = document.getElementById("btn-done");

// State
const API_BASE = "https://jobber-app.com";
let state = { accessToken: null, apiBase: API_BASE };

// Helpers
function showView(name) {
  Object.values(views).forEach((v) => v.classList.add("hidden"));
  views[name].classList.remove("hidden");
}

function showError(el, msg) {
  el.textContent = msg;
  el.classList.remove("hidden");
}

function hideError(el) {
  el.classList.add("hidden");
}

async function refreshAccessToken() {
  const { refreshToken } = await chrome.storage.local.get("refreshToken");
  if (!refreshToken) return false;

  try {
    const res = await fetch(`${state.apiBase}/api/v1/auth/refresh`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ refresh_token: refreshToken }),
    });
    if (!res.ok) return false;

    const data = await res.json();
    state.accessToken = data.access_token;
    await chrome.storage.local.set({
      accessToken: data.access_token,
      refreshToken: data.refresh_token,
    });
    return true;
  } catch {
    return false;
  }
}

async function apiFetch(path, options = {}) {
  const headers = { "Content-Type": "application/json", ...options.headers };
  if (state.accessToken) {
    headers["Authorization"] = `Bearer ${state.accessToken}`;
  }
  const response = await fetch(`${state.apiBase}${path}`, {
    ...options,
    headers,
  });

  // Auto-refresh on 401 and retry once
  if (response.status === 401) {
    const refreshed = await refreshAccessToken();
    if (refreshed) {
      headers["Authorization"] = `Bearer ${state.accessToken}`;
      return fetch(`${state.apiBase}${path}`, { ...options, headers });
    }
  }

  return response;
}

// Initialize: check if logged in
async function init() {
  const stored = await chrome.storage.local.get([
    "accessToken",
    "refreshToken",
    "apiBase",
  ]);

  // Allow dev override via chrome.storage.local.set({apiBase: "http://localhost:8080"})
  if (stored.apiBase) {
    state.apiBase = stored.apiBase;
  }

  if (stored.accessToken) {
    state.accessToken = stored.accessToken;
    await showIdleView();
  } else {
    showView("login");
  }
}

async function showIdleView() {
  // Get current tab URL
  const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });
  if (tab?.url) {
    currentUrl.textContent = new URL(tab.url).hostname;
  }
  showView("idle");
}

// Login
loginForm.addEventListener("submit", async (e) => {
  e.preventDefault();
  hideError(loginError);
  btnLogin.disabled = true;
  btnLogin.textContent = "Signing in...";

  try {
    const response = await fetch(`${state.apiBase}/api/v1/auth/login`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        email: loginEmail.value,
        password: loginPassword.value,
      }),
    });

    if (!response.ok) {
      const err = await response.json().catch(() => null);
      throw new Error(err?.error_message || "Invalid credentials");
    }

    const data = await response.json();
    state.accessToken = data.tokens.access_token;

    await chrome.storage.local.set({
      accessToken: data.tokens.access_token,
      refreshToken: data.tokens.refresh_token,
    });

    await showIdleView();
  } catch (err) {
    showError(loginError, err.message);
  } finally {
    btnLogin.disabled = false;
    btnLogin.textContent = "Sign In";
  }
});

// Parse job
btnParse.addEventListener("click", async () => {
  hideError(idleError);
  showView("loading");

  try {
    // Get the active tab
    const [tab] = await chrome.tabs.query({
      active: true,
      currentWindow: true,
    });

    if (!tab?.id) {
      throw new Error("No active tab found");
    }

    // Inject content script and extract text
    let pageData;
    try {
      const [result] = await chrome.scripting.executeScript({
        target: { tabId: tab.id },
        func: () => ({
          text: document.body.innerText.trim().substring(0, 50000),
          url: location.href,
        }),
      });
      pageData = result.result;
    } catch {
      throw new Error(
        "Cannot read this page. Try on a job posting page (LinkedIn, Indeed, etc.)",
      );
    }

    if (!pageData?.text || pageData.text.length < 10) {
      throw new Error("Not enough text on this page to parse");
    }

    // Call backend parse endpoint
    let response;
    try {
      response = await apiFetch("/api/v1/jobs/parse", {
        method: "POST",
        body: JSON.stringify({
          page_text: pageData.text,
          page_url: pageData.url,
        }),
      });
    } catch {
      throw new Error(
        "Cannot connect to Jobber server. Check your internet connection.",
      );
    }

    if (response.status === 401) {
      await chrome.storage.local.remove(["accessToken", "refreshToken"]);
      state.accessToken = null;
      showView("login");
      showError(loginError, "Session expired. Please sign in again.");
      return;
    }

    if (!response.ok) {
      const err = await response.json().catch(() => null);
      throw new Error(err?.error_message || "Failed to parse job page");
    }

    const parsed = await response.json();

    // Fill preview form
    previewTitle.value = parsed.title || "";
    previewCompany.value = parsed.company_name || "";
    previewSource.value = parsed.source || "";
    previewUrl.value = parsed.url || pageData.url;
    previewNotes.value = parsed.description || "";
    hideError(saveError);

    showView("preview");
  } catch (err) {
    await showIdleView();
    showError(idleError, err.message);
  }
});

// Save job
saveForm.addEventListener("submit", async (e) => {
  e.preventDefault();
  hideError(saveError);
  btnSave.disabled = true;
  btnSave.textContent = "Saving...";

  try {
    const body = {
      title: previewTitle.value.trim(),
      source: previewSource.value.trim() || undefined,
      url: previewUrl.value.trim() || undefined,
      notes: previewNotes.value.trim() || undefined,
    };

    // Create company first if name provided, then link by ID
    const companyName = previewCompany.value.trim();
    if (companyName) {
      const companyRes = await apiFetch("/api/v1/companies", {
        method: "POST",
        body: JSON.stringify({ name: companyName }),
      });
      if (companyRes.ok) {
        const company = await companyRes.json();
        body.company_id = company.id;
      }
    }

    const response = await apiFetch("/api/v1/jobs", {
      method: "POST",
      body: JSON.stringify(body),
    });

    if (response.status === 401) {
      await chrome.storage.local.remove(["accessToken", "refreshToken"]);
      state.accessToken = null;
      showView("login");
      showError(loginError, "Session expired. Please sign in again.");
      return;
    }

    if (!response.ok) {
      const err = await response.json().catch(() => null);
      throw new Error(err?.error_message || "Failed to save job");
    }

    showView("success");
  } catch (err) {
    showError(saveError, err.message);
  } finally {
    btnSave.disabled = false;
    btnSave.textContent = "Save to Jobber";
  }
});

// Back from preview
btnBack.addEventListener("click", () => showIdleView());

// Done
btnDone.addEventListener("click", () => window.close());

// Logout
btnLogout.addEventListener("click", async () => {
  await chrome.storage.local.remove(["accessToken", "refreshToken", "apiBase"]);
  state.accessToken = null;
  state.apiBase = null;
  showView("login");
});

// Start
init();
