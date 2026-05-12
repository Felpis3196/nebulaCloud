"use client";

import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api-client";
import type { Service } from "@/types/api";

export function useServices(projectId: string | undefined) {
  return useQuery({
    queryKey: ["services", projectId],
    enabled: !!projectId,
    queryFn: () => api<Service[]>(`/api/v1/projects/${projectId}/services`),
  });
}
