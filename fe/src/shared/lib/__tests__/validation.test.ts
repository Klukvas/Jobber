import { describe, it, expect } from "vitest";
import {
  emailSchema,
  passwordSchema,
  verificationCodeSchema,
  loginSchema,
  registerSchema,
  resetPasswordSchema,
  forgotPasswordSchema,
  createJobSchema,
  createCompanySchema,
  stageTemplateSchema,
} from "../validation";

describe("validation schemas", () => {
  describe("emailSchema", () => {
    it("accepts a valid email", () => {
      const result = emailSchema.safeParse("user@example.com");
      expect(result.success).toBe(true);
    });

    it("rejects an empty string", () => {
      const result = emailSchema.safeParse("");
      expect(result.success).toBe(false);
    });

    it("rejects an invalid email format", () => {
      const result = emailSchema.safeParse("not-an-email");
      expect(result.success).toBe(false);
    });

    it("rejects email without domain", () => {
      const result = emailSchema.safeParse("user@");
      expect(result.success).toBe(false);
    });
  });

  describe("passwordSchema", () => {
    it("accepts a password with 8+ characters", () => {
      const result = passwordSchema.safeParse("password123");
      expect(result.success).toBe(true);
    });

    it("accepts exactly 8 characters", () => {
      const result = passwordSchema.safeParse("12345678");
      expect(result.success).toBe(true);
    });

    it("rejects an empty string", () => {
      const result = passwordSchema.safeParse("");
      expect(result.success).toBe(false);
    });

    it("rejects a password shorter than 8 characters", () => {
      const result = passwordSchema.safeParse("short");
      expect(result.success).toBe(false);
    });

    it("rejects a 7-character password", () => {
      const result = passwordSchema.safeParse("1234567");
      expect(result.success).toBe(false);
    });
  });

  describe("verificationCodeSchema", () => {
    it("accepts a valid 6-digit code", () => {
      const result = verificationCodeSchema.safeParse("123456");
      expect(result.success).toBe(true);
    });

    it("rejects a code shorter than 6 digits", () => {
      const result = verificationCodeSchema.safeParse("12345");
      expect(result.success).toBe(false);
    });

    it("rejects a code longer than 6 digits", () => {
      const result = verificationCodeSchema.safeParse("1234567");
      expect(result.success).toBe(false);
    });

    it("rejects non-digit characters", () => {
      const result = verificationCodeSchema.safeParse("abcdef");
      expect(result.success).toBe(false);
    });

    it("rejects a code with mixed digits and letters", () => {
      const result = verificationCodeSchema.safeParse("12ab56");
      expect(result.success).toBe(false);
    });
  });

  describe("loginSchema", () => {
    it("validates correct email and password", () => {
      const result = loginSchema.safeParse({
        email: "user@example.com",
        password: "password123",
      });
      expect(result.success).toBe(true);
    });

    it("rejects invalid email", () => {
      const result = loginSchema.safeParse({
        email: "invalid",
        password: "password123",
      });
      expect(result.success).toBe(false);
    });

    it("rejects short password", () => {
      const result = loginSchema.safeParse({
        email: "user@example.com",
        password: "short",
      });
      expect(result.success).toBe(false);
    });

    it("rejects missing fields", () => {
      const result = loginSchema.safeParse({});
      expect(result.success).toBe(false);
    });
  });

  describe("registerSchema", () => {
    it("validates matching passwords", () => {
      const result = registerSchema.safeParse({
        email: "user@example.com",
        password: "password123",
        confirmPassword: "password123",
      });
      expect(result.success).toBe(true);
    });

    it("rejects mismatched passwords", () => {
      const result = registerSchema.safeParse({
        email: "user@example.com",
        password: "password123",
        confirmPassword: "differentpass",
      });
      expect(result.success).toBe(false);
      if (!result.success) {
        const paths = result.error.issues.map((i) => i.path.join("."));
        expect(paths).toContain("confirmPassword");
      }
    });

    it("rejects empty confirmPassword", () => {
      const result = registerSchema.safeParse({
        email: "user@example.com",
        password: "password123",
        confirmPassword: "",
      });
      expect(result.success).toBe(false);
    });

    it("rejects invalid email in register form", () => {
      const result = registerSchema.safeParse({
        email: "bad",
        password: "password123",
        confirmPassword: "password123",
      });
      expect(result.success).toBe(false);
    });
  });

  describe("resetPasswordSchema", () => {
    it("validates matching passwords", () => {
      const result = resetPasswordSchema.safeParse({
        password: "newpassword1",
        confirmPassword: "newpassword1",
      });
      expect(result.success).toBe(true);
    });

    it("rejects mismatched passwords", () => {
      const result = resetPasswordSchema.safeParse({
        password: "newpassword1",
        confirmPassword: "different123",
      });
      expect(result.success).toBe(false);
      if (!result.success) {
        const paths = result.error.issues.map((i) => i.path.join("."));
        expect(paths).toContain("confirmPassword");
      }
    });

    it("rejects short password", () => {
      const result = resetPasswordSchema.safeParse({
        password: "short",
        confirmPassword: "short",
      });
      expect(result.success).toBe(false);
    });
  });

  describe("forgotPasswordSchema", () => {
    it("validates a correct email", () => {
      const result = forgotPasswordSchema.safeParse({
        email: "user@example.com",
      });
      expect(result.success).toBe(true);
    });

    it("rejects an invalid email", () => {
      const result = forgotPasswordSchema.safeParse({
        email: "not-email",
      });
      expect(result.success).toBe(false);
    });
  });

  describe("createJobSchema", () => {
    it("validates with required title only", () => {
      const result = createJobSchema.safeParse({
        title: "Software Engineer",
      });
      expect(result.success).toBe(true);
    });

    it("validates with all optional fields", () => {
      const result = createJobSchema.safeParse({
        title: "Software Engineer",
        companyId: "company-1",
        url: "https://example.com/job",
        source: "LinkedIn",
        notes: "Great opportunity",
        description: "Full-stack role",
      });
      expect(result.success).toBe(true);
    });

    it("rejects empty title", () => {
      const result = createJobSchema.safeParse({
        title: "",
      });
      expect(result.success).toBe(false);
    });

    it("accepts empty string for url", () => {
      const result = createJobSchema.safeParse({
        title: "Job",
        url: "",
      });
      expect(result.success).toBe(true);
    });

    it("accepts a valid url", () => {
      const result = createJobSchema.safeParse({
        title: "Job",
        url: "https://example.com",
      });
      expect(result.success).toBe(true);
    });

    it("rejects an invalid url", () => {
      const result = createJobSchema.safeParse({
        title: "Job",
        url: "not-a-url",
      });
      expect(result.success).toBe(false);
    });

    it("rejects title exceeding 200 characters", () => {
      const result = createJobSchema.safeParse({
        title: "x".repeat(201),
      });
      expect(result.success).toBe(false);
    });
  });

  describe("createCompanySchema", () => {
    it("validates with required name", () => {
      const result = createCompanySchema.safeParse({
        name: "Acme Corp",
      });
      expect(result.success).toBe(true);
    });

    it("validates with all optional fields", () => {
      const result = createCompanySchema.safeParse({
        name: "Acme Corp",
        location: "New York",
        notes: "Good company",
      });
      expect(result.success).toBe(true);
    });

    it("rejects empty name", () => {
      const result = createCompanySchema.safeParse({
        name: "",
      });
      expect(result.success).toBe(false);
    });

    it("rejects name exceeding 200 characters", () => {
      const result = createCompanySchema.safeParse({
        name: "x".repeat(201),
      });
      expect(result.success).toBe(false);
    });
  });

  describe("stageTemplateSchema", () => {
    it("validates with name and order", () => {
      const result = stageTemplateSchema.safeParse({
        name: "Phone Screen",
        order: 1,
      });
      expect(result.success).toBe(true);
    });

    it("accepts order of 0", () => {
      const result = stageTemplateSchema.safeParse({
        name: "Applied",
        order: 0,
      });
      expect(result.success).toBe(true);
    });

    it("coerces string order to number", () => {
      const result = stageTemplateSchema.safeParse({
        name: "Interview",
        order: "3",
      });
      expect(result.success).toBe(true);
      if (result.success) {
        expect(result.data.order).toBe(3);
      }
    });

    it("rejects empty name", () => {
      const result = stageTemplateSchema.safeParse({
        name: "",
        order: 1,
      });
      expect(result.success).toBe(false);
    });

    it("rejects negative order", () => {
      const result = stageTemplateSchema.safeParse({
        name: "Stage",
        order: -1,
      });
      expect(result.success).toBe(false);
    });

    it("rejects non-integer order", () => {
      const result = stageTemplateSchema.safeParse({
        name: "Stage",
        order: 1.5,
      });
      expect(result.success).toBe(false);
    });

    it("rejects name exceeding 100 characters", () => {
      const result = stageTemplateSchema.safeParse({
        name: "x".repeat(101),
        order: 1,
      });
      expect(result.success).toBe(false);
    });
  });
});
