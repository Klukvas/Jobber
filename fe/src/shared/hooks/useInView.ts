import { useEffect, useRef, useState } from "react";

const hasIntersectionObserver = () =>
  typeof window !== "undefined" && "IntersectionObserver" in window;

/**
 * True once the element has entered the viewport (one-shot).
 * Environments without IntersectionObserver (jsdom, old browsers)
 * start visible so content is never hidden.
 */
export function useInView<T extends HTMLElement>(threshold = 0.15) {
  const ref = useRef<T | null>(null);
  const [isInView, setIsInView] = useState(() => !hasIntersectionObserver());

  useEffect(() => {
    const element = ref.current;
    if (!element || !hasIntersectionObserver()) return;

    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) {
          setIsInView(true);
          observer.disconnect();
        }
      },
      { threshold },
    );
    observer.observe(element);
    return () => observer.disconnect();
  }, [threshold]);

  return { ref, isInView };
}
