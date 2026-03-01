import { createBrowserRouter, Navigate } from "react-router-dom";
import { RootLayout } from "./layouts/RootLayout";
import { AuthLayout } from "./layouts/AuthLayout";
import { AppLayout } from "./layouts/AppLayout";

// Lazy load pages
import { lazy } from "react";

// Public pages
const HomePage = lazy(() => import("@/pages/Home"));
const BlogPage = lazy(() => import("@/pages/Blog"));
const BlogPostPage = lazy(() => import("@/pages/BlogPost"));
const PrivacyPage = lazy(() => import("@/pages/Privacy"));
const TermsPage = lazy(() => import("@/pages/Terms"));
const RefundPage = lazy(() => import("@/pages/Refund"));

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

export const router = createBrowserRouter([
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
        ],
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
            path: "analytics",
            element: <AnalyticsPage />,
          },
          {
            path: "settings",
            element: <SettingsPage />,
          },
        ],
      },
    ],
  },
]);
