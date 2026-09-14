import { toast } from "sonner";
// The i18next singleton, NOT "@/shared/lib/i18n": importing the app's setup
// module here would drag react-i18next's initReactI18next into every module
// that shows a toast, and with it into every unit test that mocks
// react-i18next. This is the same instance that module initialises at boot.
import i18n from "i18next";

// Wording an HTTP client produces about its own attempt: a status line, a
// transport failure, a stack frame, a method-and-URL pair. None of it is
// something a customer can act on, and it names our own infrastructure.
const INTERNAL_MESSAGE_PATTERNS = [
  /localhost(:\d+)?/i,
  /\bstatus code\b/i,
  /\bfailed to fetch\b/i,
  /\bnetworkerror\b/i,
  /\bat\s+\w+\s+\(/,
  /\b(GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS)\s+https?:\/\//,
];

const URL_IN_TEXT = /\bhttps?:\/\/[^\s<>"']+/gi;

/**
 * Whether a URL names a site on the public web, as opposed to a piece of our
 * own plumbing.
 *
 * The blanket "any URL is internal" rule this replaces was redacting messages
 * the customer had a right to read — an unreachable job posting, an external
 * resume link the server refused, anything quoting an address the customer
 * typed themselves — and replacing them with "Something went wrong", which
 * says nothing about which link was the problem.
 */
function namesAPublicSite(url: string): boolean {
  try {
    const { hostname } = new URL(url);
    if (hostname === "localhost" || hostname.endsWith(".localhost")) {
      return false;
    }
    // A bare address, in either family, is infrastructure and not a website.
    if (hostname.startsWith("[")) return false;
    if (/^\d{1,3}(\.\d{1,3}){3}$/.test(hostname)) return false;
    // A single-label host is a container or a service name, never a site.
    return hostname.includes(".");
  } catch {
    return false;
  }
}

const GENERIC_ERROR_KEY = "errors.somethingWentWrong";
/** Used only where i18n has not booted — unit tests, or a very early failure. */
const GENERIC_ERROR_FALLBACK = "Something went wrong";

/**
 * Last line of defence for user-visible error text: a blank or internal-looking
 * message is replaced with a generic localized one. Exported for tests.
 */
export function toUserFacingMessage(message: string): string {
  const trimmed = message?.trim() ?? "";
  const urls = trimmed.match(URL_IN_TEXT) ?? [];
  const isInternal =
    trimmed === "" ||
    INTERNAL_MESSAGE_PATTERNS.some((pattern) => pattern.test(trimmed)) ||
    urls.some((url) => !namesAPublicSite(url)) ||
    // A message that is nothing but an address is not copy anybody wrote for
    // a customer — it is an endpoint that escaped a client library.
    (urls.length === 1 && urls[0] === trimmed);
  if (!isInternal) return trimmed;
  return i18n.isInitialized
    ? i18n.t(GENERIC_ERROR_KEY)
    : GENERIC_ERROR_FALLBACK;
}

// Toast Notifications using sonner
export function showSuccessNotification(message: string): void {
  toast.success(message);
}

export function showErrorNotification(message: string): void {
  toast.error(toUserFacingMessage(message));
}

export function showInfoNotification(message: string): void {
  toast.info(message);
}

// Push Notifications utility functions

export async function requestNotificationPermission(): Promise<NotificationPermission> {
  if (!("Notification" in window)) {
    console.warn("This browser does not support notifications");
    return "denied";
  }

  if (Notification.permission === "granted") {
    return "granted";
  }

  if (Notification.permission !== "denied") {
    const permission = await Notification.requestPermission();
    return permission;
  }

  return Notification.permission;
}

export async function registerServiceWorker(): Promise<ServiceWorkerRegistration | null> {
  if (!("serviceWorker" in navigator)) {
    console.warn("Service Workers are not supported");
    return null;
  }

  try {
    const registration = await navigator.serviceWorker.register("/sw.js");
    return registration;
  } catch (error) {
    console.error("Service Worker registration failed:", error);
    return null;
  }
}

export async function subscribeToPushNotifications(
  registration: ServiceWorkerRegistration,
  vapidPublicKey: string,
): Promise<PushSubscription | null> {
  try {
    const applicationServerKey = urlBase64ToUint8Array(vapidPublicKey);
    const subscription = await registration.pushManager.subscribe({
      userVisibleOnly: true,
      applicationServerKey: applicationServerKey as BufferSource,
    });

    // Send this subscription to your backend
    return subscription;
  } catch (error) {
    console.error("Failed to subscribe to push notifications:", error);
    return null;
  }
}

function urlBase64ToUint8Array(base64String: string): Uint8Array {
  const padding = "=".repeat((4 - (base64String.length % 4)) % 4);
  const base64 = (base64String + padding).replace(/-/g, "+").replace(/_/g, "/");

  const rawData = window.atob(base64);
  const outputArray = new Uint8Array(rawData.length);

  for (let i = 0; i < rawData.length; ++i) {
    outputArray[i] = rawData.charCodeAt(i);
  }
  return outputArray;
}

export async function initializePushNotifications(): Promise<void> {
  // Request permission
  const permission = await requestNotificationPermission();
  if (permission !== "granted") {
    return;
  }

  // Register service worker
  const registration = await registerServiceWorker();
  if (!registration) {
    return;
  }

  // Note: In production, you would get the VAPID public key from your backend
  // and subscribe to push notifications here
}
