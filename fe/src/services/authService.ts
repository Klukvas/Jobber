import { apiClient } from "./api";
import type {
  LoginRequest,
  LoginResponse,
  RegisterRequest,
  RegisterResponse,
  AuthTokens,
  RefreshRequest,
  VerifyEmailRequest,
  ResendVerificationRequest,
  ForgotPasswordRequest,
  ResetPasswordRequest,
} from "@/shared/types/api";

export const authService = {
  async login(data: LoginRequest): Promise<LoginResponse> {
    return apiClient.post<LoginResponse>("auth/login", data);
  },

  async register(data: RegisterRequest): Promise<RegisterResponse> {
    return apiClient.post<RegisterResponse>("auth/register", data);
  },

  async refresh(data: RefreshRequest): Promise<AuthTokens> {
    return apiClient.post<AuthTokens>("auth/refresh", data);
  },

  async logout(): Promise<void> {
    return apiClient.post<void>("auth/logout");
  },

  async verifyEmail(data: VerifyEmailRequest): Promise<{ message: string }> {
    return apiClient.post<{ message: string }>("auth/verify-email", data);
  },

  async resendVerification(
    data: ResendVerificationRequest,
  ): Promise<{ message: string }> {
    return apiClient.post<{ message: string }>(
      "auth/resend-verification",
      data,
    );
  },

  async forgotPassword(
    data: ForgotPasswordRequest,
  ): Promise<{ message: string }> {
    return apiClient.post<{ message: string }>("auth/forgot-password", data);
  },

  async resetPassword(
    data: ResetPasswordRequest,
  ): Promise<{ message: string }> {
    return apiClient.post<{ message: string }>("auth/reset-password", data);
  },
};
