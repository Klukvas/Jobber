import { cn } from "@/shared/lib/utils";

interface BrowserMockupProps {
  readonly url: string;
  readonly src: string;
  readonly alt: string;
  readonly dark?: boolean;
  readonly className?: string;
}

export function BrowserMockup({ url, src, alt, dark = false, className }: BrowserMockupProps) {
  if (dark) {
    return (
      <div className={cn("overflow-hidden rounded-xl border border-white/10 shadow-2xl", className)}>
        <div className="flex items-center gap-2 border-b border-white/10 bg-slate-800/80 px-4 py-2.5">
          <div className="h-3 w-3 rounded-full bg-red-500/70" />
          <div className="h-3 w-3 rounded-full bg-yellow-500/70" />
          <div className="h-3 w-3 rounded-full bg-green-500/70" />
          <div className="ml-3 flex-1 truncate rounded bg-slate-700/60 px-3 py-0.5 text-[11px] text-slate-400">
            {url}
          </div>
        </div>
        <img src={src} alt={alt} className="block w-full" loading="lazy" />
      </div>
    );
  }

  return (
    <div className={cn("overflow-hidden rounded-xl border shadow-lg", className)}>
      <div className="flex items-center gap-2 border-b bg-muted/60 px-4 py-2.5">
        <div className="h-2.5 w-2.5 rounded-full bg-red-400/70" />
        <div className="h-2.5 w-2.5 rounded-full bg-yellow-400/70" />
        <div className="h-2.5 w-2.5 rounded-full bg-green-400/70" />
        <div className="ml-3 flex-1 truncate rounded bg-muted px-3 py-0.5 text-[11px] text-muted-foreground">
          {url}
        </div>
      </div>
      <img src={src} alt={alt} className="block w-full" loading="lazy" />
    </div>
  );
}
