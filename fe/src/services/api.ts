import ky, { type KyInstance, HTTPError } from 'ky';
import type { ErrorResponse } from '@/shared/types/api';
import { useAuthStore } from '@/stores/authStore';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1';

class ApiClient {
  private client: KyInstance;
  private refreshPromise: Promise<boolean> | null = null;

  constructor() {
    this.client = ky.create({
      prefixUrl: API_BASE_URL,
      timeout: 30000,
      hooks: {
        beforeRequest: [
          (request) => {
            // Add Authorization header if access token exists
            const { accessToken } = useAuthStore.getState();
            if (accessToken) {
              request.headers.set('Authorization', `Bearer ${accessToken}`);
            }

            // Add request_id for tracing
            const requestId = this.generateRequestId();
            request.headers.set('X-Request-ID', requestId);

            return request;
          },
        ],
        afterResponse: [
          async (request, _options, response) => {
            // Handle 401 Unauthorized - try to refresh token
            if (response.status === 401) {
              const refreshed = await this.tryRefreshToken();
              if (refreshed) {
                // Retry the original request with new token
                const { accessToken } = useAuthStore.getState();
                request.headers.set('Authorization', `Bearer ${accessToken}`);
                return ky(request);
              } else {
                // Refresh failed, clear auth and redirect to login
                useAuthStore.getState().clearAuth();
                window.location.href = '/login';
              }
            }

            return response;
          },
        ],
      },
    });
  }

  private generateRequestId(): string {
    return `${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
  }

  private async tryRefreshToken(): Promise<boolean> {
    // Deduplicate concurrent refresh attempts
    if (this.refreshPromise) {
      return this.refreshPromise;
    }

    this.refreshPromise = this.doRefreshToken();
    try {
      return await this.refreshPromise;
    } finally {
      this.refreshPromise = null;
    }
  }

  private async doRefreshToken(): Promise<boolean> {
    try {
      const { refreshToken, user } = useAuthStore.getState();

      if (!refreshToken || !user) {
        return false;
      }

      const response = await ky.post(`${API_BASE_URL}/auth/refresh`, {
        json: { refresh_token: refreshToken },
      }).json<{ access_token: string; refresh_token: string }>();

      const { setAuth } = useAuthStore.getState();
      setAuth(response.access_token, response.refresh_token, user);
      return true;
    } catch {
      return false;
    }
  }

  private async handleError(error: unknown): Promise<never> {
    if (error instanceof HTTPError) {
      try {
        const errorResponse = await error.response.json<ErrorResponse>();
        throw new ApiError(
          errorResponse.error_message,
          errorResponse.error_code,
          error.response.status
        );
      } catch (parseError) {
        if (parseError instanceof ApiError) throw parseError;
        throw new ApiError(
          error.message || 'Request failed',
          'UNKNOWN_ERROR',
          error.response.status
        );
      }
    }
    throw error;
  }

  async get<T>(url: string): Promise<T> {
    try {
      return await this.client.get(url).json<T>();
    } catch (error) {
      return this.handleError(error);
    }
  }

  async post<T>(url: string, data?: unknown): Promise<T> {
    try {
      return await this.client.post(url, { json: data }).json<T>();
    } catch (error) {
      return this.handleError(error);
    }
  }

  async patch<T>(url: string, data?: unknown): Promise<T> {
    try {
      return await this.client.patch(url, { json: data }).json<T>();
    } catch (error) {
      return this.handleError(error);
    }
  }

  async delete<T>(url: string): Promise<T> {
    try {
      return await this.client.delete(url).json<T>();
    } catch (error) {
      return this.handleError(error);
    }
  }
}

export class ApiError extends Error {
  code: string;
  status: number;

  constructor(message: string, code: string, status: number) {
    super(message);
    this.name = 'ApiError';
    this.code = code;
    this.status = status;
  }
}

export const apiClient = new ApiClient();
