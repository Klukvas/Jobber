import { useTranslation } from "react-i18next";
import { FileText } from "lucide-react";
import { usePageMeta } from "@/shared/lib/usePageMeta";
import { getAllPosts } from "@/features/blog/lib/blogLoader";
import { BlogHeader } from "@/features/blog/components/BlogHeader";
import { BlogPostCard } from "@/features/blog/components/BlogPostCard";
import { HomeNavbar } from "@/features/home/components/HomeNavbar";
import { FooterSection } from "@/features/home/components/FooterSection";
import { useAuthStore } from "@/stores/authStore";
import { useNavigate } from "react-router-dom";

export default function Blog() {
  const { t, i18n } = useTranslation();
  const navigate = useNavigate();
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);
  usePageMeta({
    titleKey: "seo.blog.title",
    descriptionKey: "seo.blog.description",
  });

  const posts = getAllPosts(i18n.language);

  return (
    <div className="flex min-h-screen flex-col">
      <HomeNavbar
        isAuthenticated={isAuthenticated}
        onLogin={() => navigate("/login")}
        onRegister={() => navigate("/register")}
        onGoPlatform={() => navigate("/app/applications")}
      />
      <main className="mx-auto w-full max-w-3xl flex-1 px-4 pt-24 pb-16">
        <BlogHeader />
        {posts.length > 0 ? (
          <div className="space-y-6">
            {posts.map((post) => (
              <BlogPostCard key={post.slug} post={post} />
            ))}
          </div>
        ) : (
          <div className="py-20 text-center">
            <FileText className="mx-auto h-12 w-12 text-muted-foreground/50" />
            <p className="mt-4 text-muted-foreground">{t("blog.noPosts")}</p>
          </div>
        )}
      </main>
      <FooterSection />
    </div>
  );
}
