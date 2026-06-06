"use client";

import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api-client";
import type { BuildLogLine } from "@/types/api";

export function useDeploymentBuildLogs(
  deploymentId: string | undefined,
  poll = false,
) {
  return useQuery({
    queryKey: ["deployment-build-logs", deploymentId],
    enabled: !!deploymentId,
    queryFn: () =>
      api<BuildLogLine[]>(
        `/api/v1/deployments/${deploymentId}/build-logs?limit=200`,
      ),
    refetchInterval: poll ? 2000 : false,
  });
}
