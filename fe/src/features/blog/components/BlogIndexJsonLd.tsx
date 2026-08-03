import { useEffect } from "react";
import type { BlogPost } from "../lib/blogLoader";

const SCRIPT_ID = "blog-index-jsonld";
const SITE_URL = "https://jobber-app.com";

interface BlogIndexJsonLdProps {
  readonly posts: readonly BlogPost[];
}

export function BlogIndexJsonLd({ posts }: BlogIndexJsonLdProps) {
  useEffect(() => {
    const existing = document.getElementById(SCRIPT_ID);
    if (existing) existing.remove();

    const collection = {
      "@context": "https://schema.org",
      "@type": "CollectionPage",
      name: "Jobber Blog",
      description:
        "Job search tips, career advice, and engineering insights for tracking your applications.",
      url: `${SITE_URL}/blog`,
      isPartOf: {
        "@type": "WebSite",
        name: "Jobber",
        url: SITE_URL,
      },
      mainEntity: {
        "@type": "ItemList",
        itemListElement: posts.map((post, index) => ({
          "@type": "ListItem",
          position: index + 1,
          url: `${SITE_URL}/blog/${post.slug}`,
          name: post.title,
        })),
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
      ],
    };

    const script = document.createElement("script");
    script.id = SCRIPT_ID;
    script.type = "application/ld+json";
    script.text = JSON.stringify([collection, breadcrumb]);
    document.head.appendChild(script);

    return () => {
      script.remove();
    };
  }, [posts]);

  return null;
}
