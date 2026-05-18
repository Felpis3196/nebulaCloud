"use client";

import { Activity, Cpu, Rocket, Timer } from "lucide-react";
import { useTranslations } from "next-intl";
import { StatCard } from "@/components/dashboard/stat-card";
import { DeployTimeline } from "@/components/dashboard/deploy-timeline";
import { ActivityFeed } from "@/components/dashboard/activity-feed";
import { ResourceMiniChart } from "@/components/dashboard/resource-mini-chart";
import { makeSeries } from "@/lib/mock-data";
import { useProjects } from "@/hooks/use-projects";
import { useDeployments } from "@/hooks/use-deployments";
import { useOrganizationStore } from "@/stores/org-store";
import type { Deployment } from "@/types/api";

function startOfTodayUtc(): number {
  const d = new Date();
  d.setUTCHours(0, 0, 0, 0);
  return d.getTime();
}

function deploysTodayUtc(deployments: Deployment[]): number {
  const t0 = startOfTodayUtc();
  return deployments.filter((d) => new Date(d.created_at).getTime() >= t0).length;
}

export function OverviewDashboardClient() {
  const t = useTranslations("dashboard.overview");
  const orgId = useOrganizationStore((s) => s.selectedOrganizationId);
  const { data: projects = [], isLoading: projectsLoading } = useProjects(orgId);
  const firstProjectId = projects[0]?.id;
  const { data: recentDeployments = [], isLoading: depLoading } = useDeployments(firstProjectId);

  const totalProjects = projects.length;
  const totalServices = projects.reduce((acc, p) => acc + (p.services_count ?? 0), 0);
  const todayDeploys = deploysTodayUtc(recentDeployments);

  const requestsSeries = makeSeries("Requests", 240, 90);
  const cpuSeries = makeSeries("CPU", 38, 22);

  return (
    <div className="space-y-6">
      <header className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">{t("title")}</h1>
          <p className="text-sm text-muted-foreground">{t("subtitle")}</p>
        </div>
      </header>

      <div className="rounded-md border border-border/80 bg-muted/30 px-3 py-2 text-xs text-muted-foreground">
        <span className="font-medium text-foreground">{t("mvpBanner")}</span> {t("mvpText")}
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
          value={
            !firstProjectId ? "—" : depLoading ? "…" : String(todayDeploys)
          }
          hint={firstProjectId ? t("deploysHintFirst") : t("deploysHintNone")}
        />
        <StatCard
          icon={Cpu}
          label={t("clusterUptime")}
          value="—"
          hint={t("clusterHint")}
        />
      </section>

      <section className="grid gap-4 lg:grid-cols-2">
        <ResourceMiniChart
          title={t("requestRate")}
          description={t("requestDesc")}
          unit=" rpm"
          series={requestsSeries}
          color="hsl(239 84% 67%)"
        />
        <ResourceMiniChart
          title={t("cpuUsage")}
          description={t("cpuDesc")}
          unit="%"
          series={cpuSeries}
          color="hsl(305 80% 65%)"
        />
      </section>

      <section className="grid gap-4 lg:grid-cols-[1.55fr_1fr]">
        <DeployTimeline
          deployments={recentDeployments}
          timelineDescription={
            firstProjectId
              ? t("timelineWithProject", {
                  name: projects[0]?.name ?? projects[0]?.slug ?? "",
                })
              : t("timelineEmpty")
          }
        />
        <ActivityFeed />
      </section>
    </div>
  );
}
