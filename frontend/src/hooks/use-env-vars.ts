"use client";

import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api-client";
import type { EnvVar } from "@/types/api";

export function useEnvVars(serviceId: string | undefined) {
  return useQuery({
    queryKey: ["env-vars", serviceId],
    enabled: !!serviceId,
    queryFn: () => api<EnvVar[]>(`/api/v1/services/${serviceId}/env-vars`),
  });
}
