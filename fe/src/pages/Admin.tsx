import { useTranslation } from "react-i18next";
import { useLayoutEffect, useRef, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { ShieldCheck, Search } from "lucide-react";
import { adminService } from "@/services/adminService";
import { useAdminAccess, useAdminScope } from "@/shared/hooks/useAdminAccess";
import { AdminUserDetail } from "@/features/admin/AdminUserDetail";
import { Button } from "@/shared/ui/Button";
import { Input } from "@/shared/ui/Input";
import { adminKeys } from "@/shared/lib/adminCapability";

export default function AdminPage() {
  const { t } = useTranslation();
  const access = useAdminAccess();
  const scope = useAdminScope();
  if (access.isPending)
    return (
      <div
        role="status"
        aria-label={t("admin.checking")}
        className="h-64 animate-pulse motion-reduce:animate-none rounded-lg bg-muted"
      />
    );
  if (access.isError || !access.data?.allowed)
    return (
      <section className="space-y-3 rounded-xl border p-6">
        <h1 className="text-xl font-semibold">{t("admin.required")}</h1>
        <p className="text-muted-foreground">{t("admin.denied")}</p>
        <Button variant="outline" onClick={() => access.refetch()}>
          {t("admin.checkAgain")}
        </Button>
      </section>
    );
  return <AdminUsers key={`${scope.actor}:${scope.session}`} />;
}

function AdminUsers() {
  const { t } = useTranslation();
  const scope = useAdminScope();
  const opener = useRef<HTMLButtonElement | null>(null);
  const restoreOpener = useRef(false);
  const list = useRef<HTMLElement>(null);
  const [input, setInput] = useState("");
  const [search, setSearch] = useState("");
  const [page, setPage] = useState(1);
  const [selected, setSelected] = useState<string | null>(null);
  useLayoutEffect(() => {
    if (selected || !restoreOpener.current) return;
    restoreOpener.current = false;
    const target = opener.current?.isConnected ? opener.current : list.current;
    target?.scrollIntoView?.({ block: "center" });
    target?.focus({ preventScroll: true });
  }, [selected]);
  const users = useQuery({
    queryKey: [...adminKeys.users(scope), search, page],
    queryFn: ({ signal }) => adminService.users(search, page, signal),
    enabled: !scope.denied && !!scope.actor,
    retry: false,
  });
  return (
    <div className="mx-auto max-w-7xl space-y-6">
      <header>
        <h1 className="flex items-center gap-2 text-2xl font-bold">
          <ShieldCheck aria-hidden="true" className="h-6 w-6" />
          {t("admin.title")}
        </h1>
        <p className="mt-1 text-sm text-muted-foreground">
          {t("admin.subtitle")}
        </p>
      </header>
      <div
        className={
          selected
            ? "grid items-start gap-6 xl:grid-cols-[minmax(0,1fr)_minmax(320px,440px)]"
            : ""
        }
      >
        <section
          ref={list}
          tabIndex={-1}
          className="min-w-0 space-y-4"
          aria-label={t("admin.users")}
        >
          <form
            className="flex gap-2"
            onSubmit={(event) => {
              event.preventDefault();
              setSearch(input.trim());
              setPage(1);
            }}
          >
            <Input
              aria-label={t("admin.searchLabel")}
              placeholder={t("admin.searchPlaceholder")}
              value={input}
              maxLength={200}
              onChange={(event) => setInput(event.target.value)}
            />
            <Button type="submit">
              <Search aria-hidden="true" className="mr-2 h-4 w-4" />
              {t("admin.search")}
            </Button>
          </form>
          <div className="overflow-x-auto rounded-xl border bg-card">
            <table className="w-full text-left text-sm">
              <thead className="border-b bg-muted/40 text-muted-foreground">
                <tr>
                  <th scope="col" className="p-3 font-medium">
                    {t("admin.user")}
                  </th>
                  <th scope="col" className="p-3 font-medium">
                    {t("admin.provider")}
                  </th>
                  <th scope="col" className="p-3 font-medium">
                    {t("admin.manual")}
                  </th>
                </tr>
              </thead>
              <tbody>
                {users.isPending ? (
                  Array.from({ length: 6 }, (_, index) => (
                    <tr
                      key={index}
                      className="h-20 border-b"
                      aria-hidden="true"
                    >
                      <td className="p-3">
                        <div className="h-10 w-40 animate-pulse motion-reduce:animate-none rounded bg-muted" />
                      </td>
                      <td className="p-3">
                        <div className="h-5 w-20 animate-pulse motion-reduce:animate-none rounded bg-muted" />
                      </td>
                      <td className="p-3">
                        <div className="h-5 w-24 animate-pulse motion-reduce:animate-none rounded bg-muted" />
                      </td>
                    </tr>
                  ))
                ) : users.isError ? (
                  <tr>
                    <td colSpan={3} className="p-5">
                      <p role="alert">{t("admin.loadError")}</p>
                      <Button
                        className="mt-3"
                        variant="outline"
                        onClick={() => users.refetch()}
                      >
                        {t("admin.retry")}
                      </Button>
                    </td>
                  </tr>
                ) : users.data.users.length === 0 ? (
                  <tr>
                    <td colSpan={3} className="p-5 text-muted-foreground">
                      {t("admin.empty")}
                    </td>
                  </tr>
                ) : (
                  users.data.users.map((user) => (
                    <tr key={user.id} className="h-20 border-b last:border-0">
                      <td className="p-3">
                        <button
                          type="button"
                          aria-pressed={selected === user.id}
                          className="min-h-11 max-w-64 rounded text-left font-medium text-foreground underline-offset-4 hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                          onClick={(event) => {
                            opener.current = event.currentTarget;
                            setSelected(user.id);
                          }}
                        >
                          <span className="block break-words">
                            {user.name || user.email}
                          </span>
                          <span className="block break-all text-xs font-normal text-muted-foreground">
                            {user.email}
                          </span>
                        </button>
                      </td>
                      <td className="p-3">
                        <span className="block capitalize">
                          {user.provider_plan}
                        </span>
                        <span className="text-xs text-muted-foreground">
                          {user.provider_status}
                        </span>
                      </td>
                      <td className="p-3 text-xs">
                        {user.manual_until ? (
                          <>
                            {t("admin.until", {
                              date: new Date(
                                user.manual_until,
                              ).toLocaleDateString(),
                            })}
                          </>
                        ) : (
                          t("admin.none")
                        )}
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
          {users.isPending && (
            <p role="status" className="sr-only">
              {t("admin.loadingUsers")}
            </p>
          )}
          <div className="flex items-center justify-between gap-3">
            <p className="text-sm tabular-nums text-muted-foreground">
              {t("admin.page", { page })}
              {users.data
                ? ` · ${t("admin.total", { total: users.data.total })}`
                : ""}
            </p>
            <div className="flex gap-2">
              <Button
                variant="outline"
                disabled={page === 1 || users.isPending}
                onClick={() => setPage((current) => current - 1)}
              >
                {t("admin.previous")}
              </Button>
              <Button
                variant="outline"
                disabled={
                  !users.data ||
                  page * users.data.limit >= users.data.total ||
                  users.isPending
                }
                onClick={() => setPage((current) => current + 1)}
              >
                {t("admin.next")}
              </Button>
            </div>
          </div>
        </section>
        {selected && (
          <AdminUserDetail
            key={selected}
            id={selected}
            onClose={() => {
              restoreOpener.current = true;
              setSelected(null);
            }}
          />
        )}
      </div>
    </div>
  );
}
