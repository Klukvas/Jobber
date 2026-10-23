import { z } from "zod";
import { apiClient, type RequestOptions } from "./api";
import {
  contactFieldsSchema,
  normalizeContact,
  type ContactFields,
} from "@/features/jobs/contacts/validation";
const contactSchema = z.object({
  id: z.uuid(),
  job_id: z.uuid(),
  name: z.string(),
  role: z.string(),
  email: z.string(),
  telegram: z.string(),
  linkedin: z.string(),
  note: z.string(),
  created_at: z.string(),
  updated_at: z.string(),
});
export type Contact = z.infer<typeof contactSchema>;
const path = (job: string) => `jobs/${encodeURIComponent(job)}/contacts`;
export const contactsService = {
  async list(job: string, options?: RequestOptions): Promise<Contact[]> {
    return contactSchema
      .array()
      .parse(await apiClient.get<unknown>(path(job), options));
  },
  async save(
    job: string,
    id: string | null,
    fields: ContactFields,
  ): Promise<Contact> {
    const input = normalizeContact(contactFieldsSchema.parse(fields));
    return contactSchema.parse(
      id
        ? await apiClient.patch<unknown>(
            `${path(job)}/${encodeURIComponent(id)}`,
            input,
          )
        : await apiClient.post<unknown>(path(job), input),
    );
  },
  async delete(job: string, id: string): Promise<void> {
    await apiClient.delete<void>(`${path(job)}/${encodeURIComponent(id)}`);
  },
};
