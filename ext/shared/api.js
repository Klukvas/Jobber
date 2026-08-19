// shared/api.js — Shared API utilities for Jobber extension
const JobberAPI = (() => {
  const API_BASE = "https://jobber-app.com";
  const ALLOWED_ORIGINS = ["https://jobber-app.com", "http://localhost:8080"];
  const MAX_TRACKED_URLS = 500;

  let _accessToken = null;
  let _apiBase = API_BASE;

  // ── Initialization ─────────────────────────────────

  function isAllowedOrigin(url) {
    return ALLOWED_ORIGINS.some(
      (origin) => url === origin || url.startsWith(origin),
    );
  }

  async function init() {
    const stored = await chrome.storage.local.get([
      "accessToken",
      "refreshToken",
      "apiBase",
    ]);
    if (stored.apiBase && isAllowedOrigin(stored.apiBase)) {
      _apiBase = stored.apiBase;
    }
    if (stored.accessToken) _accessToken = stored.accessToken;
    return !!_accessToken;
  }

  function getWebAppBase() {
    // In production, frontend and API share the same origin.
    // In dev, backend is :8080 and frontend is :3000.
    if (_apiBase === "http://localhost:8080") {
      return "http://localhost:3000";
    }
    return _apiBase || API_BASE;
  }

  function isLoggedIn() {
    return !!_accessToken;
  }

  // ── Auth ────────────────────────────────────────────

  async function login(email, password) {
    const response = await fetch(`${_apiBase}/api/v1/auth/login`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email, password }),
    });

    if (!response.ok) {
      const err = await response.json().catch(() => null);
      throw new Error(err?.error_message || "Invalid credentials");
    }

    const data = await response.json();
    _accessToken = data.tokens.access_token;

    await chrome.storage.local.set({
      accessToken: data.tokens.access_token,
      refreshToken: data.tokens.refresh_token,
    });
  }

  async function logout() {
    await chrome.storage.local.remove([
      "accessToken",
      "refreshToken",
      "autofillProfile",
    ]);
    _accessToken = null;
  }

  let _refreshPromise = null;

  async function refreshAccessToken() {
    if (_refreshPromise) return _refreshPromise;
    _refreshPromise = _doRefresh().finally(() => {
      _refreshPromise = null;
    });
    return _refreshPromise;
  }

  async function _doRefresh() {
    const { refreshToken } = await chrome.storage.local.get("refreshToken");
    if (!refreshToken) return false;

    try {
      const res = await fetch(`${_apiBase}/api/v1/auth/refresh`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ refresh_token: refreshToken }),
      });
      if (!res.ok) {
        _accessToken = null;
        // Drop the dead tokens from storage too, or other contexts (the
        // background alarm, a reopened popup/panel) would keep reviving the
        // stale access token and loop on 401.
        await chrome.storage.local.remove(["accessToken", "refreshToken"]);
        return false;
      }

      const data = await res.json();
      _accessToken = data.access_token;
      await chrome.storage.local.set({
        accessToken: data.access_token,
        refreshToken: data.refresh_token,
      });
      return true;
    } catch {
      return false;
    }
  }

  // ── API Fetch ───────────────────────────────────────

  async function apiFetch(path, options = {}) {
    const headers = { "Content-Type": "application/json", ...options.headers };
    if (_accessToken) {
      headers["Authorization"] = `Bearer ${_accessToken}`;
    }

    const response = await fetch(`${_apiBase}${path}`, {
      ...options,
      headers,
    });

    if (response.status === 401) {
      const refreshed = await refreshAccessToken();
      if (refreshed) {
        return fetch(`${_apiBase}${path}`, {
          ...options,
          headers: { ...headers, Authorization: `Bearer ${_accessToken}` },
        });
      }
    }

    return response;
  }

  // ── URL Tracking ────────────────────────────────────

  async function getImportedUrls() {
    const { importedUrls = [] } =
      await chrome.storage.local.get("importedUrls");
    return importedUrls;
  }

  async function isUrlImported(url) {
    if (!url) return false;
    const urls = await getImportedUrls();
    return urls.includes(url);
  }

  async function trackImportedUrl(url) {
    if (!url) return;
    const urls = await getImportedUrls();
    if (urls.includes(url)) return;
    const updated = [...urls, url].slice(-MAX_TRACKED_URLS);
    await chrome.storage.local.set({ importedUrls: updated });
  }

  // ── Job Operations ──────────────────────────────────

  async function parseJob(pageText, pageUrl) {
    const response = await apiFetch("/api/v1/jobs/parse", {
      method: "POST",
      body: JSON.stringify({ page_text: pageText, page_url: pageUrl }),
    });

    if (response.status === 401) throw new Error("SESSION_EXPIRED");
    if (response.status === 403) throw new Error("PARSE_LIMIT_REACHED");
    if (!response.ok) {
      const err = await response.json().catch(() => null);
      throw new Error(err?.error_message || "Failed to parse job page");
    }

    return response.json();
  }

  async function createCompany(name) {
    const response = await apiFetch("/api/v1/companies", {
      method: "POST",
      body: JSON.stringify({ name }),
    });
    // Surface an expired session instead of silently saving the job with no
    // company; other statuses stay best-effort (null => caller omits company).
    if (response.status === 401) throw new Error("SESSION_EXPIRED");
    if (!response.ok) return null;
    return response.json();
  }

  async function createJob({
    title,
    companyId,
    source,
    url,
    notes,
    description,
  }) {
    const body = {
      title,
      company_id: companyId || undefined,
      source: source || undefined,
      url: url || undefined,
      notes: notes || undefined,
      description: description || undefined,
    };

    const response = await apiFetch("/api/v1/jobs", {
      method: "POST",
      body: JSON.stringify(body),
    });

    if (response.status === 401) throw new Error("SESSION_EXPIRED");
    if (response.status === 403) throw new Error("JOBS_LIMIT_REACHED");
    if (!response.ok) {
      const err = await response.json().catch(() => null);
      throw new Error(err?.error_message || "Failed to save job");
    }

    return response.json();
  }

  async function listJobs(limit = 20, offset = 0) {
    const response = await apiFetch(
      `/api/v1/jobs?limit=${limit}&offset=${offset}`,
    );
    if (response.status === 401) throw new Error("SESSION_EXPIRED");
    if (!response.ok) throw new Error("Failed to load jobs");
    return response.json();
  }

  // ── Resumes (uploaded PDFs) ─────────────────────────

  async function listResumes(limit = 20) {
    const response = await apiFetch(`/api/v1/resumes?limit=${limit}`);
    if (response.status === 401) throw new Error("SESSION_EXPIRED");
    if (!response.ok) throw new Error("Failed to load resumes");
    return response.json();
  }

  // ── Match Score ───────────────────────────────────

  async function getMatchScore(jobId, resumeId) {
    const response = await apiFetch("/api/v1/match-score", {
      method: "POST",
      body: JSON.stringify({ job_id: jobId, resume_id: resumeId }),
    });

    if (response.status === 401) throw new Error("SESSION_EXPIRED");
    if (response.status === 403) throw new Error("PLAN_LIMIT_REACHED");
    if (!response.ok) {
      const err = await response.json().catch(() => null);
      throw new Error(err?.error_message || "Failed to analyze match");
    }

    return response.json();
  }

  // ── Resume Builder ─────────────────────────────────

  async function listResumeBuilders() {
    const response = await apiFetch("/api/v1/resume-builder");
    if (response.status === 401) throw new Error("SESSION_EXPIRED");
    if (!response.ok) throw new Error("Failed to load resumes");
    return response.json();
  }

  async function getFullResume(id) {
    const response = await apiFetch(`/api/v1/resume-builder/${id}`);
    if (response.status === 401) throw new Error("SESSION_EXPIRED");
    if (!response.ok) throw new Error("Failed to load resume");
    return response.json();
  }

  // ── Subscription ───────────────────────────────────

  async function getSubscription() {
    const response = await apiFetch("/api/v1/subscriptions");
    if (response.status === 401) throw new Error("SESSION_EXPIRED");
    if (!response.ok) return null;
    return response.json();
  }

  // ── Badge ───────────────────────────────────────────

  async function incrementSavedCount() {
    const { savedCount = 0 } = await chrome.storage.session.get("savedCount");
    const newCount = savedCount + 1;
    await chrome.storage.session.set({ savedCount: newCount });
    return newCount;
  }

  // ── Public API ──────────────────────────────────────

  return {
    API_BASE,
    init,
    getWebAppBase,
    isLoggedIn,
    login,
    logout,
    refreshAccessToken,
    apiFetch,
    getImportedUrls,
    isUrlImported,
    trackImportedUrl,
    parseJob,
    createCompany,
    createJob,
    listJobs,
    listResumes,
    getMatchScore,
    getSubscription,
    listResumeBuilders,
    getFullResume,
    incrementSavedCount,
  };
})();
