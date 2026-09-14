import { z } from "zod";

// Reusable field schemas
export const emailSchema = z
  .string()
  .min(1, "errors.required")
  .email("errors.invalidEmail");

/**
 * bcrypt hashes at most 72 bytes, so the backend rejects anything longer.
 * The limit is in BYTES, not characters: 40 Cyrillic characters are 80 bytes
 * and would be refused by a check that only counted `.length`.
 */
export const MAX_PASSWORD_BYTES = 72;

export function passwordByteLength(value: string): number {
  return new TextEncoder().encode(value).length;
}

/**
 * The published rule — "at least 8 characters" — is counted in Unicode code
 * points, matching the backend's `utf8.RuneCountInString`. `String.length`
 * counts UTF-16 units, so it sees four emoji as eight characters and would
 * wave through a password the server then rejects.
 */
export const MIN_PASSWORD_CHARS = 8;

export function passwordCharLength(value: string): number {
  return [...value].length;
}

export const passwordSchema = z
  .string()
  .min(1, "errors.required")
  .refine(
    (value) => passwordCharLength(value) >= MIN_PASSWORD_CHARS,
    "errors.passwordTooShort",
  )
  .refine(
    (value) => passwordByteLength(value) <= MAX_PASSWORD_BYTES,
    "errors.passwordTooLong",
  );

export const verificationCodeSchema = z
  .string()
  .length(6, "auth.invalidCode")
  .regex(/^\d{6}$/, "auth.invalidCode");

// Auth form schemas
export const loginSchema = z.object({
  email: emailSchema,
  password: passwordSchema,
});
export type LoginFormData = z.infer<typeof loginSchema>;

export const registerSchema = z
  .object({
    email: emailSchema,
    password: passwordSchema,
    confirmPassword: z.string().min(1, "errors.required"),
  })
  .refine((data) => data.password === data.confirmPassword, {
    message: "errors.passwordsDontMatch",
    path: ["confirmPassword"],
  });
export type RegisterFormData = z.infer<typeof registerSchema>;

export const resetPasswordSchema = z
  .object({
    password: passwordSchema,
    confirmPassword: z.string().min(1, "errors.required"),
  })
  .refine((data) => data.password === data.confirmPassword, {
    message: "errors.passwordsDontMatch",
    path: ["confirmPassword"],
  });
export type ResetPasswordFormData = z.infer<typeof resetPasswordSchema>;

export const forgotPasswordSchema = z.object({
  email: emailSchema,
});
export type ForgotPasswordFormData = z.infer<typeof forgotPasswordSchema>;

// CRUD form schemas
export const createJobSchema = z.object({
  title: z.string().min(1, "errors.required").max(200),
  companyId: z.string().optional(),
  url: z.string().url().optional().or(z.literal("")),
  source: z.string().max(100).optional(),
  description: z.string().max(10000).optional(),
});
export type CreateJobFormData = z.infer<typeof createJobSchema>;

export const createCompanySchema = z.object({
  name: z.string().min(1, "errors.required").max(200),
  location: z.string().max(200).optional(),
  notes: z.string().max(5000).optional(),
});
export type CreateCompanyFormData = z.infer<typeof createCompanySchema>;

export const stageTemplateSchema = z.object({
  name: z.string().min(1, "errors.required").max(100),
  order: z.coerce.number().int().min(0),
});
export type StageTemplateFormData = z.infer<typeof stageTemplateSchema>;
