// Popup controller (uses shared/api.js)

// ── DOM Elements ───────────────────────────────────────

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
const previewSalary = document.getElementById("preview-salary");
const previewLocation = document.getElementById("preview-location");
const previewSource = document.getElementById("preview-source");
const previewUrl = document.getElementById("preview-url");
const previewNotes = document.getElementById("preview-notes");
const btnSave = document.getElementById("btn-save");
const btnBack = document.getElementById("btn-back");
const saveError = document.getElementById("save-error");
const duplicateWarning = document.getElementById("duplicate-warning");

const idleError = document.getElementById("idle-error");
const btnDone = document.getElementById("btn-done");
const btnOpenInJobber = document.getElementById("btn-open-in-jobber");
const btnSaveAnother = document.getElementById("btn-save-another");
const successJobTitle = document.getElementById("success-job-title");

// Match score
const matchResumeSelect = document.getElementById("match-resume");
const btnCheckMatch = document.getElementById("btn-check-match");
const matchError = document.getElementById("match-error");
const matchIdle = document.getElementById("match-idle");
const matchLoading = document.getElementById("match-loading");
const matchResult = document.getElementById("match-result");
const matchScoreValue = document.getElementById("match-score-value");
const matchRing = document.getElementById("match-ring");
const matchSummary = document.getElementById("match-summary");
const matchMissing = document.getElementById("match-missing");

let lastSavedJob = null;

// ── Helpers ────────────────────────────────────────────

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

function handleSessionExpired() {
  JobberAPI.logout();
  showView("login");
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
    await showIdleView();
  } else {
    showView("login");
  }
}

async function showIdleView() {
  const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });
  if (tab?.url) {
    try {
      currentUrl.textContent = new URL(tab.url).hostname;
    } catch {
      currentUrl.textContent = tab.url;
    }
  }
  showView("idle");
}

// ── Login ──────────────────────────────────────────────

loginForm.addEventListener("submit", async (e) => {
  e.preventDefault();
  hideError(loginError);
  btnLogin.disabled = true;
  btnLogin.textContent = "Signing in\u2026";

  try {
    await JobberAPI.login(loginEmail.value, loginPassword.value);
    await showIdleView();
  } catch (err) {
    showError(loginError, err.message);
  } finally {
    btnLogin.disabled = false;
    btnLogin.textContent = "Sign In";
  }
});

// ── Parse ──────────────────────────────────────────────

btnParse.addEventListener("click", async () => {
  hideError(idleError);
  showView("loading");

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

    if (pageData?.error) {
      throw new Error(
        "Cannot read this page. Try on a job posting page (LinkedIn, Indeed, etc.)",
      );
    }

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

    showView("preview");
  } catch (err) {
    if (err.message === "SESSION_EXPIRED") {
      handleSessionExpired();
      return;
    }
    await showIdleView();
    if (err.message === "PARSE_LIMIT_REACHED") {
      showError(
        idleError,
        "Monthly AI parse limit reached. Upgrade your plan at jobber-app.com.",
      );
    } else {
      showError(idleError, err.message);
    }
  }
});

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

    // Update badge
    const count = await JobberAPI.incrementSavedCount();
    chrome.runtime.sendMessage({ action: "updateBadge", count });

    successJobTitle.textContent = job.title;
    resetMatchScore();
    loadMatchResumes();
    showView("success");
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

// ── Actions ────────────────────────────────────────────

btnBack.addEventListener("click", () => showIdleView());

btnOpenInJobber.addEventListener("click", () => {
  if (lastSavedJob) {
    chrome.tabs.create({
      url: `${JobberAPI.getWebAppBase()}/app/jobs/${lastSavedJob.id}`,
    });
  }
});

btnSaveAnother.addEventListener("click", () => showIdleView());
btnDone.addEventListener("click", () => window.close());

btnLogout.addEventListener("click", async () => {
  await JobberAPI.logout();
  showView("login");
});

// ── Match Score ────────────────────────────────────────

function getScoreColor(score) {
  if (score >= 75) return "#22c55e";
  if (score >= 50) return "#f59e0b";
  return "#ef4444";
}

function resetMatchScore() {
  matchIdle.classList.remove("hidden");
  matchLoading.classList.add("hidden");
  matchResult.classList.add("hidden");
  hideError(matchError);
}

async function loadMatchResumes() {
  const matchHint = document.getElementById("match-hint");
  try {
    const data = await JobberAPI.listResumes(50);
    const resumes = Array.isArray(data)
      ? data
      : data.items || data.resumes || data.data || [];

    matchResumeSelect.innerHTML =
      '<option value="">Select resume for match\u2026</option>';
    resumes.forEach((r) => {
      const option = document.createElement("option");
      option.value = r.id;
      option.textContent = r.title || "Untitled Resume";
      if (r.is_active) option.selected = true;
      matchResumeSelect.appendChild(option);
    });

    if (resumes.length === 0) {
      matchHint.textContent =
        "Upload a PDF resume in Jobber to use Match Score. Resume Builder resumes require PDF export.";
    } else {
      matchHint.textContent = "";
    }
  } catch (err) {
    if (err.message === "SESSION_EXPIRED") handleSessionExpired();
  }
}

btnCheckMatch.addEventListener("click", async () => {
  const resumeId = matchResumeSelect.value;
  if (!resumeId) {
    showError(matchError, "Please select a resume");
    return;
  }
  if (!lastSavedJob) return;

  hideError(matchError);
  matchIdle.classList.add("hidden");
  matchLoading.classList.remove("hidden");

  try {
    const result = await JobberAPI.getMatchScore(lastSavedJob.id, resumeId);
    const score =
      typeof result.overall_score === "number"
        ? Math.round(result.overall_score)
        : 0;
    const color = getScoreColor(score);

    matchScoreValue.textContent = score;
    matchRing.style.borderColor = color;
    matchRing.style.color = color;
    matchSummary.textContent = result.summary || "";

    matchMissing.textContent = "";
    if (result.missing_keywords?.length > 0) {
      const label = document.createElement("strong");
      label.textContent = "Missing: ";
      matchMissing.appendChild(label);
      matchMissing.appendChild(
        document.createTextNode(result.missing_keywords.slice(0, 8).join(", ")),
      );
    }

    matchLoading.classList.add("hidden");
    matchResult.classList.remove("hidden");
  } catch (err) {
    matchLoading.classList.add("hidden");
    matchIdle.classList.remove("hidden");

    if (err.message === "SESSION_EXPIRED") {
      handleSessionExpired();
      return;
    }
    if (err.message === "PLAN_LIMIT_REACHED") {
      showError(matchError, "AI usage limit reached. Upgrade your plan.");
      return;
    }
    showError(matchError, err.message);
  }
});

// ── Start ──────────────────────────────────────────────

init();
