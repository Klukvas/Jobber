import { useEffect, useState } from "react";

/**
 * rAF-throttled window scrollY, capped at `max` so consumers stop
 * re-rendering once the effect they drive is off-screen.
 */
export function useScrollY(max = Number.POSITIVE_INFINITY) {
  const [scrollY, setScrollY] = useState(0);

  useEffect(() => {
    let frame = 0;
    const read = () => {
      frame = 0;
      setScrollY(Math.min(window.scrollY, max));
    };
    const onScroll = () => {
      if (!frame) frame = requestAnimationFrame(read);
    };
    onScroll();
    window.addEventListener("scroll", onScroll, { passive: true });
    return () => {
      window.removeEventListener("scroll", onScroll);
      if (frame) cancelAnimationFrame(frame);
    };
  }, [max]);

  return scrollY;
}
