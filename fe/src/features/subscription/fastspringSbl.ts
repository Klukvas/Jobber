/**
 * The one place `window.fastspring` and the provider's global callbacks exist.
 *
 * FastSpring's Store Builder Library (SBL) is a plain script that installs
 * itself on `window` and calls global functions by *name* — names it reads from
 * `data-*` attributes on its own script tag. That is a lot of global surface, so
 * it is confined here behind a narrow typed API: nothing outside this module
 * touches `window.fastspring`, and no component ever learns a callback name.
 *
 * https://developer.fastspring.com/reference/store-builder-library-overview
 * https://developer.fastspring.com/reference/callbacks
 */

import {
  createOverlayKeyboard,
  type OverlayElements,
} from "@/features/subscription/checkoutOverlayFocus";
import { isRendered } from "@/shared/lib/visibility";

/**
 * The exact SBL build the FastSpring dashboard's own snippet pins. Pinned rather
 * than floating: this script runs with full privileges on a page where people
 * pay, so the version that ships is the version that was reviewed.
 *
 * On start it also appends a stylesheet from its own directory
 * (`…/1.0.9/fastspring.css`) for the popup chrome, which is why the CSP has to
 * name this origin under `style-src` as well as `script-src`.
 */
export const SBL_SRC =
  "https://sbl.onfastspring.com/sbl/1.0.9/fastspring-builder.min.js";

/** SBL identifies its own script tag by this id and refuses to run without it. */
export const SBL_SCRIPT_ID = "fsc-api";

/** How long to wait for the script before treating the load as failed. */
export const SCRIPT_LOAD_TIMEOUT_MS = 15_000;

/**
 * Global function names handed to SBL through `data-popup-closed` and
 * `data-error-callback`. They are constants declared here — never anything the
 * backend sends — so no response can make the provider's script call an
 * arbitrary function.
 */
export const POPUP_CLOSED_CALLBACK = "jobberFastSpringPopupClosed";
export const ERROR_CALLBACK = "jobberFastSpringError";

/** Storefront host family. Anything else is not FastSpring and is not loaded. */
const STOREFRONT_DOMAIN = "onfastspring.com";
/** The extra label a *test* storefront carries: `<store>.test.onfastspring.com`. */
const TEST_STOREFRONT_LABEL = "test";

/** `<store>[.test].onfastspring.com/<checkout-id>` and nothing else. */
const STOREFRONT_PATTERN =
  /^[a-z0-9][a-z0-9.-]*\.onfastspring\.com\/[A-Za-z0-9][A-Za-z0-9._-]*$/;

/**
 * The prefix FastSpring puts on a checkout generated as a *popup* checkout.
 *
 * This is not cosmetic. SBL decides between an on-page iframe and a full-page
 * navigation by pattern-matching the checkout URL it is about to open: only a
 * `…/popup-…` (or `…/embedded-…`) path is drawn as a frame, and anything else
 * is assigned to `window.location`. A storefront without the prefix would
 * therefore navigate the buyer off Jobber — the exact behaviour this integration
 * exists to remove — so it is refused here rather than handed to SBL.
 */
const POPUP_CHECKOUT_PREFIX = "popup-";

/** The slice of SBL Jobber actually calls. */
interface StoreBuilderLibrary {
  builder: {
    /**
     * Hands SBL a session. `checkout` takes the id created by the Sessions API,
     * which is the whole cart — nothing is rebuilt in the browser.
     *
     * https://developer.fastspring.com/reference/session-object
     */
    push(payload: { checkout: string }): void;
  };
}

declare global {
  interface Window {
    fastspring?: StoreBuilderLibrary;
  }
}

/** What the caller is told when the popup closes or SBL fails. */
export interface PopupCheckoutHandlers {
  /**
   * The popup closed. `completed` is true only when FastSpring reported an
   * order — it is a *hint to start polling*, never proof of payment.
   */
  onClose(result: { completed: boolean }): void;
  /**
   * SBL reported an error. The payload is deliberately not forwarded: it can
   * carry buyer and order details, and nothing here would act on it anyway.
   */
  onError(): void;
}

export interface OpenPopupCheckoutOptions {
  /** `data-storefront` value, as issued by the backend's checkout config. */
  storefront: string;
  /** Backend's billing environment — "test" or "live". */
  environment: string;
  /** Session id from the backend's Sessions API call. */
  sessionId: string;
  handlers: PopupCheckoutHandlers;
  /**
   * The control the buyer clicked to start this checkout, so the keyboard can
   * be handed back to it when the popup closes. Read by the caller at click
   * time: this function is reached after an await, by which point the button
   * is disabled and the page is focusing `<body>`.
   */
  returnFocusTo?: HTMLElement | null;
}

/**
 * Selectors that match the container SBL injects for its popup.
 *
 * The first three are the elements the pinned 1.0.9 build puts on the page: a
 * full-viewport host `#fscCanvas`, the checkout iframe `#fsc-popup-frame`, and
 * the dimming layer `.fs-popup-background`. Naming them is the point of this
 * list — the generic patterns below were written from guesswork and match
 * nothing this build renders, so the DOM watch had nothing to observe and an
 * X-button dismissal went unnoticed.
 *
 * The rest stay as fallbacks: the markup is the provider's, undocumented, and
 * has changed between builds. Matching any one is enough to know the popup is
 * on screen, and matching none simply means the watcher stays quiet and the
 * provider's own callback does the reporting.
 */
const POPUP_NAMED_SELECTORS = [
  "#fscCanvas",
  "#fsc-popup-frame",
  ".fs-popup-background",
  "#fsc-popup",
  ".fsc-popup",
  "#fsc-embedded-checkout-container",
  '[id^="fsc-"][class*="popup"]',
].join(",");

/**
 * The last resort: a frame recognised by where it is loaded from rather than by
 * a name SBL chose.
 *
 * CSS can only do substring matching, and `src*="onfastspring.com"` is happily
 * satisfied by `https://example.com/?ref=onfastspring.com`. So anything matched
 * by this and by nothing else has its host checked properly in
 * `belongsToPopup` — an unrelated frame read as the checkout would report a
 * purchase as abandoned the moment it was removed.
 */
const POPUP_FRAME_BY_SOURCE_SELECTOR = 'iframe[src*="onfastspring.com"]';

const POPUP_SELECTORS = [
  POPUP_NAMED_SELECTORS,
  POPUP_FRAME_BY_SOURCE_SELECTOR,
].join(",");

/** Whether a URL is actually served by FastSpring, rather than mentioning it. */
function hasStorefrontHost(src: string): boolean {
  try {
    const { hostname } = new URL(src, window.location.href);
    return (
      hostname === STOREFRONT_DOMAIN ||
      hostname.endsWith(`.${STOREFRONT_DOMAIN}`)
    );
  } catch {
    return false;
  }
}

/**
 * Whether an element the selector list matched is really part of the popup.
 *
 * Anything SBL named is taken at face value — the pinned build's own
 * `#fsc-popup-frame` included, so this check cannot hide the real popup.
 * Everything else got here on a substring of its `src` and has to prove its
 * host.
 */
function belongsToPopup(element: HTMLElement): boolean {
  if (element.matches(POPUP_NAMED_SELECTORS)) return true;
  return hasStorefrontHost((element as HTMLIFrameElement).src ?? "");
}

/** How often the popup container is re-checked, on top of DOM mutations. */
export const POPUP_WATCH_INTERVAL_MS = 400;

/**
 * How long to keep looking for a popup that has not appeared.
 *
 * If SBL never renders anything there is nothing for this watch to observe, and
 * polling forever would leave a timer running for the life of the page. Once
 * this passes the checkout is reported as failed and released: leaving it
 * "open" pinned the caller's busy state on forever and made every later click
 * fail with "a popup is already open", which is a worse outcome than telling
 * the customer it did not start and letting them try again.
 */
export const POPUP_APPEARANCE_TIMEOUT_MS = 30_000;

/**
 * In-flight or finished script load. Kept at module scope so the script is
 * inserted exactly once per page, and cleared on failure so a retry can try
 * again rather than replaying the same rejection forever.
 */
let scriptLoad: Promise<void> | null = null;
/** The storefront the loaded script was pinned to; SBL reads it only once. */
let loadedStorefront: string | null = null;
/** Handlers for the popup that is open right now, if any. */
let openHandlers: PopupCheckoutHandlers | null = null;
/**
 * Set for the whole of an open attempt, including the script load it awaits.
 *
 * `openHandlers` alone could not do this job: it is only assigned *after* the
 * await, so two hooks mounting in the same tick both read it as null, both
 * loaded the script, and both went on to open. The second overwrote the first's
 * handlers and its watcher, and the first watcher — with the keyboard it had
 * taken — was left with no way to be released. Every app root stayed `inert`
 * for the rest of the page's life.
 *
 * A plain module-level boolean is enough because the browser is single
 * threaded: it is set and read in the same synchronous step as the guard.
 */
let openInFlight = false;
/** Tears down the DOM watch for the popup that is open right now, if any. */
let stopPopupWatch: (() => void) | null = null;

/**
 * Validates a storefront before it is put in a script tag — defence in depth,
 * on top of the backend deriving it from a validated checkout path.
 *
 * Three things are checked: the value is a plain `host/path` under
 * `onfastspring.com`; its test marker agrees with the environment the backend
 * says it is in; and the checkout it names is a popup checkout. The second check
 * is what stops a test storefront from quietly taking live money's place, or a
 * live storefront from charging real cards during testing. The third is what
 * stops SBL from silently falling back to navigating the whole page.
 */
export function isValidStorefront(
  storefront: string,
  environment: string,
): boolean {
  if (!STOREFRONT_PATTERN.test(storefront)) return false;

  const separator = storefront.indexOf("/");
  if (!storefront.slice(separator + 1).startsWith(POPUP_CHECKOUT_PREFIX)) {
    return false;
  }

  const host = storefront.slice(0, separator);
  const labels = host.split(".");
  const domainLabels = STOREFRONT_DOMAIN.split(".").length;
  // "<store>.onfastspring.com" is live; "<store>.test.onfastspring.com" is test.
  // Any other shape is neither, and is refused rather than guessed at.
  const isTestHost =
    labels.length === domainLabels + 2 &&
    labels[labels.length - domainLabels - 1] === TEST_STOREFRONT_LABEL;
  const isLiveHost = labels.length === domainLabels + 1;

  if (environment === "test") return isTestHost;
  if (environment === "live") return isLiveHost;
  return false;
}

/**
 * Hands the open popup's handlers to the caller exactly once and shuts the DOM
 * watch down. Every way a checkout can end goes through here, so the callback
 * and the watcher can never both report the same close.
 */
function takeOpenHandlers(): PopupCheckoutHandlers | null {
  const handlers = openHandlers;
  openHandlers = null;
  stopPopupWatch?.();
  stopPopupWatch = null;
  return handlers;
}

/** Routes SBL's global callbacks to whichever popup is open. */
function installGlobalCallbacks(): void {
  const globals = window as unknown as Record<string, unknown>;

  globals[POPUP_CLOSED_CALLBACK] = (order: unknown) => {
    takeOpenHandlers()?.onClose({ completed: namesAnOrder(order) });
  };

  globals[ERROR_CALLBACK] = () => {
    takeOpenHandlers()?.onError();
  };
}

/**
 * Whether an element is in the document and actually on screen.
 *
 * `element.style` only ever sees the inline attribute, and SBL dismisses its
 * popup by class at least as often as by inline style — so a container taken
 * off screen by a stylesheet rule read as still open, the watch kept waiting
 * for a close that had already happened, and the buyer who dismissed the
 * checkout met "Activating your subscription…" on their next page load. The
 * shared `isRendered` reads the computed value instead, and walks up, because
 * `display: none` on a wrapper hides the subtree without changing anything
 * computed on the container itself.
 */
function isDisplayed(element: Element): boolean {
  if (!element.isConnected) return false;
  if (element.getAttribute("aria-hidden") === "true") return false;
  if (!(element instanceof HTMLElement)) return true;
  return isRendered(element);
}

/**
 * SBL's popup as it currently stands on the page: every container of its that
 * is on screen, and the checkout frame among them.
 *
 * The frame is found rather than assumed present — the canvas is appended
 * before the iframe inside it exists, and the fallback selectors may match a
 * container with no frame at all.
 */
function findOverlay(): OverlayElements {
  const roots = Array.from(
    document.querySelectorAll<HTMLElement>(POPUP_SELECTORS),
  )
    .filter(belongsToPopup)
    .filter(isDisplayed);

  let frame: HTMLElement | null = null;
  for (const root of roots) {
    const candidate =
      root instanceof HTMLIFrameElement ? root : root.querySelector("iframe");
    if (candidate && isDisplayed(candidate)) {
      frame = candidate;
      break;
    }
  }

  return { roots, frame };
}

/**
 * Reports the popup closing when SBL does not.
 *
 * Closing the popup with its own X button does not fire `data-popup-closed` —
 * so nothing cleared the "a checkout is in flight" baseline, and the next mount
 * greeted a customer who had just declined to pay with a full-screen
 * "Activating your subscription…". The provider's callback is a convenience,
 * not something to depend on; this watches the DOM for the same event.
 *
 * A dismissal is only reported once the container has actually been seen, so a
 * popup that is still rendering is never mistaken for one that closed. Both a
 * MutationObserver and a poll are used: the observer catches removals promptly,
 * and the poll catches the case where the popup is hidden by a style change the
 * observer's subtree filter misses.
 */
function watchForPopupDismissal(returnFocusTo?: HTMLElement | null): void {
  // One watch at a time, and this is the only place one is installed. The
  // teardown below is the single owner of its observer, its interval and its
  // keyboard, so a watch that somehow outlived its checkout is released here
  // rather than orphaned with the page still inert.
  stopPopupWatch?.();
  stopPopupWatch = null;

  let seen = false;
  const startedAt = Date.now();
  // While the popup is up it owns the screen, so it has to own the keyboard
  // too: the app behind it is made unreachable and focus is moved into the
  // checkout frame. Released below, on every way this checkout can end, and
  // handed back to the control the buyer opened the checkout with.
  const keyboard = createOverlayKeyboard({ returnFocusTo });

  const check = () => {
    if (!openHandlers) return;
    const overlay = findOverlay();
    if (overlay.roots.length > 0) {
      seen = true;
      keyboard.handOver(overlay);
      return;
    }
    if (!seen) {
      // Nothing has rendered yet. Keep looking for a while, then give up on
      // the checkout entirely rather than poll for the rest of the page's
      // life — and, more importantly, rather than leave the caller waiting on
      // a popup that is never going to report anything.
      if (Date.now() - startedAt > POPUP_APPEARANCE_TIMEOUT_MS) {
        takeOpenHandlers()?.onError();
      }
      return;
    }
    takeOpenHandlers()?.onClose({ completed: false });
  };

  const observer =
    typeof MutationObserver === "undefined"
      ? null
      : new MutationObserver(check);
  observer?.observe(document.body, {
    childList: true,
    subtree: true,
    attributes: true,
    attributeFilter: ["style", "class", "hidden", "aria-hidden"],
  });

  const intervalId = window.setInterval(check, POPUP_WATCH_INTERVAL_MS);

  stopPopupWatch = () => {
    observer?.disconnect();
    window.clearInterval(intervalId);
    keyboard.release();
  };

  // The container is injected asynchronously, so take a first look now and let
  // the observer and the poll carry it from there.
  check();
}

/**
 * Reports whether the close payload names an order at all.
 *
 * SBL's contract is a presence test, not a shape test: it calls the callback
 * with the popup's order references when the checkout concluded on an order, and
 * with `null` when the buyer simply closed the window. The exact fields of that
 * payload are not documented, so reading into them would be guessing — and
 * guessing wrong the *other* way would drop a real purchase on the floor.
 *
 * That single boolean is everything this module takes from it. The payload is
 * never logged and never inspected further: it is client-side data about
 * someone's purchase, and entitlement comes from the webhook, not from here.
 */
function namesAnOrder(order: unknown): boolean {
  return order !== null && order !== undefined;
}

/**
 * Inserts the SBL script once and resolves when it is ready.
 *
 * The callbacks are installed *before* insertion: SBL may fire them as soon as
 * it initialises, and a name it cannot resolve is simply a lost event.
 */
function loadScript(storefront: string): Promise<void> {
  if (scriptLoad) {
    if (loadedStorefront !== storefront) {
      return Promise.reject(
        new Error("FastSpring storefront changed after the script was loaded"),
      );
    }
    return scriptLoad;
  }

  loadedStorefront = storefront;
  installGlobalCallbacks();

  scriptLoad = new Promise<void>((resolve, reject) => {
    const script = document.createElement("script");
    script.id = SBL_SCRIPT_ID;
    script.type = "text/javascript";
    script.src = SBL_SRC;
    // Request hygiene, not a fork of the provider's loader: the id, the src and
    // the data-attributes below are exactly the ones FastSpring's own snippet
    // uses, and SBL reads nothing else off its tag.
    //
    // `anonymous` fetches the script in CORS mode with no credentials, so none
    // of the provider's cookies ride along with a static asset that has no use
    // for them. Its CDN is built for cross-origin embedding — it answers
    // `access-control-allow-origin: *` — and a CORS refusal would surface as an
    // ordinary load error, which this function already turns into a clean
    // failure the buyer can retry.
    script.crossOrigin = "anonymous";
    // The URL of the page somebody is buying from is not a static file host's
    // business. The storefront it needs is on the tag, not in the referrer.
    script.referrerPolicy = "no-referrer";
    script.dataset.storefront = storefront;
    script.dataset.popupClosed = POPUP_CLOSED_CALLBACK;
    script.dataset.errorCallback = ERROR_CALLBACK;

    const timeoutId = window.setTimeout(() => {
      fail(new Error("FastSpring checkout script timed out"));
    }, SCRIPT_LOAD_TIMEOUT_MS);

    function cleanUp() {
      window.clearTimeout(timeoutId);
      script.onload = null;
      script.onerror = null;
    }

    function fail(cause: Error) {
      cleanUp();
      script.remove();
      // Forget the failed attempt so the next click loads the script again
      // instead of replaying this rejection.
      scriptLoad = null;
      loadedStorefront = null;
      reject(cause);
    }

    script.onload = () => {
      cleanUp();
      if (!window.fastspring?.builder) {
        fail(new Error("FastSpring checkout script loaded without a builder"));
        return;
      }
      resolve();
    };
    script.onerror = () =>
      fail(new Error("FastSpring checkout script failed to load"));

    document.head.appendChild(script);
  });

  return scriptLoad;
}

/**
 * Opens the FastSpring popup on a session the backend already created.
 *
 * Rejects — without opening anything — when the storefront is not trustworthy,
 * when the script cannot be loaded, or when a popup is already open. The caller
 * may simply call again after a failure: a failed load is not cached.
 */
export async function openPopupCheckout({
  storefront,
  environment,
  sessionId,
  handlers,
  returnFocusTo,
}: OpenPopupCheckoutOptions): Promise<void> {
  if (!isValidStorefront(storefront, environment)) {
    throw new Error("FastSpring storefront is not valid for this environment");
  }
  if (!sessionId) {
    throw new Error("FastSpring checkout session carried no session id");
  }
  // Claimed before the first await and released only in the `finally` below,
  // so the loser of a two-click race is refused here rather than a hundred
  // milliseconds later, halfway through taking the keyboard off the page.
  if (openHandlers || openInFlight) {
    throw new Error("A FastSpring checkout popup is already open");
  }
  openInFlight = true;

  try {
    await loadScript(storefront);

    const builder = window.fastspring?.builder;
    if (!builder) {
      throw new Error("FastSpring checkout script is not ready");
    }

    openHandlers = handlers;
    try {
      // The session id *is* the cart. No product path, no price and no buyer
      // detail is assembled in the browser.
      builder.push({ checkout: sessionId });
    } catch (cause) {
      takeOpenHandlers();
      throw cause instanceof Error ? cause : new Error(String(cause));
    }

    // `push` can resolve the whole checkout synchronously — an immediate error
    // callback, or a session the provider rejects outright — in which case the
    // handlers have already been taken and this checkout is over. Starting a
    // watch for it would leave an interval and a MutationObserver running with
    // nothing left to report to.
    if (openHandlers === handlers) {
      watchForPopupDismissal(returnFocusTo);
    }
  } finally {
    // Every terminal path — opened, refused, thrown, or concluded inside
    // `push` — hands the latch back. `openHandlers` is what keeps a *second*
    // checkout out from here on; leaving this set as well would refuse the
    // next click forever.
    openInFlight = false;
  }
}

/** Test seam: drops the module's memory of the script and any open popup. */
export function resetSblForTests(): void {
  scriptLoad = null;
  loadedStorefront = null;
  openHandlers = null;
  openInFlight = false;
  stopPopupWatch?.();
  stopPopupWatch = null;
}
