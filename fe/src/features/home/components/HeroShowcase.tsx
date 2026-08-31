import type { CSSProperties, ReactNode } from "react";
import { useEffect, useState } from "react";
import { Bell, Sparkles, TrendingUp } from "lucide-react";
import { useScrollY } from "@/shared/hooks/useScrollY";

// Decorative demo mirroring real platform widgets: JobKanbanBoard,
// MatchScoreCard, FunnelVisualization and JobReminders. All content is
// aria-hidden stage dressing, kept in English like a product screenshot.

type BadgeVariant = "lime" | "sky" | "gray" | "teal" | "amber";

const BADGE_STYLES: Record<BadgeVariant, string> = {
  lime: "bg-lime-400/[0.08] text-lime-400 border-lime-400/[0.18]",
  sky: "bg-sky-400/[0.08] text-sky-400 border-sky-400/[0.18]",
  gray: "bg-white/[0.04] text-slate-400 border-white/[0.07]",
  teal: "bg-teal-400/[0.08] text-teal-400 border-teal-400/[0.18]",
  amber: "bg-amber-400/[0.08] text-amber-400 border-amber-400/[0.18]",
};

const MATCH_SCORE = 87;
const RING_RADIUS = 30;
const RING_CIRCUMFERENCE = 2 * Math.PI * RING_RADIUS;

const MATCH_CATEGORIES = [
  { name: "Skills", score: 92, delaySec: 1.15 },
  { name: "Experience", score: 81, delaySec: 1.3 },
] as const;

const FUNNEL_ROWS = [
  { name: "Applied", count: 12, widthPercent: 100, barClass: "bg-lime-400/80" },
  {
    name: "Interview",
    count: 6,
    widthPercent: 50,
    barClass: "bg-lime-400/50",
  },
  { name: "Offer", count: 2, widthPercent: 17, barClass: "bg-teal-400/80" },
] as const;

const prefersReducedMotion = () =>
  window.matchMedia("(prefers-reduced-motion: reduce)").matches;

function useCountUp(target: number, durationMs: number, delayMs: number) {
  const [value, setValue] = useState(() =>
    prefersReducedMotion() ? target : 0,
  );

  useEffect(() => {
    if (prefersReducedMotion()) return;
    let frame = 0;
    const timer = setTimeout(() => {
      const start = performance.now();
      const tick = (now: number) => {
        const progress = Math.min((now - start) / durationMs, 1);
        const eased = 1 - Math.pow(1 - progress, 3);
        setValue(Math.round(eased * target));
        if (progress < 1) frame = requestAnimationFrame(tick);
      };
      frame = requestAnimationFrame(tick);
    }, delayMs);
    return () => {
      clearTimeout(timer);
      cancelAnimationFrame(frame);
    };
  }, [target, durationMs, delayMs]);

  return value;
}

function KanbanDemoCard({
  company,
  role,
  badges,
  highlighted,
}: {
  company: string;
  role: string;
  badges: ReadonlyArray<{ text: string; variant: BadgeVariant }>;
  highlighted?: boolean;
}) {
  return (
    <div
      className={`rounded-lg border bg-muted p-2.5 ${
        highlighted ? "border-lime-400/25" : "border-white/[0.07]"
      }`}
    >
      <div className="mb-0.5 font-mono text-[9px] text-slate-500">
        {company}
      </div>
      <div className="mb-1.5 text-[12px] font-semibold leading-snug text-slate-100">
        {role}
      </div>
      <div className="flex flex-wrap gap-1">
        {badges.map((badge) => (
          <span
            key={badge.text}
            className={`rounded border px-1.5 py-px font-mono text-[9px] font-medium ${BADGE_STYLES[badge.variant]}`}
          >
            {badge.text}
          </span>
        ))}
      </div>
    </div>
  );
}

function KanbanColumn({
  title,
  restCount,
  movedCount,
  children,
}: {
  title: string;
  restCount: number;
  movedCount: number;
  children: ReactNode;
}) {
  return (
    <div>
      <div className="mb-2.5 flex items-center justify-between font-mono text-[10px] font-medium uppercase tracking-wider text-slate-500">
        {title}
        <span className="relative h-[18px] w-[18px]">
          <span className="hero-count-rest absolute inset-0 flex items-center justify-center rounded bg-muted text-[10px] text-slate-400">
            {restCount}
          </span>
          <span className="hero-count-moved absolute inset-0 flex items-center justify-center rounded bg-muted text-[10px] text-slate-400">
            {movedCount}
          </span>
        </span>
      </div>
      <div className="flex flex-col">{children}</div>
    </div>
  );
}

function WidgetLabel({ icon, text }: { icon: ReactNode; text: string }) {
  return (
    <div className="mb-2.5 flex items-center gap-1.5 font-mono text-[10px] font-medium uppercase tracking-wider text-slate-500">
      {icon}
      {text}
    </div>
  );
}

function MatchScoreWidget() {
  const score = useCountUp(MATCH_SCORE, 1300, 700);

  return (
    <div className="rounded-xl border border-white/[0.08] bg-card p-3.5 shadow-[0_20px_50px_-20px_rgba(0,0,0,0.7)]">
      <WidgetLabel
        icon={<Sparkles className="h-3 w-3 text-lime-400" />}
        text="AI Match Score"
      />
      <div className="flex items-center gap-3">
        <div className="relative h-[68px] w-[68px] shrink-0">
          <svg viewBox="0 0 72 72" className="h-full w-full -rotate-90">
            <circle
              cx="36"
              cy="36"
              r={RING_RADIUS}
              fill="none"
              stroke="hsl(222 30% 16%)"
              strokeWidth="6"
            />
            <circle
              cx="36"
              cy="36"
              r={RING_RADIUS}
              fill="none"
              stroke="#a3e635"
              strokeWidth="6"
              strokeLinecap="round"
              strokeDasharray={RING_CIRCUMFERENCE}
              strokeDashoffset={RING_CIRCUMFERENCE * (1 - MATCH_SCORE / 100)}
              className="hero-ring-arc"
              style={
                {
                  "--ring-circumference": `${RING_CIRCUMFERENCE}`,
                } as CSSProperties
              }
            />
          </svg>
          <span className="absolute inset-0 flex items-center justify-center text-[15px] font-bold text-lime-400">
            {score}%
          </span>
        </div>
        <div className="flex-1 space-y-2">
          {MATCH_CATEGORIES.map((category) => (
            <div key={category.name}>
              <div className="mb-1 flex justify-between text-[10px] text-slate-400">
                <span>{category.name}</span>
                <span className="text-lime-400">{category.score}%</span>
              </div>
              <div className="h-1.5 overflow-hidden rounded-full bg-muted">
                <div
                  className="h-full rounded-full bg-lime-400/80 motion-safe:animate-grow-bar"
                  style={
                    {
                      width: `${category.score}%`,
                      "--bar-w": `${category.score}%`,
                      animationDelay: `${category.delaySec}s`,
                      animationFillMode: "backwards",
                    } as CSSProperties
                  }
                />
              </div>
            </div>
          ))}
        </div>
      </div>
      <div className="mt-2.5 flex flex-wrap gap-1">
        <span
          className={`rounded border px-1.5 py-px font-mono text-[9px] ${BADGE_STYLES.lime}`}
        >
          React ✓
        </span>
        <span
          className={`rounded border px-1.5 py-px font-mono text-[9px] ${BADGE_STYLES.lime}`}
        >
          TypeScript ✓
        </span>
        <span
          className={`rounded border px-1.5 py-px font-mono text-[9px] ${BADGE_STYLES.amber}`}
        >
          + GraphQL
        </span>
      </div>
    </div>
  );
}

function FunnelWidget() {
  return (
    <div className="rounded-xl border border-white/[0.08] bg-card p-3.5 shadow-[0_20px_50px_-20px_rgba(0,0,0,0.7)]">
      <WidgetLabel
        icon={<TrendingUp className="h-3 w-3 text-lime-400" />}
        text="Pipeline · 30 days"
      />
      <div className="space-y-2">
        {FUNNEL_ROWS.map((row, index) => (
          <div key={row.name}>
            <div className="mb-1 flex justify-between text-[10px] text-slate-400">
              <span>{row.name}</span>
              <span>{row.count}</span>
            </div>
            <div className="h-1.5 overflow-hidden rounded-full bg-muted">
              <div
                className={`h-full rounded-full motion-safe:animate-grow-bar ${row.barClass}`}
                style={
                  {
                    width: `${row.widthPercent}%`,
                    "--bar-w": `${row.widthPercent}%`,
                    animationDelay: `${1.2 + index * 0.15}s`,
                    animationFillMode: "backwards",
                  } as CSSProperties
                }
              />
            </div>
          </div>
        ))}
      </div>
      <div className="mt-2.5 flex justify-between border-t border-white/[0.06] pt-2 font-mono text-[10px]">
        <span className="text-slate-500">Response rate</span>
        <span className="text-lime-400">42% ↑</span>
      </div>
    </div>
  );
}

function ReminderToast() {
  return (
    <div className="flex items-center gap-2.5 rounded-xl border border-amber-400/20 bg-card/95 p-3 shadow-[0_16px_40px_-16px_rgba(0,0,0,0.8)] backdrop-blur">
      <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-amber-400/10 text-amber-400">
        <Bell className="h-4 w-4" />
      </span>
      <div className="min-w-0">
        <div className="font-mono text-[9px] uppercase tracking-wider text-amber-400/90">
          Reminder
        </div>
        <div className="truncate text-[12px] font-semibold text-slate-100">
          Linear — Interview
        </div>
        <div className="text-[10px] text-slate-500">Tomorrow, 14:00</div>
      </div>
    </div>
  );
}

// Hero is on screen for roughly the first ~900px of scroll; beyond that
// the parallax layers are off-screen and re-renders would be wasted.
const PARALLAX_SCROLL_RANGE = 900;

export function HeroShowcase() {
  const [reduceMotion] = useState(prefersReducedMotion);
  const scrollY = useScrollY(PARALLAX_SCROLL_RANGE);
  const drift = reduceMotion ? 0 : scrollY;
  // Positive factor lags behind the scroll (background), negative rises
  // ahead of it (foreground) — the speed difference creates the depth.
  const parallaxLayer = (factor: number): CSSProperties => ({
    transform: `translate3d(0, ${(drift * factor).toFixed(1)}px, 0)`,
  });

  return (
    <div
      className="landing-fade-up landing-fade-up-3 relative mx-auto w-full max-w-[560px]"
      aria-hidden="true"
    >
      {/* Main kanban window */}
      <div
        className="rounded-2xl border border-white/[0.07] bg-card/90 p-4 shadow-[0_30px_80px_-30px_rgba(0,0,0,0.8)] backdrop-blur"
        style={parallaxLayer(0.05)}
      >
        <div className="mb-3.5 flex items-center justify-between border-b border-white/[0.06] pb-3">
          <div className="flex gap-1.5">
            <span className="h-2.5 w-2.5 rounded-full bg-white/[0.08]" />
            <span className="h-2.5 w-2.5 rounded-full bg-white/[0.08]" />
            <span className="h-2.5 w-2.5 rounded-full bg-white/[0.08]" />
          </div>
          <span className="font-mono text-[10px] uppercase tracking-wider text-slate-600">
            jobber — board
          </span>
        </div>
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-3">
          <KanbanColumn title="Applied" restCount={3} movedCount={2}>
            <div className="relative mb-2">
              {/* Drop placeholder revealed while the card is mid-flight */}
              <div className="absolute inset-0 rounded-lg border border-dashed border-white/10 bg-white/[0.02]" />
              <div className="hero-card-leave rounded-lg">
                <KanbanDemoCard
                  company="Vercel"
                  role="Staff Engineer"
                  badges={[{ text: "87% match", variant: "lime" }]}
                  highlighted
                />
              </div>
            </div>
            <div className="mb-2">
              <KanbanDemoCard
                company="Stripe"
                role="Senior Frontend Engineer"
                badges={[
                  { text: "Remote", variant: "sky" },
                  { text: "$180k", variant: "gray" },
                ]}
              />
            </div>
            <KanbanDemoCard
              company="Notion"
              role="Full-Stack Engineer"
              badges={[{ text: "Hybrid", variant: "sky" }]}
            />
          </KanbanColumn>
          <KanbanColumn title="Interview" restCount={1} movedCount={2}>
            <div className="hero-card-arrive rounded-lg">
              <KanbanDemoCard
                company="Vercel"
                role="Staff Engineer"
                badges={[{ text: "87% match", variant: "lime" }]}
                highlighted
              />
            </div>
            <KanbanDemoCard
              company="Linear"
              role="Product Engineer"
              badges={[{ text: "Tomorrow 14:00", variant: "amber" }]}
            />
          </KanbanColumn>
          <div className="hidden sm:block">
            <KanbanColumn title="Offer" restCount={1} movedCount={1}>
              <KanbanDemoCard
                company="Railway"
                role="Infrastructure Engineer"
                badges={[{ text: "\u{1F389} Offer", variant: "teal" }]}
              />
            </KanbanColumn>
          </div>
        </div>
      </div>

      {/* Overlapping satellites — placed over quiet zones of the board
          so they never cover the animated columns or their counters.
          Outer div: position + scroll parallax; inner div: CSS float
          animation (they must not share a transform). */}
      <div
        className="absolute -bottom-16 right-0 z-20 w-[188px] sm:bottom-auto sm:top-[152px] sm:-right-12 sm:w-[204px]"
        style={parallaxLayer(-0.09)}
      >
        <div
          className="hero-float"
          style={{ "--hero-tilt": "2.5deg" } as CSSProperties}
        >
          <MatchScoreWidget />
        </div>
      </div>
      <div
        className="absolute -bottom-24 left-0 z-20 hidden w-[212px] sm:-left-12 sm:block"
        style={parallaxLayer(-0.06)}
      >
        <div
          className="hero-float"
          style={
            {
              "--hero-tilt": "-2deg",
              "--hero-float-delay": "1.2s",
            } as CSSProperties
          }
        >
          <FunnelWidget />
        </div>
      </div>
      <div
        className="absolute -top-10 left-6 z-30 w-[212px]"
        style={parallaxLayer(-0.075)}
      >
        <div className="hero-toast-in">
          <ReminderToast />
        </div>
      </div>
    </div>
  );
}
