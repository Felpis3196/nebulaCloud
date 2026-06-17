"use client";

import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api-client";
import type { Deployment, DeploymentStatus } from "@/types/api";

const ACTIVE: DeploymentStatus[] = ["queued", "building", "pushing", "deploying"];

function isActive(status: DeploymentStatus | undefined) {
  return !!status && ACTIVE.includes(status);
}

export function useDeployment(
  deploymentId: string | undefined,
  enabled = true,
) {
  return useQuery({
    queryKey: ["deployment", deploymentId],
    enabled: enabled && !!deploymentId,
    queryFn: () => api<Deployment>(`/api/v1/deployments/${deploymentId}`),
    refetchInterval: (query) =>
      isActive(query.state.data?.status) ? 2000 : false,
  });
}
