import { useEffect, useRef, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { oauthService, readAuthorizationParams } from "@/services/oauthService";

export function useConsent(search: string, userId: string | undefined) {
  const [status, setStatus] = useState<"idle" | "pending" | "error">("idle");
  const [error, setError] = useState<unknown>(null);
  const submitting = useRef(false);
  const redirected = useRef(false);
  const controller = useRef<AbortController | null>(null);
  useEffect(() => {
    const lifetime = new AbortController();
    controller.current = lifetime;
    return () => lifetime.abort();
  }, []);
  const query = useQuery({
    queryKey: ["oauth-preview", userId, search],
    queryFn: ({ signal }) => oauthService.preview(search, { signal }),
    enabled: !!userId,
    retry: false,
    gcTime: 0,
    staleTime: 0,
  });
  const redirect = query.data && "redirect_to" in query.data ? query.data.redirect_to : undefined;
  useEffect(() => {
    if (redirect && !redirected.current) {
      redirected.current = true;
      window.location.assign(redirect);
    }
  }, [redirect]);
  async function decide(decision: "allow" | "deny") {
    if (submitting.current || !query.data || "redirect_to" in query.data ||
      (decision === "allow" && !query.data.plan_allowed)) return;
    const signal = controller.current?.signal;
    if (!signal || signal.aborted) return;
    submitting.current = true;
    setStatus("pending");
    setError(null);
    try {
      const result = await oauthService.authorize(readAuthorizationParams(search), decision, { signal });
      if (signal.aborted) return;
      window.location.assign(result.redirect_to);
    } catch (error) {
      if (signal.aborted) return;
      submitting.current = false;
      setError(error);
      setStatus("error");
    }
  }
  return { query, status, error, decide };
}
