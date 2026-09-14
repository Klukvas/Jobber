import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { Calendar, ArrowRight } from "lucide-react";
import { useDateLocale } from "@/shared/lib/dateFnsLocale";
import { formatPostDate } from "../lib/formatPostDate";
import type { BlogPost } from "../lib/blogLoader";

interface BlogPostCardProps {
  readonly post: BlogPost;
}

/**
 * One card, one link.
 *
 * The whole card is clickable, and it used to get there by laying a second
 * `<a>` over it — an empty one, so it had no accessible name, and there were
 * as many of them on `/blog` as there were posts. Every one duplicated the
 * title link right above it and announced itself as nothing at all.
 *
 * The title link covers the card instead, through a pseudo-element stretched
 * to the card's own bounds. Same target area, one destination, and the only
 * thing in the accessibility tree is the link that carries the post's title.
 */
export function BlogPostCard({ post }: BlogPostCardProps) {
  const { t } = useTranslation();
  // The shared map, not a local two-way branch: the local one knew about
  // Ukrainian and English only, so every Russian post was dated in English.
  const dateLocale = useDateLocale();
  const formattedDate = formatPostDate(post.date, dateLocale);

  return (
    <article className="group relative overflow-hidden rounded-xl border bg-card transition-all hover:shadow-lg hover:border-primary/30">
      <div className="absolute inset-x-0 top-0 h-1 bg-gradient-to-r from-primary/60 to-primary/20 opacity-0 transition-opacity group-hover:opacity-100" />
      <div className="p-6">
        <h2 className="text-xl font-semibold tracking-tight group-hover:text-primary transition-colors">
          <Link
            to={`/blog/${post.slug}`}
            // `after:` is what makes the card clickable: an absolutely
            // positioned box on the link, stretched to the nearest positioned
            // ancestor — the <article> above.
            className="rounded after:absolute after:inset-0 after:content-[''] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
          >
            {post.title}
          </Link>
        </h2>
        <p className="mt-2 text-muted-foreground leading-relaxed line-clamp-2">
          {post.description}
        </p>
        <div className="mt-4 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <span className="flex items-center gap-1.5 text-sm text-muted-foreground">
              <Calendar className="h-3.5 w-3.5" />
              {formattedDate}
            </span>
            {post.tags.length > 0 && (
              <div className="hidden sm:flex items-center gap-1.5">
                {post.tags.slice(0, 2).map((tag) => (
                  <span
                    key={tag}
                    className="relative z-10 rounded-full bg-secondary px-2.5 py-0.5 text-xs text-secondary-foreground"
                  >
                    {tag}
                  </span>
                ))}
              </div>
            )}
          </div>
          <span className="flex items-center gap-1 text-sm font-medium text-primary opacity-0 transition-opacity group-hover:opacity-100">
            {t("blog.readMore")}
            <ArrowRight className="h-3.5 w-3.5" />
          </span>
        </div>
      </div>
    </article>
  );
}
