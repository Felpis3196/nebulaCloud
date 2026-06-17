"use client";

import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api-client";
import type { Deployment, DeploymentStatus, Project } from "@/types/api";

const ACTIVE: DeploymentStatus[] = ["queued", "building", "pushing", "deploying"];

function startOfTodayUtc(): number {
  const d = new Date();
  d.setUTCHours(0, 0, 0, 0);
  return d.getTime();
}

function mergeDeployments(lists: Deployment[][]): Deployment[] {
  const byId = new Map<string, Deployment>();
  for (const list of lists) {
    for (const d of list) {
      byId.set(d.id, d);
    }
  }
  return Array.from(byId.values()).sort(
    (a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime(),
  );
}

function countActive(deployments: Deployment[]): number {
  return deployments.filter((d) => ACTIVE.includes(d.status)).length;
}

function countRunning(deployments: Deployment[]): number {
  return deployments.filter((d) => d.status === "running").length;
}

function deploysTodayUtc(deployments: Deployment[]): number {
  const t0 = startOfTodayUtc();
  return deployments.filter((d) => new Date(d.created_at).getTime() >= t0).length;
}

export function useWorkspaceDeployments(organizationId: string | null) {
  const query = useQuery({
    queryKey: ["workspace-deployments", organizationId],
    enabled: !!organizationId,
    refetchInterval: (q) => {
      const deps = q.state.data?.deployments ?? [];
      return countActive(deps) > 0 ? 3000 : false;
    },
    queryFn: async () => {
      const projects = await api<Project[]>(
        `/api/v1/organizations/${organizationId}/projects`,
      );
      if (projects.length === 0) {
        return { deployments: [] as Deployment[], hasProjects: false };
      }
      const lists = await Promise.all(
        projects.map((p) =>
          api<Deployment[]>(`/api/v1/projects/${p.id}/deployments?limit=50`).catch(
            () => [] as Deployment[],
          ),
        ),
      );
      return {
        deployments: mergeDeployments(lists),
        hasProjects: true,
      };
    },
  });

  const deployments = query.data?.deployments ?? [];
  const hasProjects = query.data?.hasProjects ?? false;

  return {
    deployments,
    hasProjects,
    isLoading: query.isLoading,
    isEmpty: !query.isLoading && hasProjects && deployments.length === 0,
    activeCount: countActive(deployments),
    runningCount: countRunning(deployments),
    deploysToday: deploysTodayUtc(deployments),
  };
}
