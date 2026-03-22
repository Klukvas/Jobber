import { describe, it, expect, vi, beforeEach } from "vitest";

const mockApiClient = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  patch: vi.fn(),
  put: vi.fn(),
  delete: vi.fn(),
  postFormData: vi.fn(),
}));

vi.mock("@/services/api", () => ({
  apiClient: mockApiClient,
}));

import { authService } from "../authService";

describe("authService", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("login", () => {
    it("calls POST auth/login with credentials", async () => {
      const mockResponse = {
        user: { id: "u1", email: "user@example.com" },
        tokens: {
          access_token: "at",
          refresh_token: "rt",
          expires_in: 3600,
        },
      };
      mockApiClient.post.mockResolvedValue(mockResponse);

      const result = await authService.login({
        email: "user@example.com",
        password: "password123",
      });

      expect(mockApiClient.post).toHaveBeenCalledWith("auth/login", {
        email: "user@example.com",
        password: "password123",
      });
      expect(result).toEqual(mockResponse);
    });

    it("propagates errors from apiClient", async () => {
      mockApiClient.post.mockRejectedValue(new Error("Network error"));

      await expect(
        authService.login({ email: "a@b.com", password: "pass1234" }),
      ).rejects.toThrow("Network error");
    });
  });

  describe("register", () => {
    it("calls POST auth/register with registration data", async () => {
      const mockResponse = { message: "Check your email" };
      mockApiClient.post.mockResolvedValue(mockResponse);

      const result = await authService.register({
        email: "new@example.com",
        password: "securepass",
      });

      expect(mockApiClient.post).toHaveBeenCalledWith("auth/register", {
        email: "new@example.com",
        password: "securepass",
      });
      expect(result).toEqual(mockResponse);
    });

    it("passes optional locale to register", async () => {
      mockApiClient.post.mockResolvedValue({ message: "ok" });

      await authService.register({
        email: "new@example.com",
        password: "securepass",
        locale: "en",
      });

      expect(mockApiClient.post).toHaveBeenCalledWith("auth/register", {
        email: "new@example.com",
        password: "securepass",
        locale: "en",
      });
    });
  });

  describe("refresh", () => {
    it("calls POST auth/refresh with refresh token", async () => {
      const mockTokens = {
        access_token: "new_at",
        refresh_token: "new_rt",
        expires_in: 3600,
      };
      mockApiClient.post.mockResolvedValue(mockTokens);

      const result = await authService.refresh({
        refresh_token: "old_rt",
      });

      expect(mockApiClient.post).toHaveBeenCalledWith("auth/refresh", {
        refresh_token: "old_rt",
      });
      expect(result).toEqual(mockTokens);
    });
  });

  describe("logout", () => {
    it("calls POST auth/logout without data", async () => {
      mockApiClient.post.mockResolvedValue(undefined);

      await authService.logout();

      expect(mockApiClient.post).toHaveBeenCalledWith("auth/logout");
    });

    it("calls POST exactly once", async () => {
      mockApiClient.post.mockResolvedValue(undefined);

      await authService.logout();

      expect(mockApiClient.post).toHaveBeenCalledTimes(1);
    });
  });

  describe("verifyEmail", () => {
    it("calls POST auth/verify-email with email and code", async () => {
      const mockResponse = { message: "Email verified" };
      mockApiClient.post.mockResolvedValue(mockResponse);

      const result = await authService.verifyEmail({
        email: "user@example.com",
        code: "123456",
      });

      expect(mockApiClient.post).toHaveBeenCalledWith("auth/verify-email", {
        email: "user@example.com",
        code: "123456",
      });
      expect(result).toEqual(mockResponse);
    });
  });

  describe("resendVerification", () => {
    it("calls POST auth/resend-verification with email", async () => {
      const mockResponse = { message: "Verification sent" };
      mockApiClient.post.mockResolvedValue(mockResponse);

      const result = await authService.resendVerification({
        email: "user@example.com",
      });

      expect(mockApiClient.post).toHaveBeenCalledWith(
        "auth/resend-verification",
        { email: "user@example.com" },
      );
      expect(result).toEqual(mockResponse);
    });
  });

  describe("forgotPassword", () => {
    it("calls POST auth/forgot-password with email", async () => {
      const mockResponse = { message: "Reset link sent" };
      mockApiClient.post.mockResolvedValue(mockResponse);

      const result = await authService.forgotPassword({
        email: "user@example.com",
      });

      expect(mockApiClient.post).toHaveBeenCalledWith(
        "auth/forgot-password",
        { email: "user@example.com" },
      );
      expect(result).toEqual(mockResponse);
    });
  });

  describe("resetPassword", () => {
    it("calls POST auth/reset-password with email, code, and password", async () => {
      const mockResponse = { message: "Password reset" };
      mockApiClient.post.mockResolvedValue(mockResponse);

      const result = await authService.resetPassword({
        email: "user@example.com",
        code: "654321",
        password: "newpassword123",
      });

      expect(mockApiClient.post).toHaveBeenCalledWith(
        "auth/reset-password",
        {
          email: "user@example.com",
          code: "654321",
          password: "newpassword123",
        },
      );
      expect(result).toEqual(mockResponse);
    });

    it("propagates errors from apiClient", async () => {
      mockApiClient.post.mockRejectedValue(new Error("Invalid code"));

      await expect(
        authService.resetPassword({
          email: "user@example.com",
          code: "000000",
          password: "newpass12",
        }),
      ).rejects.toThrow("Invalid code");
    });
  });
});
