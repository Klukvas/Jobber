import { useCallback, useEffect, useMemo, useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { ArrowLeft, Loader2, Palette, X } from "lucide-react";
import { Button } from "@/shared/ui/Button";
import { Sheet } from "@/shared/ui/Sheet";
import { resumeBuilderService } from "@/services/resumeBuilderService";
import { useStore } from "zustand";
import { useResumeBuilderStore } from "@/stores/resumeBuilderStore";
import { useAutoSave } from "@/features/resume-builder/hooks/useAutoSave";
import { initServerIds } from "@/features/resume-builder/hooks/useSectionPersistence";
import { PreviewPanel } from "@/features/resume-builder/components/preview/PreviewPanel";
import { PreviewErrorBoundary } from "@/features/resume-builder/components/PreviewErrorBoundary";
import { SaveIndicator } from "@/features/resume-builder/components/SaveIndicator";
import { EditorToolbar } from "@/features/resume-builder/components/EditorToolbar";
import { DesignSidebar } from "@/features/resume-builder/components/sidebar/DesignSidebar";
import { AIAssistantPanel } from "@/features/resume-builder/components/AIAssistantPanel";
import { ATSCheckerPanel } from "@/features/resume-builder/components/ATSCheckerPanel";
import { ContentLibraryPanel } from "@/features/resume-builder/components/ContentLibraryPanel";
import { usePageMeta } from "@/shared/lib/usePageMeta";
import { useMediaQuery } from "@/shared/hooks/useMediaQuery";
import { cn } from "@/shared/lib/utils";
import { Tooltip } from "@/shared/ui/Tooltip";

type SidePanel = "ai" | "ats" | "contentLibrary" | null;

export default function ResumeBuilderEditorPage() {
  const { t } = useTranslation();
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const setResume = useResumeBuilderStore((s) => s.setResume);
  const resume = useResumeBuilderStore((s) => s.resume);
  const { clear } = useStore(useResumeBuilderStore.temporal);

  const [activePanel, setActivePanel] = useState<SidePanel>(null);
  const [mobileDesignOpen, setMobileDesignOpen] = useState(false);

  const isDesktop = useMediaQuery("(min-width: 1024px)");
  const isTablet = useMediaQuery("(min-width: 768px)");

  const isDirty = useResumeBuilderStore((s) => s.isDirty);

  usePageMeta({ titleKey: "resumeBuilder.editor" });
  useAutoSave();

  // Warn user about unsaved changes before leaving
  useEffect(() => {
    const handler = (e: BeforeUnloadEvent) => {
      if (!isDirty) return;
      e.preventDefault();
    };
    window.addEventListener("beforeunload", handler);
    return () => window.removeEventListener("beforeunload", handler);
  }, [isDirty]);

  const { isLoading, error, data } = useQuery({
    queryKey: ["resume-builder", id],
    queryFn: () => resumeBuilderService.getById(id!),
    enabled: !!id,
  });

  useEffect(() => {
    if (data) {
      setResume(data);
      initServerIds(data);
      clear();
    }
  }, [data, setResume, clear]);

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

  if (error) {
    const status = (error as { response?: { status?: number } })?.response
      ?.status;
    let message = t("resumeBuilder.notFound");
    if (status === 403) {
      message = t("common.accessDenied", "Access denied");
    } else if (status && status >= 500) {
      message = t("common.serverError", "Server error");
    }
    return (
      <div className="flex h-[calc(100vh-8rem)] flex-col items-center justify-center gap-4">
        <p className="text-muted-foreground">{message}</p>
        <Button
          variant="outline"
          onClick={() => navigate("/app/resume-builder")}
        >
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
      {activePanel === "ai" && <AIAssistantPanel />}
      {activePanel === "ats" && <ATSCheckerPanel />}
      {activePanel === "contentLibrary" && <ContentLibraryPanel />}
    </>
  );

  return (
    <div className="flex h-[calc(100vh-8rem)] flex-col">
      {/* Toolbar */}
      <div className="flex min-w-0 items-center gap-2 border-b px-2 py-2 sm:px-4">
        <Tooltip content={t("resumeBuilder.backToList")} side="bottom">
          <Button
            variant="ghost"
            size="icon"
            className="shrink-0"
            onClick={() => navigate("/app/resume-builder")}
            aria-label={t("resumeBuilder.backToList")}
          >
            <ArrowLeft className="h-4 w-4" />
          </Button>
        </Tooltip>
        <h1 className="hidden shrink-0 text-lg font-semibold sm:block">
          {resume.title}
        </h1>

        <div className="flex min-w-0 flex-1 flex-wrap items-center gap-1 sm:gap-2">
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
          <SaveIndicator />
        </div>
      </div>

      {/* Main content: sidebar + document + optional right panel */}
      <div className="flex flex-1 overflow-hidden">
        {/* Design sidebar — desktop (>= md) */}
        {isTablet && <DesignSidebar />}

        {/* Centered A4 document (the editor IS the document) */}
        <div
          className={cn(
            "flex-1 overflow-y-auto bg-muted/30",
            activePanel && isDesktop && "flex-initial w-[calc(100%-320px)]",
          )}
        >
          <PreviewErrorBoundary>
            <PreviewPanel editable />
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
