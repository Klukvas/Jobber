import { endSession } from "@/shared/lib/session";
import { adoptSameAccountSession } from "@/shared/lib/sessionGeneration";
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

// Local to this tab: logout invalidates even sign-ins that began while signed out.
let logoutEpoch = 0;

export const authService = {
  async login(data: LoginRequest): Promise<LoginResponse> {
    const epoch = logoutEpoch;
    const response = await apiClient.post<LoginResponse>("auth/login", data);
    if (epoch !== logoutEpoch) throw new DOMException("Session ended", "AbortError");
    return response;
  },

  async register(data: RegisterRequest): Promise<RegisterResponse> {
    const epoch = logoutEpoch;
    const response = await apiClient.post<RegisterResponse>("auth/register", data);
    if (epoch !== logoutEpoch) throw new DOMException("Session ended", "AbortError");
    return response;
  },

  async refresh(data: RefreshRequest): Promise<AuthTokens> {
    return apiClient.post<AuthTokens>("auth/refresh", data);
  },

  async logout(): Promise<void> {
    logoutEpoch += 1;
    // While this tab still knows its account: if that same account signed in
    // again in another tab, the logout is still this user's to send. After
    // endSession the tab could no longer tell. Anything else stays as it is,
    // so a logout from a tab that is behind is still refused.
    adoptSameAccountSession();
    endSession();
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
