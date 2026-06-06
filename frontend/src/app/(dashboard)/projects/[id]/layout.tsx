"use client";

import { useEffect, useState, type ReactNode } from "react";
import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { GitBranch, Github, Loader2 } from "lucide-react";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { ProjectTabs } from "@/components/projects/project-tabs";
import { ConnectRepoDialog } from "@/components/projects/connect-repo-dialog";
import { useProject } from "@/hooks/use-project";
import { useProjectAccessGuard } from "@/hooks/use-project-access-guard";
import { useServices } from "@/hooks/use-services";
import { api, ApiError } from "@/lib/api-client";
import { useTranslations } from "next-intl";

export default function ProjectLayout({ children }: { children: ReactNode }) {
  const t = useTranslations("projects.detail");
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const id = typeof params?.id === "string" ? params.id : "";

  const [connectOpen, setConnectOpen] = useState(false);
  const { data: project, isLoading, isError, error, isFetched } = useProject(id || undefined);
  const { data: services = [] } = useServices(id || undefined);

  useProjectAccessGuard(id, isError, error, isFetched);

  useEffect(() => {
    if (typeof window === "undefined") return;
    if (new URLSearchParams(window.location.search).get("connect") === "1") {
      setConnectOpen(true);
    }
  }, [id]);

  if (!id || id === "undefined") {
    return <p className="text-sm text-muted-foreground">{t("missingProject")}</p>;
  }

  if (isLoading && !project) {
    return (
      <div className="flex items-center gap-2 text-sm text-muted-foreground">
        <Loader2 className="h-4 w-4 animate-spin" />
        {t("loadingProject")}
      </div>
    );
  }

  if (!project) {
    return (
      <div className="flex items-center gap-2 text-sm text-muted-foreground">
        <Loader2 className="h-4 w-4 animate-spin" />
        {t("loadingProject")}
      </div>
    );
  }

  const hasRepo = Boolean(project.repo_url?.trim());
  const hasService = services.length > 0;
  const canDeploy = hasRepo && hasService;

  async function deployNow() {
    if (!hasRepo) {
      toast.message(t("connectRepoFirst"));
      setConnectOpen(true);
      return;
    }
    const first = services[0];
    if (!first) {
      toast.message(t("addServiceFirst"));
      router.push(`/projects/${id}`);
      return;
    }
    try {
      await api(`/api/v1/services/${first.id}/deployments`, { method: "POST", body: {} });
      toast.success(t("deployQueued"));
      router.push(`/projects/${id}/deployments`);
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : t("deployFailed"));
    }
  }

  return (
    <div className="space-y-6">
      <header className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
        <div className="space-y-3">
          <Link
            href="/projects"
            className="text-xs text-muted-foreground transition-colors hover:text-foreground"
          >
            {t("allProjects")}
          </Link>
          <div>
            <h1 className="text-2xl font-semibold tracking-tight">{project.name}</h1>
            {project.description && (
              <p className="mt-1 text-sm text-muted-foreground">{project.description}</p>
            )}
          </div>
          <div className="flex flex-wrap items-center gap-3 text-xs text-muted-foreground">
            {project.repo_url && (
              <a
                href={project.repo_url}
                target="_blank"
                rel="noreferrer"
                className="inline-flex items-center gap-1.5 transition-colors hover:text-foreground"
              >
                <Github className="h-3 w-3" />
                <span className="font-mono">
                  {project.repo_url.replace(/^https?:\/\/(www\.)?github\.com\//, "")}
                </span>
              </a>
            )}
            <span className="inline-flex items-center gap-1.5">
              <GitBranch className="h-3 w-3" />
              <span className="font-mono">{project.default_branch}</span>
            </span>
            <Badge variant="muted">{t("servicesCount", { count: project.services_count })}</Badge>
          </div>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <Button variant="outline" size="sm" asChild>
            <Link href={`/projects/${id}/settings`}>{t("configure")}</Link>
          </Button>
          <ConnectRepoDialog projectId={id} open={connectOpen} onOpenChange={setConnectOpen} />
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger asChild>
                <span className="inline-flex">
                  <Button
                    variant="gradient"
                    size="sm"
                    disabled={!canDeploy}
                    onClick={() => void deployNow()}
                  >
                    {t("deployNow")}
                  </Button>
                </span>
              </TooltipTrigger>
              {!canDeploy && (
                <TooltipContent>
                  {!hasRepo ? t("deployNeedsRepo") : t("addServiceFirst")}
                </TooltipContent>
              )}
            </Tooltip>
          </TooltipProvider>
        </div>
      </header>
      <ProjectTabs projectId={project.id} />
      <div>{children}</div>
    </div>
  );
}
