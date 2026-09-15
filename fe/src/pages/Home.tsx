import { useEffect } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { useAuthStore } from "@/stores/authStore";
import { getAuthModalRoute } from "@/features/auth/authModalRoutes";
import { usePageMeta } from "@/shared/lib/usePageMeta";
import { scrollToSection } from "@/shared/lib/scrollToSection";
import { LoginModal } from "@/features/auth/modals/LoginModal";
import { RegisterModal } from "@/features/auth/modals/RegisterModal";
import { ForgotPasswordModal } from "@/features/auth/modals/ForgotPasswordModal";
import { HomeNavbar } from "@/features/home/components/HomeNavbar";
import { JsonLd } from "@/features/home/components/JsonLd";
import { HeroSection } from "@/features/home/components/HeroSection";
import { SocialProofBar } from "@/features/home/components/SocialProofBar";
import { FeaturesSection } from "@/features/home/components/FeaturesSection";
import { HowItWorksSection } from "@/features/home/components/HowItWorksSection";
import { AiHighlightSection } from "@/features/home/components/AiHighlightSection";
import { PricingSection } from "@/features/home/components/PricingSection";
import { FaqSection } from "@/features/home/components/FaqSection";
import { FooterCtaSection } from "@/features/home/components/FooterCtaSection";
import { FooterSection } from "@/features/home/components/FooterSection";

export default function Home() {
  const location = useLocation();
  const navigate = useNavigate();
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);
  usePageMeta({
    titleKey: "seo.home.title",
    descriptionKey: "seo.home.description",
  });

  // The `landing-page` body class this page needs is not added here. It is
  // owned by RootLayout, keyed on the route — see `app/landingBodyClass.ts`.
  // Mounting is the wrong signal for it: the prerendered home page is also the
  // SPA fallback, so every direct load of `/app/*` arrived with the class
  // already set and no landing page to remove it.

  // Keyed on `location.key`, not on the hash alone. The footer's "FAQ" link
  // points at `/#faq` from the landing page itself, so the second click
  // navigates to the hash the URL already carries: with only the hash in the
  // deps the effect never re-ran, and the link did nothing at all for anyone
  // who had scrolled away since the first one. Every navigation gets a fresh
  // key, so the same anchor works however many times it is used.
  useEffect(() => {
    if (!location.hash) return;
    const id = location.hash.slice(1);
    const timer = setTimeout(() => {
      scrollToSection(id);
    }, 100);
    return () => clearTimeout(timer);
  }, [location.hash, location.key]);

  const activeModal = getAuthModalRoute(location.pathname);

  const openLogin = () => navigate("/login");
  const openRegister = () => navigate("/register");
  const closeModal = () => navigate("/");
  const switchToRegister = () => navigate("/register");
  const switchToLogin = () => navigate("/login");
  const openForgotPassword = () => navigate("/forgot-password");
  const goPlatform = () => navigate("/app/jobs");

  return (
    <>
      {/* Landing page — forced dark theme with custom palette */}
      <div className="dark landing-override flex min-h-screen flex-col bg-background text-foreground">
        <JsonLd />
        <HomeNavbar
          isAuthenticated={isAuthenticated}
          onLogin={openLogin}
          onRegister={openRegister}
          onGoPlatform={goPlatform}
          darkHero
        />
        <main>
          <HeroSection
            isAuthenticated={isAuthenticated}
            onRegister={openRegister}
            onGoPlatform={goPlatform}
          />
          <SocialProofBar />
          <FeaturesSection />
          <HowItWorksSection />
          <AiHighlightSection
            isAuthenticated={isAuthenticated}
            onRegister={openRegister}
            onGoPlatform={goPlatform}
          />
          <PricingSection
            isAuthenticated={isAuthenticated}
            onRegister={openRegister}
            onGoPlatform={goPlatform}
          />
          <FaqSection />
          <FooterCtaSection
            isAuthenticated={isAuthenticated}
            onRegister={openRegister}
            onGoPlatform={goPlatform}
          />
        </main>
        <FooterSection />
      </div>

      {/* Modals rendered outside landing-override to use the user's theme */}
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
    </>
  );
}
