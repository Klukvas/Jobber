/**
 * A hand-cranked `requestAnimationFrame`, so a test can say which frame it is
 * on. Code that retries across frames — an overlay waiting for its transition
 * to put it on screen — is otherwise timed against the real clock, which makes
 * the test both slow and flaky.
 */
export interface ManualAnimationFrames {
  /**
   * Runs every callback queued so far. Callbacks those queue in turn wait for
   * the next call, exactly as a real frame boundary would have it.
   */
  readonly runFrame: () => void;
  /** How many callbacks are waiting — zero once a retry loop has given up. */
  readonly pendingFrames: () => number;
  readonly restore: () => void;
}

export function installManualAnimationFrames(): ManualAnimationFrames {
  const nativeRequest = window.requestAnimationFrame;
  const nativeCancel = window.cancelAnimationFrame;
  const queued = new Map<number, FrameRequestCallback>();
  let nextHandle = 1;

  window.requestAnimationFrame = ((callback: FrameRequestCallback): number => {
    const handle = nextHandle;
    nextHandle += 1;
    queued.set(handle, callback);
    return handle;
  }) as typeof window.requestAnimationFrame;

  window.cancelAnimationFrame = (handle: number): void => {
    queued.delete(handle);
  };

  return {
    runFrame: () => {
      const due = [...queued.values()];
      queued.clear();
      due.forEach((callback) => callback(0));
    },
    pendingFrames: () => queued.size,
    restore: () => {
      window.requestAnimationFrame = nativeRequest;
      window.cancelAnimationFrame = nativeCancel;
    },
  };
}
