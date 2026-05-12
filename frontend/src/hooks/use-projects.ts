"use client";

import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api-client";
import type { Project } from "@/types/api";

export function useProjects(organizationId: string | null) {
  return useQuery({
    queryKey: ["projects", organizationId],
    enabled: !!organizationId,
    queryFn: () =>
      api<Project[]>(`/api/v1/organizations/${organizationId}/projects`),
  });
}
