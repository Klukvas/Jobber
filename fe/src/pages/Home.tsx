import { useLocation, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Button } from '@/shared/ui/Button';
import { Briefcase, CheckCircle, Clock, TrendingUp } from 'lucide-react';
import { usePageTitle } from '@/shared/lib/usePageTitle';
import { LoginModal } from '@/features/auth/modals/LoginModal';
import { RegisterModal } from '@/features/auth/modals/RegisterModal';

type AuthModal = 'login' | 'register' | null;

export default function Home() {
  const { t } = useTranslation();
  const location = useLocation();
  const navigate = useNavigate();
  usePageTitle();
  
  // Derive modal state directly from URL - no useState needed
  const activeModal: AuthModal = 
    location.pathname === '/login' ? 'login' :
    location.pathname === '/register' ? 'register' : null;

  const openLogin = () => {
    navigate('/login');
  };

  const openRegister = () => {
    navigate('/register');
  };

  const closeModal = () => {
    navigate('/');
  };

  const switchToRegister = () => {
    navigate('/register');
  };

  const switchToLogin = () => {
    navigate('/login');
  };

  return (
    <div className="flex min-h-screen flex-col">
      {/* Hero Section */}
      <div className="flex flex-1 items-center justify-center bg-muted/20 p-4">
        <div className="mx-auto max-w-4xl text-center">
          <div className="mb-8 flex justify-center">
            <Briefcase className="h-20 w-20 text-primary" />
          </div>
          <h1 className="mb-4 text-4xl font-bold tracking-tight sm:text-5xl md:text-6xl">
            Track Your Job Applications
          </h1>
          <p className="mb-8 text-lg text-muted-foreground sm:text-xl">
            Organize and manage your job search as a process, not a single action.
            Keep track of every stage, stay on top of deadlines, and never miss an opportunity.
          </p>
          <div className="flex flex-col gap-4 sm:flex-row sm:justify-center">
            <Button size="lg" onClick={openRegister}>
              {t('auth.register')}
            </Button>
            <Button variant="outline" size="lg" onClick={openLogin}>
              {t('auth.login')}
            </Button>
          </div>
        </div>
      </div>

      {/* Features Section */}
      <div className="bg-background py-16">
        <div className="mx-auto max-w-5xl px-4">
          <h2 className="mb-12 text-center text-3xl font-bold">Why Jobber?</h2>
          <div className="grid gap-8 md:grid-cols-3">
            <div className="flex flex-col items-center text-center">
              <div className="mb-4 rounded-full bg-primary/10 p-4">
                <Clock className="h-8 w-8 text-primary" />
              </div>
              <h3 className="mb-2 text-xl font-semibold">Track Progress</h3>
              <p className="text-muted-foreground">
                Visualize where each application stands in your job search pipeline
              </p>
            </div>
            <div className="flex flex-col items-center text-center">
              <div className="mb-4 rounded-full bg-primary/10 p-4">
                <CheckCircle className="h-8 w-8 text-primary" />
              </div>
              <h3 className="mb-2 text-xl font-semibold">Stay Organized</h3>
              <p className="text-muted-foreground">
                Keep all your resumes, companies, and job postings in one place
              </p>
            </div>
            <div className="flex flex-col items-center text-center">
              <div className="mb-4 rounded-full bg-primary/10 p-4">
                <TrendingUp className="h-8 w-8 text-primary" />
              </div>
              <h3 className="mb-2 text-xl font-semibold">Get Insights</h3>
              <p className="text-muted-foreground">
                Understand your job search patterns and optimize your strategy
              </p>
            </div>
          </div>
        </div>
      </div>

      {/* Auth Modals */}
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
