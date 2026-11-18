import type { ReactNode } from "react";
import { useState } from "react";
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

const prefersReducedMotion = () =>
  window.matchMedia("(prefers-reduced-motion: reduce)").matches;

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
            <span className="rounded-full bg-lime-400 px-1.5 py-0.5 text-[9px] font-semibold uppercase tracking-wide text-slate-950">
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
      <div className="mb-3 flex items-center justify-between border-b border-white/[0.06] pb-3">
        <div className="flex gap-1.5" aria-hidden="true">
          <span className="h-2.5 w-2.5 rounded-full bg-white/[0.08]" />
          <span className="h-2.5 w-2.5 rounded-full bg-white/[0.08]" />
          <span className="h-2.5 w-2.5 rounded-full bg-white/[0.08]" />
        </div>
        <span className="font-mono text-[10px] uppercase tracking-wider text-slate-600">
          {t("jobs.tailoring.review")}
        </span>
      </div>
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

export function HeroShowcase() {
  const [reduceMotion] = useState(prefersReducedMotion);
  const src = HERO_REVIEW_MEDIA_SOURCES.find((item) => item.length > 0);
  const recordingSrc =
    src !== undefined && !(reduceMotion && isAnimatedMedia(src)) ? src : null;

  return (
    <div
      className="landing-fade-up landing-fade-up-3 relative mx-auto w-full min-w-0 max-w-[560px]"
      aria-hidden="true"
      data-hero-media-slot="resume-review"
    >
      {recordingSrc ? (
        <div data-hero-review="media">
          <ReviewRecording src={recordingSrc} />
        </div>
      ) : (
        <ReviewFrame>
          <StaticReview />
        </ReviewFrame>
      )}
    </div>
  );
}
