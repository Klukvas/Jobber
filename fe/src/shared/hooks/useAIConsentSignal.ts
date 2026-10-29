import { useCallback, useLayoutEffect, useRef } from "react";

/** Cancels only a pending confirmation when its originating surface disappears. */
export function useAIConsentSignal(): () => AbortSignal {
  const controller = useRef(new AbortController());
  useLayoutEffect(() => {
    const current = new AbortController();
    controller.current = current;
    return () => current.abort();
  }, []);
  return useCallback(() => controller.current.signal, []);
}
