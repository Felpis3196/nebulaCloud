"use client";

import Link from "next/link";
import { Activity, ArrowRight, FolderGit2, Rocket, Timer } from "lucide-react";
import { useTranslations } from "next-intl";
import { Button } from "@/components/ui/button";
import { StatCard } from "@/components/dashboard/stat-card";
import { DeployTimeline } from "@/components/dashboard/deploy-timeline";
import { ActivityFeed } from "@/components/dashboard/activity-feed";
import { useProjects } from "@/hooks/use-projects";
import { useWorkspaceDeployments } from "@/hooks/use-workspace-deployments";
import { useOrganizationStore } from "@/stores/org-store";

export function OverviewDashboardClient() {
  const t = useTranslations("dashboard.overview");
  const orgId = useOrganizationStore((s) => s.selectedOrganizationId);
  const { data: projects = [], isLoading: projectsLoading } = useProjects(orgId);
  const {
    deployments,
    hasProjects,
    isLoading: depLoading,
    isEmpty: noDeploys,
    activeCount,
    deploysToday,
  } = useWorkspaceDeployments(orgId);

  const totalProjects = projects.length;
  const totalServices = projects.reduce((acc, p) => acc + (p.services_count ?? 0), 0);

  const timelineDescription = !orgId
    ? t("timelineNoProjects")
    : !hasProjects
      ? t("timelineNoProjects")
      : noDeploys
        ? t("timelineNoDeploys")
        : t("timelineOrgWide");

  return (
    <div className="space-y-6">
      <header className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">{t("title")}</h1>
          <p className="text-sm text-muted-foreground">{t("subtitle")}</p>
        </div>
      </header>

      <div className="rounded-md border border-border/80 bg-muted/30 px-3 py-2 text-xs text-muted-foreground">
        <span className="font-medium text-foreground">{t("mvpBanner")}</span> {t("mvpText")}{" "}
        <Link href="/metrics" className="underline underline-offset-2">
          {t("metricsLink")}
        </Link>
        .
      </div>

      {!orgId || !hasProjects ? (
        <div className="rounded-lg border border-dashed border-border/80 bg-muted/20 px-4 py-8 text-center">
          <p className="text-sm font-medium">{t("noProjectsTitle")}</p>
          <p className="mt-1 text-xs text-muted-foreground">{t("noProjectsDesc")}</p>
          <Button asChild variant="gradient" size="sm" className="mt-4">
            <Link href="/projects">
              {t("deployCtaButton")}
              <ArrowRight className="h-4 w-4" />
            </Link>
          </Button>
        </div>
      ) : noDeploys && !depLoading ? (
        <div className="rounded-lg border border-dashed border-border/80 bg-muted/20 px-4 py-6 text-center">
          <p className="text-sm font-medium">{t("noDeploysTitle")}</p>
          <p className="mt-1 text-xs text-muted-foreground">{t("noDeploysDesc")}</p>
        </div>
      ) : null}

      <div className="flex flex-col gap-3 rounded-lg border border-primary/30 bg-primary/5 p-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-start gap-3">
          <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-primary/15 text-primary">
            <FolderGit2 className="h-4 w-4" />
          </div>
          <div>
            <p className="text-sm font-medium">{t("deployCtaTitle")}</p>
            <p className="text-xs text-muted-foreground">{t("deployCtaDesc")}</p>
          </div>
        </div>
        <Button asChild variant="gradient" size="sm" className="shrink-0">
          <Link href="/projects">
            {t("deployCtaButton")}
            <ArrowRight className="h-4 w-4" />
          </Link>
        </Button>
      </div>

      <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <StatCard
          accent
          icon={Activity}
          label={t("projects")}
          value={projectsLoading ? "…" : String(totalProjects)}
          hint={t("projectsHint")}
        />
        <StatCard
          icon={Rocket}
          label={t("servicesSum")}
          value={projectsLoading ? "…" : String(totalServices)}
          hint={t("servicesHint")}
        />
        <StatCard
          icon={Timer}
          label={t("deploysToday")}
          value={!orgId || !hasProjects ? "—" : depLoading ? "…" : String(deploysToday)}
          hint={hasProjects ? t("deploysHintAll") : t("deploysHintNone")}
        />
        <StatCard
          icon={Rocket}
          label={t("activeDeploys")}
          value={!orgId || !hasProjects ? "—" : depLoading ? "…" : String(activeCount)}
          hint={t("activeDeploysHint")}
        />
      </section>

      <section className="grid gap-4 lg:grid-cols-[1.55fr_1fr]">
        <DeployTimeline
          deployments={deployments}
          loading={depLoading}
          timelineDescription={timelineDescription}
        />
        <ActivityFeed deployments={deployments} loading={depLoading} />
      </section>
    </div>
  );
}
