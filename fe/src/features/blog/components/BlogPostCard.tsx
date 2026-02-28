import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { format, parseISO } from 'date-fns';
import { uk, enUS } from 'date-fns/locale';
import { Calendar, ArrowRight } from 'lucide-react';
import type { BlogPost } from '../lib/blogLoader';

interface BlogPostCardProps {
  readonly post: BlogPost;
}

export function BlogPostCard({ post }: BlogPostCardProps) {
  const { t, i18n } = useTranslation();
  const dateLocale = i18n.language === 'ua' ? uk : enUS;
  const formattedDate = format(parseISO(post.date), 'MMMM d, yyyy', {
    locale: dateLocale,
  });

  return (
    <article className="group relative overflow-hidden rounded-xl border bg-card transition-all hover:shadow-lg hover:border-primary/30">
      <div className="absolute inset-x-0 top-0 h-1 bg-gradient-to-r from-primary/60 to-primary/20 opacity-0 transition-opacity group-hover:opacity-100" />
      <div className="p-6">
        <h2 className="text-xl font-semibold tracking-tight group-hover:text-primary transition-colors">
          <Link to={`/blog/${post.slug}`}>{post.title}</Link>
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
            {t('blog.readMore')}
            <ArrowRight className="h-3.5 w-3.5" />
          </span>
        </div>
      </div>
      <Link
        to={`/blog/${post.slug}`}
        className="absolute inset-0"
        aria-hidden="true"
        tabIndex={-1}
      />
    </article>
  );
}
