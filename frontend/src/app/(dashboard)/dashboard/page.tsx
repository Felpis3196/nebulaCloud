import { Activity, Cpu, Rocket, Timer } from "lucide-react";
import { StatCard } from "@/components/dashboard/stat-card";
import { DeployTimeline } from "@/components/dashboard/deploy-timeline";
import { ActivityFeed } from "@/components/dashboard/activity-feed";
import { ResourceMiniChart } from "@/components/dashboard/resource-mini-chart";
import { MOCK_DEPLOYMENTS, makeSeries } from "@/lib/mock-data";

export const metadata = { title: "Overview" };

export default function DashboardOverviewPage() {
  const requestsSeries = makeSeries("Requests", 240, 90);
  const cpuSeries = makeSeries("CPU", 38, 22);

  return (
    <div className="space-y-6">
      <header className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Overview</h1>
          <p className="text-sm text-muted-foreground">
            Workspace health at a glance — deploys, throughput, and recent activity.
          </p>
        </div>
      </header>

      <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <StatCard
          accent
          icon={Activity}
          label="Active services"
          value="9"
          hint="across 3 projects"
          delta={{ direction: "up", value: "+1" }}
        />
        <StatCard
          icon={Rocket}
          label="Deploys today"
          value="12"
          hint="vs 8 yesterday"
          delta={{ direction: "up", value: "+50%" }}
        />
        <StatCard
          icon={Timer}
          label="Avg build time"
          value="42s"
          hint="last 24h"
          delta={{ direction: "down", value: "-6s" }}
        />
        <StatCard
          icon={Cpu}
          label="Uptime"
          value="99.97%"
          hint="30-day rolling"
          delta={{ direction: "flat", value: "stable" }}
        />
      </section>

      <section className="grid gap-4 lg:grid-cols-2">
        <ResourceMiniChart
          title="Request rate"
          description="Aggregated requests/min across all web services."
          unit=" rpm"
          series={requestsSeries}
          color="hsl(239 84% 67%)"
        />
        <ResourceMiniChart
          title="CPU usage"
          description="Mean CPU% across all running containers."
          unit="%"
          series={cpuSeries}
          color="hsl(305 80% 65%)"
        />
      </section>

      <section className="grid gap-4 lg:grid-cols-[1.55fr_1fr]">
        <DeployTimeline deployments={MOCK_DEPLOYMENTS} />
        <ActivityFeed />
      </section>
    </div>
  );
}
