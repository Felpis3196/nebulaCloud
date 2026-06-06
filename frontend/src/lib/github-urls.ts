import { env } from "@/lib/env";

/** GitHub blob URL for a repo file (branch from NEXT_PUBLIC_GITHUB_BRANCH). */
export function githubBlob(path: string): string {
  const normalized = path.replace(/^\//, "");
  return `${env.NEXT_PUBLIC_GITHUB_URL}/blob/${env.NEXT_PUBLIC_GITHUB_BRANCH}/${normalized}`;
}
