import { z } from "zod";

import { BLOG_LANGS } from "./existingPosts.mjs";

/** The only host generated posts may link to; anything else is likely invented. */
export const SITE_HOST = "cursopus.com";

const SLUG_PATTERN = /^[a-z0-9]+(?:-[a-z0-9]+)*$/;
const MAX_SLUG_LENGTH = 80;
const MAX_TITLE_LENGTH = 110;
const DESCRIPTION_LENGTH = { min: 70, max: 200 };
const TAG_COUNT = { min: 2, max: 6 };
const MIN_WORDS = 450;

// Letters that only one of the two Cyrillic languages uses. A post whose body
// lacks its own language's marker letters was most likely written in the other
// language (or left in English).
const UKRAINIAN_LETTERS = /[іїєґІЇЄҐ]/;
const RUSSIAN_LETTERS = /[ыэъЫЭЪ]/;
const CYRILLIC_LETTERS = /[а-яА-ЯёЁіїєґІЇЄҐ]/g;

const LocalizedPostSchema = z.strictObject({
  title: z.string(),
  slug: z.string(),
  description: z.string(),
  tags: z.array(z.string()),
  content: z.string(),
});

/** What the model returns: one article, written natively in every language. */
export const GeneratedArticleSchema = z.strictObject({
  translationKey: z.string(),
  en: LocalizedPostSchema,
  ua: LocalizedPostSchema,
  ru: LocalizedPostSchema,
});

function countWords(text) {
  return text.split(/\s+/).filter(Boolean).length;
}

// The blog's frontmatter parser strips surrounding quotes but never unescapes,
// so a value holding a newline, a backslash or a straight double quote cannot
// be written into it faithfully.
function frontmatterSafetyErrors(field, value) {
  const errors = [];
  if (/[\r\n]/.test(value)) errors.push(`${field} must be a single line`);
  if (/["\\]/.test(value)) {
    errors.push(
      `${field} must not contain straight double quotes or backslashes (use «», “” or ’ instead)`,
    );
  }
  return errors;
}

function slugErrors(field, slug) {
  if (!SLUG_PATTERN.test(slug)) {
    return [`${field} "${slug}" must be lowercase latin words joined by hyphens`];
  }
  if (slug.length > MAX_SLUG_LENGTH) {
    return [`${field} "${slug}" is longer than ${MAX_SLUG_LENGTH} characters`];
  }
  return [];
}

function foreignLinkErrors(lang, content) {
  const hosts = [...content.matchAll(/https?:\/\/([^/\s)"'>]+)/g)].map(
    ([, host]) => host.toLowerCase(),
  );
  return hosts
    .filter((host) => host !== SITE_HOST && host !== `www.${SITE_HOST}`)
    .map((host) => `${lang}.content links to ${host}; only ${SITE_HOST} links are allowed`);
}

function languageErrors(lang, content) {
  const cyrillicCount = (content.match(CYRILLIC_LETTERS) ?? []).length;
  const isMostlyCyrillic = cyrillicCount > countWords(content);

  if (lang === "en") {
    return isMostlyCyrillic ? ["en.content is not written in English"] : [];
  }
  if (!isMostlyCyrillic) return [`${lang}.content is not written in Cyrillic`];
  if (lang === "ua" && !UKRAINIAN_LETTERS.test(content)) {
    return ["ua.content does not read as Ukrainian (no і/ї/є/ґ)"];
  }
  if (lang === "ru" && !RUSSIAN_LETTERS.test(content)) {
    return ["ru.content does not read as Russian (no ы/э/ъ)"];
  }
  return [];
}

function localizedPostErrors(lang, post) {
  const descriptionLength = post.description.trim().length;
  const words = countWords(post.content);

  return [
    ...(post.title.trim() ? [] : [`${lang}.title is empty`]),
    ...(post.title.length > MAX_TITLE_LENGTH
      ? [`${lang}.title is longer than ${MAX_TITLE_LENGTH} characters`]
      : []),
    ...frontmatterSafetyErrors(`${lang}.title`, post.title),
    ...slugErrors(`${lang}.slug`, post.slug),
    ...(descriptionLength < DESCRIPTION_LENGTH.min ||
    descriptionLength > DESCRIPTION_LENGTH.max
      ? [
          `${lang}.description is ${descriptionLength} characters; expected ${DESCRIPTION_LENGTH.min}-${DESCRIPTION_LENGTH.max}`,
        ]
      : []),
    ...frontmatterSafetyErrors(`${lang}.description`, post.description),
    ...(post.tags.length < TAG_COUNT.min || post.tags.length > TAG_COUNT.max
      ? [`${lang}.tags has ${post.tags.length} tags; expected ${TAG_COUNT.min}-${TAG_COUNT.max}`]
      : []),
    ...post.tags.flatMap((tag, index) => [
      ...(tag.trim() ? [] : [`${lang}.tags[${index}] is empty`]),
      ...frontmatterSafetyErrors(`${lang}.tags[${index}]`, tag),
    ]),
    ...(post.content.trimStart().startsWith("---")
      ? [`${lang}.content must not contain its own frontmatter`]
      : []),
    ...(post.content.trimStart().startsWith("# ")
      ? []
      : [`${lang}.content must start with a "# " heading`]),
    ...(words < MIN_WORDS
      ? [`${lang}.content has ${words} words; expected at least ${MIN_WORDS}`]
      : []),
    ...foreignLinkErrors(lang, post.content),
    ...languageErrors(lang, post.content),
  ];
}

/**
 * Everything wrong with a generated article, as messages the model can act on.
 * An empty list means it is safe to publish next to `existingPosts`.
 */
export function validateArticle(article, existingPosts) {
  const takenSlugs = new Set(existingPosts.map((post) => post.slug));
  const takenKeys = new Set(
    existingPosts.map((post) => post.translationKey).filter(Boolean),
  );
  const newSlugs = BLOG_LANGS.map((lang) => article[lang].slug);

  return [
    ...slugErrors("translationKey", article.translationKey),
    ...(takenKeys.has(article.translationKey)
      ? [`translationKey "${article.translationKey}" is already used by another article`]
      : []),
    ...newSlugs
      .filter((slug) => takenSlugs.has(slug))
      .map((slug) => `slug "${slug}" is already used by a published post`),
    ...(new Set(newSlugs).size === newSlugs.length
      ? []
      : ["each language needs its own slug (the three slugs must differ)"]),
    ...BLOG_LANGS.flatMap((lang) => localizedPostErrors(lang, article[lang])),
  ];
}
