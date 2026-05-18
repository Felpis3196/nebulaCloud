"use client";

import type { ReactNode } from "react";
import Link from "next/link";
import { notFound, useParams, useRouter } from "next/navigation";
import { GitBranch, Github, Loader2 } from "lucide-react";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ProjectTabs } from "@/components/projects/project-tabs";
import { ConnectRepoDialog } from "@/components/projects/connect-repo-dialog";
import { useProject } from "@/hooks/use-project";
import { useServices } from "@/hooks/use-services";
import { api, ApiError } from "@/lib/api-client";
import { useTranslations } from "next-intl";

export default function ProjectLayout({ children }: { children: ReactNode }) {
  const t = useTranslations("projects.detail");
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const id = typeof params?.id === "string" ? params.id : "";

  const { data: project, isLoading, isError, error } = useProject(id || undefined);
  const { data: services = [] } = useServices(id || undefined);

  if (isError && error instanceof ApiError && error.status === 404) {
    notFound();
  }

  if (!id) {
    return <p className="text-sm text-muted-foreground">{t("missingProject")}</p>;
  }

  if (isLoading || !project) {
    return (
      <div className="flex items-center gap-2 text-sm text-muted-foreground">
        <Loader2 className="h-4 w-4 animate-spin" />
        {t("loadingProject")}
      </div>
    );
  }

  async function deployNow() {
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
          <ConnectRepoDialog projectId={id} />
          <Button variant="gradient" size="sm" onClick={() => void deployNow()}>
            {t("deployNow")}
          </Button>
        </div>
      </header>
      <ProjectTabs projectId={project.id} />
      <div>{children}</div>
    </div>
  );
}
