// ── Autofill ───────────────────────────────────────────

async function loadAutofillTab() {
  const session = JobberAPI.sessionId();
  // Refresh plan state on every visit — an upgrade made in the web app must
  // unlock the uploaded group without reopening the panel (a stale free
  // snapshot renders the options disabled, which the server can't overrule).
  await loadUsage();

  if (!JobberAPI.isLoggedIn() || session !== JobberAPI.sessionId()) return;
  const cached = await JobberAPI.getAutofillProfile();
  if (!JobberAPI.isLoggedIn() || session !== JobberAPI.sessionId()) return;

  // A profile is usable with a name OR an email — same threshold the backend
  // applies to extractions, so an email-only extracted profile stays active.
  if (cached?.contact?.full_name || cached?.contact?.email) {
    showActiveProfile(cached);
    return;
  }

  await showAutofillSetup();
}

function appendResumeGroup(label, prefix, resumes, enabled) {
  if (resumes.length === 0) return;
  const group = document.createElement("optgroup");
  group.label = label;
  resumes.forEach((r) => {
    const option = document.createElement("option");
    option.value = prefix + r.id;
    option.textContent = r.title || "Untitled Resume";
    option.disabled = !enabled;
    group.appendChild(option);
  });
  autofillResumeSelect.appendChild(group);
}

async function showAutofillSetup() {
  const session = JobberAPI.sessionId();
  autofillSetup.classList.remove("hidden");
  autofillActive.classList.add("hidden");
  hideError(autofillError);

  try {
    const [builderData, uploadedData] = await Promise.all([
      JobberAPI.listResumeBuilders(),
      JobberAPI.listResumes(50),
    ]);
    if (!JobberAPI.isLoggedIn() || session !== JobberAPI.sessionId()) return;
    const builders = unwrapList(builderData);
    const uploaded = unwrapList(uploadedData);
    const uploadedEnabled = true;

    autofillResumeSelect.innerHTML =
      '<option value="">Select a resume\u2026</option>';
    appendResumeGroup("Builder Resumes", "rb:", builders, true);
    appendResumeGroup("Uploaded Resumes", "up:", uploaded, uploadedEnabled);

    const isEmpty = builders.length === 0 && uploaded.length === 0;
    let hint = "";
    if (isEmpty) {
      hint =
        "Upload a resume or create one in the Resume Builder to use Autofill.";
    }
    autofillHint.textContent = hint;
    btnSetAutofill.disabled = isEmpty;
    btnOpenBuilder.classList.toggle("hidden", !isEmpty);
  } catch (err) {
    if (!JobberAPI.isLoggedIn() || session !== JobberAPI.sessionId()) return;
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

  // Extracted profiles can't be edited anywhere (accepted for v1) — bridge to
  // the editable path: the Resume Builder PDF import.
  autofillBridgeHint.textContent =
    data.source === "uploaded"
      ? "Wrong details? Import this PDF in the Resume Builder to edit."
      : "";

  const c = data.contact || {};
  profileNameEl.textContent = c.full_name || "No name";

  const details = [c.email, c.phone, c.location]
    .filter(Boolean)
    .join(" \u2022 ");
  profileInfoEl.textContent = details || "No contact info";
}

btnSetAutofill.addEventListener("click", async () => {
  const value = autofillResumeSelect.value;
  if (!value) {
    showError(autofillError, "Please select a resume");
    return;
  }

  cancelProfileSelection();
  const request = new AbortController();
  profileRequest = request;
  request.profileContext = JobberAPI.beginAutofillRequest();
  const session = JobberAPI.sessionId();
  const isCurrent = () => profileRequest === request && !request.signal.aborted && JobberAPI.isLoggedIn() && JobberAPI.sessionId() === session && autofillResumeSelect.value === value;
  hideError(autofillError);
  btnSetAutofill.disabled = true;
  const isUploaded = value.startsWith("up:");
  const resumeId = value.slice(3);
  // First extraction of an uploaded resume is an AI call (~5-15s); repeat
  // selections are instant server-side cache hits.
  btnSetAutofill.textContent = isUploaded
    ? "Parsing your resume\u2026"
    : "Loading\u2026";

  try {
    await request.profileContext.ready;
    if (!isCurrent()) return;
    const profile = isUploaded
      ? {
          ...(await JobberAPI.getUploadedAutofillProfile(resumeId, { consentSignal: request.signal })),
          source: "uploaded",
        }
      : await JobberAPI.getFullResume(resumeId);
    if (!isCurrent()) return;
    await JobberAPI.storeAutofillProfile(profile, request.profileContext);
    if (isCurrent()) showActiveProfile(profile);
  } catch (err) {
    if (!isCurrent()) return;
    if (err.message === "SESSION_EXPIRED") {
      handleSessionExpired();
      return;
    }
    const friendly = {
      PLAN_LIMIT_REACHED:
        "AI usage limit reached. Upgrade your plan at cursopus.com.",
      RESUME_UNREADABLE:
        "Couldn't read this PDF. Try the Resume Builder instead.",
      RATE_LIMITED: "Too many attempts — wait a minute and try again.",
    };
    if (err.message !== "AI_CANCELLED") showError(autofillError, friendly[err.message] || err.message);
  } finally {
    if (isCurrent()) {
      profileRequest = null;
      void loadUsage();
      btnSetAutofill.disabled = false;
      btnSetAutofill.textContent = "Set as Autofill Profile";
    }
  }
});

btnChangeProfile.addEventListener("click", () => showAutofillSetup());

btnOpenBuilder.addEventListener("click", () => {
  // /app/resume-builder is just a redirect to /app/resumes — link straight to
  // the resumes page, where "Create Resume" starts a builder resume.
  chrome.tabs.create({
    url: `${JobberAPI.getWebAppBase()}/app/resumes`,
  });
});

autofillResumeSelect.addEventListener("change", cancelProfileSelection);
