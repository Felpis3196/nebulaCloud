"use client";

import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api-client";
import type { Deployment } from "@/types/api";

export function useDeployments(projectId?: string, serviceId?: string) {
  return useQuery({
    queryKey: ["deployments", projectId ?? "none", serviceId ?? "none"],
    enabled: !!(projectId || serviceId),
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
