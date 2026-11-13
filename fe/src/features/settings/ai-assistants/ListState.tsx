import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { Skeleton } from "@/shared/ui/Skeleton";

export function ListLoading({ kind }: { kind: "tokens" | "activity" }) {
  const { t } = useTranslation();
  const [visible, setVisible] = useState(false);
  useEffect(() => {
    const timer = setTimeout(() => setVisible(true), 200);
    return () => clearTimeout(timer);
  }, []);
  return (
    <div role="status" aria-label={t("common.loading")} className="space-y-3">
      {[0, 1, 2].map((row) => (
        <div key={row} className={`space-y-3 rounded-lg border p-4 ${visible ? "" : "invisible"}`}>
          <Skeleton className="h-5 w-40" />
          <Skeleton className="h-4 w-28" />
          {kind === "tokens" && <Skeleton className="h-10 w-full" />}
        </div>
      ))}
    </div>
  );
}
