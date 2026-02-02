import { apiClient } from './api';
import type {
  LoginRequest,
  LoginResponse,
  RegisterRequest,
  RegisterResponse,
  AuthTokens,
  RefreshRequest,
} from '@/shared/types/api';

export const authService = {
  async login(data: LoginRequest): Promise<LoginResponse> {
    return apiClient.post<LoginResponse>('auth/login', data);
  },

  async register(data: RegisterRequest): Promise<RegisterResponse> {
    return apiClient.post<RegisterResponse>('auth/register', data);
  },

  async refresh(data: RefreshRequest): Promise<AuthTokens> {
    return apiClient.post<AuthTokens>('auth/refresh', data);
  },

  async logout(): Promise<void> {
    return apiClient.post<void>('auth/logout');
  },
};
