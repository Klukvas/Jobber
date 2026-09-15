import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";

import {
  ERROR_CALLBACK,
  POPUP_CLOSED_CALLBACK,
  POPUP_APPEARANCE_TIMEOUT_MS,
  POPUP_WATCH_INTERVAL_MS,
  SBL_SCRIPT_ID,
  SBL_SRC,
  SCRIPT_LOAD_TIMEOUT_MS,
  isValidStorefront,
  openPopupCheckout,
  resetSblForTests,
} from "../fastspringSbl";

const TEST_STOREFRONT = "fluxlab.test.onfastspring.com/popup-jobber";
const LIVE_STOREFRONT = "fluxlab.onfastspring.com/popup-jobber";

/** The builder stub SBL would install on `window` once it has loaded. */
function installBuilder() {
  const push = vi.fn();
  window.fastspring = { builder: { push } };
  return push;
}

/** Grabs the script element the module inserted, if any. */
function insertedScript(): HTMLScriptElement | null {
  return document.getElementById(SBL_SCRIPT_ID) as HTMLScriptElement | null;
}

/** Every SBL tag currently on the page — one is correct, more is a bug. */
function insertedScripts(): HTMLScriptElement[] {
  return Array.from(document.querySelectorAll(`script#${SBL_SCRIPT_ID}`));
}

/** Plays the part of a browser that fetched the script successfully. */
function succeedScriptLoad() {
  const script = insertedScript();
  if (!script) throw new Error("no SBL script was inserted");
  installBuilder();
  script.onload?.(new Event("load"));
}

function handlers() {
  return { onClose: vi.fn(), onError: vi.fn() };
}

function callPopupClosed(order: unknown) {
  (window as unknown as Record<string, (order: unknown) => void>)[
    POPUP_CLOSED_CALLBACK
  ](order);
}

describe("isValidStorefront", () => {
  it("accepts the test storefront in the test environment", () => {
    expect(isValidStorefront(TEST_STOREFRONT, "test")).toBe(true);
  });

  it("accepts the live storefront in the live environment", () => {
    expect(isValidStorefront(LIVE_STOREFRONT, "live")).toBe(true);
  });

  it("refuses a live storefront while the backend says test", () => {
    // Otherwise a test deployment would charge real cards.
    expect(isValidStorefront(LIVE_STOREFRONT, "test")).toBe(false);
  });

  it("refuses a test storefront while the backend says live", () => {
    expect(isValidStorefront(TEST_STOREFRONT, "live")).toBe(false);
  });

  it("refuses a checkout that is not a popup checkout", () => {
    // SBL draws an on-page frame only for a `/popup-` path; anything else is
    // assigned to window.location, which is the full-page redirect this
    // integration exists to remove.
    expect(
      isValidStorefront(
        "fluxlab.test.onfastspring.com/jobber-checkout",
        "test",
      ),
    ).toBe(false);
    expect(
      isValidStorefront(
        "fluxlab.test.onfastspring.com/jobber-popup-checkout",
        "test",
      ),
    ).toBe(false);
  });

  it("refuses anything that is not a FastSpring storefront", () => {
    for (const storefront of [
      "fluxlab.test.onfastspring.com.evil.com/popup-jobber",
      "evil.com/popup-jobber",
      "https://fluxlab.test.onfastspring.com/popup-jobber",
      "fluxlab.test.onfastspring.com/popup-jobber/../evil",
      "fluxlab.test.onfastspring.com",
      "",
    ]) {
      expect(isValidStorefront(storefront, "test")).toBe(false);
    }
  });

  it("refuses an environment the backend did not name", () => {
    expect(isValidStorefront(TEST_STOREFRONT, "")).toBe(false);
    expect(isValidStorefront(TEST_STOREFRONT, "sandbox")).toBe(false);
  });
});

describe("openPopupCheckout", () => {
  beforeEach(() => {
    resetSblForTests();
    insertedScripts().forEach((script) => script.remove());
    delete window.fastspring;
  });

  afterEach(() => {
    resetSblForTests();
    insertedScripts().forEach((script) => script.remove());
    delete window.fastspring;
    vi.useRealTimers();
  });

  it("loads the pinned SBL build pointed at the backend's storefront", async () => {
    const open = openPopupCheckout({
      storefront: TEST_STOREFRONT,
      environment: "test",
      sessionId: "sess-1",
      handlers: handlers(),
    });
    succeedScriptLoad();
    await open;

    const script = insertedScript();
    expect(script?.src).toBe(SBL_SRC);
    expect(script?.dataset.storefront).toBe(TEST_STOREFRONT);
    expect(script?.dataset.popupClosed).toBe(POPUP_CLOSED_CALLBACK);
    expect(script?.dataset.errorCallback).toBe(ERROR_CALLBACK);
  });

  /**
   * The provider's CDN serves this file for cross-origin embedding — it answers
   * `access-control-allow-origin: *` with a year-long max-age — so requesting
   * it in CORS mode costs nothing and pins the request to the credential-less,
   * referrer-less form. Without these, the browser attaches whatever cookies
   * the provider's domain has and tells it which page the buyer was on.
   */
  it("fetches the payment script without credentials or a referrer", async () => {
    const open = openPopupCheckout({
      storefront: TEST_STOREFRONT,
      environment: "test",
      sessionId: "sess-1",
      handlers: handlers(),
    });
    succeedScriptLoad();
    await open;

    const script = insertedScript();
    expect(script?.crossOrigin).toBe("anonymous");
    expect(script?.referrerPolicy).toBe("no-referrer");
    // Still the provider's own loader contract: same id, same src, same
    // data-attributes. The extra attributes are request hygiene, not a fork.
    expect(script?.id).toBe(SBL_SCRIPT_ID);
    expect(script?.getAttribute("src")).toBe(SBL_SRC);
    expect(script?.dataset.storefront).toBe(TEST_STOREFRONT);
  });

  it("hands SBL the session id and nothing else", async () => {
    const push = vi.fn();
    const open = openPopupCheckout({
      storefront: TEST_STOREFRONT,
      environment: "test",
      sessionId: "sess-1",
      handlers: handlers(),
    });
    const script = insertedScript();
    window.fastspring = { builder: { push } };
    script?.onload?.(new Event("load"));
    await open;

    // No product path, no price, no buyer detail is assembled in the browser.
    expect(push).toHaveBeenCalledWith({ checkout: "sess-1" });
    expect(push).toHaveBeenCalledTimes(1);
  });

  it("inserts the script at most once per page", async () => {
    const first = openPopupCheckout({
      storefront: TEST_STOREFRONT,
      environment: "test",
      sessionId: "sess-1",
      handlers: handlers(),
    });
    succeedScriptLoad();
    await first;
    callPopupClosed(null);

    const loadedTag = insertedScript();
    expect(insertedScripts()).toHaveLength(1);

    await openPopupCheckout({
      storefront: TEST_STOREFRONT,
      environment: "test",
      sessionId: "sess-2",
      handlers: handlers(),
    });

    // The same tag, not a second one: a re-download would reset SBL's state
    // mid-purchase and cost the buyer a page-weight for nothing.
    expect(insertedScripts()).toHaveLength(1);
    expect(insertedScript()).toBe(loadedTag);
  });

  it("refuses to open on a storefront the environment does not match", async () => {
    await expect(
      openPopupCheckout({
        storefront: LIVE_STOREFRONT,
        environment: "test",
        sessionId: "sess-1",
        handlers: handlers(),
      }),
    ).rejects.toThrow(/storefront is not valid/i);
    expect(insertedScript()).toBeNull();
  });

  it("refuses to open a checkout that would navigate the page", async () => {
    await expect(
      openPopupCheckout({
        storefront: "fluxlab.test.onfastspring.com/jobber-checkout",
        environment: "test",
        sessionId: "sess-1",
        handlers: handlers(),
      }),
    ).rejects.toThrow(/storefront is not valid/i);
    // Nothing was loaded, so SBL never got the chance to redirect.
    expect(insertedScript()).toBeNull();
  });

  it("refuses a session with no id", async () => {
    await expect(
      openPopupCheckout({
        storefront: TEST_STOREFRONT,
        environment: "test",
        sessionId: "",
        handlers: handlers(),
      }),
    ).rejects.toThrow(/no session id/i);
    expect(insertedScript()).toBeNull();
  });

  it("refuses a second popup while one is open", async () => {
    const open = openPopupCheckout({
      storefront: TEST_STOREFRONT,
      environment: "test",
      sessionId: "sess-1",
      handlers: handlers(),
    });
    succeedScriptLoad();
    await open;

    await expect(
      openPopupCheckout({
        storefront: TEST_STOREFRONT,
        environment: "test",
        sessionId: "sess-2",
        handlers: handlers(),
      }),
    ).rejects.toThrow(/already open/i);
  });

  it("lets the next click retry after the script failed to load", async () => {
    const failing = openPopupCheckout({
      storefront: TEST_STOREFRONT,
      environment: "test",
      sessionId: "sess-1",
      handlers: handlers(),
    });
    insertedScript()?.onerror?.(new Event("error"));
    await expect(failing).rejects.toThrow(/failed to load/i);
    // The failed attempt is forgotten, tag and all, so a retry is a real retry.
    expect(insertedScript()).toBeNull();

    const retry = openPopupCheckout({
      storefront: TEST_STOREFRONT,
      environment: "test",
      sessionId: "sess-1",
      handlers: handlers(),
    });
    succeedScriptLoad();
    await expect(retry).resolves.toBeUndefined();
  });

  it("gives up on a script that never loads", async () => {
    vi.useFakeTimers();
    const pending = openPopupCheckout({
      storefront: TEST_STOREFRONT,
      environment: "test",
      sessionId: "sess-1",
      handlers: handlers(),
    });
    const assertion = expect(pending).rejects.toThrow(/timed out/i);
    vi.advanceTimersByTime(SCRIPT_LOAD_TIMEOUT_MS);
    await assertion;
  });

  it("fails rather than opening when the script arrives without a builder", async () => {
    const open = openPopupCheckout({
      storefront: TEST_STOREFRONT,
      environment: "test",
      sessionId: "sess-1",
      handlers: handlers(),
    });
    insertedScript()?.onload?.(new Event("load"));
    await expect(open).rejects.toThrow(/without a builder/i);
  });

  it("reports a completed close when SBL names an order", async () => {
    const popupHandlers = handlers();
    const open = openPopupCheckout({
      storefront: TEST_STOREFRONT,
      environment: "test",
      sessionId: "sess-1",
      handlers: popupHandlers,
    });
    succeedScriptLoad();
    await open;

    callPopupClosed([{ reference: "FLUXLAB240101-1234-56789" }]);

    expect(popupHandlers.onClose).toHaveBeenCalledWith({ completed: true });
    expect(popupHandlers.onError).not.toHaveBeenCalled();
  });

  it("reports a cancelled close when SBL passes null", async () => {
    const popupHandlers = handlers();
    const open = openPopupCheckout({
      storefront: TEST_STOREFRONT,
      environment: "test",
      sessionId: "sess-1",
      handlers: popupHandlers,
    });
    succeedScriptLoad();
    await open;

    callPopupClosed(null);

    expect(popupHandlers.onClose).toHaveBeenCalledWith({ completed: false });
  });

  it("routes a provider error to the popup that is open", async () => {
    const popupHandlers = handlers();
    const open = openPopupCheckout({
      storefront: TEST_STOREFRONT,
      environment: "test",
      sessionId: "sess-1",
      handlers: popupHandlers,
    });
    succeedScriptLoad();
    await open;

    (window as unknown as Record<string, (...args: unknown[]) => void>)[
      ERROR_CALLBACK
    ]("CART_ERROR", "something went wrong");

    expect(popupHandlers.onError).toHaveBeenCalledTimes(1);
    expect(popupHandlers.onClose).not.toHaveBeenCalled();
  });

  it("delivers a close to the popup that is open, and only once", async () => {
    const first = handlers();
    const open = openPopupCheckout({
      storefront: TEST_STOREFRONT,
      environment: "test",
      sessionId: "sess-1",
      handlers: first,
    });
    succeedScriptLoad();
    await open;

    callPopupClosed(null);
    // A stray second callback — SBL is third-party code — must not re-fire.
    callPopupClosed(null);

    expect(first.onClose).toHaveBeenCalledTimes(1);
  });

  /**
   * `push` can finish the whole checkout before it returns — an error callback
   * fired synchronously, or a session the provider rejects outright. The
   * handlers are already gone by then, so installing a DOM watch would leave an
   * interval and a MutationObserver running against a checkout that is over.
   */
  it("installs no watcher when the callback already fired synchronously", async () => {
    vi.useFakeTimers();
    try {
      const h = handlers();
      const open = openPopupCheckout({
        storefront: TEST_STOREFRONT,
        environment: "test",
        sessionId: "sess-sync",
        handlers: h,
      });

      const script = insertedScript();
      if (!script) throw new Error("no SBL script was inserted");
      const push = vi.fn(() => {
        (window as unknown as Record<string, () => void>)[ERROR_CALLBACK]();
      });
      window.fastspring = { builder: { push } };
      script.onload?.(new Event("load"));
      await open;

      expect(h.onError).toHaveBeenCalledOnce();
      expect(vi.getTimerCount()).toBe(0);
    } finally {
      vi.useRealTimers();
    }
  });

  // ...and the checkout is genuinely over, so the next one may start.
  it("lets a new checkout start after a synchronous failure", async () => {
    const h = handlers();
    const open = openPopupCheckout({
      storefront: TEST_STOREFRONT,
      environment: "test",
      sessionId: "sess-sync",
      handlers: h,
    });
    const script = insertedScript();
    if (!script) throw new Error("no SBL script was inserted");
    window.fastspring = {
      builder: {
        push: () => {
          (window as unknown as Record<string, (o: unknown) => void>)[
            POPUP_CLOSED_CALLBACK
          ](null);
        },
      },
    };
    script.onload?.(new Event("load"));
    await open;

    const second = handlers();
    await expect(
      openPopupCheckout({
        storefront: TEST_STOREFRONT,
        environment: "test",
        sessionId: "sess-2",
        handlers: second,
      }),
    ).resolves.toBeUndefined();
  });

  it("refuses to reuse a script pinned to a different storefront", async () => {
    const open = openPopupCheckout({
      storefront: TEST_STOREFRONT,
      environment: "test",
      sessionId: "sess-1",
      handlers: handlers(),
    });
    succeedScriptLoad();
    await open;
    callPopupClosed(null);

    // SBL reads data-storefront once, so a second storefront cannot be honoured
    // by the already-loaded script.
    await expect(
      openPopupCheckout({
        storefront: LIVE_STOREFRONT,
        environment: "live",
        sessionId: "sess-2",
        handlers: handlers(),
      }),
    ).rejects.toThrow(/storefront changed/i);
  });
});

/**
 * Closing the popup with FastSpring's own X does not fire `data-popup-closed`.
 * Nothing then cleared the "a checkout is in flight" baseline, so the next
 * mount showed a customer who had just declined to pay a full-screen
 * "Activating your subscription…". The close is therefore also detected from
 * the DOM rather than trusted to arrive.
 */
describe("popup dismissal without a provider callback", () => {
  function removePopups() {
    document.querySelectorAll("[data-test-popup]").forEach((el) => el.remove());
  }

  beforeEach(() => {
    resetSblForTests();
    insertedScripts().forEach((script) => script.remove());
    removePopups();
    delete window.fastspring;
    vi.useFakeTimers();
  });

  afterEach(() => {
    resetSblForTests();
    insertedScripts().forEach((script) => script.remove());
    removePopups();
    delete window.fastspring;
    vi.useRealTimers();
  });

  /**
   * Stands in for the container SBL injects when the popup opens.
   *
   * `#fscCanvas` is what the pinned 1.0.9 build actually appends, so the
   * default shape here is the shape production has to recognise — a double
   * built around one of the speculative fallbacks would have passed happily
   * while the real thing went unseen.
   */
  function showPopup(shape = "#fscCanvas"): HTMLElement {
    const popup = document.createElement("div");
    if (shape.startsWith("#")) {
      popup.id = shape.slice(1);
    } else {
      popup.className = shape.slice(1);
    }
    popup.dataset.testPopup = "";
    document.body.appendChild(popup);
    return popup;
  }

  async function openWith(h: ReturnType<typeof handlers>) {
    const open = openPopupCheckout({
      storefront: TEST_STOREFRONT,
      environment: "test",
      sessionId: "sess-1",
      handlers: h,
    });
    succeedScriptLoad();
    await open;
  }

  /** Advances past one poll of the DOM watch. */
  function tickWatch() {
    vi.advanceTimersByTime(POPUP_WATCH_INTERVAL_MS + 1);
  }

  it("reports the close when the popup is removed with no callback", async () => {
    const h = handlers();
    await openWith(h);
    const popup = showPopup();
    tickWatch();
    expect(h.onClose).not.toHaveBeenCalled();

    popup.remove();
    tickWatch();

    expect(h.onClose).toHaveBeenCalledWith({ completed: false });
  });

  it("reports the close when the popup is only hidden", async () => {
    const h = handlers();
    await openWith(h);
    const popup = showPopup();
    tickWatch();

    popup.style.display = "none";
    tickWatch();

    expect(h.onClose).toHaveBeenCalledWith({ completed: false });
  });

  /**
   * SBL dismisses its popup by class as often as by inline style — and an
   * inline-only check calls a `display: none` stylesheet rule "on screen". The
   * watch then sat there believing the checkout was still up, so the buyer who
   * closed it kept their stale baseline and met "Activating your
   * subscription…" on the next page load.
   */
  it("reports the close when the popup is hidden by a stylesheet class", async () => {
    const sheet = document.createElement("style");
    sheet.textContent = ".fsc-hidden { display: none; }";
    document.head.appendChild(sheet);
    try {
      const h = handlers();
      await openWith(h);
      const popup = showPopup();
      tickWatch();
      expect(h.onClose).not.toHaveBeenCalled();

      popup.classList.add("fsc-hidden");
      tickWatch();

      expect(h.onClose).toHaveBeenCalledWith({ completed: false });
    } finally {
      sheet.remove();
    }
  });

  // `display: none` on an ancestor takes the whole subtree off the screen
  // without changing anything computed on the container itself.
  it("reports the close when a wrapper above the popup is hidden", async () => {
    const h = handlers();
    await openWith(h);
    const wrapper = document.createElement("div");
    document.body.appendChild(wrapper);
    const popup = showPopup();
    wrapper.appendChild(popup);
    tickWatch();
    expect(h.onClose).not.toHaveBeenCalled();

    wrapper.style.display = "none";
    tickWatch();

    expect(h.onClose).toHaveBeenCalledWith({ completed: false });
  });

  it("keeps watching a popup that is merely styled, not hidden", async () => {
    const sheet = document.createElement("style");
    sheet.textContent = ".fsc-shown { display: block; opacity: 0.99; }";
    document.head.appendChild(sheet);
    try {
      const h = handlers();
      await openWith(h);
      const popup = showPopup();
      popup.classList.add("fsc-shown");
      tickWatch();
      tickWatch();

      expect(h.onClose).not.toHaveBeenCalled();
      expect(h.onError).not.toHaveBeenCalled();
    } finally {
      sheet.remove();
    }
  });

  it("never claims a purchase it did not see", async () => {
    const h = handlers();
    await openWith(h);
    const popup = showPopup();
    tickWatch();

    popup.remove();
    tickWatch();

    expect(h.onClose).toHaveBeenCalledWith({ completed: false });
    expect(h.onError).not.toHaveBeenCalled();
  });

  // The container is injected asynchronously; a popup that has not rendered yet
  // must not read as one that closed.
  it("stays quiet until the popup has actually appeared", async () => {
    const h = handlers();
    await openWith(h);

    tickWatch();
    tickWatch();

    expect(h.onClose).not.toHaveBeenCalled();
  });

  /**
   * A checkout that never renders anything has to end, not hang. Leaving the
   * handlers in place kept the caller's busy state on for the life of the page
   * and made every later click fail with "a popup is already open" — the
   * customer could neither finish nor start again.
   */
  it("fails the checkout when the popup never appears", async () => {
    const h = handlers();
    await openWith(h);

    vi.advanceTimersByTime(
      POPUP_APPEARANCE_TIMEOUT_MS + POPUP_WATCH_INTERVAL_MS * 2,
    );

    expect(h.onError).toHaveBeenCalledOnce();
    // Never a close: nothing was ever seen, so nothing was dismissed.
    expect(h.onClose).not.toHaveBeenCalled();
  });

  // Nothing to observe means nothing to poll for.
  it("leaves no timer or observer behind after giving up", async () => {
    const h = handlers();
    await openWith(h);

    vi.advanceTimersByTime(
      POPUP_APPEARANCE_TIMEOUT_MS + POPUP_WATCH_INTERVAL_MS * 2,
    );

    expect(vi.getTimerCount()).toBe(0);
  });

  it("reports the failure exactly once, however long the page stays open", async () => {
    const h = handlers();
    await openWith(h);

    vi.advanceTimersByTime(POPUP_APPEARANCE_TIMEOUT_MS * 3);

    expect(h.onError).toHaveBeenCalledOnce();
  });

  // The whole point of releasing the checkout: the next click has to work.
  it("lets a later checkout start after the timeout", async () => {
    const first = handlers();
    await openWith(first);

    vi.advanceTimersByTime(
      POPUP_APPEARANCE_TIMEOUT_MS + POPUP_WATCH_INTERVAL_MS * 2,
    );

    const second = handlers();
    await expect(openWith(second)).resolves.toBeUndefined();

    const popup = showPopup();
    tickWatch();
    popup.remove();
    tickWatch();

    expect(second.onClose).toHaveBeenCalledWith({ completed: false });
    // The abandoned checkout hears nothing more.
    expect(first.onClose).not.toHaveBeenCalled();
  });

  it("reports the close exactly once", async () => {
    const h = handlers();
    await openWith(h);
    const popup = showPopup();
    tickWatch();

    popup.remove();
    tickWatch();
    tickWatch();
    tickWatch();

    expect(h.onClose).toHaveBeenCalledTimes(1);
  });

  // The provider's callback is still the primary signal; the watcher must not
  // double-report the same close, nor downgrade a completed checkout.
  it("leaves a real completion to the provider callback", async () => {
    const h = handlers();
    await openWith(h);
    const popup = showPopup();
    tickWatch();

    const closed = (window as unknown as Record<string, (o: unknown) => void>)[
      POPUP_CLOSED_CALLBACK
    ];
    closed({ reference: "ORDER-1" });
    popup.remove();
    tickWatch();

    expect(h.onClose).toHaveBeenCalledTimes(1);
    expect(h.onClose).toHaveBeenCalledWith({ completed: true });
  });

  it("does not fire after the provider reported a cancel", async () => {
    const h = handlers();
    await openWith(h);
    const popup = showPopup();
    tickWatch();

    const closed = (window as unknown as Record<string, (o: unknown) => void>)[
      POPUP_CLOSED_CALLBACK
    ];
    closed(null);
    popup.remove();
    tickWatch();

    expect(h.onClose).toHaveBeenCalledTimes(1);
    expect(h.onClose).toHaveBeenCalledWith({ completed: false });
  });

  it("does not fire after the provider reported an error", async () => {
    const h = handlers();
    await openWith(h);
    const popup = showPopup();
    tickWatch();

    const errored = (window as unknown as Record<string, () => void>)[
      ERROR_CALLBACK
    ];
    errored();
    popup.remove();
    tickWatch();

    expect(h.onError).toHaveBeenCalledTimes(1);
    expect(h.onClose).not.toHaveBeenCalled();
  });

  // A second checkout in the same page must get its own watch, not the corpse
  // of the previous one.
  it("watches each checkout independently", async () => {
    const first = handlers();
    await openWith(first);
    const popup = showPopup();
    tickWatch();
    popup.remove();
    tickWatch();
    expect(first.onClose).toHaveBeenCalledTimes(1);

    const second = handlers();
    await openWith(second);
    const secondPopup = showPopup();
    tickWatch();
    secondPopup.remove();
    tickWatch();

    expect(second.onClose).toHaveBeenCalledWith({ completed: false });
    expect(first.onClose).toHaveBeenCalledTimes(1);
  });

  /**
   * Each of these is an element SBL 1.0.9 renders. Any one of them appearing
   * and then going away is a dismissal, and the watcher has to see all three:
   * which one survives on screen depends on how the popup was closed.
   */
  it.each(["#fscCanvas", "#fsc-popup-frame", ".fs-popup-background"])(
    "recognises the popup by %s",
    async (shape) => {
      const h = handlers();
      await openWith(h);

      const popup = showPopup(shape);
      tickWatch();
      expect(h.onClose).not.toHaveBeenCalled();

      popup.remove();
      tickWatch();

      expect(h.onClose).toHaveBeenCalledWith({ completed: false });
    },
  );

  // The older guesses stay wired as fallbacks in case the provider's markup
  // moves again.
  it.each(["#fsc-popup", ".fsc-popup", "#fsc-embedded-checkout-container"])(
    "still recognises the fallback shape %s",
    async (shape) => {
      const h = handlers();
      await openWith(h);

      const popup = showPopup(shape);
      tickWatch();
      popup.remove();
      tickWatch();

      expect(h.onClose).toHaveBeenCalledWith({ completed: false });
    },
  );
});

/**
 * The popup covers the viewport, but it is the provider's iframe appended to
 * `<body>`: nothing about that moves the keyboard. Focus stayed on the app
 * control that had it, Tab walked through buttons painted over by the
 * checkout, and the payment form on screen could not be reached at all.
 */
describe("keyboard while the popup covers the page", () => {
  let app: HTMLElement;
  let upgrade: HTMLButtonElement;

  beforeEach(() => {
    resetSblForTests();
    insertedScripts().forEach((script) => script.remove());
    delete window.fastspring;
    document.body.innerHTML = "";
    app = document.createElement("div");
    app.id = "root";
    upgrade = document.createElement("button");
    app.appendChild(upgrade);
    document.body.appendChild(app);
    upgrade.focus();
    vi.useFakeTimers();
  });

  afterEach(() => {
    resetSblForTests();
    insertedScripts().forEach((script) => script.remove());
    delete window.fastspring;
    document.body.innerHTML = "";
    vi.useRealTimers();
  });

  /** What the pinned build appends: a full-viewport canvas around the frame. */
  function showPopup(): HTMLIFrameElement {
    const canvas = document.createElement("div");
    canvas.id = "fscCanvas";
    const frame = document.createElement("iframe");
    frame.id = "fsc-popup-frame";
    canvas.appendChild(frame);
    document.body.appendChild(canvas);
    return frame;
  }

  function hidePopup() {
    document.getElementById("fscCanvas")?.remove();
  }

  async function openCheckout(h: ReturnType<typeof handlers>) {
    const open = openPopupCheckout({
      storefront: TEST_STOREFRONT,
      environment: "test",
      sessionId: "sess-1",
      handlers: h,
    });
    succeedScriptLoad();
    await open;
  }

  function tickWatch() {
    vi.advanceTimersByTime(POPUP_WATCH_INTERVAL_MS + 1);
  }

  it("puts the app out of reach and the keyboard in the checkout frame", async () => {
    await openCheckout(handlers());

    const frame = showPopup();
    tickWatch();

    expect(app).toHaveAttribute("inert");
    expect(app).toHaveAttribute("aria-hidden", "true");
    expect(document.activeElement).toBe(frame);
  });

  it("leaves the provider's own overlay alone", async () => {
    await openCheckout(handlers());

    showPopup();
    tickWatch();

    expect(document.getElementById("fscCanvas")).not.toHaveAttribute("inert");
  });

  it("gives the app back its keyboard when the popup is dismissed", async () => {
    const h = handlers();
    await openCheckout(h);
    showPopup();
    tickWatch();

    hidePopup();
    tickWatch();

    expect(h.onClose).toHaveBeenCalledWith({ completed: false });
    expect(app).not.toHaveAttribute("inert");
    expect(app).not.toHaveAttribute("aria-hidden");
    expect(document.activeElement).toBe(upgrade);
  });

  it("gives it back when the provider reports the close itself", async () => {
    await openCheckout(handlers());
    showPopup();
    tickWatch();

    // SBL fires this with its frame sometimes still on the page, so the app
    // has to reclaim focus rather than wait for the browser to drop it.
    callPopupClosed([{ reference: "FLUXLAB240101-1234-56789" }]);

    expect(app).not.toHaveAttribute("inert");
    expect(document.activeElement).toBe(upgrade);
  });

  it("gives it back when the provider reports an error", async () => {
    await openCheckout(handlers());
    showPopup();
    tickWatch();

    (window as unknown as Record<string, () => void>)[ERROR_CALLBACK]();

    expect(app).not.toHaveAttribute("inert");
    expect(document.activeElement).toBe(upgrade);
  });

  it("leaves the page alone while no popup has appeared", async () => {
    await openCheckout(handlers());

    tickWatch();

    expect(app).not.toHaveAttribute("inert");
    expect(document.activeElement).toBe(upgrade);
  });

  // Two hooks mounting in the same tick — an upgrade banner and the modal it
  // opens — both used to get past the "already open" guard, because it read a
  // flag that is only set *after* the script load is awaited. The second
  // overwrote the first's handlers and its watcher, and the first watcher kept
  // the keyboard it had taken with no way left to release it: every app root
  // stayed `inert` for the rest of the page's life.
  it("refuses a second open started while the script is still loading", async () => {
    const winner = handlers();
    const loser = handlers();

    const first = openPopupCheckout({
      storefront: TEST_STOREFRONT,
      environment: "test",
      sessionId: "sess-1",
      handlers: winner,
    });
    // The tag is on the page and has not fired `load` yet: exactly the window
    // both callers used to get through.
    const second = openPopupCheckout({
      storefront: TEST_STOREFRONT,
      environment: "test",
      sessionId: "sess-2",
      handlers: loser,
    });

    await expect(second).rejects.toThrow(/already open/i);

    succeedScriptLoad();
    await first;

    // Only the winner's session reached the provider.
    expect(window.fastspring?.builder.push).toHaveBeenCalledTimes(1);
    expect(window.fastspring?.builder.push).toHaveBeenCalledWith({
      checkout: "sess-1",
    });

    showPopup();
    tickWatch();
    expect(app).toHaveAttribute("inert");

    hidePopup();
    tickWatch();

    expect(winner.onClose).toHaveBeenCalledWith({ completed: false });
    expect(loser.onClose).not.toHaveBeenCalled();
    expect(loser.onError).not.toHaveBeenCalled();
    // One owner of the watch means one release: the page is fully back.
    expect(app).not.toHaveAttribute("inert");
    expect(app).not.toHaveAttribute("aria-hidden");
    expect(document.activeElement).toBe(upgrade);
  });

  // The frame selector matches on a substring of `src`, which any URL can
  // carry. Reading an unrelated frame as the checkout would report the
  // purchase abandoned the moment that frame was removed.
  it("ignores a frame that only mentions the storefront in its URL", async () => {
    const h = handlers();
    await openCheckout(h);

    const impostor = document.createElement("iframe");
    impostor.src = "https://example.com/embed?ref=onfastspring.com";
    document.body.appendChild(impostor);
    tickWatch();

    expect(app).not.toHaveAttribute("inert");

    impostor.remove();
    tickWatch();

    // Never "seen", so its removal is not a dismissal.
    expect(h.onClose).not.toHaveBeenCalled();
  });
});
