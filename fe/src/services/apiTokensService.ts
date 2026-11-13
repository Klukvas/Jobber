import { z } from "zod";
import { apiClient, ApiError, type RequestOptions } from "./api";

const timestamp = z.iso.datetime({ offset: true });
const tokenSchema = z.object({
  id: z.string(),
  name: z.string(),
  token_prefix: z.string(),
  scopes: z.array(z.string()),
  last_used_at: timestamp.nullable(),
  expires_at: timestamp.nullable(),
  revoked_at: timestamp.nullable(),
  created_at: timestamp,
});
const activitySchema = z.object({
  id: z.string(),
  tool: z.string(),
  ok: z.boolean(),
  error_code: z.string().nullable(),
  duration_ms: z.number().nonnegative(),
  created_at: timestamp,
  api_token_id: z.string().nullable(),
  token_name: z.string().nullable(),
});

export type ApiToken = z.infer<typeof tokenSchema>;
export type AssistantActivity = z.infer<typeof activitySchema>;
export type CreateApiToken = { name: string; expires_in_days?: 30 | 90 | 365 | null };

// Never include Zod issues in an error: a create response contains a secret.
function parseResponse<T>(schema: z.ZodType<T>, value: unknown): T {
  const parsed = schema.safeParse(value);
  if (!parsed.success) throw new ApiError("", "INVALID_RESPONSE", 0);
  return parsed.data;
}

export const apiTokensService = {
  async list(options?: RequestOptions): Promise<ApiToken[]> {
    return parseResponse(z.array(tokenSchema), await apiClient.get<unknown>("api-tokens", options));
  },
  async create(input: CreateApiToken, options?: RequestOptions): Promise<ApiToken & { token: string }> {
    return parseResponse(tokenSchema.extend({ token: z.string().min(1) }), await apiClient.post<unknown>("api-tokens", input, options));
  },
  async revoke(id: string): Promise<void> {
    await apiClient.delete<void>(`api-tokens/${encodeURIComponent(id)}`);
  },
  async activity(options?: RequestOptions): Promise<AssistantActivity[]> {
    return parseResponse(z.array(activitySchema), await apiClient.get<unknown>("api-tokens/activity?limit=20", options));
  },
};
