import { Outlet, Navigate } from "react-router-dom";
import { useAuthStore } from "@/stores/authStore";
import { Sidebar } from "@/widgets/Sidebar";
import { Header } from "@/widgets/Header";
import { useOnboarding } from "@/features/onboarding/useOnboarding";
import { WelcomeWizard } from "@/features/onboarding/WelcomeWizard";

export function AppLayout() {
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);
  const { shouldShow, complete } = useOnboarding();

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }

  return (
    <div className="flex min-h-screen">
      <Sidebar />
      <div className="flex min-w-0 flex-1 flex-col">
        <Header />
        <main className="flex-1 overflow-auto p-4 md:p-6">
          <Outlet />
        </main>
      </div>
      <WelcomeWizard open={shouldShow} onComplete={complete} />
    </div>
  );
}
