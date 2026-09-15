import { Navigate, type RouteObject } from "react-router-dom";
import { sentryCreateBrowserRouter } from "@/shared/lib/sentry";
import { LegacyApplicationDetailRedirect } from "./LegacyApplicationRedirect";
import { RootLayout } from "./layouts/RootLayout";
import { LandingLayout } from "./layouts/LandingLayout";
import { AppLayout } from "./layouts/AppLayout";

import { lazy, Suspense } from "react";

// Public pages — eager so the initial HTML render carries full SEO content
// (no Suspense fallback flash, no JS chunk fetch on crawl).
import HomePage from "@/pages/Home";
import BlogPage from "@/pages/Blog";
import BlogPostPage from "@/pages/BlogPost";
import PrivacyPage from "@/pages/Privacy";
import TermsPage from "@/pages/Terms";
import RefundPage from "@/pages/Refund";
import FeatureApplicationsPage from "@/pages/FeatureApplications";
import FeatureResumeBuilderPage from "@/pages/FeatureResumeBuilder";
import FeatureCoverLettersPage from "@/pages/FeatureCoverLetters";

// Print page (no auth, no layout — used by headless Chrome for PDF export)
const ResumeBuilderPrintPage = lazy(() => import("@/pages/ResumeBuilderPrint"));

// Protected pages — kept lazy
const SettingsPage = lazy(() => import("@/pages/Settings"));
const ResumesPage = lazy(() => import("@/pages/Resumes"));
const CompaniesPage = lazy(() => import("@/pages/Companies"));
const JobsPage = lazy(() => import("@/pages/Jobs"));
const JobDetailPage = lazy(() => import("@/pages/JobDetail"));
const StageTemplatesPage = lazy(() => import("@/pages/StageTemplates"));
const AnalyticsPage = lazy(() => import("@/pages/Analytics"));
const ResumeBuilderEditorPage = lazy(
  () => import("@/pages/ResumeBuilderEditor"),
);
const CoverLettersPage = lazy(() => import("@/pages/CoverLetters"));
const CoverLetterEditorPage = lazy(() => import("@/pages/CoverLetterEditor"));
const NotFoundPage = lazy(() => import("@/pages/NotFound"));

// Password reset from a `?email=&code=` URL (no auth, noindex).
const ResetPasswordPage = lazy(() => import("@/pages/ResetPassword"));

// Public shared-stats page (no auth; opened from links posted on social media)
const SharedStatsPage = lazy(() => import("@/pages/SharedStats"));

/**
 * The route table itself, apart from the router built from it — the structure
 * is what keeps the landing page mounted across an auth modal, so it is worth
 * asserting on directly.
 */
export const routes: RouteObject[] = [
  {
    path: "/print/resume",
    element: (
      <Suspense fallback={<div />}>
        <ResumeBuilderPrintPage />
      </Suspense>
    ),
  },
  {
    path: "/",
    element: <RootLayout />,
    children: [
      {
        // The landing page and the three modal routes drawn over it are
        // siblings on purpose: same element, same depth, so opening or closing
        // an auth modal re-renders the page instead of rebuilding it. See
        // LandingLayout for what depended on that.
        path: "",
        element: <LandingLayout />,
        children: [
          {
            index: true,
            element: <HomePage />,
          },
          {
            // Login modal is shown on Home page based on URL
            path: "login",
            element: <HomePage />,
          },
          {
            // Register modal is shown on Home page based on URL
            path: "register",
            element: <HomePage />,
          },
          {
            // Forgot password modal is shown on Home page based on URL
            path: "forgot-password",
            element: <HomePage />,
          },
        ],
      },
      {
        path: "verify-email",
        element: <Navigate to="/" replace />,
      },
      {
        // A working entry point, not a redirect. The reset email carries a
        // 6-digit code rather than a link, so the modal on the landing page is
        // the usual way through — but the API takes `email` + `code` + the new
        // password, and this page is where a `?email=&code=` URL lands. It
        // redirected to `/` before, which dropped anyone who arrived with a
        // valid code at a page that could not use it.
        path: "reset-password",
        element: (
          <Suspense fallback={<div />}>
            <ResetPasswordPage />
          </Suspense>
        ),
      },
      {
        path: "blog",
        element: <BlogPage />,
      },
      {
        path: "blog/:slug",
        element: <BlogPostPage />,
      },
      {
        path: "privacy",
        element: <PrivacyPage />,
      },
      {
        path: "terms",
        element: <TermsPage />,
      },
      {
        path: "refund",
        element: <RefundPage />,
      },
      {
        path: "features",
        children: [
          {
            // "/features" has no page of its own. Without this the route
            // matched a layout-less branch and rendered a blank, indexable
            // document; sending it to the first feature page keeps the URL
            // useful and lets that page own the canonical/robots tags.
            index: true,
            element: <Navigate to="/features/applications" replace />,
          },
          {
            path: "applications",
            element: <FeatureApplicationsPage />,
          },
          {
            path: "resume-builder",
            element: <FeatureResumeBuilderPage />,
          },
          {
            path: "cover-letters",
            element: <FeatureCoverLettersPage />,
          },
        ],
      },
      {
        path: "s/:token",
        element: <SharedStatsPage />,
      },
      {
        path: "app",
        element: <AppLayout />,
        children: [
          {
            index: true,
            element: <Navigate to="/app/jobs" replace />,
          },
          {
            path: "applications",
            element: <Navigate to="/app/jobs" replace />,
          },
          {
            path: "applications/:id",
            element: <LegacyApplicationDetailRedirect />,
          },
          {
            path: "resumes",
            element: <ResumesPage />,
          },
          {
            path: "companies",
            element: <CompaniesPage />,
          },
          {
            path: "jobs",
            element: <JobsPage />,
          },
          {
            path: "jobs/:id",
            element: <JobDetailPage />,
          },
          {
            path: "stages",
            element: <StageTemplatesPage />,
          },
          {
            path: "resume-builder",
            element: <Navigate to="/app/resumes" replace />,
          },
          {
            path: "resume-builder/:id",
            element: <ResumeBuilderEditorPage />,
          },
          {
            path: "cover-letters",
            element: <CoverLettersPage />,
          },
          {
            path: "cover-letters/:id",
            element: <CoverLetterEditorPage />,
          },
          {
            path: "analytics",
            element: <AnalyticsPage />,
          },
          {
            path: "settings",
            element: <SettingsPage />,
          },
        ],
      },
      {
        path: "*",
        element: <NotFoundPage />,
      },
    ],
  },
];

export const router = sentryCreateBrowserRouter(routes);
