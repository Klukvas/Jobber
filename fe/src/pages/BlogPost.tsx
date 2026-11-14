import { useEffect, useRef } from "react";
import { useParams, Link, useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { ArrowLeft, Calendar, Tag } from "lucide-react";
import { usePageMeta } from "@/shared/lib/usePageMeta";
import { useHreflangLinks } from "@/shared/lib/useHreflangLinks";
import {
  getPostBySlug,
  getHreflangAlternates,
  postLanguageTag,
} from "@/features/blog/lib/blogLoader";
import { formatPostDate } from "@/features/blog/lib/formatPostDate";
import { useDocumentLanguage } from "@/shared/lib/useDocumentLanguage";
import { useDateLocale } from "@/shared/lib/dateFnsLocale";
import { BlogArticle } from "@/features/blog/components/BlogArticle";
import { BlogPostJsonLd } from "@/features/blog/components/BlogPostJsonLd";
import { HomeNavbar } from "@/features/home/components/HomeNavbar";
import { FooterSection } from "@/features/home/components/FooterSection";
import { useAuthStore } from "@/stores/authStore";
import { ContentLanguageNotice } from "@/shared/ui/ContentLanguageNotice";

export default function BlogPost() {
  const { slug } = useParams<{ slug: string }>();
  const { t, i18n } = useTranslation();
  const navigate = useNavigate();
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);

  const post = slug ? getPostBySlug(slug, i18n.language) : undefined;

  usePageMeta(
    post
      ? {
          title: `${post.title} — Cursopus`,
          description: post.description,
          ogType: "article" as const,
        }
      : { titleKey: "blog.notFound", noindex: true },
  );

  // A slug resolves across languages, so an RU or UA article can be rendered
  // while the interface is in English. The document has to declare the language
  // of the text it is actually showing — restored on unmount by the hook.
  useDocumentLanguage(post ? postLanguageTag(post) : undefined);

  const alternates = post ? getHreflangAlternates(post) : [];
  const uiLanguage = (i18n.language ?? "en").split("-")[0];
  const translation = alternates.find(
    (a) => a.hreflang === uiLanguage && a.slug !== post?.slug,
  );

  // Switching the language while reading follows the article into that
  // language when it has a translation. Only a switch does this — opening a
  // localized URL under another interface language (a shared link, a crawler)
  // still renders the article it names.
  const shownLanguage = useRef(uiLanguage);
  useEffect(() => {
    if (shownLanguage.current === uiLanguage) return;
    shownLanguage.current = uiLanguage;
    if (translation) navigate(`/blog/${translation.slug}`, { replace: true });
  }, [uiLanguage, translation, navigate]);

  // Link translated versions of this article for search engines; a post with no
  // translations still emits its own language.
  useHreflangLinks(
    alternates.map((a) => ({
      hreflang: a.hreflang,
      path: `/blog/${a.slug}`,
    })),
  );

  const dateLocale = useDateLocale();
  const formattedDate = post ? formatPostDate(post.date, dateLocale) : "";

  return (
    <div className="flex min-h-screen flex-col">
      <HomeNavbar
        isAuthenticated={isAuthenticated}
        onLogin={() => navigate("/login")}
        onRegister={() => navigate("/register")}
        onGoPlatform={() => navigate("/app/jobs")}
      />
      <main className="mx-auto w-full max-w-3xl flex-1 px-4 pt-24 pb-16">
        <Link
          to="/blog"
          className="mb-8 inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-primary-ink transition-colors"
        >
          <ArrowLeft className="h-4 w-4" />
          {t("blog.backToBlog")}
        </Link>

        {post ? (
          <>
            <BlogPostJsonLd post={post} />
            <ContentLanguageNotice
              contentLanguage={postLanguageTag(post)}
              action={
                translation
                  ? {
                      to: `/blog/${translation.slug}`,
                      label: t("blog.readTranslation"),
                    }
                  : { to: "/blog", label: t("blog.browseInYourLanguage") }
              }
            />
            <header className="mb-10">
              <h1 className="text-4xl font-bold tracking-tight leading-tight">
                {post.title}
              </h1>
              <div className="mt-4 flex flex-wrap items-center gap-4 text-sm text-muted-foreground">
                <span className="flex items-center gap-1.5">
                  <Calendar className="h-4 w-4" />
                  {formattedDate}
                </span>
                {post.tags.length > 0 && (
                  <span className="flex items-center gap-1.5">
                    <Tag className="h-4 w-4" />
                    {post.tags.join(", ")}
                  </span>
                )}
              </div>
              <div className="mt-6 h-px bg-gradient-to-r from-primary/50 via-border to-transparent" />
            </header>
            <BlogArticle content={post.content} />
            <footer className="mt-12 border-t pt-8">
              <Link
                to="/blog"
                className="inline-flex items-center gap-1.5 text-sm font-medium text-primary-ink hover:underline"
              >
                <ArrowLeft className="h-4 w-4" />
                {t("blog.backToBlog")}
              </Link>
            </footer>
          </>
        ) : (
          <div className="py-20 text-center">
            <h1 className="text-2xl font-bold">{t("blog.notFound")}</h1>
            <p className="mt-2 text-muted-foreground">
              {t("blog.notFoundDescription")}
            </p>
          </div>
        )}
      </main>
      <FooterSection />
    </div>
  );
}
