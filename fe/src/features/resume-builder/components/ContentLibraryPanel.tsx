import { useState, useMemo, useCallback } from "react";
import { useTranslation } from "react-i18next";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { BookOpen, Plus, Trash2, Pencil, Copy, Loader2, Search } from "lucide-react";
import { Button } from "@/shared/ui/Button";
import { Card, CardContent } from "@/shared/ui/Card";
import { Input } from "@/shared/ui/Input";
import { Label } from "@/shared/ui/Label";
import { Textarea } from "@/shared/ui/Textarea";
import { Select } from "@/shared/ui/Select";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/shared/ui/Dialog";
import { cn } from "@/shared/lib/utils";
import {
  showSuccessNotification,
  showErrorNotification,
} from "@/shared/lib/notifications";
import { contentLibraryService } from "@/services/contentLibraryService";
import type {
  ContentLibraryEntryDTO,
  ContentLibraryCategory,
} from "@/shared/types/content-library";
import { CONTENT_LIBRARY_CATEGORIES } from "@/shared/types/content-library";

const QUERY_KEY = ["content-library"] as const;

interface SnippetFormState {
  title: string;
  content: string;
  category: ContentLibraryCategory;
}

const EMPTY_FORM: SnippetFormState = {
  title: "",
  content: "",
  category: "bullet",
};

function buildFormFromEntry(entry: ContentLibraryEntryDTO): SnippetFormState {
  return {
    title: entry.title,
    content: entry.content,
    category: entry.category as ContentLibraryCategory,
  };
}

export function ContentLibraryPanel() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();

  const [searchQuery, setSearchQuery] = useState("");
  const [categoryFilter, setCategoryFilter] = useState<string>("all");
  const [dialogOpen, setDialogOpen] = useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [editingEntry, setEditingEntry] =
    useState<ContentLibraryEntryDTO | null>(null);
  const [deletingEntry, setDeletingEntry] =
    useState<ContentLibraryEntryDTO | null>(null);
  const [form, setForm] = useState<SnippetFormState>(EMPTY_FORM);

  // --- Queries & Mutations ---

  const { data: entries = [], isLoading } = useQuery({
    queryKey: QUERY_KEY,
    queryFn: contentLibraryService.list,
  });

  const createMutation = useMutation({
    mutationFn: contentLibraryService.create,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: QUERY_KEY });
      showSuccessNotification(t("contentLibrary.created"));
      closeDialog();
    },
    onError: () => {
      showErrorNotification(t("common.error"));
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({
      id,
      data,
    }: {
      id: string;
      data: Partial<SnippetFormState>;
    }) => contentLibraryService.update(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: QUERY_KEY });
      showSuccessNotification(t("contentLibrary.updated"));
      closeDialog();
    },
    onError: () => {
      showErrorNotification(t("common.error"));
    },
  });

  const deleteMutation = useMutation({
    mutationFn: contentLibraryService.remove,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: QUERY_KEY });
      showSuccessNotification(t("contentLibrary.deleted"));
      closeDeleteDialog();
    },
    onError: () => {
      showErrorNotification(t("common.error"));
    },
  });

  // --- Filtering ---

  const filteredEntries = useMemo(() => {
    const lowerSearch = searchQuery.toLowerCase();
    return entries.filter((entry) => {
      const matchesCategory =
        categoryFilter === "all" || entry.category === categoryFilter;
      const matchesSearch =
        lowerSearch === "" ||
        entry.title.toLowerCase().includes(lowerSearch);
      return matchesCategory && matchesSearch;
    });
  }, [entries, searchQuery, categoryFilter]);

  // --- Handlers ---

  const updateFormField = useCallback(
    <K extends keyof SnippetFormState>(
      field: K,
      value: SnippetFormState[K],
    ) => {
      setForm((prev) => ({ ...prev, [field]: value }));
    },
    [],
  );

  const openCreateDialog = useCallback(() => {
    setEditingEntry(null);
    setForm(EMPTY_FORM);
    setDialogOpen(true);
  }, []);

  const openEditDialog = useCallback((entry: ContentLibraryEntryDTO) => {
    setEditingEntry(entry);
    setForm(buildFormFromEntry(entry));
    setDialogOpen(true);
  }, []);

  const closeDialog = useCallback(() => {
    setDialogOpen(false);
    setEditingEntry(null);
    setForm(EMPTY_FORM);
  }, []);

  const openDeleteDialog = useCallback((entry: ContentLibraryEntryDTO) => {
    setDeletingEntry(entry);
    setDeleteDialogOpen(true);
  }, []);

  const closeDeleteDialog = useCallback(() => {
    setDeleteDialogOpen(false);
    setDeletingEntry(null);
  }, []);

  const handleSubmit = useCallback(() => {
    if (!form.title.trim() || !form.content.trim()) return;

    if (editingEntry) {
      updateMutation.mutate({
        id: editingEntry.id,
        data: {
          title: form.title,
          content: form.content,
          category: form.category,
        },
      });
    } else {
      createMutation.mutate({
        title: form.title,
        content: form.content,
        category: form.category,
      });
    }
  }, [form, editingEntry, createMutation, updateMutation]);

  const handleDelete = useCallback(() => {
    if (!deletingEntry) return;
    deleteMutation.mutate(deletingEntry.id);
  }, [deletingEntry, deleteMutation]);

  const handleInsert = useCallback(
    async (entry: ContentLibraryEntryDTO) => {
      try {
        await navigator.clipboard.writeText(entry.content);
        showSuccessNotification(t("contentLibrary.insertSuccess"));
      } catch {
        showErrorNotification(t("common.error"));
      }
    },
    [t],
  );

  const isSaving = createMutation.isPending || updateMutation.isPending;

  // --- Render ---

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <BookOpen className="h-5 w-5 text-primary" />
          <h2 className="text-lg font-semibold">
            {t("contentLibrary.title")}
          </h2>
        </div>
        <Button size="sm" onClick={openCreateDialog}>
          <Plus className="h-4 w-4" />
          {t("contentLibrary.add")}
        </Button>
      </div>

      {/* Search & Filter */}
      <div className="space-y-3">
        <div className="relative">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder={t("contentLibrary.search")}
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-9"
          />
        </div>
        <Select
          value={categoryFilter}
          onChange={(e) => setCategoryFilter(e.target.value)}
        >
          <option value="all">{t("contentLibrary.category")}</option>
          {CONTENT_LIBRARY_CATEGORIES.map((cat) => (
            <option key={cat} value={cat}>
              {t(`contentLibrary.categories.${cat}`)}
            </option>
          ))}
        </Select>
      </div>

      {/* List */}
      {isLoading ? (
        <div className="flex items-center justify-center py-8">
          <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
        </div>
      ) : filteredEntries.length === 0 ? (
        <p className="py-8 text-center text-sm text-muted-foreground">
          {t("contentLibrary.empty")}
        </p>
      ) : (
        <div className="space-y-3">
          {filteredEntries.map((entry) => (
            <SnippetCard
              key={entry.id}
              entry={entry}
              onEdit={openEditDialog}
              onDelete={openDeleteDialog}
              onInsert={handleInsert}
            />
          ))}
        </div>
      )}

      {/* Create / Edit Dialog */}
      <Dialog open={dialogOpen} onOpenChange={closeDialog}>
        <DialogContent onClose={closeDialog}>
          <DialogHeader>
            <DialogTitle>
              {editingEntry
                ? t("contentLibrary.edit")
                : t("contentLibrary.add")}
            </DialogTitle>
          </DialogHeader>

          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label>{t("contentLibrary.snippetTitle")}</Label>
              <Input
                value={form.title}
                onChange={(e) => updateFormField("title", e.target.value)}
              />
            </div>

            <div className="space-y-2">
              <Label>{t("contentLibrary.content")}</Label>
              <Textarea
                value={form.content}
                onChange={(e) => updateFormField("content", e.target.value)}
                rows={5}
              />
            </div>

            <div className="space-y-2">
              <Label>{t("contentLibrary.category")}</Label>
              <Select
                value={form.category}
                onChange={(e) =>
                  updateFormField(
                    "category",
                    e.target.value as ContentLibraryCategory,
                  )
                }
              >
                {CONTENT_LIBRARY_CATEGORIES.map((cat) => (
                  <option key={cat} value={cat}>
                    {t(`contentLibrary.categories.${cat}`)}
                  </option>
                ))}
              </Select>
            </div>
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={closeDialog}>
              {t("common.cancel")}
            </Button>
            <Button
              onClick={handleSubmit}
              disabled={isSaving || !form.title.trim() || !form.content.trim()}
            >
              {isSaving && <Loader2 className="h-4 w-4 animate-spin" />}
              {editingEntry ? t("common.save") : t("contentLibrary.add")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation Dialog */}
      <Dialog open={deleteDialogOpen} onOpenChange={closeDeleteDialog}>
        <DialogContent onClose={closeDeleteDialog}>
          <DialogHeader>
            <DialogTitle>{t("contentLibrary.delete")}</DialogTitle>
          </DialogHeader>

          <p className="py-4 text-sm text-muted-foreground">
            {t("contentLibrary.confirmDelete")}
          </p>

          <DialogFooter>
            <Button variant="outline" onClick={closeDeleteDialog}>
              {t("common.cancel")}
            </Button>
            <Button
              variant="destructive"
              onClick={handleDelete}
              disabled={deleteMutation.isPending}
            >
              {deleteMutation.isPending && (
                <Loader2 className="h-4 w-4 animate-spin" />
              )}
              {t("contentLibrary.delete")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

// --- Snippet Card Sub-Component ---

interface SnippetCardProps {
  entry: ContentLibraryEntryDTO;
  onEdit: (entry: ContentLibraryEntryDTO) => void;
  onDelete: (entry: ContentLibraryEntryDTO) => void;
  onInsert: (entry: ContentLibraryEntryDTO) => void;
}

function SnippetCard({ entry, onEdit, onDelete, onInsert }: SnippetCardProps) {
  const { t } = useTranslation();

  return (
    <Card>
      <CardContent className="p-4">
        <div className="flex items-start justify-between gap-2">
          <div className="min-w-0 flex-1">
            <h3 className="truncate text-sm font-medium">{entry.title}</h3>
            <span
              className={cn(
                "mt-1 inline-block rounded-full px-2 py-0.5 text-xs font-medium",
                "bg-primary/10 text-primary",
              )}
            >
              {t(`contentLibrary.categories.${entry.category}`)}
            </span>
            <p className="mt-2 line-clamp-3 text-sm text-muted-foreground">
              {entry.content}
            </p>
          </div>

          <div className="flex shrink-0 gap-1">
            <Button
              size="sm"
              variant="ghost"
              onClick={() => onInsert(entry)}
              title={t("contentLibrary.insert")}
            >
              <Copy className="h-4 w-4" />
            </Button>
            <Button
              size="sm"
              variant="ghost"
              onClick={() => onEdit(entry)}
              title={t("contentLibrary.edit")}
            >
              <Pencil className="h-4 w-4" />
            </Button>
            <Button
              size="sm"
              variant="ghost"
              onClick={() => onDelete(entry)}
              title={t("contentLibrary.delete")}
            >
              <Trash2 className="h-4 w-4 text-destructive" />
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
