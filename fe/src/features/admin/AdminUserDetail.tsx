import { useTranslation } from "react-i18next";
import { useEffect, useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { adminService, type AdminDetail } from "@/services/adminService";
import { ApiError } from "@/services/api";
import { Button } from "@/shared/ui/Button";
import { Input } from "@/shared/ui/Input";
import { Select } from "@/shared/ui/Select";
import { useAdminScope } from "@/shared/hooks/useAdminAccess";
import {
  adminKeys,
  assertCurrentAdminRequest,
  currentAdminRequest,
  isCurrentAdminRequest,
} from "@/shared/lib/adminCapability";

const date = (value: string | null) =>
  value ? new Date(value).toLocaleString() : "—";

export function AdminUserDetail({
  id,
  onClose,
}: {
  id: string;
  onClose: () => void;
}) {
  const { t } = useTranslation();
  const scope = useAdminScope();
  const detail = useQuery({
    queryKey: adminKeys.detail(scope, id),
    queryFn: ({ signal }) => adminService.detail(id, signal),
    enabled: !scope.denied && !!scope.actor,
    retry: false,
  });
  if (scope.denied || !scope.actor) return null;
  if (detail.isPending)
    return (
      <div
        role="status"
        aria-label={t("admin.loadingDetail")}
        className="h-96 animate-pulse motion-reduce:animate-none rounded-lg bg-muted"
      />
    );
  if (!detail.data)
    return (
      <div role="alert" className="rounded-lg border p-5">
        {t("admin.detailError")}{" "}
        <Button variant="outline" onClick={() => detail.refetch()}>
          {t("admin.retry")}
        </Button>
        <Button variant="ghost" onClick={onClose}>
          {t("admin.close")}
        </Button>
      </div>
    );
  return <DetailForm key={id} detail={detail.data} onClose={onClose} />;
}

function DetailForm({
  detail,
  onClose,
}: {
  detail: AdminDetail;
  onClose: () => void;
}) {
  const { t } = useTranslation();
  const { user } = detail;
  const panel = useRef<HTMLElement>(null);
  useEffect(() => {
    panel.current?.scrollIntoView?.({ block: "start" });
    panel.current?.focus({ preventScroll: true });
  }, []);
  const queryClient = useQueryClient();
  const scope = useAdminScope();
  const requestScope = useRef(currentAdminRequest());
  const [name, setName] = useState(user.name);
  const [locale, setLocale] = useState(user.locale);
  const [days, setDays] = useState(30);
  const [reason, setReason] = useState("");
  const [revokeReason, setRevokeReason] = useState("");
  const [notice, setNotice] = useState("");
  // Unknown outcomes retain their immutable receipt until the same request succeeds.
  const grantDraft = useRef<{
    id: string;
    days: number;
    reason: string;
  } | null>(null);
  const inFlight = useRef(false);
  const [lockedDraft, setLockedDraft] = useState(false);
  const [needsRefresh, setNeedsRefresh] = useState(false);
  const [refreshing, setRefreshing] = useState(false);
  const clearGrantReceipt = () => {
    grantDraft.current = null;
    setLockedDraft(false);
  };
  const refreshConflict = async () => {
    if (!isCurrentAdminRequest(requestScope.current)) return;
    setRefreshing(true);
    try {
      await queryClient.fetchQuery({
        queryKey: adminKeys.detail(scope, user.id),
        queryFn: ({ signal }) => {
          assertCurrentAdminRequest(requestScope.current);
          return adminService.detail(user.id, signal);
        },
        staleTime: 0,
      });
      if (!isCurrentAdminRequest(requestScope.current)) return;
      // A definitive conflict did not apply this draft. Reconcile current access
      // before allowing a corrected request with a new receipt.
      clearGrantReceipt();
      setNeedsRefresh(false);
      setNotice(t("admin.grantConflictRefreshed"));
    } catch {
      setNotice(t("admin.grantConflictRefreshError"));
    } finally {
      setRefreshing(false);
    }
  };
  const mutation = useMutation({
    mutationFn: async (action: string | { isTest: boolean }) => {
      assertCurrentAdminRequest(requestScope.current);
      if (typeof action !== "string")
        return adminService.setTest(user.id, action.isTest);
      if (action === "profile")
        return adminService.update(user.id, { name: name.trim(), locale });
      if (action === "grant") {
        const draft = grantDraft.current ?? {
          id: crypto.randomUUID(),
          days,
          reason: reason.trim(),
        };
        grantDraft.current = draft;
        setLockedDraft(true);
        return adminService.grant(user.id, draft);
      }
      return adminService.revoke(user.id, action, revokeReason.trim());
    },
    onSuccess: async (_data, action) => {
      if (!isCurrentAdminRequest(requestScope.current)) return;
      setNotice(
        typeof action !== "string"
          ? t("admin.testSaved")
          : action === "profile"
            ? t("admin.profileSaved")
            : action === "grant"
              ? t("admin.granted")
              : t("admin.revokedNotice"),
      );
      await Promise.all([
        queryClient.cancelQueries({
          queryKey: adminKeys.detail(scope, user.id),
        }),
        queryClient.cancelQueries({ queryKey: adminKeys.users(scope) }),
      ]);
      if (!isCurrentAdminRequest(requestScope.current)) return;
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: adminKeys.detail(scope, user.id),
        }),
        queryClient.invalidateQueries({ queryKey: adminKeys.users(scope) }),
        queryClient.invalidateQueries({ queryKey: ["subscription"] }),
      ]);
      if (action === "grant") {
        clearGrantReceipt();
        setReason("");
      }
    },
    onError: async (error, action) => {
      if (!isCurrentAdminRequest(requestScope.current)) return;
      if (
        action === "grant" &&
        error instanceof ApiError &&
        !error.unstructuredResponse
      ) {
        if (error.status === 400) {
          clearGrantReceipt();
          setNotice(t("admin.grantValidationError"));
          return;
        }
        if (error.status === 409) {
          setNeedsRefresh(true);
          await refreshConflict();
          return;
        }
      }
      setNotice(t("admin.saveError"));
    },
    onSettled: () => {
      inFlight.current = false;
    },
  });
  const save = (action: string) => {
    if (inFlight.current || !isCurrentAdminRequest(requestScope.current))
      return;
    inFlight.current = true;
    setNotice("");
    // Capture the desired value before an offline mutation can pause and the
    // detail cache can refresh to a different flag value.
    mutation.mutate(
      action === "test-account" ? { isTest: !user.is_test } : action,
    );
  };
  const active = detail.manual_grants.find((grant) => grant.active);
  const pending = mutation.isPending;
  return (
    <section
      ref={panel}
      tabIndex={-1}
      aria-labelledby="admin-detail-title"
      className="min-w-0 scroll-mt-20 space-y-6 rounded-xl border bg-card p-4 sm:p-6"
    >
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0">
          <h2 id="admin-detail-title" className="text-lg font-semibold">
            {t("admin.detailTitle")}
          </h2>
          <p className="break-all text-sm text-muted-foreground">
            {user.email}
          </p>
        </div>
        <Button variant="ghost" disabled={pending} onClick={onClose}>
          {t("admin.close")}
        </Button>
      </div>
      <p className="break-all text-xs text-muted-foreground">
        ID: {user.id} ·{" "}
        {user.email_verified ? t("admin.verified") : t("admin.unverified")}
      </p>
      <div className="space-y-3 rounded-lg border bg-muted/20 p-4">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <h3 className="font-semibold">{t("admin.testAccount")}</h3>
          {user.is_test && (
            <span className="rounded border border-amber-500/30 bg-amber-500/10 px-2 py-1 text-xs font-medium">
              {t("admin.testBadge")}
            </span>
          )}
        </div>
        <p className="text-sm text-muted-foreground">
          {t("admin.testDescription")}
        </p>
        <Button
          type="button"
          variant="outline"
          aria-pressed={user.is_test}
          disabled={pending}
          onClick={() => save("test-account")}
        >
          {pending && typeof mutation.variables === "object"
            ? t("admin.saving")
            : user.is_test
              ? t("admin.unmarkTest")
              : t("admin.markTest")}
        </Button>
      </div>
      <form
        className="space-y-3"
        onSubmit={(event) => {
          event.preventDefault();
          save("profile");
        }}
      >
        <label className="block text-sm font-medium">
          {t("admin.name")}
          <Input
            value={name}
            onChange={(event) => setName(event.target.value)}
            required
            maxLength={100}
            disabled={pending}
          />
        </label>
        <label className="block text-sm font-medium">
          {t("admin.locale")}
          <Select
            className="min-h-11"
            value={locale}
            onChange={(event) => setLocale(event.target.value)}
            disabled={pending}
          >
            <option value="en">English</option>
            <option value="uk">Українська</option>
            <option value="ru">Русский</option>
          </Select>
        </label>
        <Button type="submit" disabled={pending || !name.trim()}>
          {pending && mutation.variables === "profile"
            ? t("admin.saving")
            : t("admin.saveProfile")}
        </Button>
      </form>
      <div className="space-y-2 border-t pt-4">
        <h3 className="font-semibold">{t("admin.accessBilling")}</h3>
        <p>
          {t("admin.effectivePlan")}:{" "}
          <strong className="capitalize">{detail.effective_plan}</strong>
        </p>
        <dl className="space-y-2 text-sm">
          <div>
            <dt className="text-muted-foreground">
              {t("admin.providerStatus")}
            </dt>
            <dd>
              {user.provider_plan} / {user.provider_status}
            </dd>
          </div>
          <div>
            <dt className="text-muted-foreground">{t("admin.providerIds")}</dt>
            <dd className="break-all">
              {detail.provider_subscription_id ?? "—"} /{" "}
              {detail.provider_account_id ?? "—"}
            </dd>
          </div>
          <div>
            <dt className="text-muted-foreground">
              {t("admin.providerDates")}
            </dt>
            <dd>
              {date(detail.provider_period_end)} /{" "}
              {date(detail.provider_cancel_at)}
            </dd>
          </div>
        </dl>
      </div>
      <div className="space-y-3 border-t pt-4">
        <h3 className="font-semibold">{t("admin.manualTitle")}</h3>
        <p className="text-sm text-muted-foreground">
          {t("admin.manualDescription")}
        </p>
        {active ? (
          <form
            className="space-y-3"
            onSubmit={(event) => {
              event.preventDefault();
              save(active.id);
            }}
          >
            <p>{t("admin.activeUntil", { date: date(active.period_end) })}</p>
            <label className="block text-sm font-medium">
              {t("admin.revokeReason")}
              <Input
                value={revokeReason}
                onChange={(event) => setRevokeReason(event.target.value)}
                required
                minLength={3}
                maxLength={500}
                disabled={pending}
              />
            </label>
            <Button
              variant="destructive"
              type="submit"
              disabled={pending || revokeReason.trim().length < 3}
            >
              {t("admin.revoke")}
            </Button>
          </form>
        ) : (
          <form
            className="space-y-3"
            onSubmit={(event) => {
              event.preventDefault();
              save("grant");
            }}
          >
            <label className="block text-sm font-medium">
              {t("admin.days")}
              <Input
                type="number"
                min={1}
                max={31}
                step={1}
                value={days}
                onChange={(event) => setDays(Number(event.target.value))}
                required
                disabled={pending || lockedDraft}
              />
            </label>
            <label className="block text-sm font-medium">
              {t("admin.grantReason")}
              <Input
                value={reason}
                onChange={(event) => setReason(event.target.value)}
                minLength={3}
                maxLength={500}
                required
                disabled={pending || lockedDraft}
              />
            </label>
            <Button
              type="submit"
              disabled={
                pending ||
                needsRefresh ||
                reason.trim().length < 3 ||
                !Number.isInteger(days) ||
                days < 1 ||
                days > 31
              }
            >
              {lockedDraft ? t("admin.retryGrant") : t("admin.grant")}
            </Button>
          </form>
        )}
      </div>
      {needsRefresh && (
        <Button
          variant="outline"
          disabled={pending || refreshing}
          onClick={() => void refreshConflict()}
        >
          {t("admin.refreshAccess")}
        </Button>
      )}
      {notice && (
        <p
          role={mutation.isError ? "alert" : "status"}
          className="rounded-md border p-3 text-sm"
        >
          {notice}
        </p>
      )}
      <div className="space-y-3 border-t pt-4">
        <h3 className="font-semibold">
          {t("admin.history")}{" "}
          <span className="text-sm font-normal text-muted-foreground">
            {t("admin.latest")}
          </span>
        </h3>
        {detail.manual_grants.length === 0 ? (
          <p className="text-sm text-muted-foreground">{t("admin.noGrants")}</p>
        ) : (
          detail.manual_grants.map((grant) => (
            <article key={grant.id} className="rounded-lg border p-3 text-sm">
              <p className="font-medium">
                {grant.revoked_at
                  ? t("admin.revoked")
                  : grant.active
                    ? t("admin.active")
                    : t("admin.expired")}{" "}
                · {t("admin.scheduledEnd", { date: date(grant.period_end) })}
              </p>
              <p className="tabular-nums">
                {t("admin.creditsUsed", {
                  used: grant.used,
                  credits: grant.credits,
                })}
              </p>
              <p className="break-words">{grant.reason}</p>
              <p className="break-all text-xs text-muted-foreground">
                {t("admin.grantedBy", {
                  actor: grant.actor_id,
                  date: date(grant.period_start),
                })}
              </p>
              {grant.revoked_at && (
                <p className="break-words">
                  {t("admin.revokedBy", {
                    date: date(grant.revoked_at),
                    actor: grant.revoked_by,
                    reason: grant.revoke_reason,
                  })}
                </p>
              )}
            </article>
          ))
        )}
      </div>
    </section>
  );
}
