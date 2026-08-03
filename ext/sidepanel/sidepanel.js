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
const previewNotes = document.getElementById("preview-notes");
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
const btnSetAutofill = document.getElementById("btn-set-autofill");
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

// ── Helpers ────────────────────────────────────────────

function showError(el, msg) {
  el.textContent = msg;
  el.classList.remove("hidden");
}

function hideError(el) {
  el.classList.add("hidden");
}

function showSaveState(state) {
  [saveIdle, saveLoading, savePreview, saveSuccess].forEach((el) =>
    el.classList.add("hidden"),
  );
  document.getElementById(`save-${state}`).classList.remove("hidden");
}

function escapeHtml(str) {
  const div = document.createElement("div");
  div.textContent = str;
  return div.innerHTML;
}

function handleSessionExpired() {
  JobberAPI.logout();
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
  const isLoggedIn = await JobberAPI.init();
  if (isLoggedIn) {
    showMainView();
    const consumed = await checkPendingParse();
    if (!consumed) {
      // Panel opened before background stored data — listen for it
      listenForPendingParse();
    }
  } else {
    viewLogin.classList.remove("hidden");
    viewMain.classList.add("hidden");
  }
}

function listenForPendingParse() {
  const handler = async (changes, area) => {
    if (area === "session" && changes.pendingParse?.newValue) {
      chrome.storage.onChanged.removeListener(handler);
      await checkPendingParse();
    }
  };
  chrome.storage.onChanged.addListener(handler);
}

function showMainView() {
  viewLogin.classList.add("hidden");
  viewMain.classList.remove("hidden");
  updateCurrentUrl();
  loadUsage();
}

async function updateCurrentUrl() {
  const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });
  if (tab?.url) {
    try {
      currentUrlEl.textContent = new URL(tab.url).hostname;
    } catch {
      currentUrlEl.textContent = tab.url;
    }
  }
}

// ── Pending Parse (from context menu / shortcut) ───────

async function checkPendingParse() {
  const { pendingParse } = await chrome.storage.session.get("pendingParse");
  if (!pendingParse || Date.now() - pendingParse.timestamp > 10000) {
    return false;
  }

  await chrome.storage.session.remove("pendingParse");

  if (pendingParse.pageData?.text) {
    await parseWithData(pendingParse.pageData);
  } else {
    await parseCurrentPage();
  }
  return true;
}

// ── Usage ──────────────────────────────────────────────

async function loadUsage() {
  try {
    const sub = await JobberAPI.getSubscription();
    if (!sub) return;

    const limits = sub.limits || {};
    const usage = sub.usage || {};
    const parts = [];

    if (limits.max_jobs > 0) {
      parts.push(`Jobs: ${usage.jobs ?? 0}/${limits.max_jobs}`);
    }
    if (limits.max_job_parses > 0) {
      parts.push(
        `Parses: ${usage.job_parses ?? 0}/${limits.max_job_parses}/mo`,
      );
    }
    if (limits.max_ai_requests > 0) {
      parts.push(`AI: ${usage.ai_requests ?? 0}/${limits.max_ai_requests}/mo`);
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
    showMainView();
  } catch (err) {
    showError(loginError, err.message);
  } finally {
    btnLogin.disabled = false;
    btnLogin.textContent = "Sign In";
  }
});

btnLogout.addEventListener("click", async () => {
  await JobberAPI.logout();
  viewMain.classList.add("hidden");
  viewLogin.classList.remove("hidden");
});

// ── Parse ──────────────────────────────────────────────

async function parseCurrentPage() {
  hideError(idleError);
  showSaveState("loading");

  try {
    const [tab] = await chrome.tabs.query({
      active: true,
      currentWindow: true,
    });
    if (!tab?.id) throw new Error("No active tab found");

    const pageData = await chrome.runtime.sendMessage({
      action: "extractPageText",
      tabId: tab.id,
    });

    if (pageData?.error) throw new Error(pageData.error);
    await parseWithData(pageData);
  } catch (err) {
    if (err.message === "SESSION_EXPIRED") {
      handleSessionExpired();
      return;
    }
    showSaveState("idle");
    showError(idleError, err.message);
  }
}

async function parseWithData(pageData) {
  showSaveState("loading");

  try {
    if (!pageData?.text || pageData.text.length < 10) {
      throw new Error("Not enough text on this page to parse");
    }

    const parsed = await JobberAPI.parseJob(pageData.text, pageData.url);

    previewTitle.value = parsed.title || "";
    previewCompany.value = parsed.company_name || "";
    previewSalary.value = "";
    previewLocation.value = "";
    previewSource.value = parsed.source || "";
    previewUrl.value = parsed.url || pageData.url;
    previewNotes.value = parsed.description || "";
    hideError(saveError);

    const isDuplicate = await JobberAPI.isUrlImported(previewUrl.value);
    duplicateWarning.classList.toggle("hidden", !isDuplicate);

    showSaveState("preview");
  } catch (err) {
    if (err.message === "SESSION_EXPIRED") {
      handleSessionExpired();
      return;
    }
    showSaveState("idle");
    if (err.message === "PARSE_LIMIT_REACHED") {
      showError(
        idleError,
        "Monthly AI parse limit reached. Upgrade your plan at jobber-app.com.",
      );
    } else {
      showError(idleError, err.message);
    }
  }
}

btnParse.addEventListener("click", parseCurrentPage);

// ── Save ───────────────────────────────────────────────

saveForm.addEventListener("submit", async (e) => {
  e.preventDefault();
  hideError(saveError);
  btnSave.disabled = true;
  btnSave.textContent = "Saving\u2026";

  try {
    let companyId;
    const companyName = previewCompany.value.trim();
    if (companyName) {
      const company = await JobberAPI.createCompany(companyName);
      if (company) companyId = company.id;
    }

    // Build notes from salary + location
    const salary = previewSalary.value.trim();
    const loc = previewLocation.value.trim();
    const noteParts = [];
    if (salary) noteParts.push(`Salary: ${salary}`);
    if (loc) noteParts.push(`Location: ${loc}`);
    const notes = noteParts.length > 0 ? noteParts.join("\n") : undefined;

    const job = await JobberAPI.createJob({
      title: previewTitle.value.trim(),
      companyId,
      source: previewSource.value.trim(),
      url: previewUrl.value.trim(),
      notes,
      description: previewNotes.value.trim(),
    });

    lastSavedJob = job;
    await JobberAPI.trackImportedUrl(job.url);

    const count = await JobberAPI.incrementSavedCount();
    chrome.runtime.sendMessage({ action: "updateBadge", count });

    successJobTitle.textContent = job.title;
    resetSpMatchScore();
    loadSpMatchResumes();
    loadUsage();
    showSaveState("success");
  } catch (err) {
    if (err.message === "SESSION_EXPIRED") {
      handleSessionExpired();
      return;
    }
    if (err.message === "JOBS_LIMIT_REACHED") {
      showError(
        saveError,
        "Jobs limit reached for your plan. Upgrade at jobber-app.com.",
      );
    } else {
      showError(saveError, err.message);
    }
  } finally {
    btnSave.disabled = false;
    btnSave.textContent = "Save to Jobber";
  }
});

btnBack.addEventListener("click", () => {
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
  spMatchIdle.classList.remove("hidden");
  spMatchLoading.classList.add("hidden");
  spMatchResult.classList.add("hidden");
  hideError(spMatchError);
}

async function loadSpMatchResumes() {
  const hint = document.getElementById("sp-match-hint");
  try {
    const data = await JobberAPI.listResumes(50);
    const resumes = Array.isArray(data)
      ? data
      : data.items || data.resumes || data.data || [];

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
        "Upload a PDF resume in Jobber to use Match Score. Resume Builder resumes require PDF export.";
    } else {
      hint.textContent = "";
    }
  } catch (err) {
    if (err.message === "SESSION_EXPIRED") handleSessionExpired();
  }
}

spBtnCheckMatch.addEventListener("click", async () => {
  const resumeId = spMatchResumeSelect.value;
  if (!resumeId) {
    showError(spMatchError, "Please select a resume");
    return;
  }
  if (!lastSavedJob) return;

  hideError(spMatchError);
  spMatchIdle.classList.add("hidden");
  spMatchLoading.classList.remove("hidden");

  try {
    const result = await JobberAPI.getMatchScore(lastSavedJob.id, resumeId);
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
    showError(spMatchError, err.message);
  }
});

// ── My Jobs ────────────────────────────────────────────

async function loadJobs() {
  jobsList.innerHTML = "";
  jobsEmpty.classList.add("hidden");
  hideError(jobsError);
  jobsLoading.classList.remove("hidden");

  try {
    const data = await JobberAPI.listJobs(20);
    const jobs = Array.isArray(data)
      ? data
      : data.items || data.jobs || data.data || [];

    jobsLoading.classList.add("hidden");

    if (jobs.length === 0) {
      jobsEmpty.classList.remove("hidden");
      return;
    }

    const baseUrl = JobberAPI.getWebAppBase();
    jobs.forEach((job) => {
      const el = document.createElement("a");
      el.className = "job-item";
      el.href = `${baseUrl}/app/jobs/${job.id}`;
      el.target = "_blank";
      el.rel = "noopener";
      el.innerHTML = `
        <div class="job-item-title">${escapeHtml(job.title)}</div>
        <div class="job-item-meta">
          ${job.company_name ? `<span>${escapeHtml(job.company_name)}</span>` : ""}
          ${job.source ? `<span class="job-item-source">${escapeHtml(job.source)}</span>` : ""}
        </div>
      `;
      jobsList.appendChild(el);
    });
  } catch (err) {
    jobsLoading.classList.add("hidden");
    if (err.message === "SESSION_EXPIRED") {
      handleSessionExpired();
      return;
    }
    showError(jobsError, err.message);
  }
}

// ── Autofill ───────────────────────────────────────────

async function loadAutofillTab() {
  const { autofillProfile: cached } =
    await chrome.storage.local.get("autofillProfile");

  if (cached?.contact?.full_name) {
    showActiveProfile(cached);
    return;
  }

  await showAutofillSetup();
}

async function showAutofillSetup() {
  autofillSetup.classList.remove("hidden");
  autofillActive.classList.add("hidden");
  hideError(autofillError);

  try {
    const data = await JobberAPI.listResumeBuilders();
    const resumes = Array.isArray(data)
      ? data
      : data.items || data.resumes || data.data || [];

    autofillResumeSelect.innerHTML =
      '<option value="">Select a resume\u2026</option>';
    resumes.forEach((r) => {
      const option = document.createElement("option");
      option.value = r.id;
      option.textContent = r.title || "Untitled Resume";
      autofillResumeSelect.appendChild(option);
    });
  } catch (err) {
    if (err.message === "SESSION_EXPIRED") {
      handleSessionExpired();
      return;
    }
    showError(autofillError, err.message);
  }
}

function showActiveProfile(data) {
  autofillSetup.classList.add("hidden");
  autofillActive.classList.remove("hidden");

  const c = data.contact || {};
  profileNameEl.textContent = c.full_name || "No name";

  const details = [c.email, c.phone, c.location]
    .filter(Boolean)
    .join(" \u2022 ");
  profileInfoEl.textContent = details || "No contact info";
}

btnSetAutofill.addEventListener("click", async () => {
  const resumeId = autofillResumeSelect.value;
  if (!resumeId) {
    showError(autofillError, "Please select a resume");
    return;
  }

  hideError(autofillError);
  btnSetAutofill.disabled = true;
  btnSetAutofill.textContent = "Loading\u2026";

  try {
    const fullResume = await JobberAPI.getFullResume(resumeId);
    await chrome.storage.local.set({ autofillProfile: fullResume });
    showActiveProfile(fullResume);
  } catch (err) {
    if (err.message === "SESSION_EXPIRED") {
      handleSessionExpired();
      return;
    }
    showError(autofillError, err.message);
  } finally {
    btnSetAutofill.disabled = false;
    btnSetAutofill.textContent = "Set as Autofill Profile";
  }
});

btnChangeProfile.addEventListener("click", () => showAutofillSetup());

// ── Start ──────────────────────────────────────────────

init();
