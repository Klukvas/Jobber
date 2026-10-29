// shared/api.js — Shared API utilities for Cursopus extension
const JobberAPI = (() => {
  const API_BASE = "https://cursopus.com";
  const MAX_TRACKED_URLS = 500;

  let _accessToken = null;
  let _apiBase = API_BASE;

  // ── Initialization ─────────────────────────────────

  let _session = null;
  async function authMessage(action, data = {}) {
    const result = await chrome.runtime.sendMessage({ action, ...data });
    if (result?.error) throw new Error(result.error);
    if (!result?.data && action !== "authLogout" && action !== "authStoreProfile" && action !== "authTakePending") throw new Error("Authentication unavailable");
    return result?.data;
  }
  function adopt(session) {
    _session = session;
    _accessToken = session?.accessToken || null;
    _apiBase = session?.apiBase || API_BASE;
    return session;
  }
  async function sessionSnapshot() { return adopt(await authMessage("authSnapshot")); }
  async function init() { await sessionSnapshot(); return !!_accessToken; }
  function sessionId() { return _session?.authSessionId; }
  async function assertSession(session, consentSignal) {
    checkConsentSignal(consentSignal);
    const current = await sessionSnapshot();
    if (!session?.accessToken || !current.accessToken || current.authSessionId !== session.authSessionId || current.apiBase !== session.apiBase) throw new Error("AI_CANCELLED");
    checkConsentSignal(consentSignal);
    return current;
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
    adopt(await authMessage("authLogin", { email, password }));
  }
  async function logout({ onlyIfCurrent = false } = {}) {
    const session = onlyIfCurrent ? _session : null;
    _accessToken = null;
    _session = null;
    await authMessage("authLogout", { session });
  }
  async function refreshAccessToken(session = _session) {
    adopt(await authMessage("authRefresh", { session }));
    return true;
  }
  let profileOwner = null;
  let profileRevision = 0;
  function beginAutofillRequest() {
    profileOwner ||= crypto.randomUUID();
    const request = { owner: profileOwner, revision: ++profileRevision, sessionId: sessionId(), apiBase: _apiBase };
    return {
      request,
      ready: authMessage("authBeginProfile", { request }),
      cancel: () => { void authMessage("authCancelProfile", { request }).catch(() => {}); },
    };
  }
  async function storeAutofillProfile(profile, context) {
    await authMessage("authStoreProfile", { profile, request: context.request });
  }
  async function getAutofillProfile() {
    return (await chrome.runtime.sendMessage({ action: "getAutofillProfile" }))?.profile || null;
  }

  // Requests remain bound to their originating account, including 401 replay.
  async function apiFetch(path, options = {}, original, consentSignal) {
    const session = original ? await assertSession(original, consentSignal) : await sessionSnapshot();
    if (!session.accessToken) throw new Error("SESSION_EXPIRED");
    const send = (token) => fetch(`${session.apiBase}${path}`, {
      ...options,
      headers: { "Content-Type": "application/json", ...options.headers, Authorization: `Bearer ${token}` },
    });
    checkConsentSignal(consentSignal);
    let response = await send(session.accessToken);
    await assertSession(session);
    if (response.status === 401) {
      await refreshAccessToken(session);
      const fresh = await assertSession(session);
      response = await send(fresh.accessToken);
      await assertSession(session);
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

  async function trackImportedUrl(url, { session, consentSignal } = {}) {
    if (!url) return;
    if (session) {
      await assertSession(session, consentSignal);
      checkConsentSignal(consentSignal);
      await authMessage("authTrackImportedUrl", { url, session });
      return;
    }
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

  async function confirmCredits(operation, consentSignal, session) {
    checkConsentSignal(consentSignal);
    await assertSession(session, consentSignal);
    const copy = creditCopy();
    const subscription = await getSubscription(session);
    await assertSession(session, consentSignal);
    const cost = subscription?.credit_costs?.[operation];
    const remaining = subscription?.credits?.remaining;
    if (!Number.isSafeInteger(cost) || cost <= 0 || !Number.isSafeInteger(remaining)) throw new Error(copy.unavailable);
    if (remaining < cost) throw new Error(copy.CREDIT_UNAVAILABLE);
    if (typeof globalThis.confirm !== "function" || !globalThis.confirm(`${copy.cost(cost, remaining)}\n\n${copy.policy}`)) throw new Error("AI_CANCELLED");
    await assertSession(session, consentSignal);
    return crypto.randomUUID();
  }
  async function checkCreditError(response) {
    if (response.ok) return;
    const error = await response.clone().json().catch(() => null);
    const message = creditCopy()[error?.error_code];
    if (typeof message === "string") throw new Error(message);
  }

  // ── Job Operations ──────────────────────────────────

  async function parseJob(pageText, pageUrl, { consentSignal } = {}) {
    const session = await sessionSnapshot();
    const key = await confirmCredits("job_parse", consentSignal, session);
    await assertSession(session, consentSignal);
    const response = await apiFetch("/api/v1/jobs/parse", {
      method: "POST",
      headers: { "Idempotency-Key": key },
      body: JSON.stringify({ page_text: pageText, page_url: pageUrl }),
    }, session, consentSignal);

    if (response.status === 401) throw new Error("SESSION_EXPIRED");
    await checkCreditError(response);
    if (response.status === 403) throw new Error("PARSE_LIMIT_REACHED");
    if (!response.ok) {
      const err = await response.json().catch(() => null);
      throw new Error(err?.error_message || "Failed to parse job page");
    }

    return response.json();
  }

  async function createCompany(name, { session, consentSignal } = {}) {
    const response = await apiFetch("/api/v1/companies", {
      method: "POST",
      body: JSON.stringify({ name }),
    }, session, consentSignal);
    // Surface an expired session instead of silently saving the job with no
    // company; other statuses stay best-effort (null => caller omits company).
    if (response.status === 401) throw new Error("SESSION_EXPIRED");
    if (!response.ok) return null;
    return response.json();
  }

  async function createJob({ title, companyId, source, url, description }, { session, consentSignal } = {}) {
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
    }, session, consentSignal);

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
  async function getUploadedAutofillProfile(resumeId, { consentSignal } = {}) {
    const session = await sessionSnapshot();
    await assertSession(session, consentSignal);
    const path = `/api/v1/resumes/${encodeURIComponent(resumeId)}/autofill-profile`;
    let response = await apiFetch(path, {}, session);
    await assertSession(session, consentSignal);
    if (response.status === 404) {
      const key = await confirmCredits("autofill_import", consentSignal, session);
      await assertSession(session, consentSignal);
      response = await apiFetch(path, { method: "POST", headers: { "Idempotency-Key": key } }, session, consentSignal);
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
    const session = await sessionSnapshot();
    await assertSession(session, consentSignal);
    const cached = await apiFetch(`/api/v1/match-score?job_id=${encodeURIComponent(jobId)}&resume_id=${encodeURIComponent(resumeId)}`, {}, session);
    await assertSession(session, consentSignal);
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
    const key = await confirmCredits("match", consentSignal, session);
    await assertSession(session, consentSignal);
    // Origin cancellation stops lookup/consent; an already sent paid request keeps its transport.
    const response = await apiFetch("/api/v1/match-score", {
      method: "POST",
      headers: { "Idempotency-Key": key },
      body: JSON.stringify({ job_id: jobId, resume_id: resumeId }),
    }, session, consentSignal);

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

  async function getSubscription(session) {
    const response = await apiFetch("/api/v1/subscription", {}, session);
    if (response.status === 401) throw new Error("SESSION_EXPIRED");
    if (!response.ok) return null;
    return response.json();
  }

  // ── Badge ───────────────────────────────────────────

  async function incrementSavedCount({ session, consentSignal } = {}) {
    if (session) {
      await assertSession(session, consentSignal);
      checkConsentSignal(consentSignal);
      return authMessage("authIncrementSavedCount", { session });
    }
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
    sessionId,
    sessionSnapshot,
    takePendingParse: () => authMessage("authTakePending", { sessionId: sessionId() }),
    storeAutofillProfile,
    beginAutofillRequest,
    getAutofillProfile,
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
