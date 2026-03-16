import { useState, useEffect, useCallback } from "react";
import { useMutation } from "@tanstack/react-query";

const MAX_RESENDS = 3;
const COOLDOWN_SECONDS = 60;
const BLOCK_DURATION_MS = 24 * 60 * 60 * 1000; // 1 day
const SUCCESS_DISPLAY_MS = 5000;
const STORAGE_PREFIX = "resend_block_";

function getStorageKey(storageKey: string): string {
  return `${STORAGE_PREFIX}${storageKey}`;
}

function getBlockedUntil(storageKey: string): number {
  try {
    const raw = localStorage.getItem(getStorageKey(storageKey));
    if (!raw) return 0;
    const ts = Number(raw);
    return ts > Date.now() ? ts : 0;
  } catch {
    return 0;
  }
}

function setBlockedUntil(storageKey: string): void {
  try {
    localStorage.setItem(
      getStorageKey(storageKey),
      String(Date.now() + BLOCK_DURATION_MS),
    );
  } catch {
    // localStorage unavailable — block is client-side only, backend still enforces
  }
}

interface UseResendCodeOptions {
  /** Function to call when resending */
  mutationFn: () => Promise<unknown>;
  /** Key for localStorage block (e.g. "verify_user@example.com") */
  storageKey: string;
  /** Max resend attempts before 24h block (default 3) */
  maxResends?: number;
}

export function useResendCode({
  mutationFn,
  storageKey,
  maxResends = MAX_RESENDS,
}: UseResendCodeOptions) {
  const [cooldown, setCooldown] = useState(COOLDOWN_SECONDS);
  const [resendCount, setResendCount] = useState(0);
  const [resendError, setResendError] = useState("");
  const [blockedUntil, setBlockedUntilState] = useState(() =>
    getBlockedUntil(storageKey),
  );

  // Countdown timer
  useEffect(() => {
    if (cooldown <= 0) return;
    const timer = setTimeout(() => setCooldown((c) => c - 1), 1000);
    return () => clearTimeout(timer);
  }, [cooldown]);

  // Check block expiry periodically (every second while blocked)
  useEffect(() => {
    if (blockedUntil <= 0) return;
    const remaining = blockedUntil - Date.now();
    if (remaining <= 0) {
      setBlockedUntilState(0);
      return;
    }
    const timer = setTimeout(
      () => setBlockedUntilState(getBlockedUntil(storageKey)),
      Math.min(remaining, 60_000),
    );
    return () => clearTimeout(timer);
  }, [blockedUntil, storageKey]);

  const mutation = useMutation({
    mutationFn,
    onSuccess: () => {
      const newCount = resendCount + 1;
      setResendCount(newCount);
      setCooldown(COOLDOWN_SECONDS);
      setResendError("");
      setTimeout(() => mutation.reset(), SUCCESS_DISPLAY_MS);

      if (newCount >= maxResends) {
        setBlockedUntil(storageKey);
        setBlockedUntilState(Date.now() + BLOCK_DURATION_MS);
      }
    },
    onError: () => {
      setResendError("auth.resendFailed");
    },
  });

  const isBlocked = blockedUntil > Date.now();
  const isLimitReached = resendCount >= maxResends || isBlocked;
  const isDisabled = mutation.isPending || cooldown > 0 || isLimitReached;

  const resend = useCallback(() => {
    if (!isDisabled) {
      setResendError("");
      mutation.mutate();
    }
  }, [isDisabled, mutation]);

  return {
    cooldown,
    resendCount,
    resendError,
    isLimitReached,
    isDisabled,
    isPending: mutation.isPending,
    isSuccess: mutation.isSuccess,
    resend,
  };
}
