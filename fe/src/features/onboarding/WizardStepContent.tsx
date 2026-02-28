import { useTranslation } from "react-i18next";
import {
  Briefcase,
  Building2,
  FileText,
  Target,
  ListOrdered,
  BarChart3,
  CheckCircle,
} from "lucide-react";

const STEPS = [
  { key: "welcome", icon: Briefcase, color: "text-primary bg-primary/10" },
  { key: "company", icon: Building2, color: "text-blue-500 bg-blue-500/10" },
  { key: "resume", icon: FileText, color: "text-green-500 bg-green-500/10" },
  { key: "job", icon: Target, color: "text-orange-500 bg-orange-500/10" },
  {
    key: "stages",
    icon: ListOrdered,
    color: "text-violet-500 bg-violet-500/10",
  },
  { key: "analytics", icon: BarChart3, color: "text-pink-500 bg-pink-500/10" },
  {
    key: "done",
    icon: CheckCircle,
    color: "text-emerald-500 bg-emerald-500/10",
  },
] as const;

interface WizardStepContentProps {
  step: number;
}

export function WizardStepContent({ step }: WizardStepContentProps) {
  const { t } = useTranslation();
  const { key, icon: Icon, color } = STEPS[step];

  const title =
    key === "welcome"
      ? t("onboarding.welcome.title")
      : t(`onboarding.steps.${key}.title`);

  const description =
    key === "welcome"
      ? t("onboarding.welcome.subtitle")
      : t(`onboarding.steps.${key}.description`);

  return (
    <div className="flex flex-col items-center py-6 text-center">
      <div className={`mb-6 rounded-full p-4 ${color}`}>
        <Icon className="h-10 w-10" />
      </div>
      <h3 className="mb-3 text-xl font-semibold">{title}</h3>
      <p className="max-w-sm text-sm text-muted-foreground">{description}</p>
    </div>
  );
}

export const TOTAL_STEPS = STEPS.length;
