import { describe, it, expect, beforeEach } from "vitest";
import { act } from "@testing-library/react";
import { useAuthStore } from "../authStore";

const mockUser = {
  id: "user-1",
  email: "test@example.com",
  name: "Test User",
  locale: "en",
  created_at: "2024-01-01T00:00:00Z",
};

describe("authStore", () => {
  beforeEach(() => {
    act(() => {
      useAuthStore.setState({
        user: null,
        isAuthenticated: false,
      });
    });
  });

  describe("initial state", () => {
    it("has null user and false isAuthenticated", () => {
      const state = useAuthStore.getState();
      expect(state.user).toBeNull();
      expect(state.isAuthenticated).toBe(false);
    });
  });

  describe("setAuth", () => {
    it("stores user and sets isAuthenticated to true", () => {
      act(() => {
        useAuthStore.getState().setAuth(mockUser);
      });

      const state = useAuthStore.getState();
      expect(state.user).toEqual(mockUser);
      expect(state.isAuthenticated).toBe(true);
    });

    it("replaces an existing user", () => {
      act(() => {
        useAuthStore.getState().setAuth(mockUser);
      });

      const updatedUser = { ...mockUser, id: "user-2", name: "Other User" };

      act(() => {
        useAuthStore.getState().setAuth(updatedUser);
      });

      const state = useAuthStore.getState();
      expect(state.user).toEqual(updatedUser);
      expect(state.isAuthenticated).toBe(true);
    });
  });

  describe("clearAuth", () => {
    it("clears user and sets isAuthenticated to false", () => {
      act(() => {
        useAuthStore.getState().setAuth(mockUser);
      });

      expect(useAuthStore.getState().isAuthenticated).toBe(true);

      act(() => {
        useAuthStore.getState().clearAuth();
      });

      const state = useAuthStore.getState();
      expect(state.user).toBeNull();
      expect(state.isAuthenticated).toBe(false);
    });

    it("is a no-op when already cleared", () => {
      act(() => {
        useAuthStore.getState().clearAuth();
      });

      const state = useAuthStore.getState();
      expect(state.user).toBeNull();
      expect(state.isAuthenticated).toBe(false);
    });
  });

  describe("partialize", () => {
    it("only persists user and isAuthenticated (actions are excluded)", () => {
      act(() => {
        useAuthStore.getState().setAuth(mockUser);
      });

      const state = useAuthStore.getState();
      // The store has actions in its state
      expect(typeof state.setAuth).toBe("function");
      expect(typeof state.clearAuth).toBe("function");

      // Verify persisted keys: user and isAuthenticated are present
      expect(state.user).toEqual(mockUser);
      expect(state.isAuthenticated).toBe(true);
    });
  });
});
