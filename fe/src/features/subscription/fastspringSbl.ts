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
}

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

/** Routes SBL's global callbacks to whichever popup is open. */
function installGlobalCallbacks(): void {
  const globals = window as unknown as Record<string, unknown>;

  globals[POPUP_CLOSED_CALLBACK] = (order: unknown) => {
    const handlers = openHandlers;
    openHandlers = null;
    handlers?.onClose({ completed: namesAnOrder(order) });
  };

  globals[ERROR_CALLBACK] = () => {
    const handlers = openHandlers;
    openHandlers = null;
    handlers?.onError();
  };
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
}: OpenPopupCheckoutOptions): Promise<void> {
  if (!isValidStorefront(storefront, environment)) {
    throw new Error("FastSpring storefront is not valid for this environment");
  }
  if (!sessionId) {
    throw new Error("FastSpring checkout session carried no session id");
  }
  if (openHandlers) {
    throw new Error("A FastSpring checkout popup is already open");
  }

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
    openHandlers = null;
    throw cause instanceof Error ? cause : new Error(String(cause));
  }
}

/** Test seam: drops the module's memory of the script and any open popup. */
export function resetSblForTests(): void {
  scriptLoad = null;
  loadedStorefront = null;
  openHandlers = null;
}
