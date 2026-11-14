import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { Skeleton } from "@/shared/ui/Skeleton";

export function ConsentLoading() {
  const { t } = useTranslation();
  const [visible, setVisible] = useState(false);
  useEffect(() => {
    const timer = setTimeout(() => setVisible(true), 200);
    return () => clearTimeout(timer);
  }, []);
  return (
    <div role="status" aria-label={t("common.loading")}>
      <div aria-hidden className={`space-y-5 ${visible ? "" : "invisible"}`}>
        <Skeleton className="h-7 w-4/5" />
        <Skeleton className="h-16 w-full" />
        <div className="space-y-2 rounded-lg border p-4">
          <Skeleton className="h-5 w-1/2" /><Skeleton className="h-7 w-3/4" />
        </div>
        <div className="space-y-3">
          <Skeleton className="h-6 w-2/3" />
          {[0, 1, 2, 3].map((row) => <Skeleton key={row} className="h-10 w-full" />)}
        </div>
        <Skeleton className="h-10 w-full" />
        <div className="flex justify-end gap-3"><Skeleton className="h-11 w-24" /><Skeleton className="h-11 w-24" /></div>
      </div>
    </div>
  );
}
