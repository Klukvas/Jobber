// Side panel controller (uses shared/api.js)

// ── DOM Elements ───────────────────────────────────────

const viewLogin = document.getElementById("view-login");
const viewMain = document.getElementById("view-main");

const loginForm = document.getElementById("login-form");
const loginEmail = document.getElementById("login-email");
const loginPassword = document.getElementById("login-password");
const loginError = document.getElementById("login-error");
const btnLogin = document.getElementById("btn-login");
const btnLogout = document.getElementById("btn-logout");

const tabButtons = document.querySelectorAll(".tab");
const tabSave = document.getElementById("tab-save");
const tabJobs = document.getElementById("tab-jobs");
const tabAutofill = document.getElementById("tab-autofill");

// Save tab
const saveIdle = document.getElementById("save-idle");
const saveLoading = document.getElementById("save-loading");
const savePreview = document.getElementById("save-preview");
const saveSuccess = document.getElementById("save-success");
const currentUrlEl = document.getElementById("current-url");
const btnParse = document.getElementById("btn-parse");
const idleError = document.getElementById("idle-error");

const saveForm = document.getElementById("save-form");
const previewTitle = document.getElementById("preview-title");
const previewCompany = document.getElementById("preview-company");
const previewSalary = document.getElementById("preview-salary");
const previewLocation = document.getElementById("preview-location");
const previewSource = document.getElementById("preview-source");
const previewUrl = document.getElementById("preview-url");
const previewDescription = document.getElementById("preview-description");
const btnSave = document.getElementById("btn-save");
const btnBack = document.getElementById("btn-back");
const saveError = document.getElementById("save-error");
const duplicateWarning = document.getElementById("duplicate-warning");

const successJobTitle = document.getElementById("success-job-title");
const btnOpenInJobber = document.getElementById("btn-open-in-jobber");
const btnSaveAnother = document.getElementById("btn-save-another");

// Jobs tab
const jobsLoading = document.getElementById("jobs-loading");
const jobsEmpty = document.getElementById("jobs-empty");
const jobsList = document.getElementById("jobs-list");
const jobsError = document.getElementById("jobs-error");

// Autofill tab
const autofillSetup = document.getElementById("autofill-setup");
const autofillActive = document.getElementById("autofill-active");
const autofillResumeSelect = document.getElementById("autofill-resume-select");
const autofillHint = document.getElementById("autofill-hint");
const autofillBridgeHint = document.getElementById("autofill-bridge-hint");
const btnSetAutofill = document.getElementById("btn-set-autofill");
const btnOpenBuilder = document.getElementById("btn-open-builder");
const autofillError = document.getElementById("autofill-error");
const profileNameEl = document.getElementById("profile-name");
const profileInfoEl = document.getElementById("profile-info");
const btnChangeProfile = document.getElementById("btn-change-profile");

// Match score
const spMatchResumeSelect = document.getElementById("sp-match-resume");
const spBtnCheckMatch = document.getElementById("sp-btn-check-match");
const spMatchError = document.getElementById("sp-match-error");
const spMatchIdle = document.getElementById("sp-match-idle");
const spMatchLoading = document.getElementById("sp-match-loading");
const spMatchResult = document.getElementById("sp-match-result");
const spMatchScore = document.getElementById("sp-match-score");
const spMatchRing = document.getElementById("sp-match-ring");
const spMatchSummary = document.getElementById("sp-match-summary");
const spMatchCategories = document.getElementById("sp-match-categories");
const spMatchStrengths = document.getElementById("sp-match-strengths");
const spMatchMissing = document.getElementById("sp-match-missing");

const usageBar = document.getElementById("usage-bar");

let lastSavedJob = null;
let matchRequest = null;

function cancelSpMatchRequest() {
  if (!matchRequest) return;
  matchRequest.abort();
  matchRequest = null;
  spMatchLoading.classList.add("hidden");
  spMatchIdle.classList.remove("hidden");
}

let parseRequest = null;
let profileRequest = null;
let saveRequest = null;
function cancelProfileSelection() {
  profileRequest?.profileContext?.cancel();
  profileRequest?.abort();
  profileRequest = null;
  btnSetAutofill.disabled = false;
  btnSetAutofill.textContent = "Set as Autofill Profile";
}
function cancelPageRequests() {
  cancelSpMatchRequest();
  parseRequest?.abort();
  parseRequest = null;
  if (!saveLoading.classList.contains("hidden")) showSaveState("idle");
  cancelProfileSelection();
  if (saveRequest) {
    saveRequest.abort();
    saveRequest = null;
    btnSave.disabled = false;
    btnSave.textContent = "Save to Cursopus";
  }
}
function clearPassword() {
  loginPassword.value = "";
  loginPassword.type = "password";
}
function resetPrivateView() {
  cancelPageRequests();
  clearPassword();
  lastSavedJob = null;
  saveForm.reset();
  profileNameEl.textContent = "";
  profileInfoEl.textContent = "";
  autofillResumeSelect.innerHTML = "";
  btnSetAutofill.disabled = false;
  btnSetAutofill.textContent = "Set as Autofill Profile";
  autofillActive.classList.add("hidden");
  jobsList.textContent = "";
  usageBar.textContent = "";
  showSaveState("idle");
  viewMain.classList.add("hidden");
  viewLogin.classList.remove("hidden");
}
window.addEventListener("pagehide", cancelPageRequests);
chrome.storage.onChanged.addListener((changes, area) => {
  if (area !== "local") return;
  if (changes.authSessionId || (changes.accessToken && !changes.accessToken.newValue)) resetPrivateView();
  if (changes.authSessionId || changes.accessToken) {
    JobberAPI.init().then((loggedIn) => { if (loggedIn) { showMainView(); void checkPendingParse(); } }).catch(() => {});
  }
});
// Full URL of the active tab, cached so the parse click can request host access
// for it synchronously (permissions.request must run inside a user gesture).
let currentTabUrl = null;
// ── Helpers ────────────────────────────────────────────

function showError(el, msg) {
  el.textContent = msg;
  el.classList.remove("hidden");
}

function hideError(el) {
  el.classList.add("hidden");
}

// Unwrap a list endpoint response into an array, tolerant of the shapes the API
// has returned over time (bare array, {items}, {resumes}, {jobs}, {data}) and of
// a null body — Go serializes an empty slice as null, which used to crash the
// callers with "Cannot read properties of null (reading 'items')".
function unwrapList(data) {
  if (Array.isArray(data)) return data;
  return data?.items || data?.resumes || data?.jobs || data?.data || [];
}

function showSaveState(state) {
  if (state !== "success") cancelSpMatchRequest();
  [saveIdle, saveLoading, savePreview, saveSuccess].forEach((el) =>
    el.classList.add("hidden"),
  );
  document.getElementById(`save-${state}`).classList.remove("hidden");
}

function handleSessionExpired() {
  resetPrivateView();
  void JobberAPI.logout({ onlyIfCurrent: true });
  viewMain.classList.add("hidden");
  viewLogin.classList.remove("hidden");
  showError(loginError, "Session expired. Please sign in again.");
}

// ── Password Toggle ────────────────────────────────────

document.getElementById("btn-toggle-password").addEventListener("click", () => {
  const isPassword = loginPassword.type === "password";
  loginPassword.type = isPassword ? "text" : "password";
});

// ── Init ───────────────────────────────────────────────

async function init() {
  listenForPendingParse();
  const isLoggedIn = await JobberAPI.init();
  if (isLoggedIn) {
    showMainView();
    await checkPendingParse();
  } else {
    viewLogin.classList.remove("hidden");
    viewMain.classList.add("hidden");
  }
}

let pendingParseListening = false;
function listenForPendingParse() {
  // Register exactly one persistent listener for the panel's lifetime (the flag
  // guards against stacking one on every reopen, which would double-parse). It
  // stays armed rather than removing itself, so a pendingParse written after a
  // stale/missed one is still handled; checkPendingParse guards freshness and
  // clears the item.
  if (pendingParseListening) return;
  pendingParseListening = true;
  chrome.storage.onChanged.addListener((changes, area) => {
    if (area === "session" && changes.pendingParse?.newValue) {
      checkPendingParse();
    }
  });
}

function showMainView() {
  viewLogin.classList.add("hidden");
  viewMain.classList.remove("hidden");
  updateCurrentUrl();
  loadUsage();
}

async function updateCurrentUrl() {
  const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });
  // Without the "tabs" permission, tab.url is only exposed for origins we hold
  // host access to. A hidden URL must reset the cache — keeping the previous
  // tab's URL made requestHostAccess() ask for the wrong origin, so extraction
  // then failed on the actual page with "Cannot read this page".
  currentTabUrl = tab?.url || null;
  if (!currentTabUrl) {
    currentUrlEl.textContent = "current page";
    return;
  }
  try {
    currentUrlEl.textContent = new URL(currentTabUrl).hostname;
  } catch {
    currentUrlEl.textContent = currentTabUrl;
  }
}

// The panel outlives tab switches and in-tab navigations, so the cached URL has
// to track them — it was previously refreshed only when the panel (re)opened.
chrome.tabs.onActivated.addListener(() => {
  cancelPageRequests();
  updateCurrentUrl();
});
chrome.tabs.onUpdated.addListener((tabId, changeInfo, tab) => {
  if (tab.active && (changeInfo.url || changeInfo.status === "complete")) {
    cancelPageRequests();
    updateCurrentUrl();
  }
});

// ── Pending Parse (from context menu / shortcut) ───────

let pendingCheck = false;
let pendingRecheck = false;
async function checkPendingParse() {
  if (!JobberAPI.isLoggedIn()) return false;
  if (pendingCheck) { pendingRecheck = true; return false; }
  pendingCheck = true;
  let pending;
  const session = JobberAPI.sessionId();
  try { pending = await JobberAPI.takePendingParse(); }
  finally {
    pendingCheck = false;
    if (pendingRecheck) { pendingRecheck = false; void checkPendingParse(); }
  }
  if (!pending || !JobberAPI.isLoggedIn() || JobberAPI.sessionId() !== session) return false;
  if (pending.pageData?.text) await parseWithData(pending.pageData);
  else await parseCurrentPage();
  return true;
}

// ── Usage ──────────────────────────────────────────────

async function loadUsage() {
  const session = JobberAPI.sessionId();
  try {
    const sub = await JobberAPI.getSubscription();
    if (!sub || !JobberAPI.isLoggedIn() || session !== JobberAPI.sessionId()) return;

    const limits = sub.limits || {};
    const usage = sub.usage || {};
    const parts = [];

    if (limits.max_jobs > 0) {
      parts.push(`Jobs: ${usage.jobs ?? 0}/${limits.max_jobs}`);
    }
    if (sub.credits) {
      parts.push(`AI credits: ${sub.credits.remaining}/${sub.credits.granted}`);
    }


    if (parts.length > 0) {
      usageBar.textContent = parts.join("  \u2022  ");
      usageBar.classList.remove("hidden");
    }
  } catch {
    // Usage display is optional
  }
}

// ── Tabs ───────────────────────────────────────────────

tabButtons.forEach((btn) => {
  btn.addEventListener("click", () => {
    tabButtons.forEach((t) => t.classList.remove("active"));
    btn.classList.add("active");

    const name = btn.dataset.tab;
    cancelPageRequests();
    tabSave.classList.toggle("hidden", name !== "save");
    tabJobs.classList.toggle("hidden", name !== "jobs");
    tabAutofill.classList.toggle("hidden", name !== "autofill");

    if (name === "jobs") loadJobs();
    if (name === "autofill") loadAutofillTab();
  });
});

// ── Login ──────────────────────────────────────────────

loginForm.addEventListener("submit", async (e) => {
  e.preventDefault();
  hideError(loginError);
  btnLogin.disabled = true;
  btnLogin.textContent = "Signing in\u2026";

  try {
    await JobberAPI.login(loginEmail.value, loginPassword.value);
    clearPassword();
    showMainView();
    await checkPendingParse();
  } catch (err) {
    showError(loginError, err.message);
  } finally {
    btnLogin.disabled = false;
    btnLogin.textContent = "Sign In";
  }
});

btnLogout.addEventListener("click", async () => {
  resetPrivateView();
  await JobberAPI.logout();
  viewMain.classList.add("hidden");
  viewLogin.classList.remove("hidden");
});

// ── Parse ──────────────────────────────────────────────

function beginParse() {
  parseRequest?.abort();
  const request = new AbortController();
  parseRequest = request;
  return request;
}
function isCurrentParse(request) { return parseRequest === request && !request.signal.aborted; }
async function parseCurrentPage() {
  const request = beginParse();
  hideError(idleError);
  showSaveState("loading");

  try {
    const [tab] = await chrome.tabs.query({
      active: true,
      currentWindow: true,
    });
    if (!isCurrentParse(request)) return;
    if (!tab?.id) throw new Error("No active tab found");

    const pageData = await chrome.runtime.sendMessage({
      action: "extractPageText",
      tabId: tab.id,
    });

    if (!isCurrentParse(request)) return;
    if (pageData?.error) throw new Error(pageData.error);
    await parseWithData(pageData, request);
  } catch (err) {
    if (!isCurrentParse(request)) return;
    if (err.message === "SESSION_EXPIRED") {
      handleSessionExpired();
      return;
    }
    showSaveState("idle");
    if (err.message !== "AI_CANCELLED") showError(idleError, err.message);
  }
}

async function parseWithData(pageData, request = beginParse()) {
  showSaveState("loading");

  try {
    if (!pageData?.text || pageData.text.length < 10) {
      throw new Error("Not enough text on this page to parse");
    }

    const parsed = await JobberAPI.parseJob(pageData.text, pageData.url, { consentSignal: request.signal });
    if (!isCurrentParse(request)) return;

    previewTitle.value = parsed.title || "";
    previewCompany.value = parsed.company_name || "";
    previewSalary.value = "";
    previewLocation.value = "";
    previewSource.value = parsed.source || "";
    previewUrl.value = parsed.url || pageData.url;
    previewDescription.value = parsed.description || "";
    hideError(saveError);

    const isDuplicate = await JobberAPI.isUrlImported(previewUrl.value);
    if (!isCurrentParse(request)) return;
    duplicateWarning.classList.toggle("hidden", !isDuplicate);

    showSaveState("preview");
  } catch (err) {
    if (!isCurrentParse(request)) return;
    if (err.message === "SESSION_EXPIRED") {
      handleSessionExpired();
      return;
    }
    showSaveState("idle");
    if (err.message === "PARSE_LIMIT_REACHED") {
      showError(
        idleError,
        "Not enough AI credits. Check your balance at cursopus.com.",
      );
    } else {
      if (err.message !== "AI_CANCELLED") showError(idleError, err.message);
    }
  } finally {
    if (isCurrentParse(request)) void loadUsage();
  }
}

// The page can be any origin, but we only hold host access for cursopus.com (the API host) by
// default — so extraction rode the gesture-scoped `activeTab` grant and broke on
// the second parse (after "Save Another"). Request persistent host access to the
// current tab's origin first, inside the click gesture, so re-parsing works.
btnParse.addEventListener("click", async () => {
  try {
    const granted = await requestHostAccess();
    if (!granted) {
      showError(idleError, "Grant access to this page to parse it.");
      return;
    }
  } catch {
    // Non-web scheme (chrome://, file:…) or an origin Chrome refuses to grant.
    showError(idleError, "Cannot read this page. Try a job posting page.");
    return;
  }
  // A fresh grant makes the tab's URL visible — refresh the shown hostname.
  updateCurrentUrl();
  parseCurrentPage();
});

// Promise for host access to the active tab's origin. Must be the first call
// in the click handler to satisfy permissions.request()'s user-gesture
// requirement; an already-granted origin resolves true without a prompt.
// When the URL is hidden from us (a tab we hold no host access to yet), we
// cannot name its origin — fall back to requesting all-https access, the
// pattern optional_host_permissions declares. Rejects for pages that can never
// be parsed (chrome://, file:// and other non-web schemes).
function requestHostAccess() {
  let originPattern = "https://*/*";
  if (currentTabUrl) {
    let url = null;
    try {
      url = new URL(currentTabUrl);
    } catch {
      // Fall through — treat an unparseable URL as an unreadable page.
    }
    if (!url || (url.protocol !== "https:" && url.protocol !== "http:")) {
      return Promise.reject(new Error("Page scheme cannot be granted"));
    }
    originPattern = `${url.origin}/*`;
  }
  return chrome.permissions.request({ origins: [originPattern] });
}

// ── Save ───────────────────────────────────────────────

saveForm.addEventListener("submit", async (e) => {
  e.preventDefault();
  cancelPageRequests();
  const request = new AbortController();
  saveRequest = request;
  const originSessionId = JobberAPI.sessionId();
  const isCurrent = () => saveRequest === request && !request.signal.aborted && JobberAPI.isLoggedIn() && JobberAPI.sessionId() === originSessionId;
  hideError(saveError);
  btnSave.disabled = true;
  btnSave.textContent = "Saving\u2026";

  // Fold salary + location into the description. The job model no longer has a
  // separate notes field — comments are the place for free-form annotations.
  const salary = previewSalary.value.trim();
  const loc = previewLocation.value.trim();
  const meta = [];
  if (salary) meta.push(`Salary: ${salary}`);
  if (loc) meta.push(`Location: ${loc}`);
  const body = previewDescription.value.trim();
  const description =
    [meta.join("\n"), body].filter(Boolean).join("\n\n") || undefined;

  const submitted = { title: previewTitle.value.trim(), companyName: previewCompany.value.trim(),
    source: previewSource.value.trim(), url: previewUrl.value.trim(), description };
  try {
    const session = await JobberAPI.sessionSnapshot();
    if (!isCurrent() || session.authSessionId !== originSessionId) return;
    const context = { session, consentSignal: request.signal };
    let companyId;
    if (submitted.companyName) {
      const company = await JobberAPI.createCompany(submitted.companyName, context);
      if (!isCurrent()) return;
      if (company) companyId = company.id;
    }

    if (!isCurrent()) return;
    const job = await JobberAPI.createJob({ ...submitted, companyId }, context);
    if (!isCurrent()) return;
    await JobberAPI.trackImportedUrl(job.url, context);
    if (!isCurrent()) return;
    const count = await JobberAPI.incrementSavedCount(context);
    if (!isCurrent()) return;
    chrome.runtime.sendMessage({ action: "updateBadge", count });

    lastSavedJob = job;
    successJobTitle.textContent = job.title;
    resetSpMatchScore();
    loadSpMatchResumes();
    loadUsage();
    showSaveState("success");
  } catch (err) {
    if (!isCurrent()) return;
    if (err.message === "SESSION_EXPIRED") {
      handleSessionExpired();
      return;
    }
    if (err.message === "JOBS_LIMIT_REACHED") {
      showError(
        saveError,
        "Jobs limit reached for your plan. Upgrade at cursopus.com.",
      );
    } else if (err.message !== "AI_CANCELLED") {
      showError(saveError, err.message);
    }
  } finally {
    if (isCurrent()) {
      saveRequest = null;
      btnSave.disabled = false;
      btnSave.textContent = "Save to Cursopus";
    }
  }
});

btnBack.addEventListener("click", () => {
  cancelPageRequests();
  showSaveState("idle");
  updateCurrentUrl();
});

btnOpenInJobber.addEventListener("click", () => {
  if (lastSavedJob) {
    chrome.tabs.create({
      url: `${JobberAPI.getWebAppBase()}/app/jobs/${lastSavedJob.id}`,
    });
  }
});

btnSaveAnother.addEventListener("click", () => {
  cancelPageRequests();
  showSaveState("idle");
  updateCurrentUrl();
  loadUsage();
});

// ── Match Score ────────────────────────────────────────

function getScoreColor(score) {
  if (score >= 75) return "#22c55e";
  if (score >= 50) return "#f59e0b";
  return "#ef4444";
}

function resetSpMatchScore() {
  cancelSpMatchRequest();
  spMatchIdle.classList.remove("hidden");
  spMatchLoading.classList.add("hidden");
  spMatchResult.classList.add("hidden");
  hideError(spMatchError);
}

async function loadSpMatchResumes() {
  const session = JobberAPI.sessionId();
  const hint = document.getElementById("sp-match-hint");
  try {
    const data = await JobberAPI.listResumes(50);
    if (!JobberAPI.isLoggedIn() || session !== JobberAPI.sessionId()) return;
    const resumes = unwrapList(data);

    resetSpMatchScore();
    spMatchResumeSelect.innerHTML =
      '<option value="">Select resume for match\u2026</option>';
    resumes.forEach((r) => {
      const option = document.createElement("option");
      option.value = r.id;
      option.textContent = r.title || "Untitled Resume";
      if (r.is_active) option.selected = true;
      spMatchResumeSelect.appendChild(option);
    });

    if (resumes.length === 0) {
      hint.textContent =
        "Upload a PDF resume in Cursopus to use Match Score. Resume Builder resumes require PDF export.";
    } else {
      hint.textContent = "";
    }
  } catch (err) {
    if (!JobberAPI.isLoggedIn() || session !== JobberAPI.sessionId()) return;
    if (err.message === "SESSION_EXPIRED") handleSessionExpired();
  }
}

spMatchResumeSelect.addEventListener("change", resetSpMatchScore);

spBtnCheckMatch.addEventListener("click", async () => {
  const resumeId = spMatchResumeSelect.value;
  if (!resumeId) {
    showError(spMatchError, "Please select a resume");
    return;
  }
  if (!lastSavedJob) return;

  cancelSpMatchRequest();
  const request = new AbortController();
  matchRequest = request;
  const jobId = lastSavedJob.id;
  const isCurrent = () => matchRequest === request && !request.signal.aborted && lastSavedJob?.id === jobId && spMatchResumeSelect.value === resumeId;
  hideError(spMatchError);
  spMatchIdle.classList.add("hidden");
  spMatchLoading.classList.remove("hidden");

  try {
    const result = await JobberAPI.getMatchScore(jobId, resumeId, { consentSignal: request.signal });
    if (!isCurrent()) return;
    const score =
      typeof result.overall_score === "number"
        ? Math.round(result.overall_score)
        : 0;
    const color = getScoreColor(score);

    spMatchScore.textContent = score;
    spMatchRing.style.borderColor = color;
    spMatchRing.style.color = color;
    spMatchSummary.textContent = result.summary || "";

    // Categories
    spMatchCategories.textContent = "";
    if (result.categories?.length > 0) {
      result.categories.forEach((c) => {
        const safeScore = typeof c.score === "number" ? Math.round(c.score) : 0;
        const row = document.createElement("div");
        row.className = "match-cat";
        const nameSpan = document.createElement("span");
        nameSpan.className = "match-cat-name";
        nameSpan.textContent = c.name || "";
        const scoreSpan = document.createElement("span");
        scoreSpan.className = "match-cat-score";
        scoreSpan.style.color = getScoreColor(safeScore);
        scoreSpan.textContent = `${safeScore}%`;
        row.appendChild(nameSpan);
        row.appendChild(scoreSpan);
        spMatchCategories.appendChild(row);
      });
    } else {
      spMatchCategories.textContent = "";
    }

    // Strengths
    spMatchStrengths.textContent = "";
    if (result.strengths?.length > 0) {
      const label = document.createElement("strong");
      label.textContent = "Strengths: ";
      spMatchStrengths.appendChild(label);
      spMatchStrengths.appendChild(
        document.createTextNode(result.strengths.slice(0, 5).join(" \u2022 ")),
      );
    }

    // Missing
    spMatchMissing.textContent = "";
    if (result.missing_keywords?.length > 0) {
      const label = document.createElement("strong");
      label.textContent = "Missing: ";
      spMatchMissing.appendChild(label);
      spMatchMissing.appendChild(
        document.createTextNode(result.missing_keywords.slice(0, 8).join(", ")),
      );
    }

    spMatchLoading.classList.add("hidden");
    spMatchResult.classList.remove("hidden");
  } catch (err) {
    if (!isCurrent()) return;
    spMatchLoading.classList.add("hidden");
    spMatchIdle.classList.remove("hidden");

    if (err.message === "SESSION_EXPIRED") {
      handleSessionExpired();
      return;
    }
    if (err.message === "PLAN_LIMIT_REACHED") {
      showError(spMatchError, "AI usage limit reached. Upgrade your plan.");
      return;
    }
    if (err.message !== "AI_CANCELLED") showError(spMatchError, err.message);
  } finally {
    if (isCurrent()) {
      matchRequest = null;
      void loadUsage();
    }
  }
});

// ── My Jobs ────────────────────────────────────────────

let jobsRequest = 0;
async function loadJobs() {
  const request = ++jobsRequest;
  const session = JobberAPI.sessionId();
  const isCurrent = () => request === jobsRequest && JobberAPI.isLoggedIn() && session === JobberAPI.sessionId();
  jobsList.innerHTML = "";
  jobsEmpty.classList.add("hidden");
  hideError(jobsError);
  jobsLoading.classList.remove("hidden");

  try {
    const data = await JobberAPI.listJobs(20);
    if (!isCurrent()) return;
    const jobs = unwrapList(data);

    jobsLoading.classList.add("hidden");

    if (jobs.length === 0) {
      jobsEmpty.classList.remove("hidden");
      return;
    }

    const baseUrl = JobberAPI.getWebAppBase();
    jobs.forEach((job) => {
      const el = document.createElement("a");
      el.className = "job-item";
      el.href = `${baseUrl}/app/jobs/${encodeURIComponent(job.id)}`;
      el.target = "_blank";
      el.rel = "noopener";

      const title = document.createElement("div");
      title.className = "job-item-title";
      title.textContent = job.title || "";
      el.appendChild(title);

      const meta = document.createElement("div");
      meta.className = "job-item-meta";
      if (job.company_name) {
        const company = document.createElement("span");
        company.textContent = job.company_name;
        meta.appendChild(company);
      }
      if (job.source) {
        const source = document.createElement("span");
        source.className = "job-item-source";
        source.textContent = job.source;
        meta.appendChild(source);
      }
      el.appendChild(meta);

      jobsList.appendChild(el);
    });
  } catch (err) {
    if (!isCurrent()) return;
    jobsLoading.classList.add("hidden");
    if (err.message === "SESSION_EXPIRED") {
      handleSessionExpired();
      return;
    }
    if (err.message !== "AI_CANCELLED") showError(jobsError, err.message);
  }
}

// ── Start ──────────────────────────────────────────────

init();
