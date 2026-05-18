"use client";

import { useQueries } from "@tanstack/react-query";
import { useTranslations } from "next-intl";
import { MetricsPanel } from "@/components/metrics/metrics-panel";
import { useOrganizationStore } from "@/stores/org-store";
import { useProjects } from "@/hooks/use-projects";
import { api } from "@/lib/api-client";
import type { Service } from "@/types/api";

export default function MetricsPage() {
  const t = useTranslations("dashboard.metrics");
  const orgId = useOrganizationStore((s) => s.selectedOrganizationId);
  const { data: projects = [], isLoading: projLoading } = useProjects(orgId);

  const serviceQueries = useQueries({
    queries: projects.map((p) => ({
      queryKey: ["services", p.id] as const,
      queryFn: () => api<Service[]>(`/api/v1/projects/${p.id}/services`),
      enabled: !!orgId && projects.length > 0,
    })),
  });

  const loading = projLoading || serviceQueries.some((q) => q.isPending);
  const flat = projects.flatMap((p, i) => {
    const rows = serviceQueries[i]?.data ?? [];
    return rows.map((s) => ({ id: s.id, name: `${p.slug}/${s.slug}` }));
  });

  return (
    <div className="space-y-6">
      <header>
        <h1 className="text-2xl font-semibold tracking-tight">{t("title")}</h1>
        <p className="text-sm text-muted-foreground">{t("subtitle")}</p>
      </header>

      {loading ? (
        <p className="text-sm text-muted-foreground">{t("loadingServices")}</p>
      ) : flat.length === 0 ? (
        <p className="text-sm text-muted-foreground">{t("orgNoServices")}</p>
      ) : (
        <MetricsPanel variant="live" services={flat} />
      )}
    </div>
  );
}
