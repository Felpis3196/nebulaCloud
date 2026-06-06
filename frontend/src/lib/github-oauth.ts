import { env } from "@/lib/env";

export type GitHubRepoOption = {
  full_name: string;
  html_url: string;
  clone_url: string;
  default_branch: string;
  private: boolean;
};

export const GITHUB_REPOS_SESSION_KEY = "nebula_github_repos";
export const GITHUB_RETURN_PROJECT_KEY = "nebula_github_return_project";

/** Builds the API OAuth start URL; `returnTo` must be same-origin as the dashboard. */
export function githubOAuthStartUrl(returnTo: string): string {
  const u = new URL(`${env.NEXT_PUBLIC_API_URL}/api/v1/auth/github`);
  u.searchParams.set("return_to", returnTo);
  return u.toString();
}

export function readGitHubReposFromSession(): GitHubRepoOption[] {
  if (typeof window === "undefined") return [];
  try {
    const raw = sessionStorage.getItem(GITHUB_REPOS_SESSION_KEY);
    if (!raw) return [];
    const parsed = JSON.parse(raw) as GitHubRepoOption[];
    return Array.isArray(parsed) ? parsed : [];
  } catch {
    return [];
  }
}

export function storeGitHubReposInSession(repos: GitHubRepoOption[]): void {
  sessionStorage.setItem(GITHUB_REPOS_SESSION_KEY, JSON.stringify(repos));
  sessionStorage.removeItem(GITHUB_RETURN_PROJECT_KEY);
}

export function clearGitHubReposSession(): void {
  sessionStorage.removeItem(GITHUB_REPOS_SESSION_KEY);
  sessionStorage.removeItem(GITHUB_RETURN_PROJECT_KEY);
}
