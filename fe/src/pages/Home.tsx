import { useEffect } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { useAuthStore } from "@/stores/authStore";
import { usePageMeta } from "@/shared/lib/usePageMeta";
import { LoginModal } from "@/features/auth/modals/LoginModal";
import { RegisterModal } from "@/features/auth/modals/RegisterModal";
import { ForgotPasswordModal } from "@/features/auth/modals/ForgotPasswordModal";
import { HomeNavbar } from "@/features/home/components/HomeNavbar";
import { JsonLd } from "@/features/home/components/JsonLd";
import { HeroSection } from "@/features/home/components/HeroSection";
import { FeaturesSection } from "@/features/home/components/FeaturesSection";
import { HowItWorksSection } from "@/features/home/components/HowItWorksSection";
import { DemoPreviewSection } from "@/features/home/components/DemoPreviewSection";
import { FooterSection } from "@/features/home/components/FooterSection";

type AuthModal = "login" | "register" | "forgot-password" | null;

export default function Home() {
  const location = useLocation();
  const navigate = useNavigate();
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);
  usePageMeta({
    titleKey: "seo.home.title",
    descriptionKey: "seo.home.description",
  });

  useEffect(() => {
    if (!location.hash) return;
    const id = location.hash.slice(1);
    const timer = setTimeout(() => {
      document.getElementById(id)?.scrollIntoView({ behavior: "smooth" });
    }, 100);
    return () => clearTimeout(timer);
  }, [location.hash]);

  const activeModal: AuthModal =
    location.pathname === "/login"
      ? "login"
      : location.pathname === "/register"
        ? "register"
        : location.pathname === "/forgot-password"
          ? "forgot-password"
          : null;

  const openLogin = () => navigate("/login");
  const openRegister = () => navigate("/register");
  const closeModal = () => navigate("/");
  const switchToRegister = () => navigate("/register");
  const switchToLogin = () => navigate("/login");
  const openForgotPassword = () => navigate("/forgot-password");
  const goPlatform = () => navigate("/app/applications");

  return (
    <div className="flex min-h-screen flex-col">
      <JsonLd />
      <HomeNavbar
        isAuthenticated={isAuthenticated}
        onLogin={openLogin}
        onRegister={openRegister}
        onGoPlatform={goPlatform}
      />
      <HeroSection
        isAuthenticated={isAuthenticated}
        onRegister={openRegister}
        onLogin={openLogin}
        onGoPlatform={goPlatform}
      />
      <FeaturesSection />
      <HowItWorksSection />
      <DemoPreviewSection />
      <FooterSection />

      <LoginModal
        open={activeModal === "login"}
        onOpenChange={(open) => !open && closeModal()}
        onSwitchToRegister={switchToRegister}
        onForgotPassword={openForgotPassword}
      />
      <RegisterModal
        open={activeModal === "register"}
        onOpenChange={(open) => !open && closeModal()}
        onSwitchToLogin={switchToLogin}
      />
      <ForgotPasswordModal
        open={activeModal === "forgot-password"}
        onOpenChange={(open) => !open && closeModal()}
        onBackToLogin={switchToLogin}
      />
    </div>
  );
}
