import { describe, it, expect } from "vitest";
import { ApiError } from "../api";

describe("ApiError", () => {
  it("extends Error", () => {
    const error = new ApiError("Something went wrong", "BAD_REQUEST", 400);
    expect(error).toBeInstanceOf(Error);
  });

  it("has the correct name property", () => {
    const error = new ApiError("Not found", "NOT_FOUND", 404);
    expect(error.name).toBe("ApiError");
  });

  it("stores the message", () => {
    const error = new ApiError("Unauthorized access", "UNAUTHORIZED", 401);
    expect(error.message).toBe("Unauthorized access");
  });

  it("stores the code", () => {
    const error = new ApiError("Forbidden", "FORBIDDEN", 403);
    expect(error.code).toBe("FORBIDDEN");
  });

  it("stores the status", () => {
    const error = new ApiError("Server error", "INTERNAL_ERROR", 500);
    expect(error.status).toBe(500);
  });

  it("has all properties set correctly", () => {
    const error = new ApiError("Conflict", "CONFLICT", 409);
    expect(error).toMatchObject({
      name: "ApiError",
      message: "Conflict",
      code: "CONFLICT",
      status: 409,
    });
  });

  it("can be caught as an Error", () => {
    const error = new ApiError("Test", "TEST", 400);
    try {
      throw error;
    } catch (caught) {
      expect(caught).toBeInstanceOf(Error);
      expect(caught).toBeInstanceOf(ApiError);
    }
  });

  it("has a stack trace", () => {
    const error = new ApiError("Test", "TEST", 400);
    expect(error.stack).toBeDefined();
  });
});
