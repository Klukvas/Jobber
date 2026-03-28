import { useParams, Link, useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { ArrowLeft, Calendar, Tag } from "lucide-react";
import { format, parseISO } from "date-fns";
import { uk, ru, enUS } from "date-fns/locale";
import { usePageMeta } from "@/shared/lib/usePageMeta";
import { getPostBySlug } from "@/features/blog/lib/blogLoader";
import { BlogArticle } from "@/features/blog/components/BlogArticle";
import { BlogPostJsonLd } from "@/features/blog/components/BlogPostJsonLd";
import { HomeNavbar } from "@/features/home/components/HomeNavbar";
import { FooterSection } from "@/features/home/components/FooterSection";
import { useAuthStore } from "@/stores/authStore";

export default function BlogPost() {
  const { slug } = useParams<{ slug: string }>();
  const { t, i18n } = useTranslation();
  const navigate = useNavigate();
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);

  const post = slug ? getPostBySlug(slug, i18n.language) : undefined;

  usePageMeta(
    post
      ? {
          title: `${post.title} — Jobber`,
          description: post.description,
        }
      : { titleKey: "blog.notFound", noindex: true },
  );

  const dateLocale =
    i18n.language === "ua" ? uk : i18n.language === "ru" ? ru : enUS;
  const formattedDate = post
    ? format(parseISO(post.date), "MMMM d, yyyy", { locale: dateLocale })
    : "";

  return (
    <div className="flex min-h-screen flex-col">
      <HomeNavbar
        isAuthenticated={isAuthenticated}
        onLogin={() => navigate("/login")}
        onRegister={() => navigate("/register")}
        onGoPlatform={() => navigate("/app/applications")}
      />
      <main className="mx-auto w-full max-w-3xl flex-1 px-4 pt-24 pb-16">
        <Link
          to="/blog"
          className="mb-8 inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-primary transition-colors"
        >
          <ArrowLeft className="h-4 w-4" />
          {t("blog.backToBlog")}
        </Link>

        {post ? (
          <>
            <BlogPostJsonLd post={post} />
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
                className="inline-flex items-center gap-1.5 text-sm font-medium text-primary hover:underline"
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
