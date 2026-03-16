import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Sparkles, Loader2, Check, X, Briefcase } from "lucide-react";
import { useResumeBuilderStore } from "@/stores/resumeBuilderStore";
import { useAISuggestions } from "../hooks/useAISuggestions";
import { useSubscription } from "@/shared/hooks/useSubscription";
import { UpgradeBanner } from "@/features/subscription/components/UpgradeBanner";
import { Button } from "@/shared/ui/Button";
import { Textarea } from "@/shared/ui/Textarea";
import { Label } from "@/shared/ui/Label";
import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/Card";
import { cn } from "@/shared/lib/utils";

type InstructionKey = "concise" | "metrics" | "professional" | "action_verbs";

const INSTRUCTION_KEYS: InstructionKey[] = [
  "concise",
  "metrics",
  "professional",
  "action_verbs",
];

export function AIAssistantPanel() {
  const { t } = useTranslation();
  const resume = useResumeBuilderStore((s) => s.resume);
  const updateSummary = useResumeBuilderStore((s) => s.updateSummary);
  const updateExperience = useResumeBuilderStore((s) => s.updateExperience);
  const { suggestSummary, suggestBullets, improveText } = useAISuggestions();
  const { canCreate } = useSubscription();
  const aiLimitReached = !canCreate("ai");

  const [textToImprove, setTextToImprove] = useState("");
  const [selectedInstruction, setSelectedInstruction] =
    useState<InstructionKey>("concise");
  const [selectedExpId, setSelectedExpId] = useState<string>("");

  const experiences = resume?.experiences ?? [];

  const handleSuggestSummary = () => {
    if (!resume) return;
    suggestSummary.mutate({ resume_id: resume.id });
  };

  const handleApplySummary = () => {
    if (!suggestSummary.data) return;
    updateSummary({ content: suggestSummary.data.summary });
    suggestSummary.reset();
  };

  const handleSuggestBullets = () => {
    const exp = experiences.find((e) => e.id === selectedExpId);
    if (!exp) return;
    suggestBullets.mutate({
      job_title: exp.position,
      company: exp.company,
      current_description: exp.description,
    });
  };

  const handleApplyBullets = () => {
    if (!suggestBullets.data || !selectedExpId) return;
    const description = suggestBullets.data.bullets
      .map((b) => `• ${b}`)
      .join("\n");
    updateExperience(selectedExpId, { description });
    suggestBullets.reset();
  };

  const handleImproveText = () => {
    if (!textToImprove.trim()) return;
    const instruction = t(
      `resumeBuilder.ai.instructions.${selectedInstruction}`,
    );
    improveText.mutate({ text: textToImprove, instruction });
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-2">
        <Sparkles className="h-5 w-5 text-primary" />
        <h2 className="text-lg font-semibold">{t("resumeBuilder.ai.title")}</h2>
      </div>

      {aiLimitReached && <UpgradeBanner resource="ai" />}

      {/* Suggest Summary */}
      <section className="space-y-3">
        <Button
          onClick={handleSuggestSummary}
          disabled={suggestSummary.isPending || !resume || aiLimitReached}
          variant="outline"
          className="w-full"
        >
          {suggestSummary.isPending ? (
            <>
              <Loader2 className="h-4 w-4 animate-spin" />
              {t("resumeBuilder.ai.generating")}
            </>
          ) : (
            <>
              <Sparkles className="h-4 w-4" />
              {t("resumeBuilder.ai.suggestSummary")}
            </>
          )}
        </Button>

        {suggestSummary.isError && (
          <p className="text-sm text-destructive">{t("common.error")}</p>
        )}

        {suggestSummary.data && (
          <SuggestionCard
            title={t("resumeBuilder.ai.result")}
            content={suggestSummary.data.summary}
            onApply={handleApplySummary}
            onDismiss={() => suggestSummary.reset()}
          />
        )}
      </section>

      {/* Suggest Bullets for Work Experience */}
      {experiences.length > 0 && (
        <section className="space-y-3">
          <div className="flex items-center gap-2">
            <Briefcase className="h-4 w-4 text-muted-foreground" />
            <Label>{t("resumeBuilder.ai.suggestBullets")}</Label>
          </div>

          <select
            value={selectedExpId}
            onChange={(e) => {
              setSelectedExpId(e.target.value);
              suggestBullets.reset();
            }}
            className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
          >
            <option value="">{t("resumeBuilder.ai.selectExperience")}</option>
            {experiences.map((exp) => (
              <option key={exp.id} value={exp.id}>
                {exp.position
                  ? `${exp.position}${exp.company ? ` — ${exp.company}` : ""}`
                  : exp.company || exp.id.slice(0, 8)}
              </option>
            ))}
          </select>

          <Button
            onClick={handleSuggestBullets}
            disabled={
              suggestBullets.isPending || !selectedExpId || aiLimitReached
            }
            variant="outline"
            className="w-full"
          >
            {suggestBullets.isPending ? (
              <>
                <Loader2 className="h-4 w-4 animate-spin" />
                {t("resumeBuilder.ai.generating")}
              </>
            ) : (
              <>
                <Sparkles className="h-4 w-4" />
                {t("resumeBuilder.ai.suggestBullets")}
              </>
            )}
          </Button>

          {suggestBullets.isError && (
            <p className="text-sm text-destructive">{t("common.error")}</p>
          )}

          {suggestBullets.data && (
            <Card className="border-primary/20 bg-primary/5">
              <CardHeader className="p-4 pb-2">
                <CardTitle className="flex items-center gap-2 text-sm font-medium">
                  <Sparkles className="h-3.5 w-3.5 text-primary" />
                  {t("resumeBuilder.ai.result")}
                </CardTitle>
              </CardHeader>
              <CardContent className="p-4 pt-0">
                <ul className="mb-3 space-y-1.5">
                  {suggestBullets.data.bullets.map((bullet, i) => (
                    <li key={i} className="flex items-start gap-2 text-sm">
                      <span className="mt-1.5 block h-1.5 w-1.5 shrink-0 rounded-full bg-primary" />
                      {bullet}
                    </li>
                  ))}
                </ul>
                <div className="flex gap-2">
                  <Button size="sm" onClick={handleApplyBullets}>
                    <Check className="h-3.5 w-3.5" />
                    {t("resumeBuilder.ai.apply")}
                  </Button>
                  <Button
                    size="sm"
                    variant="ghost"
                    onClick={() => suggestBullets.reset()}
                  >
                    <X className="h-3.5 w-3.5" />
                    {t("resumeBuilder.ai.dismiss")}
                  </Button>
                </div>
              </CardContent>
            </Card>
          )}
        </section>
      )}

      {/* Improve Text */}
      <section className="space-y-3">
        <Label>{t("resumeBuilder.ai.improveText")}</Label>

        <Textarea
          value={textToImprove}
          onChange={(e) => setTextToImprove(e.target.value)}
          placeholder={t("resumeBuilder.ai.improveText")}
          rows={4}
        />

        <div className="space-y-2">
          <Label>{t("resumeBuilder.ai.instruction")}</Label>
          <div className="flex flex-wrap gap-2">
            {INSTRUCTION_KEYS.map((key) => (
              <button
                key={key}
                onClick={() => setSelectedInstruction(key)}
                className={cn(
                  "rounded-full border px-3 py-1 text-xs transition-colors",
                  selectedInstruction === key
                    ? "border-primary bg-primary text-primary-foreground"
                    : "border-border text-muted-foreground hover:border-primary/50 hover:text-foreground",
                )}
              >
                {t(`resumeBuilder.ai.instructions.${key}`)}
              </button>
            ))}
          </div>
        </div>

        <Button
          onClick={handleImproveText}
          disabled={
            improveText.isPending || !textToImprove.trim() || aiLimitReached
          }
          variant="outline"
          className="w-full"
        >
          {improveText.isPending ? (
            <>
              <Loader2 className="h-4 w-4 animate-spin" />
              {t("resumeBuilder.ai.generating")}
            </>
          ) : (
            <>
              <Sparkles className="h-4 w-4" />
              {t("resumeBuilder.ai.improveText")}
            </>
          )}
        </Button>

        {improveText.isError && (
          <p className="text-sm text-destructive">{t("common.error")}</p>
        )}

        {improveText.data && (
          <SuggestionCard
            title={t("resumeBuilder.ai.result")}
            content={improveText.data.improved}
            onApply={() => {
              setTextToImprove(improveText.data.improved);
              improveText.reset();
            }}
            onDismiss={() => improveText.reset()}
          />
        )}
      </section>
    </div>
  );
}

interface SuggestionCardProps {
  title: string;
  content: string;
  onApply: () => void;
  onDismiss: () => void;
}

function SuggestionCard({
  title,
  content,
  onApply,
  onDismiss,
}: SuggestionCardProps) {
  const { t } = useTranslation();

  return (
    <Card className="border-primary/20 bg-primary/5">
      <CardHeader className="p-4 pb-2">
        <CardTitle className="flex items-center gap-2 text-sm font-medium">
          <Sparkles className="h-3.5 w-3.5 text-primary" />
          {title}
        </CardTitle>
      </CardHeader>
      <CardContent className="p-4 pt-0">
        <p className="mb-3 whitespace-pre-wrap text-sm text-foreground">
          {content}
        </p>
        <div className="flex gap-2">
          <Button size="sm" onClick={onApply}>
            <Check className="h-3.5 w-3.5" />
            {t("resumeBuilder.ai.apply")}
          </Button>
          <Button size="sm" variant="ghost" onClick={onDismiss}>
            <X className="h-3.5 w-3.5" />
            {t("resumeBuilder.ai.dismiss")}
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}
