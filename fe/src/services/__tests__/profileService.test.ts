import { describe, it, expect, vi, beforeEach } from "vitest";

const mockApiClient = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  patch: vi.fn(),
  delete: vi.fn(),
}));

vi.mock("@/services/api", () => ({ apiClient: mockApiClient }));

import { profileService } from "../profileService";

describe("profileService", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("reads the caller's own profile", async () => {
    const user = { id: "u1", email: "a@b.c", name: "Alex", locale: "en" };
    mockApiClient.get.mockResolvedValue(user);

    await expect(profileService.get()).resolves.toEqual(user);
    expect(mockApiClient.get).toHaveBeenCalledWith("profile", undefined);
  });

  // React Query hands every queryFn a signal; passing it on is what makes an
  // unmount or a key change actually cancel the request.
  it("forwards the caller's cancellation signal", async () => {
    const { signal } = new AbortController();
    mockApiClient.get.mockResolvedValue({});

    await profileService.get({ signal });

    expect(mockApiClient.get).toHaveBeenCalledWith("profile", { signal });
  });

  // The account is the token's. Sending an id would be meaningless at best and
  // misleading at worst, so the client never assembles one.
  it("addresses the caller and nothing else", async () => {
    mockApiClient.get.mockResolvedValue({});

    await profileService.get();

    expect(mockApiClient.get.mock.calls[0][0]).toBe("profile");
  });

  // `GET /profile` is the whole API surface. A write method here would have to
  // exist on the server too, and it does not.
  it("exposes no way to write a profile", () => {
    expect(Object.keys(profileService)).toEqual(["get"]);
  });
});
