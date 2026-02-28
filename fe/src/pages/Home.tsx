import { useLocation, useNavigate } from 'react-router-dom';
import { useAuthStore } from '@/stores/authStore';
import { usePageTitle } from '@/shared/lib/usePageTitle';
import { LoginModal } from '@/features/auth/modals/LoginModal';
import { RegisterModal } from '@/features/auth/modals/RegisterModal';
import { HomeNavbar } from '@/features/home/components/HomeNavbar';
import { HeroSection } from '@/features/home/components/HeroSection';
import { FeaturesSection } from '@/features/home/components/FeaturesSection';
import { HowItWorksSection } from '@/features/home/components/HowItWorksSection';
import { DemoPreviewSection } from '@/features/home/components/DemoPreviewSection';
import { FooterSection } from '@/features/home/components/FooterSection';

type AuthModal = 'login' | 'register' | null;

export default function Home() {
  const location = useLocation();
  const navigate = useNavigate();
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);
  usePageTitle();

  const activeModal: AuthModal =
    location.pathname === '/login' ? 'login' :
    location.pathname === '/register' ? 'register' : null;

  const openLogin = () => navigate('/login');
  const openRegister = () => navigate('/register');
  const closeModal = () => navigate('/');
  const switchToRegister = () => navigate('/register');
  const switchToLogin = () => navigate('/login');
  const goPlatform = () => navigate('/app/applications');

  return (
    <div className="flex min-h-screen flex-col">
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
        open={activeModal === 'login'}
        onOpenChange={(open) => !open && closeModal()}
        onSwitchToRegister={switchToRegister}
      />
      <RegisterModal
        open={activeModal === 'register'}
        onOpenChange={(open) => !open && closeModal()}
        onSwitchToLogin={switchToLogin}
      />
    </div>
  );
}
