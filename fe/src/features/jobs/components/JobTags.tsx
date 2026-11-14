import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useSingleFlightMutation } from "@/shared/hooks/useSingleFlightMutation";
import { Tag as TagIcon, Plus, X } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/Card";
import { Button } from "@/shared/ui/Button";
import { Input } from "@/shared/ui/Input";
import { tagsService } from "@/services/tagsService";
import {
  showSuccessNotification,
  showErrorNotification,
} from "@/shared/lib/notifications";
import type { TagDTO } from "@/shared/types/api";

interface JobTagsProps {
  readonly jobId: string;
}

// A colored chip uses the tag color as a translucent background and border.
// The text is the tag color mixed half-way towards the theme's foreground:
// the raw color (a light lime, a mid blue) fell to 1.8–3.3:1 on the card,
// while the mix stays in the same hue and reads at well over 4.5:1 in both themes.
function chipStyle(color?: string): React.CSSProperties | undefined {
  if (!color) return undefined;
  return {
    backgroundColor: `${color}1a`,
    color: `color-mix(in srgb, ${color} 50%, hsl(var(--foreground)))`,
    borderColor: `${color}40`,
  };
}

export function JobTags({ jobId }: JobTagsProps) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();

  const [name, setName] = useState("");
  const [color, setColor] = useState("#2563eb");

  const jobTagsKey = ["job-tags", jobId];
  const allTagsKey = ["tags"];

  const { data: jobTagsData } = useQuery({
    queryKey: jobTagsKey,
    queryFn: () => tagsService.listByJob(jobId),
    enabled: !!jobId,
  });
  const jobTags = jobTagsData ?? [];

  const { data: allTagsData } = useQuery({
    queryKey: allTagsKey,
    queryFn: () => tagsService.list(),
  });
  const allTags = allTagsData ?? [];

  const invalidateJobTags = () =>
    queryClient.invalidateQueries({ queryKey: jobTagsKey });

  const attachMutation = useMutation({
    mutationFn: (tagId: string) =>
      tagsService.attach(tagId, { entity_type: "job", entity_id: jobId }),
    onSuccess: invalidateJobTags,
    onError: (err: Error) =>
      showErrorNotification(err.message || t("jobs.tagAttachError")),
  });

  const detachMutation = useMutation({
    mutationFn: (tagId: string) => tagsService.detach(tagId, "job", jobId),
    onSuccess: invalidateJobTags,
    onError: (err: Error) =>
      showErrorNotification(err.message || t("jobs.tagDetachError")),
  });

  // Single-flight: a double-click on Create sent two POSTs, and the second one
  // came back 409 and showed an error next to the success toast.
  const createMutation = useSingleFlightMutation({
    mutationFn: async () => {
      const tag = await tagsService.create({ name: name.trim(), color });
      await tagsService.attach(tag.id, {
        entity_type: "job",
        entity_id: jobId,
      });
      return tag;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: allTagsKey });
      invalidateJobTags();
      setName("");
      showSuccessNotification(t("jobs.tagCreatedSuccess"));
    },
    onError: (err: Error) =>
      showErrorNotification(err.message || t("jobs.tagCreateError")),
  });

  const attachedIds = new Set(jobTags.map((tag) => tag.id));
  const unattached = allTags.filter((tag) => !attachedIds.has(tag.id));

  const canCreate = name.trim().length > 0 && !createMutation.isPending;

  const handleCreate = (e: React.FormEvent) => {
    e.preventDefault();
    if (!canCreate) return;
    createMutation.mutate();
  };

  const renderChip = (tag: TagDTO, onRemove: () => void) => (
    <span
      key={tag.id}
      style={chipStyle(tag.color)}
      className="inline-flex min-w-0 max-w-full items-center gap-1 rounded-full border border-border bg-muted px-2.5 py-0.5 text-xs font-medium max-sm:min-h-11 max-sm:py-0 max-sm:pl-3 max-sm:pr-0"
    >
      <span className="min-w-0 wrap-anywhere">{tag.name}</span>
      <button
        type="button"
        onClick={onRemove}
        disabled={detachMutation.isPending}
        aria-label={t("common.delete")}
        className="flex shrink-0 items-center justify-center opacity-60 transition-opacity hover:opacity-100 disabled:opacity-40 max-sm:h-11 max-sm:w-11"
      >
        <X className="h-3 w-3" />
      </button>
    </span>
  );

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-lg">
          <TagIcon className="h-4 w-4" />
          {t("jobs.tags")}
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="flex flex-wrap items-center gap-2">
          {jobTags.length > 0 ? (
            jobTags.map((tag) =>
              renderChip(tag, () => detachMutation.mutate(tag.id)),
            )
          ) : (
            <p className="text-sm text-muted-foreground">{t("jobs.noTags")}</p>
          )}
        </div>

        {unattached.length > 0 && (
          <div className="space-y-1.5">
            <p className="text-xs text-muted-foreground">
              {t("jobs.addExistingTag")}
            </p>
            <div className="flex flex-wrap gap-2">
              {unattached.map((tag) => (
                <button
                  key={tag.id}
                  type="button"
                  onClick={() => attachMutation.mutate(tag.id)}
                  disabled={attachMutation.isPending}
                  style={chipStyle(tag.color)}
                  className="inline-flex min-w-0 max-w-full items-center gap-1 rounded-full border border-dashed border-border px-2.5 py-0.5 text-left text-xs font-medium transition-opacity hover:opacity-80 disabled:opacity-40 max-sm:min-h-11 max-sm:px-3"
                >
                  <Plus className="h-3 w-3 shrink-0" />
                  <span className="min-w-0 wrap-anywhere">{tag.name}</span>
                </button>
              ))}
            </div>
          </div>
        )}

        <form onSubmit={handleCreate} className="flex items-center gap-2">
          <Input
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder={t("jobs.newTagPlaceholder")}
            maxLength={100}
            className="flex-1"
          />
          <input
            type="color"
            value={color}
            onChange={(e) => setColor(e.target.value)}
            aria-label={t("jobs.tagColor")}
            className="h-9 w-10 shrink-0 cursor-pointer rounded-md border border-input bg-background p-1 max-sm:h-11 max-sm:w-11"
          />
          <Button type="submit" size="sm" disabled={!canCreate}>
            <Plus className="mr-1 h-4 w-4" />
            {t("common.create")}
          </Button>
        </form>
      </CardContent>
    </Card>
  );
}
