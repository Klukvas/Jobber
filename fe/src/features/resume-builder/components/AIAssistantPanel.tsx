import { isAIActionCancelled } from "@/shared/lib/aiCredits";
import { useAIAvailability, aiErrorKey } from "@/shared/hooks/useAIAvailability";
import { AIAvailabilityNotice } from "@/shared/ui/AIAvailabilityNotice";
import { useState } from "react";
import { flushResumeDocument } from "../lib/resumeDocumentSync";
import { resolveResumeSectionIds } from "../hooks/useSectionPersistence";
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
  const { unavailable, isChecking, markUnavailable } = useAIAvailability();
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

  const [isSyncing, setIsSyncing] = useState(false);
  const [saveError, setSaveError] = useState(false);
  const [summaryResult, setSummaryResult] = useState<{
    revision: string;
    summary: string;
  } | null>(null);
  const [bulletResult, setBulletResult] = useState<{
    revision: string;
    entryId: string;
    bullets: string[];
  } | null>(null);
  const [textResult, setTextResult] = useState<{
    revision: string;
    source: string;
    improved: string;
  } | null>(null);
  const revision = resume ? JSON.stringify(resume) : null;
  const isCurrent = (origin: string) =>
    JSON.stringify(useResumeBuilderStore.getState().resume) === origin;
  const summary = summaryResult?.revision === revision ? summaryResult : null;
  const bullets =
    bulletResult?.revision === revision &&
    bulletResult.entryId === selectedExpId
      ? bulletResult
      : null;
  const improved =
    textResult?.revision === revision && textResult.source === textToImprove
      ? textResult
      : null;

  // Analysis reads the stored document. Capture its revision only after all
  // writes have landed, and never let a result cross a document/edit boundary.
  const prepareAnalysis = async () => {
    if (!resume || unavailable || isChecking || isSyncing || suggestSummary.isPending || suggestBullets.isPending || improveText.isPending) return null;
    setIsSyncing(true);
    setSaveError(false);
    try {
      await flushResumeDocument(resume.id);
      const live = useResumeBuilderStore.getState().resume;
      if (!live || live.id !== resume.id) return null;
      return { resume: live, revision: JSON.stringify(live) };
    } catch {
      if (useResumeBuilderStore.getState().resume?.id === resume.id)
        setSaveError(true);
      return null;
    } finally {
      setIsSyncing(false);
    }
  };

  const handleSuggestSummary = async () => {
    const origin = await prepareAnalysis();
    if (!origin) return;
    setSummaryResult(null);
    suggestSummary.mutate(
      { resume_id: origin.resume.id },
      {
        onError: markUnavailable,
        onSuccess: (data) => {
          if (isCurrent(origin.revision))
            setSummaryResult({ ...data, revision: origin.revision });
        },
      },
    );
  };

  const handleApplySummary = () => {
    if (!summary || !isCurrent(summary.revision)) return;
    updateSummary({ content: summary.summary });
    setSummaryResult(null);
    suggestSummary.reset();
  };

  const handleSuggestBullets = async () => {
    if (!resume) return;
    const selectedIndex = experiences.findIndex(
      (entry) => entry.id === selectedExpId,
    );
    if (selectedIndex < 0) return;
    const selectedDocument = resume;
    const origin = await prepareAnalysis();
    if (!origin) return;
    // A pending CREATE can acquire its server ID during the flush. Resolve
    // that alias in the captured selection instead of losing the selection.
    const selectedId =
      resolveResumeSectionIds(selectedDocument).experiences[selectedIndex]?.id;
    const exp = origin.resume.experiences.find(
      (entry) => entry.id === selectedId,
    );
    if (!exp) return;
    setSelectedExpId(exp.id);
    setBulletResult(null);
    suggestBullets.mutate(
      {
        job_title: exp.position,
        company: exp.company,
        current_description: exp.description,
      },
      {
        onError: markUnavailable,
        onSuccess: (data) => {
          if (isCurrent(origin.revision))
            setBulletResult({
              ...data,
              revision: origin.revision,
              entryId: exp.id,
            });
        },
      },
    );
  };

  const handleApplyBullets = () => {
    if (!bullets || !isCurrent(bullets.revision)) return;
    updateExperience(bullets.entryId, {
      description: bullets.bullets.map((bullet) => `• ${bullet}`).join("\n"),
    });
    setBulletResult(null);
    suggestBullets.reset();
  };

  const handleImproveText = async () => {
    if (!textToImprove.trim()) return;
    const source = textToImprove;
    const instruction = t(
      `resumeBuilder.ai.instructions.${selectedInstruction}`,
    );
    const origin = await prepareAnalysis();
    if (!origin) return;
    setTextResult(null);
    improveText.mutate(
      { text: source, instruction },
      {
        onError: markUnavailable,
        onSuccess: (data) => {
          if (isCurrent(origin.revision))
            setTextResult({ ...data, source, revision: origin.revision });
        },
      },
    );
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-2">
        <Sparkles className="h-5 w-5 text-primary" />
        <h2 className="text-lg font-semibold">{t("resumeBuilder.ai.title")}</h2>
      </div>

      <AIAvailabilityNotice unavailable={unavailable} />
      {aiLimitReached && <UpgradeBanner resource="ai" />}
      {saveError && (
        <p role="alert" className="text-sm text-destructive">
          {t("resumeBuilder.analysis.saveError")}
        </p>
      )}

      {/* Suggest Summary */}
      <section className="space-y-3">
        <Button
          onClick={handleSuggestSummary}
          disabled={
            unavailable || isChecking || isSyncing || suggestSummary.isPending || !resume || aiLimitReached
          }
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

        {suggestSummary.isError && !isAIActionCancelled(suggestSummary.error) && (
          <p role="alert" className="text-sm text-destructive">{t(aiErrorKey(suggestSummary.error))}</p>
        )}

        {summary && (
          <SuggestionCard
            title={t("resumeBuilder.ai.result")}
            content={summary.summary}
            onApply={handleApplySummary}
            onDismiss={() => {
              setSummaryResult(null);
              suggestSummary.reset();
            }}
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
              unavailable || isChecking || isSyncing ||
              suggestBullets.isPending ||
              !selectedExpId ||
              aiLimitReached
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

          {suggestBullets.isError && !isAIActionCancelled(suggestBullets.error) && (
            <p role="alert" className="text-sm text-destructive">{t(aiErrorKey(suggestBullets.error))}</p>
          )}

          {bullets && (
            <Card className="border-primary/20 bg-primary/5">
              <CardHeader className="p-4 pb-2">
                <CardTitle className="flex items-center gap-2 text-sm font-medium">
                  <Sparkles className="h-3.5 w-3.5 text-primary" />
                  {t("resumeBuilder.ai.result")}
                </CardTitle>
              </CardHeader>
              <CardContent className="p-4 pt-0">
                <ul className="mb-3 space-y-1.5">
                  {bullets.bullets.map((bullet, i) => (
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
                    onClick={() => {
                      setBulletResult(null);
                      suggestBullets.reset();
                    }}
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
            unavailable || isChecking || isSyncing ||
            improveText.isPending ||
            !textToImprove.trim() ||
            aiLimitReached
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

        {improveText.isError && !isAIActionCancelled(improveText.error) && (
          <p role="alert" className="text-sm text-destructive">{t(aiErrorKey(improveText.error))}</p>
        )}

        {improved && (
          <SuggestionCard
            title={t("resumeBuilder.ai.result")}
            content={improved.improved}
            onApply={() => {
              if (!isCurrent(improved.revision)) return;
              setTextToImprove(improved.improved);
              setTextResult(null);
              improveText.reset();
            }}
            onDismiss={() => {
              setTextResult(null);
              improveText.reset();
            }}
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
