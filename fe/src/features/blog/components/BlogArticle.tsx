import { useMemo } from 'react';
import { marked } from 'marked';
import DOMPurify from 'dompurify';

interface BlogArticleProps {
  readonly content: string;
}

// Strip the first H1 line from markdown — the page header already renders it
function stripFirstH1(md: string): string {
  return md.replace(/^# .+$/m, '').trimStart();
}

export function BlogArticle({ content }: BlogArticleProps) {
  const html = useMemo(
    () =>
      DOMPurify.sanitize(
        marked.parse(stripFirstH1(content), { async: false }),
      ),
    [content],
  );

  return (
    <article
      className="prose prose-lg prose-neutral dark:prose-invert max-w-none
        prose-headings:font-semibold prose-headings:tracking-tight
        prose-h2:mt-10 prose-h2:mb-4 prose-h2:text-2xl prose-h2:border-b prose-h2:pb-2 prose-h2:border-border
        prose-h3:mt-8 prose-h3:mb-3
        prose-p:leading-relaxed
        prose-a:text-primary prose-a:no-underline hover:prose-a:underline prose-a:font-medium
        prose-strong:text-foreground
        prose-blockquote:border-l-primary prose-blockquote:bg-muted/50 prose-blockquote:py-1 prose-blockquote:px-4 prose-blockquote:rounded-r-lg prose-blockquote:not-italic
        prose-li:marker:text-primary
        prose-hr:border-border
        prose-code:rounded prose-code:bg-muted prose-code:px-1.5 prose-code:py-0.5 prose-code:text-sm prose-code:font-normal prose-code:before:content-none prose-code:after:content-none
        prose-pre:bg-muted prose-pre:border prose-pre:border-border prose-pre:rounded-lg
        prose-img:rounded-lg prose-img:shadow-md"
      dangerouslySetInnerHTML={{ __html: html }}
    />
  );
}
