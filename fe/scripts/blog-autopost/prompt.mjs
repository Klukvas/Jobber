import { SITE_HOST } from "./articleSchema.mjs";

/**
 * The product's public name: the "# " heading of llms.txt, so a rebrand on the
 * site reaches the posts without touching this code.
 */
export function productNameFrom(productFacts) {
  const name = productFacts.match(/^#\s+(.+?)\s*$/m)?.[1];
  if (!name) throw new Error('llms.txt must start with a "# <product name>" heading');
  return name;
}

/**
 * The editorial brief the scheduled Claude routine reads before writing.
 * Product facts come from `public/llms.txt` so the posts describe the product
 * the way the site itself does, and follow it as it changes.
 */
export function buildEditorialBrief(productFacts) {
  const productName = productNameFrom(productFacts);
  return `You write articles for the blog of ${productName} (https://${SITE_HOST}), a personal job application tracker with AI assistance. Readers are job seekers, many of them in IT and in Ukraine or other Russian- and Ukrainian-speaking markets, others applying internationally.

What the product is and does (this is the whole truth about it; never claim features, prices or limits beyond it):
<product_facts>
${productFacts.trim()}
</product_facts>

Each article you write must:
- Teach something concrete and useful about job searching: practical steps, examples, templates, checklists or numbers a reader can act on today. Advice comes first; it is not an advert.
- Mention ${productName} at most a couple of times where it genuinely helps, and end with one short italic call to action that links to a relevant ${SITE_HOST} page from the product facts.
- Be 900-1400 words per language, in markdown: start with a single "# " heading equal to the title, use "##" sections, lists and an occasional table where they help scanning.
- Be written natively in each language, not translated word for word: English (en), Ukrainian (ua) and Russian (ru). Adapt examples to the reader where it matters (for example Djinni and DOU for the Ukrainian IT market, LinkedIn for international roles).
- Never invent statistics, studies, quotes, people or company policies. If you give a figure, make it a clearly labelled rule of thumb.
- Contain no links other than ${SITE_HOST} pages, and no raw HTML.

Metadata rules:
- title: at most 110 characters; description: 70-200 characters, a search-snippet summary. Neither may contain straight double quotes or backslashes; use «» in Ukrainian and Russian and “” or ’ in English.
- slug: lowercase latin words joined by hyphens, unique per language. English slugs are short English phrases; Ukrainian and Russian slugs are transliterations of their own titles (for example "yak-napysaty-suprovodniy-lyst", "kak-napisat-soprovoditelnoe-pismo").
- translationKey: a short lowercase hyphenated id for the article, shared by all three languages.
- tags: 3-5 short tags in the post's own language.`;
}

function describeExistingArticles(existingPosts) {
  // Map.groupBy needs Node 21; CI runs Node 20.
  const groups = existingPosts.reduce((acc, post) => {
    const key = post.translationKey ?? `${post.lang}:${post.slug}`;
    return new Map(acc).set(key, [...(acc.get(key) ?? []), post]);
  }, new Map());
  return [...groups.entries()]
    .map(([key, posts]) => {
      const titles = posts.map((post) => `[${post.lang}] ${post.title}`).join(" / ");
      const tags = [...new Set(posts.flatMap((post) => post.tags))].join(", ");
      return `- ${key} (${posts[0].date}): ${titles} — tags: ${tags}`;
    })
    .join("\n");
}

/** The request for one new article, given everything already published. */
export function buildTopicRequest({ existingPosts, date }) {
  return `Today is ${date}. Write the next article for the blog.

Articles already published (translationKey, date, titles, tags):
${describeExistingArticles(existingPosts)}

Choose a topic that none of these articles covers yet, that a job seeker would actually search for, and that you can write about accurately. Prefer topics that widen the blog (stages of the search, specific situations, negotiation, remote work, career changes, the Ukrainian IT market) over a new angle on a topic already covered. Do not reuse any slug or translationKey listed above.`;
}

/** The file publish.mjs expects, so the article can be validated before it is written. */
export function describeArticleFile() {
  return `Save the article as a JSON file of exactly this shape (no other keys):
{
  "translationKey": "...",
  "en": { "title": "...", "slug": "...", "description": "...", "tags": ["..."], "content": "# Title\\n\\n..." },
  "ua": { ...same fields... },
  "ru": { ...same fields... }
}
"content" is the full markdown body starting with the "# " heading, without frontmatter.`;
}
