import { useNavigate } from "react-router-dom";
import { authService } from "@/services/authService";
import { useAuthStore } from "@/stores/authStore";
import { useSingleFlightMutation } from "@/shared/hooks/useSingleFlightMutation";

/** Both logout controls end local work before the server revokes the cookie. */
export function useLogout() {
  const navigate = useNavigate();
  return useSingleFlightMutation({
    mutationFn: authService.logout,
    onSettled: () => {
      if (!useAuthStore.getState().isAuthenticated) navigate("/");
    },
  });
}
