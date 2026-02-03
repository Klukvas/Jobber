import { createBrowserRouter, Navigate } from 'react-router-dom';
import { RootLayout } from './layouts/RootLayout';
import { AuthLayout } from './layouts/AuthLayout';
import { AppLayout } from './layouts/AppLayout';

// Lazy load pages
import { lazy } from 'react';

// Public pages
const HomePage = lazy(() => import('@/pages/Home'));
const LoginPage = lazy(() => import('@/pages/Login'));
const RegisterPage = lazy(() => import('@/pages/Register'));

// Protected pages
const ApplicationsPage = lazy(() => import('@/pages/Applications'));
const ApplicationDetailPage = lazy(() => import('@/pages/ApplicationDetail'));
const ResumesPage = lazy(() => import('@/pages/Resumes'));
const CompaniesPage = lazy(() => import('@/pages/Companies'));
const JobsPage = lazy(() => import('@/pages/Jobs'));
const StageTemplatesPage = lazy(() => import('@/pages/StageTemplates'));
const AnalyticsPage = lazy(() => import('@/pages/Analytics'));
const SettingsPage = lazy(() => import('@/pages/Settings'));

export const router = createBrowserRouter([
  {
    path: '/',
    element: <RootLayout />,
    children: [
      {
        path: '',
        element: <AuthLayout />,
        children: [
          {
            index: true,
            element: <HomePage />,
          },
          {
            path: 'login',
            element: <LoginPage />,
          },
          {
            path: 'register',
            element: <RegisterPage />,
          },
        ],
      },
      {
        path: 'app',
        element: <AppLayout />,
        children: [
          {
            index: true,
            element: <Navigate to="/app/applications" replace />,
          },
          {
            path: 'applications',
            element: <ApplicationsPage />,
          },
          {
            path: 'applications/:id',
            element: <ApplicationDetailPage />,
          },
          {
            path: 'resumes',
            element: <ResumesPage />,
          },
          {
            path: 'companies',
            element: <CompaniesPage />,
          },
          {
            path: 'jobs',
            element: <JobsPage />,
          },
          {
            path: 'stages',
            element: <StageTemplatesPage />,
          },
          {
            path: 'analytics',
            element: <AnalyticsPage />,
          },
          {
            path: 'settings',
            element: <SettingsPage />,
          },
        ],
      },
    ],
  },
]);
