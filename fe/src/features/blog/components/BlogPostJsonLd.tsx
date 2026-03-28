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

    const schema = {
      "@context": "https://schema.org",
      "@type": "BlogPosting",
      headline: post.title,
      description: post.description,
      datePublished: post.date,
      url: `${SITE_URL}/blog/${post.slug}`,
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

    const script = document.createElement("script");
    script.id = SCRIPT_ID;
    script.type = "application/ld+json";
    script.text = JSON.stringify(schema);
    document.head.appendChild(script);

    return () => {
      script.remove();
    };
  }, [post]);

  return null;
}
