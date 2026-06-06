"use client";

import { Suspense, useEffect, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { Loader2 } from "lucide-react";
import { useTranslations } from "next-intl";
import { toast } from "sonner";

import { apiWithCredentials, ApiError } from "@/lib/api-client";
import {
  GITHUB_RETURN_PROJECT_KEY,
  type GitHubRepoOption,
  storeGitHubReposInSession,
} from "@/lib/github-oauth";

function GitHubCallbackInner() {
  const t = useTranslations("auth.githubCallback");
  const router = useRouter();
  const searchParams = useSearchParams();
  const [message, setMessage] = useState(t("loading"));

  useEffect(() => {
    const status = searchParams.get("github");
    const reason = searchParams.get("reason");
    const next = searchParams.get("next") ?? "/dashboard";
    const projectId = searchParams.get("projectId");

    if (status === "error") {
      toast.error(t("oauthFailed", { reason: reason ?? "unknown" }));
      router.replace(next);
      return;
    }
    if (status !== "connected") {
      router.replace(next);
      return;
    }

    let cancelled = false;
    (async () => {
      try {
        const repos = await apiWithCredentials<GitHubRepoOption[]>("/api/v1/auth/github/repos");
        if (cancelled) return;
        storeGitHubReposInSession(repos);
        if (projectId) {
          sessionStorage.setItem(GITHUB_RETURN_PROJECT_KEY, projectId);
          router.replace(`/projects/${projectId}?connect=1`);
          return;
        }
        toast.success(t("connected"));
        router.replace(next);
      } catch (err) {
        if (cancelled) return;
        const msg = err instanceof ApiError ? err.message : t("reposFailed");
        setMessage(msg);
        toast.error(msg);
        router.replace(next);
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [router, searchParams, t]);

  return (
    <div className="flex flex-col items-center gap-3 text-center text-sm text-muted-foreground">
      <Loader2 className="h-6 w-6 animate-spin text-primary" />
      <p>{message}</p>
    </div>
  );
}

export default function GitHubCallbackPage() {
  return (
    <Suspense fallback={null}>
      <GitHubCallbackInner />
    </Suspense>
  );
}
