import { Navigate } from "react-router-dom";
import { sentryCreateBrowserRouter } from "@/shared/lib/sentry";
import { LegacyApplicationDetailRedirect } from "./LegacyApplicationRedirect";
import { RootLayout } from "./layouts/RootLayout";
import { AuthLayout } from "./layouts/AuthLayout";
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

// Public shared-stats page (no auth; opened from links posted on social media)
const SharedStatsPage = lazy(() => import("@/pages/SharedStats"));

export const router = sentryCreateBrowserRouter([
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
        index: true,
        element: <HomePage />,
      },
      {
        path: "",
        element: <AuthLayout />,
        children: [
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
        path: "reset-password",
        element: <Navigate to="/" replace />,
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
]);
