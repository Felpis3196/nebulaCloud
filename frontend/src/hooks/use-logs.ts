"use client";

import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api-client";
import type { LogLine } from "@/types/api";

export function useLogs(serviceId?: string, window = "30m") {
  return useQuery({
    queryKey: ["logs", serviceId, window],
    enabled: !!serviceId,
    refetchInterval: 5000,
    queryFn: () =>
      api<LogLine[]>(`/api/v1/services/${serviceId}/logs?window=${encodeURIComponent(window)}`),
  });
}
