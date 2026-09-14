import { cn } from "@/shared/lib/utils";

export const FLUXLAB_URL = "https://flux-lab.dev/en";

interface PoweredByFluxLabProps {
  readonly className?: string;
}

/**
 * Required build attribution, shown in the landing footer and the in-app
 * footer.
 *
 * It is a real external anchor rather than a styled span so it is keyboard
 * reachable, middle-clickable and announced as a link; `target="_blank"`
 * always ships with `rel="noopener noreferrer"`.
 *
 * The wording is a brand mark, so it stays "Powered by FluxLab" verbatim in
 * every locale — routing it through i18n would only invite drift.
 */
export function PoweredByFluxLab({ className }: PoweredByFluxLabProps) {
  return (
    <a
      href={FLUXLAB_URL}
      target="_blank"
      rel="noopener noreferrer"
      className={cn(
        "inline-flex items-center rounded-sm underline-offset-4 transition-colors",
        // 13px type in a wrapped footer row measured 119x20 — wide enough, but
        // half the height a thumb needs. The minimum is dropped from `sm` up so
        // the pointer footers keep their line height.
        "min-h-11 sm:min-h-0",
        "hover:underline focus-visible:underline",
        "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background",
        className,
      )}
    >
      Powered by FluxLab
    </a>
  );
}
