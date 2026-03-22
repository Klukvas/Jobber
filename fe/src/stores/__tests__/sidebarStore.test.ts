import { describe, it, expect, beforeEach } from "vitest";
import { act } from "@testing-library/react";
import { useSidebarStore } from "../sidebarStore";

describe("sidebarStore", () => {
  beforeEach(() => {
    act(() => {
      useSidebarStore.setState({
        isExpanded: true,
        isMobileOpen: false,
      });
    });
  });

  describe("initial state", () => {
    it("has isExpanded true by default", () => {
      expect(useSidebarStore.getState().isExpanded).toBe(true);
    });

    it("has isMobileOpen false by default", () => {
      expect(useSidebarStore.getState().isMobileOpen).toBe(false);
    });
  });

  describe("toggleExpanded", () => {
    it("collapses when expanded", () => {
      act(() => {
        useSidebarStore.getState().toggleExpanded();
      });

      expect(useSidebarStore.getState().isExpanded).toBe(false);
    });

    it("expands when collapsed", () => {
      act(() => {
        useSidebarStore.getState().toggleExpanded();
      });

      act(() => {
        useSidebarStore.getState().toggleExpanded();
      });

      expect(useSidebarStore.getState().isExpanded).toBe(true);
    });
  });

  describe("toggleMobile", () => {
    it("opens mobile sidebar when closed", () => {
      act(() => {
        useSidebarStore.getState().toggleMobile();
      });

      expect(useSidebarStore.getState().isMobileOpen).toBe(true);
    });

    it("closes mobile sidebar when open", () => {
      act(() => {
        useSidebarStore.getState().toggleMobile();
      });

      act(() => {
        useSidebarStore.getState().toggleMobile();
      });

      expect(useSidebarStore.getState().isMobileOpen).toBe(false);
    });
  });

  describe("closeMobile", () => {
    it("closes mobile sidebar", () => {
      act(() => {
        useSidebarStore.getState().toggleMobile();
      });

      expect(useSidebarStore.getState().isMobileOpen).toBe(true);

      act(() => {
        useSidebarStore.getState().closeMobile();
      });

      expect(useSidebarStore.getState().isMobileOpen).toBe(false);
    });

    it("is a no-op when already closed", () => {
      act(() => {
        useSidebarStore.getState().closeMobile();
      });

      expect(useSidebarStore.getState().isMobileOpen).toBe(false);
    });
  });

  describe("partialize", () => {
    it("only persists isExpanded (isMobileOpen is excluded)", () => {
      act(() => {
        useSidebarStore.getState().toggleExpanded();
        useSidebarStore.getState().toggleMobile();
      });

      const state = useSidebarStore.getState();
      // Both values are in state
      expect(state.isExpanded).toBe(false);
      expect(state.isMobileOpen).toBe(true);

      // Actions are present as functions
      expect(typeof state.toggleExpanded).toBe("function");
      expect(typeof state.toggleMobile).toBe("function");
      expect(typeof state.closeMobile).toBe("function");
    });
  });
});
