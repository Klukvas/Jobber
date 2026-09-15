import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

import {
  createOverlayKeyboard,
  type OverlayElements,
} from "../checkoutOverlayFocus";

/**
 * The provider's checkout is a cross-origin iframe appended to `<body>`. While
 * it covers the viewport the keyboard used to stay in the app underneath it:
 * Tab walked through controls nobody could see, and the payment form could not
 * be reached at all. These are the promises that made that impossible.
 */
describe("checkout overlay keyboard", () => {
  let app: HTMLElement;
  let startedCheckout: HTMLButtonElement;

  beforeEach(() => {
    document.body.innerHTML = "";
    app = document.createElement("div");
    app.id = "root";
    startedCheckout = document.createElement("button");
    startedCheckout.textContent = "Upgrade";
    app.appendChild(startedCheckout);
    document.body.appendChild(app);
    startedCheckout.focus();
  });

  afterEach(() => {
    document.body.innerHTML = "";
  });

  /** Stands in for what SBL appends: a full-viewport canvas around the frame. */
  function showOverlay({ withFrame = true } = {}): OverlayElements {
    const canvas = document.createElement("div");
    canvas.id = "fscCanvas";
    document.body.appendChild(canvas);

    if (!withFrame) return { roots: [canvas], frame: null };

    const frame = document.createElement("iframe");
    frame.id = "fsc-popup-frame";
    canvas.appendChild(frame);
    return { roots: [canvas, frame], frame };
  }

  it("takes the app out of the tab order and the accessibility tree", () => {
    const keyboard = createOverlayKeyboard();

    keyboard.handOver(showOverlay());

    expect(app).toHaveAttribute("inert");
    expect(app).toHaveAttribute("aria-hidden", "true");
  });

  it("leaves the provider's own overlay reachable", () => {
    const keyboard = createOverlayKeyboard();
    const overlay = showOverlay();

    keyboard.handOver(overlay);

    for (const root of overlay.roots) {
      expect(root).not.toHaveAttribute("inert");
      expect(root).not.toHaveAttribute("aria-hidden");
    }
  });

  it("moves focus into the checkout frame", () => {
    const keyboard = createOverlayKeyboard();
    const overlay = showOverlay();

    keyboard.handOver(overlay);

    expect(document.activeElement).toBe(overlay.frame);
  });

  // The canvas is appended before the frame inside it exists, and the caller
  // hands over on every tick of its DOM watch.
  it("suspends the page even before the frame exists, then focuses it", () => {
    const keyboard = createOverlayKeyboard();

    keyboard.handOver(showOverlay({ withFrame: false }));
    expect(app).toHaveAttribute("inert");

    const frame = document.createElement("iframe");
    document.getElementById("fscCanvas")!.appendChild(frame);
    keyboard.handOver({
      roots: [document.getElementById("fscCanvas")!, frame],
      frame,
    });

    expect(document.activeElement).toBe(frame);
  });

  // Focus inside a cross-origin frame reads as the frame element itself, so
  // this is also "the buyer is part-way through typing a card number".
  it("does not pull focus back once it is already in the overlay", () => {
    const keyboard = createOverlayKeyboard();
    const overlay = showOverlay();
    keyboard.handOver(overlay);

    keyboard.handOver(overlay);

    expect(document.activeElement).toBe(overlay.frame);
  });

  // The app keeps rendering behind the checkout — a toast portal, say — and
  // whatever it appends must not become the one reachable thing on the page.
  it("suspends what the app appends while the overlay is up", () => {
    const keyboard = createOverlayKeyboard();
    const overlay = showOverlay();
    keyboard.handOver(overlay);

    const toast = document.createElement("div");
    document.body.appendChild(toast);
    keyboard.handOver(overlay);

    expect(toast).toHaveAttribute("inert");
    expect(toast).toHaveAttribute("aria-hidden", "true");

    keyboard.release();
    expect(toast).not.toHaveAttribute("inert");
  });

  it("gives the app back its keyboard when the overlay goes", () => {
    const keyboard = createOverlayKeyboard();
    keyboard.handOver(showOverlay());

    keyboard.release();

    expect(app).not.toHaveAttribute("inert");
    expect(app).not.toHaveAttribute("aria-hidden");
  });

  it("returns focus to the control that started the checkout", () => {
    const keyboard = createOverlayKeyboard();
    keyboard.handOver(showOverlay());
    expect(document.activeElement).not.toBe(startedCheckout);

    keyboard.release();

    expect(document.activeElement).toBe(startedCheckout);
  });

  // The app puts up its own "activating your subscription" overlay the moment
  // the checkout reports an order, and focuses its dismiss button. That
  // decision is newer than this one.
  it("leaves focus alone when the app has already placed it", () => {
    const keyboard = createOverlayKeyboard();
    keyboard.handOver(showOverlay());
    const dismiss = document.createElement("button");
    app.appendChild(dismiss);

    keyboard.release();
    dismiss.focus();
    keyboard.release();

    expect(document.activeElement).toBe(dismiss);
  });

  // Anything already hidden — a modal's own backdrop, a sibling overlay — has
  // to come back exactly as it was, not merely "not hidden".
  it("restores attributes it did not set", () => {
    const keyboard = createOverlayKeyboard();
    const decoration = document.createElement("div");
    decoration.setAttribute("aria-hidden", "true");
    decoration.setAttribute("inert", "");
    document.body.appendChild(decoration);

    keyboard.handOver(showOverlay());
    keyboard.release();

    expect(decoration).toHaveAttribute("inert");
    expect(decoration).toHaveAttribute("aria-hidden", "true");
  });

  it("does nothing at all when no overlay is on screen", () => {
    const keyboard = createOverlayKeyboard();

    keyboard.handOver({ roots: [], frame: null });

    expect(app).not.toHaveAttribute("inert");
    expect(document.activeElement).toBe(startedCheckout);
  });

  it("survives a release with nothing to release", () => {
    const keyboard = createOverlayKeyboard();

    expect(() => keyboard.release()).not.toThrow();
    expect(app).not.toHaveAttribute("inert");
  });
});

/**
 * Where the buyer lands when the checkout goes away.
 *
 * The control that opened it is disabled for as long as the purchase is in
 * flight, and the browser blurs a disabled control — so by the time the
 * provider's popup exists, `document.activeElement` is `<body>` and the page no
 * longer knows where the buyer came from. Dismissing the checkout therefore put
 * focus back on `<body>`: the modal they had been reading was still on screen,
 * and the keyboard was nowhere near it.
 */
describe("checkout overlay keyboard — returning the buyer to the trigger", () => {
  let app: HTMLElement;
  let trigger: HTMLButtonElement;

  beforeEach(() => {
    document.body.innerHTML = "";
    app = document.createElement("div");
    trigger = document.createElement("button");
    trigger.textContent = "Get Started";
    app.appendChild(trigger);
    document.body.appendChild(app);
    // What the page actually looks like once a checkout has started: the
    // trigger is busy, and nothing holds focus.
    trigger.disabled = true;
    trigger.blur();
  });

  afterEach(() => {
    document.body.innerHTML = "";
    vi.useRealTimers();
  });

  function showOverlay(): OverlayElements {
    const canvas = document.createElement("div");
    canvas.id = "fscCanvas";
    const frame = document.createElement("iframe");
    canvas.appendChild(frame);
    document.body.appendChild(canvas);
    return { roots: [canvas, frame], frame };
  }

  /** Lets the MutationObserver callbacks the release armed actually run. */
  const flushObservers = () => new Promise((resolve) => setTimeout(resolve, 0));

  it("remembers the named trigger even though the page was focusing nothing", async () => {
    const keyboard = createOverlayKeyboard({ returnFocusTo: trigger });
    keyboard.handOver(showOverlay());
    expect(document.activeElement).not.toBe(trigger);

    keyboard.release();
    // The app re-enables the button as part of the same close handling.
    trigger.disabled = false;
    await flushObservers();

    expect(document.activeElement).toBe(trigger);
  });

  // Without the wait this looked correct and did nothing: `focus()` on a
  // disabled control is silently ignored.
  it("does not give up while the trigger is still busy", () => {
    const keyboard = createOverlayKeyboard({ returnFocusTo: trigger });
    keyboard.handOver(showOverlay());

    keyboard.release();

    expect(document.activeElement).not.toBe(trigger);
    expect(trigger.disabled).toBe(true);
  });

  it("focuses a trigger that was never disabled straight away", () => {
    trigger.disabled = false;
    const keyboard = createOverlayKeyboard({ returnFocusTo: trigger });
    keyboard.handOver(showOverlay());

    keyboard.release();

    expect(document.activeElement).toBe(trigger);
  });

  // The app puts up its own "activating your subscription" overlay on a
  // completed purchase and focuses its dismiss button. That is the newer
  // decision, and a late return must not overrule it.
  it("leaves focus alone if the app placed it while the trigger was busy", async () => {
    const keyboard = createOverlayKeyboard({ returnFocusTo: trigger });
    keyboard.handOver(showOverlay());
    keyboard.release();

    const dismiss = document.createElement("button");
    app.appendChild(dismiss);
    dismiss.focus();
    trigger.disabled = false;
    await flushObservers();

    expect(document.activeElement).toBe(dismiss);
  });

  it("stops waiting for a trigger that never comes back", async () => {
    vi.useFakeTimers();
    const keyboard = createOverlayKeyboard({ returnFocusTo: trigger });
    keyboard.handOver(showOverlay());
    keyboard.release();

    await vi.advanceTimersByTimeAsync(5_000);
    trigger.disabled = false;
    await vi.advanceTimersByTimeAsync(0);

    expect(document.activeElement).not.toBe(trigger);
  });

  it("still falls back to whatever had focus when no trigger is named", () => {
    trigger.disabled = false;
    trigger.focus();
    const keyboard = createOverlayKeyboard();
    keyboard.handOver(showOverlay());

    keyboard.release();

    expect(document.activeElement).toBe(trigger);
  });

  // A hidden control takes `focus()` exactly as silently as a disabled one.
  // Treating it as focusable ended the checkout with the buyer on `<body>` and
  // no wait armed, which is the failure the whole wait exists to prevent.
  it("waits for a trigger its own container has hidden", async () => {
    trigger.disabled = false;
    app.style.display = "none";

    const keyboard = createOverlayKeyboard({ returnFocusTo: trigger });
    keyboard.handOver(showOverlay());
    keyboard.release();

    expect(document.activeElement).not.toBe(trigger);

    // The modal the trigger lives in comes back as the checkout closes.
    app.style.display = "";
    await flushObservers();

    expect(document.activeElement).toBe(trigger);
  });

  it("waits for a trigger that is merely invisible, not undisplayed", async () => {
    trigger.disabled = false;
    trigger.style.visibility = "hidden";

    const keyboard = createOverlayKeyboard({ returnFocusTo: trigger });
    keyboard.handOver(showOverlay());
    keyboard.release();

    expect(document.activeElement).not.toBe(trigger);

    trigger.style.visibility = "";
    await flushObservers();

    expect(document.activeElement).toBe(trigger);
  });

  it("does not chase a trigger that left the page with its modal", async () => {
    const keyboard = createOverlayKeyboard({ returnFocusTo: trigger });
    keyboard.handOver(showOverlay());
    trigger.remove();

    keyboard.release();
    await flushObservers();

    expect(document.activeElement).not.toBe(trigger);
  });
});
