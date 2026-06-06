import { z } from "zod";

/**
 * Public env vars consumed by the dashboard. Validated at module load so a
 * misconfigured deployment fails loudly rather than producing odd 4xx errors.
 *
 * Only NEXT_PUBLIC_* values may be referenced from the browser bundle.
 */
const schema = z.object({
  NEXT_PUBLIC_API_URL: z
    .string()
    .url()
    .default("http://api.nebula.localhost"),
  NEXT_PUBLIC_APP_URL: z
    .string()
    .url()
    .default("http://app.nebula.localhost"),
  NEXT_PUBLIC_GITHUB_URL: z
    .string()
    .url()
    .default("https://github.com/Felpis3196/nebulaCloud"),
  NEXT_PUBLIC_GITHUB_BRANCH: z.string().min(1).default("master"),
});

const raw = {
  NEXT_PUBLIC_API_URL: process.env.NEXT_PUBLIC_API_URL,
  NEXT_PUBLIC_APP_URL: process.env.NEXT_PUBLIC_APP_URL,
  NEXT_PUBLIC_GITHUB_URL: process.env.NEXT_PUBLIC_GITHUB_URL,
  NEXT_PUBLIC_GITHUB_BRANCH: process.env.NEXT_PUBLIC_GITHUB_BRANCH,
};

export const env = schema.parse(raw);
export type Env = z.infer<typeof schema>;
