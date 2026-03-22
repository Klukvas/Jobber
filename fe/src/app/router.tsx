import { Navigate } from "react-router-dom";
import { sentryCreateBrowserRouter } from "@/shared/lib/sentry";
import { RootLayout } from "./layouts/RootLayout";
import { AuthLayout } from "./layouts/AuthLayout";
import { AppLayout } from "./layouts/AppLayout";

// Lazy load pages
import { lazy, Suspense } from "react";

// Print page (no auth, no layout — used by headless Chrome for PDF export)
const ResumeBuilderPrintPage = lazy(() => import("@/pages/ResumeBuilderPrint"));

// Public pages
const HomePage = lazy(() => import("@/pages/Home"));
const BlogPage = lazy(() => import("@/pages/Blog"));
const BlogPostPage = lazy(() => import("@/pages/BlogPost"));
const PrivacyPage = lazy(() => import("@/pages/Privacy"));
const TermsPage = lazy(() => import("@/pages/Terms"));
const RefundPage = lazy(() => import("@/pages/Refund"));
const FeatureApplicationsPage = lazy(
  () => import("@/pages/FeatureApplications"),
);
const FeatureResumeBuilderPage = lazy(
  () => import("@/pages/FeatureResumeBuilder"),
);
const FeatureCoverLettersPage = lazy(
  () => import("@/pages/FeatureCoverLetters"),
);

// Protected pages
const SettingsPage = lazy(() => import("@/pages/Settings"));
const ApplicationsPage = lazy(() => import("@/pages/Applications"));
const ApplicationDetailPage = lazy(() => import("@/pages/ApplicationDetail"));
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
        path: "app",
        element: <AppLayout />,
        children: [
          {
            index: true,
            element: <Navigate to="/app/applications" replace />,
          },
          {
            path: "applications",
            element: <ApplicationsPage />,
          },
          {
            path: "applications/:id",
            element: <ApplicationDetailPage />,
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
