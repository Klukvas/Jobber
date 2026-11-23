import type { KeyboardEvent, ReactNode } from "react";
import { useEffect, useId, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Check } from "lucide-react";

// Paths under fe/public for a short recording of the resume-review screen.
// Leave this empty to keep the static mock. A .mp4 or .webm plays inline;
// a .gif or still image is shown as a picture. Example:
// "/landing/hero-review.mp4"
const HERO_REVIEW_MEDIA_SOURCES: readonly string[] = [];

const SAMPLE_CHANGES = [
  {
    id: "summary",
    kindKey: "jobs.tailoring.summary",
    checked: true,
    beforeKey: "home.showcase.summaryBefore",
    afterKey: "home.showcase.summaryAfter",
    reasonKey: "home.showcase.summaryReason",
  },
  {
    id: "experience",
    kindKey: "jobs.tailoring.experience",
    checked: false,
    beforeKey: "home.showcase.experienceBefore",
    afterKey: "home.showcase.experienceAfter",
    reasonKey: "home.showcase.experienceReason",
  },
] as const;

const HERO_FRAMES = ["edits", "check", "builder", "stages"] as const;
type HeroFrame = (typeof HERO_FRAMES)[number];

/** How long each product frame stays before the next one. */
export const HERO_FRAME_MS = 4000;

const prefersReducedMotion = () =>
  window.matchMedia("(prefers-reduced-motion: reduce)").matches;

function nextFrame(frame: HeroFrame): HeroFrame {
  return HERO_FRAMES[(HERO_FRAMES.indexOf(frame) + 1) % HERO_FRAMES.length];
}

function mediaKind(src: string): "video" | "image" {
  return /\.(mp4|webm)(?:\?.*)?$/i.test(src) ? "video" : "image";
}

function isAnimatedMedia(src: string) {
  return /\.(mp4|webm|gif)(?:\?.*)?$/i.test(src);
}

function CheckMark({ checked }: { checked: boolean }) {
  return (
    <span
      className={`flex h-3.5 w-3.5 shrink-0 items-center justify-center rounded-[3px] border ${
        checked
          ? "hero-review-check border-lime-400 bg-lime-400 text-slate-950"
          : "border-white/25 bg-transparent"
      }`}
    >
      {checked ? <Check className="h-2.5 w-2.5" strokeWidth={3} /> : null}
    </span>
  );
}

function ReviewChangeCard({
  kindKey,
  checked,
  beforeKey,
  afterKey,
  reasonKey,
  index,
}: (typeof SAMPLE_CHANGES)[number] & { readonly index: number }) {
  const { t } = useTranslation();

  return (
    <article
      className={`hero-review-card hero-review-card-${index + 1} min-w-0 rounded-lg border ${
        checked
          ? "border-lime-400/40 bg-lime-400/[0.05]"
          : "border-white/[0.07] bg-muted/40"
      }`}
    >
      <div className="flex items-center gap-2.5 border-b border-white/[0.06] px-3 py-2.5">
        <CheckMark checked={checked} />
        <span className="min-w-0 text-[12px] font-semibold text-slate-100">
          {t(kindKey)}
        </span>
      </div>
      <div className="grid min-w-0 grid-cols-1 divide-y divide-white/[0.06] sm:grid-cols-2 sm:divide-x sm:divide-y-0">
        <div className="min-w-0 p-3">
          <p className="mb-1 text-[10px] font-medium uppercase tracking-wide text-slate-500">
            {t("jobs.tailoring.before")}
          </p>
          <p className="break-words text-[11px] leading-snug text-slate-400">
            {t(beforeKey)}
          </p>
        </div>
        <div
          className={`hero-review-proposed min-w-0 p-3 ${checked ? "hero-review-after" : ""}`}
        >
          <p className="mb-1.5 flex flex-wrap items-center gap-1.5">
            <span className="rounded-full bg-lime-400 px-1.5 py-0.5 text-[9px] font-semibold text-slate-950">
              {t("home.showcase.proposedEyebrow")}
            </span>
            <span className="text-[10px] font-medium uppercase tracking-wide text-lime-300">
              {t("home.showcase.proposedTitle")}
            </span>
          </p>
          <p className="break-words text-[11px] font-medium leading-snug text-slate-50">
            {t(afterKey)}
          </p>
        </div>
      </div>
      <p className="border-t border-white/[0.06] px-3 py-2.5 text-[11px] leading-snug text-slate-400">
        <span className="font-medium text-slate-300">
          {t("jobs.tailoring.reason")}{" "}
        </span>
        {t(reasonKey)}
      </p>
    </article>
  );
}

function StaticReview() {
  const { t } = useTranslation();

  return (
    <div className="rounded-2xl border border-white/[0.07] bg-card/90 p-4 shadow-[0_30px_80px_-30px_rgba(0,0,0,0.8)] backdrop-blur">
      <FrameChrome label={t("jobs.tailoring.review")} />
      <p className="text-[13px] font-semibold leading-snug text-slate-100">
        Staff Engineer — Vercel
      </p>
      <p className="mb-3 mt-1 text-[11px] leading-snug text-slate-500">
        {t("jobs.tailoring.reviewHint")}
      </p>
      <div className="mb-3 flex items-center gap-2 text-[11px] text-slate-400">
        <CheckMark checked={false} />
        {t("jobs.tailoring.selectAll")}
      </div>
      <div className="flex flex-col gap-2.5">
        {SAMPLE_CHANGES.map((change, index) => (
          <ReviewChangeCard key={change.id} index={index} {...change} />
        ))}
      </div>
      <div className="mt-3 flex flex-wrap items-center justify-between gap-2 border-t border-white/[0.06] pt-3">
        <p className="text-[11px] text-slate-500">
          {t("jobs.tailoring.selected", {
            count: 1,
            total: SAMPLE_CHANGES.length,
          })}
        </p>
        <span className="rounded-md bg-lime-400 px-2.5 py-1 text-[11px] font-semibold text-slate-950">
          {t("jobs.tailoring.saveCopy")}
        </span>
      </div>
    </div>
  );
}

function ReviewRecording({ src }: { src: string }) {
  const [failed, setFailed] = useState(false);
  if (failed) return <StaticReview />;

  const frame =
    "block h-auto max-h-[640px] w-full rounded-2xl border border-white/[0.07] bg-card object-contain shadow-[0_30px_80px_-30px_rgba(0,0,0,0.8)]";

  if (mediaKind(src) === "video") {
    return (
      <video
        className={frame}
        src={src}
        autoPlay
        muted
        loop
        playsInline
        onError={() => setFailed(true)}
      />
    );
  }

  return (
    <img className={frame} src={src} alt="" onError={() => setFailed(true)} />
  );
}

function ReviewFrame({ children }: { children: ReactNode }) {
  return <div data-hero-review="static">{children}</div>;
}

const CHECK_ITEMS = [
  { key: "home.showcase.check.item1", checked: true },
  { key: "home.showcase.check.item2", checked: true },
  { key: "home.showcase.check.item3", checked: false },
] as const;

function CheckFrame() {
  const { t } = useTranslation();

  return (
    <div className="rounded-2xl border border-white/[0.07] bg-card/90 p-4 shadow-[0_30px_80px_-30px_rgba(0,0,0,0.8)] backdrop-blur">
      <FrameChrome label={t("home.showcase.check.label")} />
      <p className="text-[13px] font-semibold leading-snug text-slate-100">
        {t("home.showcase.check.title")}
      </p>
      <ul className="mt-3 flex flex-col gap-2.5">
        {CHECK_ITEMS.map((item) => (
          <li
            key={item.key}
            className="flex items-start gap-2 text-[12px] leading-snug text-slate-300"
          >
            <CheckMark checked={item.checked} />
            {t(item.key)}
          </li>
        ))}
      </ul>
      <p className="mt-3 text-[11px] leading-snug text-slate-500">
        {t("home.showcase.check.note")}
      </p>
    </div>
  );
}

function BuilderFrame() {
  const { t } = useTranslation();

  return (
    <div className="rounded-2xl border border-white/[0.07] bg-card/90 p-4 shadow-[0_30px_80px_-30px_rgba(0,0,0,0.8)] backdrop-blur">
      <FrameChrome label={t("home.showcase.builder.label")} />
      <div className="rounded-lg bg-[#f6f4ee] px-4 py-3.5 text-slate-900">
        <p className="text-[15px] font-bold leading-tight">
          {t("home.showcase.builder.name")}
        </p>
        <p className="mt-0.5 text-[11px] text-slate-600">
          {t("home.showcase.builder.role")}
        </p>
        <p className="mb-1 mt-3 text-[10px] font-semibold uppercase tracking-wide text-slate-500">
          {t("home.showcase.builder.experience")}
        </p>
        <p className="text-[12px] leading-snug">
          {t("home.showcase.builder.experienceLine")}
        </p>
        <p className="mb-1.5 mt-3 text-[10px] font-semibold uppercase tracking-wide text-slate-500">
          {t("home.showcase.builder.skills")}
        </p>
        <div className="flex flex-wrap gap-1.5">
          <span className="rounded-full bg-[#e4efe0] px-2 py-0.5 text-[10px] font-medium text-slate-800">
            {t("home.showcase.builder.skillReact")}
          </span>
          <span className="rounded-full bg-[#e4efe0] px-2 py-0.5 text-[10px] font-medium text-slate-800">
            {t("home.showcase.builder.skillTypeScript")}
          </span>
        </div>
      </div>
      <div className="mt-3 flex flex-wrap items-center justify-between gap-2 border-t border-white/[0.06] pt-3">
        <p className="text-[11px] text-slate-500">
          {t("home.showcase.builder.template")}
        </p>
        <span className="rounded-md bg-lime-400 px-2.5 py-1 text-[11px] font-semibold text-slate-950">
          {t("home.showcase.builder.export")}
        </span>
      </div>
    </div>
  );
}

const STAGE_COLUMNS = [
  {
    nameKey: "home.showcase.stages.saved",
    jobKey: "home.showcase.stages.savedJob",
  },
  {
    nameKey: "home.showcase.stages.applied",
    jobKey: "home.showcase.stages.appliedJob",
    hot: true,
  },
  {
    nameKey: "home.showcase.stages.interview",
    jobKey: "home.showcase.stages.interviewJob",
  },
  { nameKey: "home.showcase.stages.offer" },
] as const;

function StagesFrame() {
  const { t } = useTranslation();

  return (
    <div className="rounded-2xl border border-white/[0.07] bg-card/90 p-4 shadow-[0_30px_80px_-30px_rgba(0,0,0,0.8)] backdrop-blur">
      <FrameChrome label={t("home.showcase.stages.label")} />
      <div className="grid grid-cols-4 gap-2">
        {STAGE_COLUMNS.map((column) => (
          <div key={column.nameKey} className="min-w-0">
            <p className="mb-2 text-[10px] font-semibold uppercase tracking-wide text-slate-500">
              {t(column.nameKey)}
            </p>
            {"jobKey" in column ? (
              <p
                className={`rounded-md border px-2 py-2 text-[11px] font-medium leading-snug text-slate-100 ${
                  "hot" in column
                    ? "border-lime-400/40 bg-lime-400/[0.08]"
                    : "border-white/[0.08] bg-white/[0.03]"
                }`}
              >
                {t(column.jobKey)}
              </p>
            ) : null}
          </div>
        ))}
      </div>
    </div>
  );
}

function FrameChrome({ label }: { readonly label: string }) {
  return (
    <div className="mb-3 flex items-center justify-between border-b border-white/[0.06] pb-3">
      <div className="flex gap-1.5" aria-hidden="true">
        <span className="h-2.5 w-2.5 rounded-full bg-white/[0.08]" />
        <span className="h-2.5 w-2.5 rounded-full bg-white/[0.08]" />
        <span className="h-2.5 w-2.5 rounded-full bg-white/[0.08]" />
      </div>
      <span className="font-mono text-[10px] uppercase tracking-wider text-slate-400">
        {label}
      </span>
    </div>
  );
}

function EditsFrame({
  recordingSrc,
}: {
  readonly recordingSrc: string | null;
}) {
  if (recordingSrc) {
    return (
      <div data-hero-review="media">
        <ReviewRecording src={recordingSrc} />
      </div>
    );
  }

  return (
    <ReviewFrame>
      <StaticReview />
    </ReviewFrame>
  );
}

function framePanel(frame: HeroFrame, recordingSrc: string | null) {
  if (frame === "edits") return <EditsFrame recordingSrc={recordingSrc} />;
  if (frame === "check") return <CheckFrame />;
  if (frame === "builder") return <BuilderFrame />;
  return <StagesFrame />;
}

export function HeroShowcase() {
  const { t } = useTranslation();
  const baseId = useId();
  const [reduceMotion] = useState(prefersReducedMotion);
  const [frame, setFrame] = useState<HeroFrame>("edits");
  const [hovered, setHovered] = useState(false);
  const [keyboardPaused, setKeyboardPaused] = useState(false);
  const [timerEpoch, setTimerEpoch] = useState(0);
  const fromPointer = useRef(false);
  const src = HERO_REVIEW_MEDIA_SOURCES.find((item) => item.length > 0);
  const recordingSrc =
    src !== undefined && !(reduceMotion && isAnimatedMedia(src)) ? src : null;
  const paused = hovered || keyboardPaused;

  useEffect(() => {
    if (reduceMotion || paused) return;
    const id = window.setInterval(() => {
      setFrame((current) => nextFrame(current));
    }, HERO_FRAME_MS);
    return () => window.clearInterval(id);
  }, [reduceMotion, paused, timerEpoch]);

  const selectFrame = (next: HeroFrame) => {
    setFrame(next);
    setTimerEpoch((epoch) => epoch + 1);
  };

  const onTabsKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    const index = HERO_FRAMES.indexOf(frame);
    let next: HeroFrame | null = null;
    if (event.key === "ArrowRight") {
      next = HERO_FRAMES[(index + 1) % HERO_FRAMES.length];
    } else if (event.key === "ArrowLeft") {
      next = HERO_FRAMES[(index - 1 + HERO_FRAMES.length) % HERO_FRAMES.length];
    } else if (event.key === "Home") {
      next = HERO_FRAMES[0];
    } else if (event.key === "End") {
      next = HERO_FRAMES[HERO_FRAMES.length - 1];
    }
    if (!next) return;
    event.preventDefault();
    selectFrame(next);
    document.getElementById(tabId(baseId, next))?.focus();
  };

  return (
    <div
      className="landing-fade-up landing-fade-up-3 relative mx-auto w-full min-w-0 max-w-[560px]"
      data-hero-media-slot="resume-review"
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      onPointerDown={() => {
        fromPointer.current = true;
      }}
      onPointerUp={() => {
        fromPointer.current = false;
      }}
      onPointerCancel={() => {
        fromPointer.current = false;
      }}
      onFocus={() => {
        const viaPointer = fromPointer.current;
        fromPointer.current = false;
        if (viaPointer) return;
        setKeyboardPaused(true);
      }}
      onBlur={(event) => {
        const next = event.relatedTarget;
        if (next instanceof Node && event.currentTarget.contains(next)) return;
        setKeyboardPaused(false);
      }}
    >
      <div className="relative">
        <div aria-hidden inert className="pointer-events-none invisible">
          <div className="grid">
            {HERO_FRAMES.map((id) => (
              <div key={id} className="col-start-1 row-start-1 self-start">
                {framePanel(id, recordingSrc)}
              </div>
            ))}
          </div>
          <div className="mt-3 flex flex-wrap gap-2">
            {HERO_FRAMES.map((id) => (
              <span
                key={id}
                className="rounded-full border px-2.5 py-1 text-[12px] leading-tight"
              >
                {t(`home.showcase.frames.${id}`)}
              </span>
            ))}
          </div>
        </div>
        <div className="absolute inset-x-0 top-0">
          {HERO_FRAMES.map((id) => {
            const active = frame === id;
            return (
              <div
                key={id}
                id={panelId(baseId, id)}
                role="tabpanel"
                aria-labelledby={tabId(baseId, id)}
                data-hero-frame={id}
                data-active={active ? "true" : "false"}
                hidden={!active}
              >
                {framePanel(id, recordingSrc)}
              </div>
            );
          })}
          <div
            className="mt-3 flex flex-wrap gap-2"
            role="tablist"
            aria-label={t("home.showcase.framesLabel")}
            onKeyDown={onTabsKeyDown}
          >
            {HERO_FRAMES.map((id) => {
              const selected = frame === id;
              return (
                <button
                  key={id}
                  id={tabId(baseId, id)}
                  type="button"
                  role="tab"
                  aria-selected={selected}
                  aria-controls={panelId(baseId, id)}
                  tabIndex={selected ? 0 : -1}
                  className={`rounded-full border px-2.5 py-1 text-[12px] leading-tight transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-lime-400 ${
                    selected
                      ? "border-lime-400/70 bg-lime-400/10 text-lime-100"
                      : "border-white/10 text-slate-400 hover:border-white/20 hover:text-slate-200"
                  }`}
                  onClick={() => selectFrame(id)}
                >
                  {t(`home.showcase.frames.${id}`)}
                </button>
              );
            })}
          </div>
        </div>
      </div>
    </div>
  );
}

function tabId(baseId: string, frame: HeroFrame) {
  return `${baseId}-tab-${frame}`;
}

function panelId(baseId: string, frame: HeroFrame) {
  return `${baseId}-panel-${frame}`;
}
