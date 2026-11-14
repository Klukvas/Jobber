import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { Languages } from "lucide-react";

interface ContentLanguageNoticeProps {
  /** BCP 47 tag of the language the page's own text is written in. */
  readonly contentLanguage: string;
  /** Where to go instead, labelled in the interface language. */
  readonly action?: { readonly to: string; readonly label: string };
}

const baseLanguage = (tag: string) => tag.split("-")[0].toLowerCase();

function languageName(tag: string, inLanguage: string): string {
  try {
    return (
      new Intl.DisplayNames([inLanguage], { type: "language" }).of(tag) ?? tag
    );
  } catch {
    return tag;
  }
}

/**
 * Says so when a page's text is in a different language from the interface.
 *
 * Switching the language translated the navigation around an untranslated
 * article or legal page and left a page in two languages with no explanation.
 * Renders nothing when the languages already match.
 */
export function ContentLanguageNotice({
  contentLanguage,
  action,
}: ContentLanguageNoticeProps) {
  const { t, i18n } = useTranslation();
  const uiLanguage = baseLanguage(i18n.language ?? "en");
  if (uiLanguage === baseLanguage(contentLanguage)) return null;

  return (
    <div
      role="note"
      lang={uiLanguage}
      className="mb-8 flex flex-col gap-2 rounded-lg border bg-muted/50 px-4 py-3 text-sm sm:flex-row sm:items-center sm:justify-between"
    >
      <span className="flex items-center gap-2 text-muted-foreground">
        <Languages className="h-4 w-4 shrink-0" aria-hidden="true" />
        {t("common.contentLanguageOnly", {
          language: languageName(contentLanguage, uiLanguage),
        })}
      </span>
      {action && (
        <Link
          to={action.to}
          className="inline-flex items-center font-medium text-primary-ink hover:underline max-sm:min-h-11"
        >
          {action.label}
        </Link>
      )}
    </div>
  );
}
