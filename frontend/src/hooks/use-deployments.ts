"use client";

import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api-client";
import type { Deployment, DeploymentStatus } from "@/types/api";

const ACTIVE: DeploymentStatus[] = ["queued", "building", "pushing", "deploying"];

function hasActiveDeployment(items: Deployment[] | undefined) {
  return items?.some((d) => ACTIVE.includes(d.status)) ?? false;
}

export function useDeployments(projectId?: string, serviceId?: string) {
  return useQuery({
    queryKey: ["deployments", projectId ?? "none", serviceId ?? "none"],
    enabled: !!(projectId || serviceId),
    refetchInterval: (query) =>
      hasActiveDeployment(query.state.data) ? 3000 : false,
    queryFn: async () => {
      if (serviceId) {
        return api<Deployment[]>(
          `/api/v1/services/${serviceId}/deployments?limit=100`,
        );
      }
      if (projectId) {
        return api<Deployment[]>(
          `/api/v1/projects/${projectId}/deployments?limit=100`,
        );
      }
      return [];
    },
  });
}
