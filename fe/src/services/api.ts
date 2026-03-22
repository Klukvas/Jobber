import ky, { type KyInstance, HTTPError } from "ky";
import type { ErrorResponse } from "@/shared/types/api";
import { useAuthStore } from "@/stores/authStore";

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || "/api/v1";

class ApiClient {
  private client: KyInstance;
  private refreshPromise: Promise<boolean> | null = null;

  constructor() {
    this.client = ky.create({
      prefixUrl: API_BASE_URL,
      timeout: 30000,
      credentials: "include",
      hooks: {
        beforeRequest: [
          (request) => {
            // Add request_id for tracing
            const requestId = this.generateRequestId();
            request.headers.set("X-Request-ID", requestId);

            return request;
          },
        ],
        afterResponse: [
          async (request, _options, response) => {
            // Handle 401 Unauthorized - try to refresh token via cookie
            // Skip for auth endpoints (login/register return 401 on bad credentials)
            const url = new URL(request.url);
            const isAuthEndpoint = url.pathname.includes("/auth/");

            const isRetry = request.headers.get("X-Retry") === "1";
            if (response.status === 401 && !isAuthEndpoint && !isRetry) {
              const refreshed = await this.tryRefreshToken();
              if (refreshed) {
                // Retry the original request — mark to prevent infinite loop
                const retryReq = new Request(request, {
                  credentials: "include",
                });
                retryReq.headers.set("X-Retry", "1");
                return ky(retryReq);
              } else {
                // Refresh failed, clear auth and redirect to login
                useAuthStore.getState().clearAuth();
                window.location.href = "/";
              }
            }

            return response;
          },
        ],
      },
    });
  }

  private generateRequestId(): string {
    return crypto.randomUUID();
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
      // Refresh token is sent automatically via httpOnly cookie
      await ky.post(`${API_BASE_URL}/auth/refresh`, {
        credentials: "include",
      });
      return true;
    } catch (err) {
      console.error(
        "[ApiClient] token refresh failed:",
        err instanceof Error ? err.message : "unknown error",
      );
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
          error.response.status,
        );
      } catch (parseError) {
        if (parseError instanceof ApiError) throw parseError;
        throw new ApiError(
          error.message || "Request failed",
          "UNKNOWN_ERROR",
          error.response.status,
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

  async put<T>(url: string, data?: unknown): Promise<T> {
    try {
      return await this.client.put(url, { json: data }).json<T>();
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

  async postBlob(url: string, data?: unknown, timeout?: number): Promise<Blob> {
    try {
      const response = await this.client.post(url, {
        json: data,
        timeout: timeout ?? 60000,
      });
      return await response.blob();
    } catch (error) {
      return this.handleError(error);
    }
  }

  async postFormData<T>(
    url: string,
    formData: FormData,
    timeout?: number,
  ): Promise<T> {
    try {
      return await this.client
        .post(url, {
          body: formData,
          timeout: timeout ?? 60000,
        })
        .json<T>();
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
    this.name = "ApiError";
    this.code = code;
    this.status = status;
  }
}

export const apiClient = new ApiClient();
