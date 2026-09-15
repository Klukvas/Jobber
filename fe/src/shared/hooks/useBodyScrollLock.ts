import * as React from "react";

/**
 * How many overlays are holding the page still right now.
 *
 * Module scope, because the question cannot be answered from inside one
 * overlay. Each used to save `document.body.style.overflow` on open and write
 * it back on close, which is correct only while overlays close in the order
 * they opened. A drawer opened first and closed second — a dialog raised over
 * it, then the drawer dismissed behind it — restored the value it had captured
 * *before* the dialog locked, and the page scrolled underneath a modal that
 * was still up.
 */
let lockCount = 0;
/** What `<body>` scrolled like before the first lock. */
let overflowBeforeLock: string | null = null;

function lock(): void {
  if (lockCount === 0) {
    overflowBeforeLock = document.body.style.overflow;
    document.body.style.overflow = "hidden";
  }
  lockCount += 1;
}

function release(): void {
  if (lockCount === 0) return;

  lockCount -= 1;
  if (lockCount > 0) return;

  document.body.style.overflow = overflowBeforeLock ?? "";
  overflowBeforeLock = null;
}

/**
 * Freezes the page behind an overlay for as long as `active` is true, and
 * gives it back — to whatever it was, not to a guess — when the last overlay
 * lets go. Unmounting while active releases too.
 */
export function useBodyScrollLock(active: boolean): void {
  React.useEffect(() => {
    if (!active) return;

    lock();
    return release;
  }, [active]);
}
