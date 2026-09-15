import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";

vi.mock("sonner", () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
    info: vi.fn(),
  },
}));

import { toast } from "sonner";
import {
  showSuccessNotification,
  showErrorNotification,
  showInfoNotification,
  toUserFacingMessage,
  requestNotificationPermission,
  registerServiceWorker,
  subscribeToPushNotifications,
  initializePushNotifications,
} from "../notifications";

describe("showSuccessNotification", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("calls toast.success with the message", () => {
    showSuccessNotification("Item saved");
    expect(toast.success).toHaveBeenCalledWith("Item saved");
  });

  it("calls toast.success exactly once", () => {
    showSuccessNotification("Done");
    expect(toast.success).toHaveBeenCalledTimes(1);
  });
});

describe("showErrorNotification", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("calls toast.error with the message", () => {
    showErrorNotification("Something failed");
    expect(toast.error).toHaveBeenCalledWith("Something failed");
  });

  it("calls toast.error exactly once", () => {
    showErrorNotification("Oops");
    expect(toast.error).toHaveBeenCalledTimes(1);
  });
});

describe("requestNotificationPermission", () => {
  const originalNotification = globalThis.Notification;

  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    Object.defineProperty(globalThis, "Notification", {
      value: originalNotification,
      writable: true,
      configurable: true,
    });
  });

  it("returns 'denied' when Notification is not supported", async () => {
    const saved = globalThis.Notification;
    delete (globalThis as Record<string, unknown>).Notification;

    const result = await requestNotificationPermission();
    expect(result).toBe("denied");

    Object.defineProperty(globalThis, "Notification", {
      value: saved,
      writable: true,
      configurable: true,
    });
  });

  it("returns 'granted' when already granted", async () => {
    Object.defineProperty(globalThis, "Notification", {
      value: { permission: "granted", requestPermission: vi.fn() },
      writable: true,
      configurable: true,
    });

    const result = await requestNotificationPermission();
    expect(result).toBe("granted");
  });

  it("requests permission when status is 'default'", async () => {
    const mockRequest = vi.fn().mockResolvedValue("granted");
    Object.defineProperty(globalThis, "Notification", {
      value: { permission: "default", requestPermission: mockRequest },
      writable: true,
      configurable: true,
    });

    const result = await requestNotificationPermission();
    expect(mockRequest).toHaveBeenCalled();
    expect(result).toBe("granted");
  });

  it("returns 'denied' when user denies the permission request", async () => {
    const mockRequest = vi.fn().mockResolvedValue("denied");
    Object.defineProperty(globalThis, "Notification", {
      value: { permission: "default", requestPermission: mockRequest },
      writable: true,
      configurable: true,
    });

    const result = await requestNotificationPermission();
    expect(result).toBe("denied");
  });

  it("returns 'denied' when permission is already denied", async () => {
    Object.defineProperty(globalThis, "Notification", {
      value: { permission: "denied", requestPermission: vi.fn() },
      writable: true,
      configurable: true,
    });

    const result = await requestNotificationPermission();
    expect(result).toBe("denied");
  });
});

describe("registerServiceWorker", () => {
  const savedNavigator = globalThis.navigator;

  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    Object.defineProperty(globalThis, "navigator", {
      value: savedNavigator,
      writable: true,
      configurable: true,
    });
  });

  it("returns null when serviceWorker is not supported", async () => {
    Object.defineProperty(globalThis, "navigator", {
      value: {},
      writable: true,
      configurable: true,
    });

    const result = await registerServiceWorker();
    expect(result).toBeNull();
  });

  it("returns registration when serviceWorker registers successfully", async () => {
    const mockRegistration = { scope: "/" };
    Object.defineProperty(globalThis, "navigator", {
      value: {
        serviceWorker: {
          register: vi.fn().mockResolvedValue(mockRegistration),
        },
      },
      writable: true,
      configurable: true,
    });

    const result = await registerServiceWorker();
    expect(result).toBe(mockRegistration);
    expect(navigator.serviceWorker.register).toHaveBeenCalledWith("/sw.js");
  });

  it("returns null when serviceWorker registration fails", async () => {
    Object.defineProperty(globalThis, "navigator", {
      value: {
        serviceWorker: {
          register: vi.fn().mockRejectedValue(new Error("SW failed")),
        },
      },
      writable: true,
      configurable: true,
    });

    const result = await registerServiceWorker();
    expect(result).toBeNull();
  });
});

describe("subscribeToPushNotifications", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("subscribes with correct options and returns subscription", async () => {
    const mockSubscription = { endpoint: "https://push.example.com" };
    const mockSubscribe = vi.fn().mockResolvedValue(mockSubscription);
    const mockRegistration = {
      pushManager: { subscribe: mockSubscribe },
    } as unknown as ServiceWorkerRegistration;

    // Mock window.atob for base64 decoding
    const originalAtob = window.atob;
    window.atob = vi.fn().mockReturnValue("decoded");

    const result = await subscribeToPushNotifications(
      mockRegistration,
      "BEl62iUYgUivxIkv69yViEuiBIa",
    );

    expect(result).toBe(mockSubscription);
    expect(mockSubscribe).toHaveBeenCalledWith(
      expect.objectContaining({
        userVisibleOnly: true,
      }),
    );

    window.atob = originalAtob;
  });

  it("returns null when subscription fails", async () => {
    const mockSubscribe = vi
      .fn()
      .mockRejectedValue(new Error("Subscribe failed"));
    const mockRegistration = {
      pushManager: { subscribe: mockSubscribe },
    } as unknown as ServiceWorkerRegistration;

    const originalAtob = window.atob;
    window.atob = vi.fn().mockReturnValue("decoded");

    const result = await subscribeToPushNotifications(
      mockRegistration,
      "BEl62iUYgUivxIkv69yViEuiBIa",
    );

    expect(result).toBeNull();

    window.atob = originalAtob;
  });
});

describe("initializePushNotifications", () => {
  const savedNotification = globalThis.Notification;
  const savedNavigator = globalThis.navigator;

  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    Object.defineProperty(globalThis, "Notification", {
      value: savedNotification,
      writable: true,
      configurable: true,
    });
    Object.defineProperty(globalThis, "navigator", {
      value: savedNavigator,
      writable: true,
      configurable: true,
    });
  });

  it("does nothing when permission is not granted", async () => {
    Object.defineProperty(globalThis, "Notification", {
      value: { permission: "denied", requestPermission: vi.fn() },
      writable: true,
      configurable: true,
    });

    // Should not throw
    await initializePushNotifications();
  });

  it("stops when service worker registration fails", async () => {
    Object.defineProperty(globalThis, "Notification", {
      value: { permission: "granted", requestPermission: vi.fn() },
      writable: true,
      configurable: true,
    });
    Object.defineProperty(globalThis, "navigator", {
      value: {},
      writable: true,
      configurable: true,
    });

    // Should not throw even though SW is not supported
    await initializePushNotifications();
  });

  it("completes successfully when permission is granted and SW registers", async () => {
    Object.defineProperty(globalThis, "Notification", {
      value: { permission: "granted", requestPermission: vi.fn() },
      writable: true,
      configurable: true,
    });
    Object.defineProperty(globalThis, "navigator", {
      value: {
        serviceWorker: {
          register: vi.fn().mockResolvedValue({ scope: "/" }),
        },
      },
      writable: true,
      configurable: true,
    });

    // Should complete without errors
    await initializePushNotifications();
  });
});

// The API client hands over blank messages for anything it could not quote
// safely, and libraries occasionally leak their own wording. Neither may reach
// a customer.
describe("toUserFacingMessage", () => {
  it("passes a real, customer-readable message through untouched", () => {
    expect(toUserFacingMessage("You have reached your plan limit.")).toBe(
      "You have reached your plan limit.",
    );
  });

  it("trims surrounding whitespace", () => {
    expect(toUserFacingMessage("  Limit reached  ")).toBe("Limit reached");
  });

  it.each([
    ["an empty message", ""],
    ["whitespace only", "   "],
    ["a fetch failure", "Failed to fetch"],
    ["a status line with a URL", "Request failed with status code 404 Not Found: POST http://localhost:8080/api/v1/support"],
    ["a bare backend URL", "https://api.jobber-app.com/api/v1/jobs"],
    ["a localhost origin", "connect ECONNREFUSED localhost:8080"],
    ["a stack frame", "TypeError at handleSubmit (Jobs.tsx:12)"],
  ])("replaces %s with a generic line", (_label, message) => {
    const result = toUserFacingMessage(message);

    expect(result).not.toBe(message);
    expect(result).not.toMatch(/https?:\/\//);
    expect(result).not.toMatch(/localhost/i);
    expect(result).not.toMatch(/status code/i);
    expect(result.trim()).not.toBe("");
  });

  // The blanket "any URL is internal" rule was throwing away the one detail
  // these messages exist to carry: which link the customer needs to fix.
  it.each([
    [
      "an unreachable posting the customer pasted",
      "We could not read https://jobs.example.com/postings/42 — check the link.",
    ],
    [
      "a refused resume link",
      "That file URL is not reachable: https://drive.example.com/file/abc",
    ],
  ])("keeps %s intact", (_label, message) => {
    expect(toUserFacingMessage(message)).toBe(message);
  });

  it("still redacts a private address inside an otherwise readable sentence", () => {
    const result = toUserFacingMessage(
      "We could not read http://10.0.0.7:8080/internal — check the link.",
    );

    expect(result).not.toContain("10.0.0.7");
  });
});

describe("showErrorNotification hygiene", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("never sends a raw transport message to the toast", () => {
    showErrorNotification(
      "Request failed with status code 500: GET http://localhost:8080/api/v1/jobs",
    );

    const shown = (toast.error as unknown as { mock: { calls: string[][] } })
      .mock.calls[0][0];
    expect(shown).not.toContain("localhost");
    expect(shown).not.toContain("status code");
  });

  it("still shows a message the API wrote for the customer", () => {
    showErrorNotification("Cover letters are not available on your plan.");
    expect(toast.error).toHaveBeenCalledWith(
      "Cover letters are not available on your plan.",
    );
  });
});

describe("showInfoNotification", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("uses the neutral toast, not the success one", () => {
    showInfoNotification("We are confirming your payment");

    expect(toast.info).toHaveBeenCalledWith("We are confirming your payment");
    expect(toast.success).not.toHaveBeenCalled();
  });
});
