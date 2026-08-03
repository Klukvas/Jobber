// Content script: adds "Save to Jobber" buttons on job board listing pages
(() => {
  const BOARD_CONFIGS = {
    "linkedin.com": {
      domains: ["linkedin.com"],
      name: "LinkedIn",
      cardSelector:
        ".job-card-container, .jobs-search-results__list-item, [data-job-id], .scaffold-layout__list-item, li.jobs-search-results__list-item, .job-card-list",
      titleSelector:
        ".job-card-list__title, .artdeco-entity-lockup__title a, a[href*='/jobs/view/'], [class*='job-card'] a[href*='/jobs/'], .job-card-container__link",
      companySelector:
        ".job-card-container__primary-description, .artdeco-entity-lockup__subtitle span, [class*='job-card'] [class*='company'], .job-card-container__company-name",
      linkSelector: "a[href*='/jobs/view/'], a[href*='/jobs/collections/']",
      getJobUrl(link) {
        const href = link?.getAttribute("href");
        if (!href) return null;
        try {
          const url = new URL(href, location.origin);
          url.search = "";
          return url.href;
        } catch {
          return null;
        }
      },
    },
    "indeed.com": {
      domains: ["indeed.com"],
      name: "Indeed",
      cardSelector: ".job_seen_beacon, .resultContent, [data-jk]",
      titleSelector: ".jobTitle a span, h2.jobTitle a, .jcs-JobTitle span",
      companySelector:
        "[data-testid='company-name'], .companyName span, .company_location .companyName",
      linkSelector: ".jobTitle a, a[data-jk], .jcs-JobTitle a",
      getJobUrl(link) {
        const href = link?.getAttribute("href");
        if (!href) return null;
        try {
          return new URL(href, location.origin).href;
        } catch {
          return null;
        }
      },
    },
    "glassdoor.com": {
      domains: ["glassdoor.com", "glassdoor.co.uk"],
      name: "Glassdoor",
      cardSelector: "[data-test='jobListing'], .react-job-listing, .jobCard",
      titleSelector: "a[data-test='job-link'], .job-title a, .jobCard a",
      companySelector:
        ".EmployerProfile_compactEmployerName__LE242, .employer-name, [data-test='employer-short-name']",
      linkSelector: "a[data-test='job-link'], .job-title a",
      getJobUrl(link) {
        const href = link?.getAttribute("href");
        if (!href) return null;
        try {
          return new URL(href, location.origin).href;
        } catch {
          return null;
        }
      },
    },
  };

  // Detect current board
  const hostname = location.hostname.replace("www.", "");
  let config = null;
  for (const [, cfg] of Object.entries(BOARD_CONFIGS)) {
    const domains = cfg.domains || [];
    if (domains.some((d) => hostname.includes(d))) {
      config = cfg;
      break;
    }
  }
  if (!config) return;

  let importedUrls = [];

  async function loadImportedUrls() {
    const data = await chrome.storage.local.get("importedUrls");
    importedUrls = data.importedUrls || [];
  }

  // Find first matching element from comma-separated selectors
  function querySelector(parent, selectors) {
    for (const sel of selectors.split(",")) {
      try {
        const el = parent.querySelector(sel.trim());
        if (el) return el;
      } catch {
        // Invalid selector — skip
      }
    }
    return null;
  }

  function getTextContent(el) {
    return el?.textContent?.trim() || "";
  }

  function showToast(message) {
    // Remove existing toast
    const existing = document.getElementById("jobber-toast");
    if (existing) existing.remove();

    const toast = document.createElement("div");
    toast.id = "jobber-toast";
    toast.className = "jobber-toast";

    const icon = document.createElement("span");
    icon.className = "jobber-toast-icon";
    icon.textContent = "\uD83D\uDCBC";

    const text = document.createElement("span");
    text.className = "jobber-toast-text";
    text.textContent = message;

    const close = document.createElement("button");
    close.className = "jobber-toast-close";
    close.textContent = "\u2715";
    close.addEventListener("click", () => toast.remove());

    toast.appendChild(icon);
    toast.appendChild(text);
    toast.appendChild(close);
    document.body.appendChild(toast);

    // Auto-dismiss after 6s
    setTimeout(() => toast.remove(), 6000);
  }

  function processCards() {
    let cards;
    try {
      cards = document.querySelectorAll(config.cardSelector);
    } catch {
      return;
    }

    cards.forEach((card) => {
      if (card.dataset.jobberProcessed) return;
      card.dataset.jobberProcessed = "true";

      const linkEl = querySelector(card, config.linkSelector);
      const titleEl = querySelector(card, config.titleSelector);
      const companyEl = querySelector(card, config.companySelector);

      const url = config.getJobUrl(linkEl);
      const title = getTextContent(titleEl);
      const company = getTextContent(companyEl);

      if (!title) return;

      const isSaved = url && importedUrls.includes(url);

      const btn = document.createElement("button");
      btn.className = `jobber-save-btn${isSaved ? " jobber-saved" : ""}`;
      btn.textContent = isSaved ? "\u2713 Saved" : "\uD83D\uDCBC Save";
      btn.title = isSaved ? "Already saved to Jobber" : "Save to Jobber";

      if (!isSaved) {
        btn.addEventListener("click", async (e) => {
          e.preventDefault();
          e.stopPropagation();

          btn.disabled = true;
          btn.textContent = "Saving\u2026";

          const result = await chrome.runtime.sendMessage({
            action: "quickSave",
            title,
            company,
            url,
            source: config.name,
          });

          if (result?.ok) {
            btn.className = "jobber-save-btn jobber-saved";
            btn.textContent = "\u2713 Saved";
            btn.title = "Saved to Jobber";
            if (url) importedUrls = [...importedUrls, url];
          } else {
            btn.disabled = false;
            btn.textContent = "\u274C Error";
            btn.title = result?.error || "Failed to save";
            showToast(result?.error || "Failed to save");
            setTimeout(() => {
              btn.textContent = "\uD83D\uDCBC Save";
              btn.title = "Save to Jobber";
            }, 3000);
          }
        });
      }

      // Position the button inside the card
      const computed = getComputedStyle(card);
      if (computed.position === "static") {
        card.style.position = "relative";
      }
      card.appendChild(btn);
    });
  }

  // Debounce MutationObserver callbacks
  let processTimeout = null;
  function scheduleProcess() {
    if (processTimeout) return;
    processTimeout = setTimeout(() => {
      processTimeout = null;
      processCards();
    }, 500);
  }

  async function init() {
    await loadImportedUrls();
    processCards();

    // Re-scan when DOM changes (infinite scroll, dynamic loading)
    const observer = new MutationObserver(scheduleProcess);
    observer.observe(document.body, { childList: true, subtree: true });

    // Re-check imported URLs when storage changes
    chrome.storage.onChanged.addListener((changes, area) => {
      if (area === "local" && changes.importedUrls) {
        importedUrls = changes.importedUrls.newValue || [];
      }
    });
  }

  init();
})();
