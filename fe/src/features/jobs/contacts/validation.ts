import { z } from "zod";

export const CONTACT_LIMITS = {
  name: 120,
  role: 120,
  email: 254,
  telegram: 512,
  linkedin: 512,
  note: 2000,
} as const;
export type ContactFields = Record<keyof typeof CONTACT_LIMITS, string>;
export const EMPTY_CONTACT: ContactFields = {
  name: "",
  role: "",
  email: "",
  telegram: "",
  linkedin: "",
  note: "",
};
const emailPattern =
  /^[A-Za-z0-9.!#$%&'*+/=?^_`{|}~-]+@[A-Za-z0-9](?:[A-Za-z0-9-]*[A-Za-z0-9])?(?:\.[A-Za-z0-9](?:[A-Za-z0-9-]*[A-Za-z0-9])?)+$/;

export function channelLink(
  kind: "telegram" | "linkedin",
  input: string,
): string | null {
  if (!input) return "";
  if (kind === "telegram") {
    const isUrl = /^(?:https?:\/\/)?t\.me\//.test(input);
    const handle = isUrl
      ? input.replace(/^(?:https?:\/\/)?t\.me\//, "").replace(/\/$/, "")
      : input.replace(/^@/, "");
    return /^[A-Za-z][A-Za-z0-9_]{4,31}$/.test(handle)
      ? `https://t.me/${handle}`
      : null;
  }
  const prefix = /^(?:https?:\/\/)?(?:www\.)?linkedin\.com\/in\//;
  if (!prefix.test(input)) return null;
  const slug = input.replace(prefix, "").replace(/\/$/, "");
  if (slug === input || !/^[A-Za-z0-9_-]{1,100}$/.test(slug)) return null;
  return `https://www.linkedin.com/in/${slug}`;
}
// Match encoding/json's scalar text decoding for a lone UTF-16 surrogate.
// Array.from preserves valid surrogate pairs as one character.
function contactText(value: string): string {
  return Array.from(value, (char) => {
    const code = char.codePointAt(0) ?? 0;
    return code >= 0xd800 && code <= 0xdfff ? "\ufffd" : char;
  })
    .join("")
    .trim();
}

export function contactErrors(
  fields: ContactFields,
): Partial<Record<keyof ContactFields, string>> {
  return Object.fromEntries(
    (Object.keys(CONTACT_LIMITS) as (keyof ContactFields)[]).flatMap((key) => {
      const value = contactText(fields[key]);
      const hasControls = Array.from(value).some((char) => {
        const code = char.codePointAt(0) ?? 0;
        return (
          (code < 32 || (code >= 127 && code <= 159)) &&
          !(key === "note" && [9, 10, 13].includes(code))
        );
      });
      const invalid =
        Array.from(value).length > CONTACT_LIMITS[key] ||
        hasControls ||
        (key === "name" && !value) ||
        (key === "email" && value && !emailPattern.test(value)) ||
        ((key === "telegram" || key === "linkedin") &&
          channelLink(key, value) === null);
      return invalid
        ? [[key, key === "name" && !value ? "required" : "invalid"]]
        : [];
    }),
  );
}
export const contactFieldsSchema = z
  .object({
    name: z.string(),
    role: z.string(),
    email: z.string(),
    telegram: z.string(),
    linkedin: z.string(),
    note: z.string(),
  })
  .refine((value) => Object.keys(contactErrors(value)).length === 0);
export function normalizeContact(fields: ContactFields): ContactFields {
  return {
    name: contactText(fields.name),
    role: contactText(fields.role),
    email: contactText(fields.email),
    telegram: channelLink("telegram", contactText(fields.telegram)) ?? "",
    linkedin: channelLink("linkedin", contactText(fields.linkedin)) ?? "",
    note: contactText(fields.note),
  };
}
export function emailLink(email: string): string | null {
  return email &&
    email.length <= CONTACT_LIMITS.email &&
    emailPattern.test(email)
    ? `mailto:${encodeURIComponent(email)}`
    : null;
}
