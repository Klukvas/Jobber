import path from "node:path";
import { fileURLToPath } from "node:url";

export const FE_ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
export const BLOG_DIR = path.join(FE_ROOT, "src/content/blog");
export const PRODUCT_FACTS_PATH = path.join(FE_ROOT, "public/llms.txt");

// Posts are dated in the readers' time zone, not the runner's.
const PUBLICATION_TIME_ZONE = "Europe/Kyiv";

export function publicationDateToday() {
  // en-CA formats dates as YYYY-MM-DD.
  return new Intl.DateTimeFormat("en-CA", {
    timeZone: PUBLICATION_TIME_ZONE,
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).format(new Date());
}
