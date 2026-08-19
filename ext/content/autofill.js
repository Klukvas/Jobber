// content/autofill.js — Autofill job application forms with Jobber profile data
(() => {
  if (window.__jobberAutofillInit) return;
  window.__jobberAutofillInit = true;

  let profile = null;

  // Autofill trigger from the background (right-click context menu). Registered
  // synchronously — not inside the async init — and over runtime messaging (not
  // a page-dispatchable window event) so a page script can't trigger autofill of
  // the user's PII.
  chrome.runtime.onMessage.addListener((message) => {
    if (message?.action === "jobber-autofill") runAutofillFromTrigger();
  });

  async function runAutofillFromTrigger() {
    if (!profile) {
      const data = await chrome.storage.local.get("autofillProfile");
      profile = data.autofillProfile || null;
    }
    if (!profile?.contact) return;

    const { filled } = autofillPage();
    createButton();

    const btn = document.getElementById("jobber-autofill-btn");
    if (!btn) return;
    btn.textContent =
      filled > 0 ? `✓ Filled ${filled} fields` : "No matching fields";
    if (filled > 0) btn.classList.add("jobber-autofill-success");
    setTimeout(() => resetButton(btn), 3000);
  }

  // ── Field Detection Patterns ─────────────────────────

  const PATTERNS = {
    firstName: /first[\s_-]*name|given[\s_-]*name|fname|prénom/i,
    lastName: /last[\s_-]*name|family[\s_-]*name|surname|lname/i,
    // The bare "name" alternative excludes "<word> name" (Company Name, User
    // Name, …) via the lookbehind, so it only matches a standalone Name field.
    fullName:
      /full[\s_-]*name|your[\s_-]*name|applicant[\s_-]*name|legal[\s_-]*name|(?<![a-z][\s_-])\bname\b/i,
    email: /email|e[\s_-]*mail/i,
    phone: /phone|tel(?:ephone)?|mobile|cell/i,
    location: /city|location|address(?!.*email)/i,
    linkedin: /linkedin/i,
    github: /github/i,
    website: /website|portfolio|personal[\s_-]*(?:url|site|page)/i,
    currentCompany:
      /current[\s_-]*(?:company|employer|organization)|most[\s_-]*recent[\s_-]*(?:company|employer)/i,
    currentTitle:
      /current[\s_-]*(?:title|position|role)|most[\s_-]*recent[\s_-]*(?:title|position)/i,
    summary:
      /cover[\s_-]*letter|about[\s_-]*(?:you|yourself)|additional[\s_-]*info/i,
    school: /school|university|college|institution/i,
    degree: /degree|qualification/i,
    fieldOfStudy: /field[\s_-]*of[\s_-]*study|major|concentration/i,
  };

  const AUTOCOMPLETE_MAP = {
    "given-name": "firstName",
    "family-name": "lastName",
    name: "fullName",
    email: "email",
    tel: "phone",
    "address-level2": "location",
    url: "website",
    organization: "currentCompany",
    "organization-title": "currentTitle",
  };

  // ── Field Identification ─────────────────────────────

  function getFieldIdentifiers(field) {
    const ids = [];

    if (field.name) ids.push(field.name);
    if (field.id) ids.push(field.id);
    if (field.placeholder) ids.push(field.placeholder);

    const ariaLabel = field.getAttribute("aria-label");
    if (ariaLabel) ids.push(ariaLabel);

    // Direct <label>
    const directLabel =
      field.closest("label") ||
      (field.id &&
        document.querySelector(`label[for="${CSS.escape(field.id)}"]`));
    if (directLabel) ids.push(directLabel.textContent.trim());

    // Parent container label (common in ATS form builders)
    const parent = field.closest(
      ".field, .form-group, .form-field, [class*='field'], [class*='Field']",
    );
    if (parent) {
      const labelEl = parent.querySelector(
        "label, .label, [class*='label'], [class*='Label']",
      );
      if (labelEl && labelEl !== directLabel) {
        ids.push(labelEl.textContent.trim());
      }
    }

    return ids.filter(Boolean);
  }

  function identifyField(field) {
    // Check autocomplete attribute first
    const autocomplete = field.getAttribute("autocomplete");
    if (autocomplete && AUTOCOMPLETE_MAP[autocomplete]) {
      return AUTOCOMPLETE_MAP[autocomplete];
    }

    const combined = getFieldIdentifiers(field).join(" ");
    for (const [type, pattern] of Object.entries(PATTERNS)) {
      if (pattern.test(combined)) return type;
    }

    return null;
  }

  // ── Value Resolution ─────────────────────────────────

  function getValueForField(type) {
    const c = profile?.contact || {};
    const exp = profile?.experiences?.[0];
    const edu = profile?.educations?.[0];

    const map = {
      firstName: c.full_name?.split(" ")[0],
      lastName: c.full_name?.split(" ").slice(1).join(" "),
      fullName: c.full_name,
      email: c.email,
      phone: c.phone,
      location: c.location,
      linkedin: c.linkedin,
      github: c.github,
      website: c.website,
      currentCompany: exp?.company,
      currentTitle: exp?.position,
      summary: profile?.summary?.content,
      school: edu?.institution,
      degree: edu?.degree,
      fieldOfStudy: edu?.field_of_study,
    };

    return map[type] || null;
  }

  // ── Form Filling ─────────────────────────────────────

  function fillField(field, value) {
    if (!value || field.disabled || field.readOnly) return false;
    if (field.value && field.value.trim().length > 0) return false;

    if (field.tagName === "SELECT") return fillSelect(field, value);
    if (field.tagName === "TEXTAREA" || field.tagName === "INPUT") {
      return fillInput(field, value);
    }

    return false;
  }

  function fillInput(input, value) {
    // Native setter bypasses React controlled input restrictions
    const proto =
      input.tagName === "TEXTAREA"
        ? HTMLTextAreaElement.prototype
        : HTMLInputElement.prototype;
    const descriptor = Object.getOwnPropertyDescriptor(proto, "value");

    if (descriptor?.set) {
      descriptor.set.call(input, value);
    } else {
      input.value = value;
    }

    // Use InputEvent for React 17+ compatibility (Workday, Greenhouse)
    input.dispatchEvent(new Event("focus", { bubbles: true }));
    input.dispatchEvent(
      new InputEvent("input", {
        bubbles: true,
        data: value,
        inputType: "insertText",
      }),
    );
    input.dispatchEvent(new Event("change", { bubbles: true }));
    input.dispatchEvent(new Event("blur", { bubbles: true }));

    return true;
  }

  function fillSelect(select, value) {
    const lower = value.toLowerCase();
    const opts = Array.from(select.options);
    const optText = (o) => o.textContent.trim().toLowerCase();
    // Prefer exact match, then prefix, then substring — so "US" doesn't land on
    // "Australia".
    const option =
      opts.find(
        (o) => o.value.toLowerCase() === lower || optText(o) === lower,
      ) ||
      opts.find(
        (o) =>
          o.value.toLowerCase().startsWith(lower) ||
          optText(o).startsWith(lower),
      ) ||
      opts.find(
        (o) =>
          o.value.toLowerCase().includes(lower) || optText(o).includes(lower),
      );

    if (option) {
      select.value = option.value;
      select.dispatchEvent(new Event("change", { bubbles: true }));
      return true;
    }

    return false;
  }

  // ── Main Autofill ────────────────────────────────────

  function autofillPage() {
    if (!profile) return { filled: 0, total: 0 };

    const selector = [
      'input:not([type="hidden"]):not([type="submit"]):not([type="button"])',
      ':not([type="checkbox"]):not([type="radio"]):not([type="file"])',
      ", textarea, select",
    ].join("");

    const fields = document.querySelectorAll(selector);
    let filled = 0;
    let total = 0;

    fields.forEach((field) => {
      if (!isVisible(field)) return;

      const type = identifyField(field);
      if (!type) return;

      total++;
      const value = getValueForField(type);
      if (value && fillField(field, value)) {
        filled++;
        field.classList.add("jobber-autofilled");
      }
    });

    return { filled, total };
  }

  // ── Floating Button ──────────────────────────────────

  function createButton() {
    if (document.getElementById("jobber-autofill-btn")) return;

    const btn = document.createElement("button");
    btn.id = "jobber-autofill-btn";
    btn.textContent = "\uD83D\uDCBC Autofill";
    btn.title = "Autofill with Jobber profile";

    btn.addEventListener("click", async () => {
      if (!profile) {
        const data = await chrome.storage.local.get("autofillProfile");
        profile = data.autofillProfile;
      }

      if (!profile?.contact) {
        btn.textContent = "\u26A0\uFE0F No profile";
        btn.title = "Open Jobber side panel → Autofill tab to set up";
        setTimeout(() => resetButton(btn), 3000);
        return;
      }

      const { filled } = autofillPage();

      if (filled > 0) {
        btn.textContent = `\u2713 Filled ${filled} fields`;
        btn.classList.add("jobber-autofill-success");
      } else {
        btn.textContent = "No matching fields";
      }

      setTimeout(() => resetButton(btn), 3000);
    });

    document.body.appendChild(btn);
  }

  function resetButton(btn) {
    btn.textContent = "\uD83D\uDCBC Autofill";
    btn.title = "Autofill with Jobber profile";
    btn.classList.remove("jobber-autofill-success");
  }

  // ── Form Detection ───────────────────────────────────

  function isVisible(el) {
    return (
      el.offsetParent !== null ||
      el.closest(
        '[role="dialog"], [style*="position: fixed"], [style*="position:fixed"]',
      ) !== null
    );
  }

  function hasApplicationForm() {
    const inputs = document.querySelectorAll(
      'input[type="text"], input[type="email"], input[type="tel"], input:not([type]), textarea',
    );

    let visible = 0;
    inputs.forEach((el) => {
      if (isVisible(el)) visible++;
    });

    return visible >= 2;
  }

  // ── Init ─────────────────────────────────────────────

  async function init() {
    const data = await chrome.storage.local.get("autofillProfile");
    profile = data.autofillProfile || null;

    // Delay to let dynamic forms render
    setTimeout(() => {
      if (hasApplicationForm()) createButton();
    }, 1500);

    // Watch for dynamically added forms
    let checkTimeout = null;
    const observer = new MutationObserver(() => {
      if (document.getElementById("jobber-autofill-btn")) return;
      if (checkTimeout) return;
      checkTimeout = setTimeout(() => {
        checkTimeout = null;
        if (hasApplicationForm()) createButton();
      }, 1000);
    });
    observer.observe(document.body, { childList: true, subtree: true });

    // Listen for profile updates from side panel
    chrome.storage.onChanged.addListener((changes, area) => {
      if (area === "local" && changes.autofillProfile) {
        profile = changes.autofillProfile.newValue || null;
      }
    });
  }

  init();
})();
