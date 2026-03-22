import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";

vi.mock("sonner", () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
  },
}));

import { toast } from "sonner";
import {
  showSuccessNotification,
  showErrorNotification,
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
