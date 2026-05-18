"use client";

import { useSearchParams } from "next/navigation";
import { useTranslations } from "next-intl";
import { DeploymentsTable } from "@/components/deployments/deployments-table";
import { useDeployments } from "@/hooks/use-deployments";
import { useProjects } from "@/hooks/use-projects";
import { useOrganizationStore } from "@/stores/org-store";

export default function DeploymentsPage() {
  const t = useTranslations("dashboard.deployments");
  const tCommon = useTranslations("common");
  const search = useSearchParams();
  const serviceId = search.get("s") ?? undefined;

  const orgId = useOrganizationStore((x) => x.selectedOrganizationId);
  const { data: projects = [] } = useProjects(orgId);
  const firstProject = projects[0]?.id;

  const pid = serviceId ? undefined : firstProject;
  const { data = [], isLoading } = useDeployments(pid, serviceId);

  return (
    <div className="space-y-6">
      <header className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">{t("title")}</h1>
          <p className="text-sm text-muted-foreground">
            {serviceId ? t("subtitleFilter") : t("subtitleDefault")}
          </p>
        </div>
      </header>
      {isLoading ? (
        <p className="text-sm text-muted-foreground">{tCommon("loading")}</p>
      ) : (
        <DeploymentsTable deployments={data} />
      )}
    </div>
  );
}
