import { useCallback, useLayoutEffect, useRef } from "react";

/** Async responses may refresh caches after dismissal, but must not affect a new form. */
export function useCurrentFormSession(): () => boolean {
  const active = useRef(false);
  useLayoutEffect(() => {
    active.current = true;
    return () => { active.current = false; };
  }, []);
  return useCallback(() => active.current, []);
}
