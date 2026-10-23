import { MobileTextEditor } from "@/features/resume-builder/components/preview/MobileTextEditor";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { ApiError } from "@/services/api";
import { useTranslation } from "react-i18next";
import { ArrowLeft, Loader2, Palette, X } from "lucide-react";
import { Button } from "@/shared/ui/Button";
import { Sheet } from "@/shared/ui/Sheet";
import { useStore } from "zustand";
import { useResumeBuilderStore } from "@/stores/resumeBuilderStore";
import { useAutoSave } from "@/features/resume-builder/hooks/useAutoSave";
import {
  enqueueOutgoingResumeSave,
  fetchResumeDocument,
  reconcileLoadedResume,
} from "@/features/resume-builder/lib/resumeDocumentSync";
import { PreviewPanel } from "@/features/resume-builder/components/preview/PreviewPanel";
import { PreviewErrorBoundary } from "@/features/resume-builder/components/PreviewErrorBoundary";
import { SaveIndicator } from "@/features/resume-builder/components/SaveIndicator";
import { EditorToolbar } from "@/features/resume-builder/components/EditorToolbar";
import { DesignSidebar } from "@/features/resume-builder/components/sidebar/DesignSidebar";
import { AIAssistantPanel } from "@/features/resume-builder/components/AIAssistantPanel";
import { ATSCheckerPanel } from "@/features/resume-builder/components/ATSCheckerPanel";
import { ContentLibraryPanel } from "@/features/resume-builder/components/ContentLibraryPanel";
import { usePageMeta } from "@/shared/lib/usePageMeta";
import { resumeQueryKeys } from "@/shared/lib/resumeQueryKeys";
import { useMediaQuery } from "@/shared/hooks/useMediaQuery";
import { cn } from "@/shared/lib/utils";
import { Tooltip } from "@/shared/ui/Tooltip";

type SidePanel = "ai" | "ats" | "contentLibrary" | null;

export default function ResumeBuilderEditorPage() {
  const { t } = useTranslation();
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const setResume = useResumeBuilderStore((s) => s.setResume);
  const setSaveStatus = useResumeBuilderStore((s) => s.setSaveStatus);
  const markDirty = useResumeBuilderStore((s) => s.markDirty);
  const resume = useResumeBuilderStore((s) => s.resume);
  const { clear } = useStore(useResumeBuilderStore.temporal);

  const [activePanel, setActivePanel] = useState<SidePanel>(null);
  const [mobileTextMode, setMobileTextMode] = useState(true);
  const [mobileDesignOpen, setMobileDesignOpen] = useState(false);

  const isDesktop = useMediaQuery("(min-width: 1024px)");
  const isTablet = useMediaQuery("(min-width: 768px)");

  const isDirty = useResumeBuilderStore((s) => s.isDirty);

  usePageMeta({ titleKey: "resumeBuilder.editor", noindex: true });
  const { flush } = useAutoSave();

  const handleRetrySave = useCallback(() => {
    void flush().catch(() => {
      // The failure is already on screen through saveStatus.
    });
  }, [flush]);

  // Warn user about unsaved changes before leaving
  useEffect(() => {
    const handler = (e: BeforeUnloadEvent) => {
      if (!isDirty) return;
      e.preventDefault();
    };
    window.addEventListener("beforeunload", handler);
    return () => window.removeEventListener("beforeunload", handler);
  }, [isDirty]);

  // The fetch goes through the sync module, which dates the request: a
  // response the server composed before a save of this client's must not
  // replace it, however late it arrives.
  const { isLoading, error, data } = useQuery({
    queryKey: resumeQueryKeys.builderDetail(id!),
    queryFn: () => fetchResumeDocument(id!),
    enabled: !!id,
  });

  useEffect(() => {
    if (!data) return;

    // Opening another resume from here — a job screen's link, the browser's
    // history — changes the route parameter without unmounting this page, so
    // the auto-save unmount flush never runs. The document on its way out is
    // handed to its queue first, while the store still holds it.
    enqueueOutgoingResumeSave(data.id);

    // The payload never replaces work the server has not got: section writes
    // still on their way, edits typed while it was being fetched, the draft a
    // failed save left behind, and a cache entry older than this editor's own
    // last save are all reconciled — along with the known server ids, which are
    // seeded from the payload only when it is the newer one.
    const { resume: reconciled, unsaved } = reconcileLoadedResume(data);
    setResume(reconciled);
    clear();

    // `setResume` opens the document as stored. Anything the server has not
    // accepted has to stay dirty so auto-save retries it, and a write that
    // failed has to say so instead of looking saved.
    if (unsaved !== "none") markDirty();
    if (unsaved === "failed") setSaveStatus("error");
  }, [data, setResume, setSaveStatus, markDirty, clear]);

  const handleTogglePanel = useCallback((panel: SidePanel) => {
    setActivePanel((current) => (current === panel ? null : panel));
  }, []);

  const handleToggleAI = useCallback(
    () => handleTogglePanel("ai"),
    [handleTogglePanel],
  );

  const handleToggleATS = useCallback(
    () => handleTogglePanel("ats"),
    [handleTogglePanel],
  );

  const handleToggleContentLibrary = useCallback(
    () => handleTogglePanel("contentLibrary"),
    [handleTogglePanel],
  );

  const handleClosePanel = useCallback(() => {
    setActivePanel(null);
  }, []);

  const panelTitle = useMemo(() => {
    if (activePanel === "ai") return t("resumeBuilder.toolbar.aiAssistant");
    if (activePanel === "ats") return t("resumeBuilder.toolbar.atsCheck");
    if (activePanel === "contentLibrary")
      return t("resumeBuilder.toolbar.contentLibrary");
    return "";
  }, [activePanel, t]);

  if (isLoading) {
    return (
      <div className="flex h-[calc(100vh-8rem)] items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  // ApiError carries the status flat. Reaching for `error.response.status`
  // always found undefined, so a 403 and a 500 both told the customer the
  // resume did not exist.
  const status = error instanceof ApiError ? error.status : undefined;
  // Opening a resume again refetches it, and a refetch that fails leaves
  // `error` set next to a payload this editor already has. Reporting that the
  // resume does not exist would be wrong then — and would take away a document
  // whose save may still be on its way. Only the server saying the resume is
  // gone, or not this customer's, is final.
  const resumeIsGone = status === 404 || status === 403;
  if (error && (resumeIsGone || !data)) {
    let message = t("resumeBuilder.notFound");
    if (status === 403) {
      message = t("common.accessDenied", "Access denied");
    } else if (status && status >= 500) {
      message = t("common.serverError", "Server error");
    }
    return (
      <div className="flex h-[calc(100vh-8rem)] flex-col items-center justify-center gap-4">
        <p className="text-muted-foreground">{message}</p>
        <Button variant="outline" onClick={() => navigate("/app/resumes")}>
          <ArrowLeft className="mr-2 h-4 w-4" />
          {t("resumeBuilder.backToList")}
        </Button>
      </div>
    );
  }

  if (!resume) {
    return (
      <div className="flex h-[calc(100vh-8rem)] items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  const activePanelContent = (
    <>
      {activePanel === "ai" && <AIAssistantPanel key={resume.id} />}
      {activePanel === "ats" && <ATSCheckerPanel key={resume.id} />}
      {activePanel === "contentLibrary" && (
        <ContentLibraryPanel key={resume.id} />
      )}
    </>
  );

  return (
    <div className="flex h-[calc(100vh-8rem)] min-w-0 flex-col">
      {/* Toolbar */}
      <div className="flex min-w-0 flex-wrap items-center gap-2 border-b px-2 py-2 sm:px-4">
        <Tooltip content={t("resumeBuilder.backToList")} side="bottom">
          <Button
            variant="ghost"
            size="icon"
            className="shrink-0"
            onClick={() => navigate("/app/resumes")}
            aria-label={t("resumeBuilder.backToList")}
          >
            <ArrowLeft className="h-4 w-4" />
          </Button>
        </Tooltip>
        <h1
          title={resume.title}
          className="hidden min-w-0 max-w-full flex-1 truncate text-lg font-semibold sm:block"
        >
          {resume.title}
        </h1>

        <div className="order-3 flex w-full min-w-0 flex-wrap items-center gap-1 sm:gap-2 xl:order-none xl:w-auto xl:flex-1">
          {/* Design button — mobile only (< md) */}
          {!isTablet && (
            <Tooltip content={t("resumeBuilder.sections.design")} side="bottom">
              <Button
                variant={mobileDesignOpen ? "default" : "outline"}
                size="sm"
                className="shrink-0"
                onClick={() => setMobileDesignOpen(true)}
                aria-label={t("resumeBuilder.sections.design")}
                aria-pressed={mobileDesignOpen}
              >
                <Palette className="h-4 w-4" />
              </Button>
            </Tooltip>
          )}

          <EditorToolbar
            showAI={activePanel === "ai"}
            onToggleAI={handleToggleAI}
            showATS={activePanel === "ats"}
            onToggleATS={handleToggleATS}
            showContentLibrary={activePanel === "contentLibrary"}
            onToggleContentLibrary={handleToggleContentLibrary}
          />
        </div>

        <div className="shrink-0">
          <SaveIndicator onRetry={handleRetrySave} />
        </div>
      </div>

      {!isTablet && <div className="flex shrink-0 gap-2 border-b px-3 py-2" role="group" aria-label={t("resumeBuilder.editor")}>
        <Button className="min-h-11 flex-1" size="sm" variant={mobileTextMode ? "default" : "outline"} aria-pressed={mobileTextMode} onClick={() => setMobileTextMode(true)}>{t("resumeBuilder.mobile.text")}</Button>
        <Button className="min-h-11 flex-1" size="sm" variant={!mobileTextMode ? "default" : "outline"} aria-pressed={!mobileTextMode} onClick={() => setMobileTextMode(false)}>{t("resumeBuilder.mobile.preview")}</Button>
      </div>}
      {/* Main content: sidebar + document + optional right panel */}
      <div className="flex flex-1 overflow-hidden">
        {/* Design sidebar — desktop (>= md) */}
        {isTablet && <DesignSidebar />}

        {/* Centered A4 document (the editor IS the document) */}
        <div
          className={cn(
            "min-w-0 flex-1 overflow-y-auto bg-muted/30",
            activePanel && isDesktop && "flex-initial w-[calc(100%-320px)]",
          )}
        >
          <PreviewErrorBoundary>
            {!isTablet && mobileTextMode ? <MobileTextEditor key={resume.id} /> : <PreviewPanel editable={isTablet} fitToWidth />}
          </PreviewErrorBoundary>
        </div>

        {/* Side Panel — desktop (>= lg) */}
        {activePanel !== null && isDesktop && (
          <div className="w-[320px] min-w-[320px] overflow-y-auto border-l bg-background">
            <div className="flex items-center justify-between border-b px-4 py-2">
              <h2 className="text-sm font-semibold">{panelTitle}</h2>
              <Button
                variant="ghost"
                size="icon"
                onClick={handleClosePanel}
                aria-label={t("common.close")}
              >
                <X className="h-4 w-4" />
              </Button>
            </div>
            <div className="p-4">{activePanelContent}</div>
          </div>
        )}
      </div>

      {/* Mobile sheet for right panels (< lg) */}
      {!isDesktop && (
        <Sheet
          open={activePanel !== null}
          onOpenChange={(open) => {
            if (!open) handleClosePanel();
          }}
          title={panelTitle}
        >
          {activePanelContent}
        </Sheet>
      )}

      {/* Mobile sheet for design tools (< md) */}
      {!isTablet && (
        <Sheet
          open={mobileDesignOpen}
          onOpenChange={setMobileDesignOpen}
          title={t("resumeBuilder.sections.design")}
        >
          <DesignSidebar layout="grid" />
        </Sheet>
      )}
    </div>
  );
}
