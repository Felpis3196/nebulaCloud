"use client";

import { useCallback, useEffect, useState } from "react";
import { AlertTriangle, Github, Loader2 } from "lucide-react";
import { useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { api, ApiError } from "@/lib/api-client";
import { env } from "@/lib/env";
import {
  clearGitHubReposSession,
  githubOAuthStartUrl,
  readGitHubReposFromSession,
  type GitHubRepoOption,
} from "@/lib/github-oauth";
import type { Project, RepoAnalysis } from "@/types/api";
import { useTranslations } from "next-intl";

const webhookURL = `${env.NEXT_PUBLIC_API_URL}/api/v1/webhooks/github`;

interface Props {
  /** Required to PATCH repo_url / default_branch on the project. */
  projectId?: string;
  trigger?: React.ReactNode;
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
}

export function ConnectRepoDialog({ projectId, trigger, open: controlledOpen, onOpenChange }: Props) {
  const t = useTranslations("projects.connectRepo");
  const tCommon = useTranslations("common");
  const qc = useQueryClient();
  const [internalOpen, setInternalOpen] = useState(false);
  const isControlled = controlledOpen !== undefined;
  const open = isControlled ? controlledOpen : internalOpen;
  const setOpen = (next: boolean) => {
    if (isControlled) onOpenChange?.(next);
    else setInternalOpen(next);
  };

  const [submitting, setSubmitting] = useState(false);
  const [repo, setRepo] = useState("");
  const [branch, setBranch] = useState("main");
  const [installationId, setInstallationId] = useState("");
  const [githubRepos, setGithubRepos] = useState<GitHubRepoOption[]>([]);
  const [selectedRepo, setSelectedRepo] = useState("");
  const [analyzing, setAnalyzing] = useState(false);
  const [analysis, setAnalysis] = useState<RepoAnalysis | null>(null);

  const runAnalysis = useCallback(async () => {
    if (!projectId || !repo.trim()) {
      setAnalysis(null);
      return;
    }
    setAnalyzing(true);
    try {
      const result = await api<RepoAnalysis>(`/api/v1/projects/${projectId}/analyze-repo`, {
        method: "POST",
        body: {
          repo_url: repo.trim(),
          default_branch: branch.trim() || "main",
        },
      });
      setAnalysis(result);
    } catch (err) {
      setAnalysis(null);
      toast.error(err instanceof ApiError ? err.message : t("analysisFailed"));
    } finally {
      setAnalyzing(false);
    }
  }, [projectId, repo, branch, t]);

  useEffect(() => {
    if (!open) return;
    const stored = readGitHubReposFromSession();
    setGithubRepos(stored);
    if (stored.length === 1) {
      applyRepo(stored[0]!);
      setSelectedRepo(stored[0]!.html_url);
    }
  }, [open]);

  function applyRepo(r: GitHubRepoOption) {
    setRepo(r.html_url);
    setBranch(r.default_branch || "main");
  }

  function resetForm() {
    setRepo("");
    setBranch("main");
    setInstallationId("");
    setSelectedRepo("");
    setAnalysis(null);
    setGithubRepos(readGitHubReposFromSession());
  }

  function browseGitHub() {
    if (!projectId) {
      toast.error(t("needProject"));
      return;
    }
    const returnTo = `${window.location.origin}/auth/github/callback?projectId=${encodeURIComponent(projectId)}`;
    window.location.href = githubOAuthStartUrl(returnTo);
  }

  async function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (!projectId) {
      toast.error(t("needProject"));
      return;
    }
    setSubmitting(true);
    try {
      const trimmedInst = installationId.trim();
      if (trimmedInst !== "") {
        const n = Number.parseInt(trimmedInst, 10);
        if (!Number.isFinite(n) || n <= 0) {
          toast.error(t("invalidInstallation"));
          return;
        }
      }
      await api<Project>(`/api/v1/projects/${projectId}`, {
        method: "PATCH",
        body: {
          repo_url: repo.trim(),
          default_branch: branch.trim() || "main",
        },
      });
      if (trimmedInst !== "") {
        const n = Number.parseInt(trimmedInst, 10);
        await api<Project>(`/api/v1/projects/${projectId}/github-installation`, {
          method: "POST",
          body: { installation_id: n },
        });
      }
      await qc.invalidateQueries({ queryKey: ["project", projectId] });
      await qc.invalidateQueries({ queryKey: ["projects"] });
      clearGitHubReposSession();
      setOpen(false);
      const hasWarnings = (analysis?.warnings?.length ?? 0) > 0;
      toast.success(trimmedInst ? t("savedWithInstall") : t("savedRepoOnly"));
      if (hasWarnings) {
        toast.warning(t("savedWithWarnings"), { duration: 10000 });
      }
      toast.info(t("nextStepAddService"), { duration: 8000 });
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : t("saveFailed"));
    } finally {
      setSubmitting(false);
    }
  }

  const dialogBody = (
    <>
      <DialogHeader>
        <DialogTitle>{t("dialogTitle")}</DialogTitle>
        <DialogDescription>{t("dialogDescription")}</DialogDescription>
      </DialogHeader>
      <form onSubmit={handleSubmit} className="space-y-4">
        <div className="flex flex-wrap gap-2">
          <Button type="button" variant="secondary" size="sm" onClick={browseGitHub} disabled={!projectId}>
            <Github className="h-4 w-4" />
            {t("browseGitHub")}
          </Button>
        </div>
        {githubRepos.length > 0 && (
          <div className="space-y-2">
            <Label htmlFor="github-repo-pick">{t("pickRepo")}</Label>
            <Select
              value={selectedRepo}
              onValueChange={(v) => {
                setSelectedRepo(v);
                const found = githubRepos.find((r) => r.html_url === v);
                if (found) applyRepo(found);
              }}
            >
              <SelectTrigger id="github-repo-pick">
                <SelectValue placeholder={t("pickRepoPlaceholder")} />
              </SelectTrigger>
              <SelectContent>
                {githubRepos.map((r) => (
                  <SelectItem key={r.full_name} value={r.html_url}>
                    {r.full_name}
                    {r.private ? " · private" : ""}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        )}
        <div className="space-y-2">
          <Label htmlFor="nebula-webhook-url">{t("webhookEndpoint")}</Label>
          <Input id="nebula-webhook-url" readOnly className="font-mono text-xs" value={webhookURL} />
          <p className="text-xs text-muted-foreground">{t("webhookHint")}</p>
        </div>
        <div className="space-y-2">
          <Label htmlFor="github-installation-id">{t("installationId")}</Label>
          <Input
            id="github-installation-id"
            inputMode="numeric"
            placeholder={t("installationPlaceholder")}
            value={installationId}
            onChange={(e) => setInstallationId(e.target.value)}
          />
          <p className="text-xs text-muted-foreground">{t("installationHintLong")}</p>
        </div>
        <div className="space-y-2">
          <Label htmlFor="repo">{t("repoUrl")}</Label>
          <Input
            id="repo"
            required
            placeholder={t("repoPlaceholder")}
            value={repo}
            onChange={(e) => {
              setRepo(e.target.value);
              setAnalysis(null);
            }}
            onBlur={() => {
              if (repo.trim() && projectId) void runAnalysis();
            }}
          />
        </div>
        <div className="flex flex-wrap gap-2">
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={!projectId || !repo.trim() || analyzing}
            onClick={() => void runAnalysis()}
          >
            {analyzing ? <Loader2 className="h-4 w-4 animate-spin" /> : null}
            {analyzing ? t("analyzing") : t("analyzeRepo")}
          </Button>
        </div>
        {analysis && (analysis.warnings?.length || analysis.hints?.length) ? (
          <div className="space-y-2 rounded-md border border-amber-500/40 bg-amber-500/10 px-3 py-3 text-sm">
            <div className="flex items-center gap-2 font-medium text-amber-200">
              <AlertTriangle className="h-4 w-4 shrink-0" />
              {t("analysisWarnings")}
            </div>
            {analysis.warnings?.map((w) => (
              <p key={w} className="text-sm leading-relaxed text-amber-100/90">
                {w}
              </p>
            ))}
            {analysis.hints && analysis.hints.length > 0 && (
              <div className="space-y-1 border-t border-amber-500/20 pt-2">
                <p className="text-xs font-semibold uppercase tracking-wide text-amber-200/80">
                  {t("analysisHints")}
                </p>
                {analysis.hints.map((h) => (
                  <p key={h} className="text-xs leading-relaxed text-amber-100/80">
                    {h}
                  </p>
                ))}
              </div>
            )}
          </div>
        ) : null}
        <div className="rounded-md border border-border/60 bg-muted/20 px-3 py-2 text-xs text-muted-foreground">
          <p className="mb-1 font-medium text-foreground">{t("recommendedRepos")}</p>
          <ul className="list-inside list-disc space-y-0.5">
            <li>
              <a
                href="https://github.com/docker/welcome-to-docker"
                target="_blank"
                rel="noopener noreferrer"
                className="text-primary hover:underline"
              >
                docker/welcome-to-docker
              </a>
            </li>
            <li>
              <a
                href="https://github.com/tiangolo/uvicorn-gunicorn-fastapi-docker"
                target="_blank"
                rel="noopener noreferrer"
                className="text-primary hover:underline"
              >
                tiangolo/uvicorn-gunicorn-fastapi-docker
              </a>
            </li>
          </ul>
        </div>
        <div className="space-y-2">
          <Label htmlFor="branch">{t("defaultBranch")}</Label>
          <Input
            id="branch"
            placeholder="main"
            value={branch}
            onChange={(e) => setBranch(e.target.value)}
          />
          <p className="text-xs text-muted-foreground">{t("defaultBranchHint")}</p>
        </div>
        <DialogFooter>
          <DialogClose asChild>
            <Button type="button" variant="outline">
              {tCommon("cancel")}
            </Button>
          </DialogClose>
          <Button
            type="submit"
            variant="gradient"
            disabled={submitting || !projectId}
            className="inline-flex items-center gap-2"
          >
            {submitting ? <Loader2 className="h-4 w-4 animate-spin" /> : null}
            {submitting ? t("saving") : tCommon("save")}
          </Button>
        </DialogFooter>
      </form>
    </>
  );

  if (isControlled) {
    return (
      <Dialog
        open={open}
        onOpenChange={(next) => {
          setOpen(next);
          if (next) resetForm();
        }}
      >
        <DialogContent>{dialogBody}</DialogContent>
      </Dialog>
    );
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        setOpen(next);
        if (next) resetForm();
      }}
    >
      <DialogTrigger asChild>
        {trigger ?? (
          <Button variant="outline" size="sm" type="button" disabled={!projectId}>
            <Github className="h-4 w-4" /> {t("connectButton")}
          </Button>
        )}
      </DialogTrigger>
      <DialogContent>{dialogBody}</DialogContent>
    </Dialog>
  );
}
