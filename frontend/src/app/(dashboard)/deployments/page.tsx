"use client";

import { useSearchParams } from "next/navigation";
import { DeploymentsTable } from "@/components/deployments/deployments-table";
import { useDeployments } from "@/hooks/use-deployments";
import { useProjects } from "@/hooks/use-projects";
import { useOrganizationStore } from "@/stores/org-store";

export default function DeploymentsPage() {
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
          <h1 className="text-2xl font-semibold tracking-tight">Deployments</h1>
          <p className="text-sm text-muted-foreground">
            Audit releases in this workspace ({serviceId ? "single service filter" : "first project"})
          </p>
        </div>
      </header>
      {isLoading ? (
        <p className="text-sm text-muted-foreground">Loading…</p>
      ) : (
        <DeploymentsTable deployments={data} />
      )}
    </div>
  );
}
