// shared/api.js — Shared API utilities for Cursopus extension
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

  // Every provider-start call asks once, before dispatch. Auth refresh preserves
  // the same key; a user-initiated new attempt receives a new key and consent.
  const CREDIT_COPY = {
    en: {
      cost: (cost, remaining) => `Cost: ${cost} AI credits. Available: ${remaining}.`,
      policy: "Credits are reserved before processing and charged once AI starts. Errors after AI starts still use credits; checks before AI starts return them. A new AI operation uses a new credit charge. Repeated delivery of the same submitted request is not charged twice. Retrying an existing interview operation is included in its original credit charge (up to 3 attempts); each attempt remains subject to safety spending limits. Input and safety spending limits may stop AI before all credits are used. Continue?",
      unavailable: "Could not verify AI credits. Try again later.",
      CREDIT_UNAVAILABLE: "Not enough AI credits. Free grants 30 once; Pro grants 500 per confirmed paid period, without rollover.",
      AI_BUDGET_REACHED: "AI is paused by a safety spending limit. Credits may remain. Try later or contact support.",
      AI_OPERATION_CONFLICT: "This operation is already submitted or running. Check its status before retrying.",
    },
    ru: {
      cost: (cost, remaining) => `Стоимость: ${cost} ИИ-кредитов. Доступно: ${remaining}.`,
      policy: "Кредиты резервируются до обработки и списываются один раз при запуске ИИ. Ошибка после запуска также расходует кредиты; при остановке до запуска они возвращаются. Новая ИИ-операция оплачивается отдельно. Повторная доставка уже отправленного запроса не оплачивается дважды. Повтор существующей операции подготовки к интервью включён в её исходную стоимость в кредитах (до 3 попыток); каждая попытка ограничена безопасным бюджетом расходов. Лимиты ввода и безопасных расходов могут остановить ИИ до исчерпания кредитов. Продолжить?",
      unavailable: "Не удалось проверить ИИ-кредиты. Повторите позже.",
      CREDIT_UNAVAILABLE: "Недостаточно ИИ-кредитов. Free: 30 однократно; Pro: 500 на подтверждённый оплаченный период, без переноса.",
      AI_BUDGET_REACHED: "ИИ приостановлен из-за безопасного лимита расходов. Кредиты могут оставаться. Попробуйте позже или обратитесь в поддержку.",
      AI_OPERATION_CONFLICT: "Операция уже отправлена или выполняется. Проверьте её статус перед повтором.",
    },
    uk: {
      cost: (cost, remaining) => `Вартість: ${cost} ШІ-кредитів. Доступно: ${remaining}.`,
      policy: "Кредити резервуються до обробки та списуються один раз із запуском ШІ. Помилка після запуску також витрачає кредити; при зупинці до запуску вони повертаються. Нова ШІ-операція оплачується окремо. Повторне доставлення вже надісланого запиту не оплачується двічі. Повтор наявної операції підготовки до інтерв’ю включено до її початкової вартості в кредитах (до 3 спроб); кожна спроба обмежена безпечним бюджетом витрат. Ліміти вводу та безпечних витрат можуть зупинити ШІ до вичерпання кредитів. Продовжити?",
      unavailable: "Не вдалося перевірити ШІ-кредити. Повторіть пізніше.",
      CREDIT_UNAVAILABLE: "Недостатньо ШІ-кредитів. Free: 30 одноразово; Pro: 500 на підтверджений оплачений період, без перенесення.",
      AI_BUDGET_REACHED: "ШІ призупинено через безпечний ліміт витрат. Кредити можуть залишатися. Спробуйте пізніше або зверніться до підтримки.",
      AI_OPERATION_CONFLICT: "Операцію вже надіслано або вона виконується. Перевірте її статус перед повтором.",
    },
  };
  function creditCopy() {
    return CREDIT_COPY[globalThis.navigator?.language?.split("-")[0]] || CREDIT_COPY.en;
  }
  function checkConsentSignal(signal) {
    if (signal?.aborted) throw new Error("AI_CANCELLED");
  }

  async function confirmCredits(operation, consentSignal) {
    checkConsentSignal(consentSignal);
    const copy = creditCopy();
    const subscription = await getSubscription();
    checkConsentSignal(consentSignal);
    const cost = subscription?.credit_costs?.[operation];
    const remaining = subscription?.credits?.remaining;
    if (!Number.isSafeInteger(cost) || cost <= 0 || !Number.isSafeInteger(remaining)) throw new Error(copy.unavailable);
    if (remaining < cost) throw new Error(copy.CREDIT_UNAVAILABLE);
    if (typeof globalThis.confirm !== "function" || !globalThis.confirm(`${copy.cost(cost, remaining)}\n\n${copy.policy}`)) throw new Error("AI_CANCELLED");
    checkConsentSignal(consentSignal);
    return crypto.randomUUID();
  }
  async function checkCreditError(response) {
    if (response.ok) return;
    const error = await response.clone().json().catch(() => null);
    const message = creditCopy()[error?.error_code];
    if (typeof message === "string") throw new Error(message);
  }

  // ── Job Operations ──────────────────────────────────

  async function parseJob(pageText, pageUrl) {
    const key = await confirmCredits("job_parse");
    const response = await apiFetch("/api/v1/jobs/parse", {
      method: "POST",
      headers: { "Idempotency-Key": key },
      body: JSON.stringify({ page_text: pageText, page_url: pageUrl }),
    });

    if (response.status === 401) throw new Error("SESSION_EXPIRED");
    await checkCreditError(response);
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

  async function createJob({ title, companyId, source, url, description }) {
    const body = {
      title,
      company_id: companyId || undefined,
      source: source || undefined,
      url: url || undefined,
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

  // Read an existing profile for free; only an explicit POST may start AI.
  async function getUploadedAutofillProfile(resumeId) {
    const path = `/api/v1/resumes/${encodeURIComponent(resumeId)}/autofill-profile`;
    let response = await apiFetch(path);
    if (response.status === 404) {
      const key = await confirmCredits("autofill_import");
      response = await apiFetch(path, { method: "POST", headers: { "Idempotency-Key": key } });
    }
    await checkCreditError(response);
    if (response.status === 401) throw new Error("SESSION_EXPIRED");
    if (response.status === 403) {
      const err = await response.json().catch(() => null);
      throw new Error(
        err?.error_code === "PAID_FEATURE"
          ? "PAID_FEATURE"
          : "PLAN_LIMIT_REACHED",
      );
    }
    if (response.status === 422) throw new Error("RESUME_UNREADABLE");
    if (response.status === 429) throw new Error("RATE_LIMITED");
    if (!response.ok) throw new Error("Failed to prepare autofill profile");
    return response.json();
  }

  // ── Match Score ───────────────────────────────────

  function isCachedMatch(result) {
    const score = (value) => Number.isInteger(value) && value >= 0 && value <= 100;
    const strings = (value) => value === null || (Array.isArray(value) && value.every((item) => typeof item === "string"));
    return result && result.from_cache === true && score(result.overall_score) && typeof result.summary === "string" &&
      strings(result.missing_keywords) && strings(result.strengths) &&
      (result.categories === null || (Array.isArray(result.categories) && result.categories.every((category) =>
        category && typeof category.name === "string" && score(category.score) && typeof category.details === "string")));
  }

  async function getMatchScore(jobId, resumeId, { consentSignal } = {}) {
    checkConsentSignal(consentSignal);
    const cached = await apiFetch(`/api/v1/match-score?job_id=${encodeURIComponent(jobId)}&resume_id=${encodeURIComponent(resumeId)}`);
    checkConsentSignal(consentSignal);
    if (cached.status === 401) throw new Error("SESSION_EXPIRED");
    if (cached.ok) {
      const result = await cached.json();
      checkConsentSignal(consentSignal);
      if (!isCachedMatch(result)) throw new Error("Failed to read saved match analysis");
      return result;
    }
    const error = await cached.json().catch(() => null);
    checkConsentSignal(consentSignal);
    if (cached.status !== 404 || error?.error_code !== "MATCH_SCORE_NOT_READY") throw new Error("Failed to read saved match analysis");
    const key = await confirmCredits("match", consentSignal);
    checkConsentSignal(consentSignal);
    // Origin cancellation stops lookup/consent; an already sent paid request keeps its transport.
    const response = await apiFetch("/api/v1/match-score", {
      method: "POST",
      headers: { "Idempotency-Key": key },
      body: JSON.stringify({ job_id: jobId, resume_id: resumeId }),
    });

    if (response.status === 401) throw new Error("SESSION_EXPIRED");
    await checkCreditError(response);
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
    const response = await apiFetch("/api/v1/subscription");
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
    getUploadedAutofillProfile,
    getMatchScore,
    getSubscription,
    listResumeBuilders,
    getFullResume,
    incrementSavedCount,
  };
})();
