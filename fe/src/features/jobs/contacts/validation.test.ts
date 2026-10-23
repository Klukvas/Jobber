import { describe, expect, it } from "vitest";
import {
  channelLink,
  contactErrors,
  EMPTY_CONTACT,
  emailLink,
  normalizeContact,
} from "./validation";

describe("contact validation", () => {
  it("normalizes safe known channel shapes", () => {
    expect(
      normalizeContact({
        ...EMPTY_CONTACT,
        name: " Ada ",
        telegram: "@ada_hr",
        linkedin: "http://linkedin.com/in/ada/",
      }),
    ).toMatchObject({
      name: "Ada",
      telegram: "https://t.me/ada_hr",
      linkedin: "https://www.linkedin.com/in/ada",
    });
    expect(channelLink("telegram", "t.me/ada_hr/")).toBe("https://t.me/ada_hr");
    expect(channelLink("telegram", "https://t.me/ada_hr")).toBe(
      "https://t.me/ada_hr",
    );
  });
  it("matches server whitespace and scalar Unicode normalization", () => {
    expect(
      normalizeContact({ ...EMPTY_CONTACT, name: "\ufeffAda\ufeff" }).name,
    ).toBe("Ada");
    expect(
      contactErrors({ ...EMPTY_CONTACT, name: "\u0085Ada" }),
    ).toHaveProperty("name");
    expect(normalizeContact({ ...EMPTY_CONTACT, name: "\ud800" }).name).toBe(
      "\ufffd",
    );
    expect(contactErrors({ ...EMPTY_CONTACT, name: "\ud800" })).toEqual({});
  });
  it("counts Unicode code points consistently with the server", () => {
    expect(
      contactErrors({
        ...EMPTY_CONTACT,
        name: "😀".repeat(120),
        note: "😀".repeat(2000),
      }),
    ).toEqual({});
    expect(
      contactErrors({ ...EMPTY_CONTACT, name: "😀".repeat(121) }),
    ).toHaveProperty("name");
    expect(
      contactErrors({ ...EMPTY_CONTACT, name: "Ada", note: "x".repeat(2001) }),
    ).toHaveProperty("note");
  });
  it("rejects missing names, controls and malformed emails", () => {
    expect(contactErrors(EMPTY_CONTACT)).toEqual({ name: "required" });
    expect(
      contactErrors({
        ...EMPTY_CONTACT,
        name: "N\u0000",
        email: "a@example.com\r\nbcc:x@example.com",
      }),
    ).toMatchObject({ name: "invalid", email: "invalid" });
  });
  it.each([
    "javascript:alert(1)",
    "https://t.me.evil/abcde",
    "https://evil@t.me/abcde",
    "https://t.me:443/abcde",
    "t.me/abcde?x=1",
    "t.me/%61bcde",
    "t.me/@abcde",
    "@abcde/",
    "abcd",
    "12345",
  ])("rejects unsafe Telegram %s", (value) =>
    expect(channelLink("telegram", value)).toBeNull(),
  );
  it.each([
    "javascript:alert(1)",
    "https://linkedin.com.evil/in/name",
    "https://evil@linkedin.com/in/name",
    "https://linkedin.com:443/in/name",
    "linkedin.com/in/a%2fb",
    "linkedin.com/in/name?x=1",
    "linkedin.com/in/name#x",
    "linkedin.com/company/name",
    "name",
    "recruiter/",
  ])("rejects unsafe LinkedIn %s", (value) =>
    expect(channelLink("linkedin", value)).toBeNull(),
  );
  it("encodes email metacharacters so they cannot inject headers or fragments", () => {
    expect(emailLink("a?subject=x@example.com")).toBe(
      "mailto:a%3Fsubject%3Dx%40example.com",
    );
    expect(emailLink("a#tag%20@example.com")).toBe(
      "mailto:a%23tag%2520%40example.com",
    );
    expect(emailLink("a@example.com\nBcc:x@example.com")).toBeNull();
  });
});
