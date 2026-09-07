import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";

import {
  ERROR_CALLBACK,
  POPUP_CLOSED_CALLBACK,
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
