import { z } from "zod";
import { apiClient, ApiError, type RequestOptions } from "./api";

const fields = ["response_type", "client_id", "redirect_uri", "code_challenge", "code_challenge_method", "state", "scope", "resource"] as const;
export type AuthorizationParams = Partial<Record<typeof fields[number], string>>;

export function readAuthorizationParams(search: string): AuthorizationParams {
  if (search.length > 16_384) throw new ApiError("", "INVALID_REQUEST", 0);
  const query = new URLSearchParams(search);
  if (fields.some((field) => query.getAll(field).length > 1)) {
    throw new ApiError("", "INVALID_REQUEST", 0);
  }
  return Object.fromEntries(fields.flatMap((field) => {
    const value = query.get(field);
    return value === null ? [] : [[field, value]];
  }));
}

function isCallback(value: string): boolean {
  try {
    const url = new URL(value);
    return !url.username && !url.password && !url.hash &&
      (url.protocol === "https:" || (url.protocol === "http:" &&
        ["localhost", "127.0.0.1", "[::1]"].includes(url.hostname)));
  } catch {
    return false;
  }
}
// The callback and state can each be 2048 bytes before percent-encoding.
const redirectSchema = z.object({ redirect_to: z.string().max(32_768).refine(isCallback) });
const clientNameSchema = z.string().refine((value) => [...value].length <= 100);
const previewSchema = z.object({
  client: z.object({
    client_id: z.string().min(1).max(2048),
    name: clientNameSchema,
    redirect_host: z.string().min(1).max(2048),
    is_metadata_document: z.boolean(),
  }),
  scope: z.literal("mcp"),
  resource: z.string().url(),
  plan_allowed: z.boolean(),
});
const grantSchema = z.object({
  id: z.uuid(),
  client_name: clientNameSchema,
  client_id: z.string().min(1).max(2048),
  redirect_host: z.string().min(1).max(2048),
  created_at: z.iso.datetime({ offset: true }),
  last_used_at: z.iso.datetime({ offset: true }).nullable(),
});
export type ConsentPreview = z.infer<typeof previewSchema>;
export type OAuthGrant = z.infer<typeof grantSchema>;

function parseResponse<T>(schema: z.ZodType<T>, value: unknown): T {
  const parsed = schema.safeParse(value);
  if (!parsed.success) throw new ApiError("", "INVALID_RESPONSE", 0);
  return parsed.data;
}

// Only navigate to the callback sent for this request, even if a response is malformed.
function readRedirect(value: unknown, params: AuthorizationParams) {
  const result = parseResponse(redirectSchema, value);
  if (!params.redirect_uri || !isCallback(params.redirect_uri)) throw new ApiError("", "INVALID_RESPONSE", 0);
  const expected = new URL(params.redirect_uri);
  const actual = new URL(result.redirect_to);
  if (expected.origin !== actual.origin || expected.pathname !== actual.pathname) {
    throw new ApiError("", "INVALID_RESPONSE", 0);
  }
  return result;
}

export const oauthService = {
  async preview(search: string, options?: RequestOptions): Promise<ConsentPreview | { redirect_to: string }> {
    const params = readAuthorizationParams(search);
    const value = await apiClient.get<unknown>(`oauth/authorize/preview${search}`, options);
    if (typeof value === "object" && value !== null && "redirect_to" in value) return readRedirect(value, params);
    return parseResponse(previewSchema, value);
  },
  async authorize(params: AuthorizationParams, decision: "allow" | "deny", options?: RequestOptions) {
    return readRedirect(await apiClient.post<unknown>("oauth/authorize", { ...params, decision }, options), params);
  },
  async grants(options?: RequestOptions): Promise<OAuthGrant[]> {
    const rows = parseResponse(z.array(z.unknown()), await apiClient.get<unknown>("oauth/grants", options));
    // One malformed app must not remove the other apps' disconnect controls.
    return rows.flatMap((row) => {
      const parsed = grantSchema.safeParse(row);
      return parsed.success ? [parsed.data] : [];
    });
  },
  async revoke(id: string): Promise<void> {
    const parsed = z.uuid().safeParse(id);
    if (!parsed.success) throw new ApiError("", "INVALID_REQUEST", 0);
    await apiClient.delete<void>(`oauth/grants/${encodeURIComponent(parsed.data)}`);
  },
};
