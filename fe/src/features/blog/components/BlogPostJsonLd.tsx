import { useEffect } from "react";
import type { BlogPost } from "../lib/blogLoader";

const SCRIPT_ID = "blog-post-jsonld";
const SITE_URL = "https://jobber-app.com";

interface BlogPostJsonLdProps {
  readonly post: BlogPost;
}

export function BlogPostJsonLd({ post }: BlogPostJsonLdProps) {
  useEffect(() => {
    const existing = document.getElementById(SCRIPT_ID);
    if (existing) existing.remove();

    const imageUrl = post.image
      ? post.image.startsWith("http")
        ? post.image
        : `${SITE_URL}${post.image.startsWith("/") ? "" : "/"}${post.image}`
      : `${SITE_URL}/og-image.png`;
    const canonicalUrl = `${SITE_URL}/blog/${post.slug}`;

    const blogPosting = {
      "@context": "https://schema.org",
      "@type": "BlogPosting",
      headline: post.title,
      description: post.description,
      image: {
        "@type": "ImageObject",
        url: imageUrl,
        width: 1200,
        height: 630,
      },
      datePublished: post.date,
      dateModified: post.dateModified ?? post.date,
      url: canonicalUrl,
      mainEntityOfPage: {
        "@type": "WebPage",
        "@id": canonicalUrl,
      },
      inLanguage: post.lang === "ua" ? "uk" : post.lang,
      keywords: post.tags.join(", "),
      author: {
        "@type": "Organization",
        name: "Jobber",
        url: SITE_URL,
      },
      publisher: {
        "@type": "Organization",
        name: "Jobber",
        url: SITE_URL,
        logo: {
          "@type": "ImageObject",
          url: `${SITE_URL}/favicon.png`,
        },
      },
    };

    const breadcrumb = {
      "@context": "https://schema.org",
      "@type": "BreadcrumbList",
      itemListElement: [
        {
          "@type": "ListItem",
          position: 1,
          name: "Home",
          item: SITE_URL,
        },
        {
          "@type": "ListItem",
          position: 2,
          name: "Blog",
          item: `${SITE_URL}/blog`,
        },
        {
          "@type": "ListItem",
          position: 3,
          name: post.title,
          item: canonicalUrl,
        },
      ],
    };

    const script = document.createElement("script");
    script.id = SCRIPT_ID;
    script.type = "application/ld+json";
    script.text = JSON.stringify([blogPosting, breadcrumb]);
    document.head.appendChild(script);

    return () => {
      script.remove();
    };
  }, [post]);

  return null;
}
