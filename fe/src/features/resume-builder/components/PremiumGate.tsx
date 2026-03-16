import type { ReactNode } from "react";
import { useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { Lock } from "lucide-react";
import { useSubscription } from "@/shared/hooks/useSubscription";
import { Button } from "@/shared/ui/Button";
import { Card, CardContent } from "@/shared/ui/Card";

interface PremiumGateProps {
  feature: string;
  children: ReactNode;
}

export function PremiumGate({ feature, children }: PremiumGateProps) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { isFree } = useSubscription();

  if (!isFree) {
    return <>{children}</>;
  }

  return (
    <Card className="border-dashed">
      <CardContent className="flex flex-col items-center gap-4 p-8 text-center">
        <div className="flex h-12 w-12 items-center justify-center rounded-full bg-muted">
          <Lock className="h-6 w-6 text-muted-foreground" />
        </div>

        <div className="space-y-1.5">
          <h3 className="text-lg font-semibold">{t("premium.title")}</h3>
          <p className="text-sm text-muted-foreground">
            {t("premium.description", { feature })}
          </p>
        </div>

        <Button onClick={() => navigate("/app/settings")}>
          {t("premium.upgrade")}
        </Button>
      </CardContent>
    </Card>
  );
}
