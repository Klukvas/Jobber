import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useMutation } from '@tanstack/react-query';
import { authService } from '@/services/authService';
import { useAuthStore } from '@/stores/authStore';
import { Button } from '@/shared/ui/Button';
import { Input } from '@/shared/ui/Input';
import { PasswordInput } from '@/shared/ui/PasswordInput';
import { Label } from '@/shared/ui/Label';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/shared/ui/Dialog';
import { ApiError } from '@/services/api';

interface LoginModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSwitchToRegister: () => void;
}

function ModalContent({ onOpenChange, onSwitchToRegister }: Omit<LoginModalProps, 'open'>) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const setAuth = useAuthStore((state) => state.setAuth);
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [errors, setErrors] = useState<{ email?: string; password?: string }>({});

  const loginMutation = useMutation({
    mutationFn: authService.login,
    onSuccess: (data) => {
      setAuth(data.tokens.access_token, data.tokens.refresh_token, data.user);
      onOpenChange(false);
      navigate('/app');
    },
    onError: (error: ApiError) => {
      setErrors({ password: error.message });
    },
  });

  const validate = () => {
    const newErrors: { email?: string; password?: string } = {};
    
    if (!email) {
      newErrors.email = t('errors.required');
    } else if (!/\S+@\S+\.\S+/.test(email)) {
      newErrors.email = t('errors.invalidEmail');
    }
    
    if (!password) {
      newErrors.password = t('errors.required');
    } else if (password.length < 8) {
      newErrors.password = t('errors.passwordTooShort');
    }
    
    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (validate()) {
      loginMutation.mutate({ email, password });
    }
  };

  return (
    <>
      <DialogHeader>
        <DialogTitle className="text-2xl font-bold">{t('auth.login')}</DialogTitle>
        <DialogDescription>
          Enter your email and password to access your account
        </DialogDescription>
      </DialogHeader>
      <form onSubmit={handleSubmit} className="mt-4">
        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="login-email">{t('auth.email')}</Label>
            <Input
              id="login-email"
              type="email"
              placeholder="you@example.com"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              aria-invalid={!!errors.email}
              aria-describedby={errors.email ? 'login-email-error' : undefined}
            />
            {errors.email && (
              <p id="login-email-error" className="text-sm text-destructive">
                {errors.email}
              </p>
            )}
          </div>
          <div className="space-y-2">
            <Label htmlFor="login-password">{t('auth.password')}</Label>
            <PasswordInput
              id="login-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              aria-invalid={!!errors.password}
              aria-describedby={errors.password ? 'login-password-error' : undefined}
            />
            {errors.password && (
              <p id="login-password-error" className="text-sm text-destructive">
                {errors.password}
              </p>
            )}
          </div>
        </div>
        <div className="mt-6 flex flex-col gap-4">
          <Button
            type="submit"
            className="w-full"
            disabled={loginMutation.isPending}
          >
            {loginMutation.isPending ? t('common.loading') : t('auth.login')}
          </Button>
          <p className="text-center text-sm text-muted-foreground">
            {t('auth.dontHaveAccount')}{' '}
            <button
              type="button"
              onClick={onSwitchToRegister}
              className="font-medium text-primary hover:underline"
            >
              {t('auth.register')}
            </button>
          </p>
        </div>
      </form>
    </>
  );
}

export function LoginModal({ open, onOpenChange, onSwitchToRegister }: LoginModalProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent onClose={() => onOpenChange(false)}>
        <ModalContent
          key={open ? 'open' : 'closed'}
          onOpenChange={onOpenChange}
          onSwitchToRegister={onSwitchToRegister}
        />
      </DialogContent>
    </Dialog>
  );
}
