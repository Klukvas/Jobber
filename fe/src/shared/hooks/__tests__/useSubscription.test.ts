import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import React from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

const mockGetSubscription = vi.hoisted(() => vi.fn());

vi.mock("@/services/subscriptionService", () => ({
  subscriptionService: {
    getSubscription: mockGetSubscription,
  },
}));

import { useSubscription } from "../useSubscription";

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        gcTime: 0,
      },
    },
  });
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return React.createElement(
      QueryClientProvider,
      { client: queryClient },
      children,
    );
  };
}

describe("useSubscription", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("returns free plan defaults when no subscription data", async () => {
    mockGetSubscription.mockResolvedValue(undefined);

    const { result } = renderHook(() => useSubscription(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.isFree).toBe(true);
    });

    expect(result.current.plan).toBe("free");
    expect(result.current.isPro).toBe(false);
    expect(result.current.isEnterprise).toBe(false);
    expect(result.current.nextPlan).toBe("pro");
  });

  it("returns correct flags for pro plan", async () => {
    mockGetSubscription.mockResolvedValue({
      plan: "pro",
      limits: {
        max_jobs: -1,
        max_resumes: 10,
        max_applications: -1,
        max_ai_requests: 50,
        max_job_parses: -1,
        max_resume_builders: 5,
        max_cover_letters: 10,
      },
      usage: {
        jobs: 3,
        resumes: 2,
        applications: 5,
        ai_requests: 10,
        job_parses: 0,
        resume_builders: 1,
        cover_letters: 2,
      },
    });

    const { result } = renderHook(() => useSubscription(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.plan).toBe("pro");
    });

    expect(result.current.isPro).toBe(true);
    expect(result.current.isFree).toBe(false);
    expect(result.current.isEnterprise).toBe(false);
    expect(result.current.nextPlan).toBe("enterprise");
  });

  it("returns correct flags for enterprise plan", async () => {
    mockGetSubscription.mockResolvedValue({
      plan: "enterprise",
      limits: {
        max_jobs: -1,
        max_resumes: -1,
        max_applications: -1,
        max_ai_requests: -1,
        max_job_parses: -1,
        max_resume_builders: -1,
        max_cover_letters: -1,
      },
      usage: {
        jobs: 0,
        resumes: 0,
        applications: 0,
        ai_requests: 0,
        job_parses: 0,
        resume_builders: 0,
        cover_letters: 0,
      },
    });

    const { result } = renderHook(() => useSubscription(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.plan).toBe("enterprise");
    });

    expect(result.current.isPro).toBe(true);
    expect(result.current.isEnterprise).toBe(true);
    expect(result.current.isFree).toBe(false);
    expect(result.current.nextPlan).toBeNull();
  });

  describe("canCreate", () => {
    it("allows creation when usage is below limit", async () => {
      mockGetSubscription.mockResolvedValue({
        plan: "free",
        limits: {
          max_jobs: 5,
          max_resumes: 1,
          max_applications: 5,
          max_ai_requests: 1,
          max_job_parses: 5,
          max_resume_builders: 1,
          max_cover_letters: 0,
        },
        usage: {
          jobs: 2,
          resumes: 0,
          applications: 1,
          ai_requests: 0,
          job_parses: 0,
          resume_builders: 0,
          cover_letters: 0,
        },
      });

      const { result } = renderHook(() => useSubscription(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.subscription).toBeDefined();
      });

      expect(result.current.canCreate("jobs")).toBe(true);
      expect(result.current.canCreate("resumes")).toBe(true);
      expect(result.current.canCreate("applications")).toBe(true);
    });

    it("blocks creation when usage equals limit", async () => {
      mockGetSubscription.mockResolvedValue({
        plan: "free",
        limits: {
          max_jobs: 5,
          max_resumes: 1,
          max_applications: 5,
          max_ai_requests: 1,
          max_job_parses: 5,
          max_resume_builders: 1,
          max_cover_letters: 0,
        },
        usage: {
          jobs: 5,
          resumes: 1,
          applications: 5,
          ai_requests: 1,
          job_parses: 5,
          resume_builders: 1,
          cover_letters: 0,
        },
      });

      const { result } = renderHook(() => useSubscription(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.subscription).toBeDefined();
      });

      expect(result.current.canCreate("jobs")).toBe(false);
      expect(result.current.canCreate("resumes")).toBe(false);
      expect(result.current.canCreate("applications")).toBe(false);
      expect(result.current.canCreate("ai")).toBe(false);
      expect(result.current.canCreate("resume_builders")).toBe(false);
    });

    it("allows unlimited creation when limit is -1", async () => {
      mockGetSubscription.mockResolvedValue({
        plan: "enterprise",
        limits: {
          max_jobs: -1,
          max_resumes: -1,
          max_applications: -1,
          max_ai_requests: -1,
          max_job_parses: -1,
          max_resume_builders: -1,
          max_cover_letters: -1,
        },
        usage: {
          jobs: 100,
          resumes: 50,
          applications: 200,
          ai_requests: 999,
          job_parses: 100,
          resume_builders: 50,
          cover_letters: 100,
        },
      });

      const { result } = renderHook(() => useSubscription(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.subscription).toBeDefined();
      });

      expect(result.current.canCreate("jobs")).toBe(true);
      expect(result.current.canCreate("resumes")).toBe(true);
      expect(result.current.canCreate("applications")).toBe(true);
      expect(result.current.canCreate("ai")).toBe(true);
      expect(result.current.canCreate("resume_builders")).toBe(true);
      expect(result.current.canCreate("cover_letters")).toBe(true);
    });

    it("blocks cover_letters when max is 0", async () => {
      mockGetSubscription.mockResolvedValue({
        plan: "free",
        limits: {
          max_jobs: 5,
          max_resumes: 1,
          max_applications: 5,
          max_ai_requests: 0,
          max_job_parses: 5,
          max_resume_builders: 1,
          max_cover_letters: 0,
        },
        usage: {
          jobs: 0,
          resumes: 0,
          applications: 0,
          ai_requests: 0,
          job_parses: 0,
          resume_builders: 0,
          cover_letters: 0,
        },
      });

      const { result } = renderHook(() => useSubscription(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.subscription).toBeDefined();
      });

      expect(result.current.canCreate("cover_letters")).toBe(false);
    });

    it("blocks ai when max_ai_requests is 0", async () => {
      mockGetSubscription.mockResolvedValue({
        plan: "free",
        limits: {
          max_jobs: 5,
          max_resumes: 1,
          max_applications: 5,
          max_ai_requests: 0,
          max_job_parses: 5,
          max_resume_builders: 1,
          max_cover_letters: 0,
        },
        usage: {
          jobs: 0,
          resumes: 0,
          applications: 0,
          ai_requests: 0,
          job_parses: 0,
          resume_builders: 0,
          cover_letters: 0,
        },
      });

      const { result } = renderHook(() => useSubscription(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.subscription).toBeDefined();
      });

      expect(result.current.canCreate("ai")).toBe(false);
    });
  });

  it("provides default limits when subscription has no limits", async () => {
    mockGetSubscription.mockResolvedValue(undefined);

    const { result } = renderHook(() => useSubscription(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.isFree).toBe(true);
    });

    expect(result.current.limits).toEqual({
      max_jobs: 5,
      max_resumes: 1,
      max_applications: 5,
      max_ai_requests: 1,
      max_job_parses: 5,
      max_resume_builders: 1,
      max_cover_letters: 0,
    });
  });

  it("provides default usage when subscription has no usage", async () => {
    mockGetSubscription.mockResolvedValue(undefined);

    const { result } = renderHook(() => useSubscription(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.isFree).toBe(true);
    });

    expect(result.current.usage).toEqual({
      jobs: 0,
      resumes: 0,
      applications: 0,
      ai_requests: 0,
      job_parses: 0,
      resume_builders: 0,
      cover_letters: 0,
    });
  });
});
