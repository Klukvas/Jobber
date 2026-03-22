import { describe, it, expect, beforeEach } from "vitest";
import { act } from "@testing-library/react";
import { useThemeStore } from "../themeStore";

describe("themeStore", () => {
  beforeEach(() => {
    act(() => {
      useThemeStore.setState({ theme: "light" });
    });
  });

  describe("initial state", () => {
    it("defaults to light theme", () => {
      expect(useThemeStore.getState().theme).toBe("light");
    });
  });

  describe("setTheme", () => {
    it("changes theme to dark", () => {
      act(() => {
        useThemeStore.getState().setTheme("dark");
      });

      expect(useThemeStore.getState().theme).toBe("dark");
    });

    it("changes theme to light", () => {
      act(() => {
        useThemeStore.getState().setTheme("dark");
      });

      act(() => {
        useThemeStore.getState().setTheme("light");
      });

      expect(useThemeStore.getState().theme).toBe("light");
    });

    it("sets the same theme without error", () => {
      act(() => {
        useThemeStore.getState().setTheme("light");
      });

      expect(useThemeStore.getState().theme).toBe("light");
    });
  });

  describe("toggleTheme", () => {
    it("switches from light to dark", () => {
      act(() => {
        useThemeStore.getState().toggleTheme();
      });

      expect(useThemeStore.getState().theme).toBe("dark");
    });

    it("switches from dark to light", () => {
      act(() => {
        useThemeStore.getState().setTheme("dark");
      });

      act(() => {
        useThemeStore.getState().toggleTheme();
      });

      expect(useThemeStore.getState().theme).toBe("light");
    });

    it("toggles back and forth correctly", () => {
      act(() => {
        useThemeStore.getState().toggleTheme();
      });
      expect(useThemeStore.getState().theme).toBe("dark");

      act(() => {
        useThemeStore.getState().toggleTheme();
      });
      expect(useThemeStore.getState().theme).toBe("light");

      act(() => {
        useThemeStore.getState().toggleTheme();
      });
      expect(useThemeStore.getState().theme).toBe("dark");
    });
  });
});
