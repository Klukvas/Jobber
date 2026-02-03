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

interface RegisterModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSwitchToLogin: () => void;
}

function ModalContent({ onOpenChange, onSwitchToLogin }: Omit<RegisterModalProps, 'open'>) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const setAuth = useAuthStore((state) => state.setAuth);
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [errors, setErrors] = useState<{
    email?: string;
    password?: string;
    confirmPassword?: string;
  }>({});

  const registerMutation = useMutation({
    mutationFn: authService.register,
    onSuccess: (data) => {
      setAuth(data.tokens.access_token, data.tokens.refresh_token, data.user);
      onOpenChange(false);
      navigate('/app');
    },
    onError: (error: ApiError) => {
      setErrors({ email: error.message });
    },
  });

  const validate = () => {
    const newErrors: {
      email?: string;
      password?: string;
      confirmPassword?: string;
    } = {};

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
    
    if (password !== confirmPassword) {
      newErrors.confirmPassword = t('errors.passwordsDontMatch');
    }
    
    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (validate()) {
      registerMutation.mutate({ email, password });
    }
  };

  return (
    <>
      <DialogHeader>
        <DialogTitle className="text-2xl font-bold">{t('auth.register')}</DialogTitle>
        <DialogDescription>
          Create an account to start tracking your job applications
        </DialogDescription>
      </DialogHeader>
      <form onSubmit={handleSubmit} className="mt-4">
        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="register-email">{t('auth.email')}</Label>
            <Input
              id="register-email"
              type="email"
              placeholder="you@example.com"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              aria-invalid={!!errors.email}
              aria-describedby={errors.email ? 'register-email-error' : undefined}
            />
            {errors.email && (
              <p id="register-email-error" className="text-sm text-destructive">
                {errors.email}
              </p>
            )}
          </div>
          <div className="space-y-2">
            <Label htmlFor="register-password">{t('auth.password')}</Label>
            <PasswordInput
              id="register-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              aria-invalid={!!errors.password}
              aria-describedby={errors.password ? 'register-password-error' : undefined}
            />
            {errors.password && (
              <p id="register-password-error" className="text-sm text-destructive">
                {errors.password}
              </p>
            )}
          </div>
          <div className="space-y-2">
            <Label htmlFor="register-confirmPassword">{t('auth.confirmPassword')}</Label>
            <PasswordInput
              id="register-confirmPassword"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              aria-invalid={!!errors.confirmPassword}
              aria-describedby={errors.confirmPassword ? 'register-confirmPassword-error' : undefined}
            />
            {errors.confirmPassword && (
              <p id="register-confirmPassword-error" className="text-sm text-destructive">
                {errors.confirmPassword}
              </p>
            )}
          </div>
        </div>
        <div className="mt-6 flex flex-col gap-4">
          <Button
            type="submit"
            className="w-full"
            disabled={registerMutation.isPending}
          >
            {registerMutation.isPending ? t('common.loading') : t('auth.register')}
          </Button>
          <p className="text-center text-sm text-muted-foreground">
            {t('auth.alreadyHaveAccount')}{' '}
            <button
              type="button"
              onClick={onSwitchToLogin}
              className="font-medium text-primary hover:underline"
            >
              {t('auth.login')}
            </button>
          </p>
        </div>
      </form>
    </>
  );
}

export function RegisterModal({ open, onOpenChange, onSwitchToLogin }: RegisterModalProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent onClose={() => onOpenChange(false)}>
        <ModalContent
          key={open ? 'open' : 'closed'}
          onOpenChange={onOpenChange}
          onSwitchToLogin={onSwitchToLogin}
        />
      </DialogContent>
    </Dialog>
  );
}
