import { afterEach, expect, it, vi } from "vitest";
import { act, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { authService } from "@/services/authService";
import { useAuthStore } from "@/stores/authStore";
import { LoginModal } from "../LoginModal";

const mocks = vi.hoisted(() => ({ post: vi.fn(), navigate: vi.fn() }));
vi.mock("@/services/api", async (importOriginal) => ({
  ...await importOriginal<typeof import("@/services/api")>(),
  apiClient: { post: mocks.post },
}));
vi.mock("react-router-dom", () => ({
  useNavigate: () => mocks.navigate,
  useLocation: () => ({ pathname: "/login", search: "?next=/app/jobs" }),
}));
vi.mock("react-i18next", () => ({ useTranslation: () => ({ t: (key: string) => key }) }));

afterEach(() => {
  useAuthStore.getState().clearAuth();
  vi.clearAllMocks();
});

it.each([true, false])("does not authenticate or navigate after logout (callback: %s)", async (withCallback) => {
  const user = userEvent.setup();
  const queryClient = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
  const onOpenChange = vi.fn();
  const onAuthenticated = vi.fn();
  let finishLogin: (value: unknown) => void = () => {};
  mocks.post.mockImplementation((path: string) => path === "auth/login"
    ? new Promise((resolve) => { finishLogin = resolve; })
    : Promise.resolve());
  render(<QueryClientProvider client={queryClient}>
    <LoginModal open onOpenChange={onOpenChange} onSwitchToRegister={vi.fn()} onAuthenticated={withCallback ? onAuthenticated : undefined} />
  </QueryClientProvider>);
  await user.type(screen.getByLabelText("auth.email"), "user@example.com");
  await user.type(screen.getByLabelText("auth.password"), "password123");
  await user.click(screen.getByRole("button", { name: "auth.login" }));
  await waitFor(() => expect(mocks.post).toHaveBeenCalledWith("auth/login", expect.anything()));
  await act(async () => { await authService.logout(); });
  await act(async () => { finishLogin({ user: { id: "old", email: "user@example.com", name: "Old", locale: "en", created_at: "" } }); });
  await waitFor(() => expect(queryClient.isMutating()).toBe(0));
  expect(useAuthStore.getState().isAuthenticated).toBe(false);
  expect(onAuthenticated).not.toHaveBeenCalled();
  expect(onOpenChange).not.toHaveBeenCalled();
  expect(mocks.navigate).not.toHaveBeenCalled();
});
