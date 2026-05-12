"use client";

import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api-client";
import type { MetricSeries } from "@/types/api";

export function useMetrics(serviceId?: string, window = "60m") {
  return useQuery({
    queryKey: ["metrics", serviceId, window],
    enabled: !!serviceId,
    refetchInterval: 15000,
    queryFn: () =>
      api<MetricSeries[]>(
        `/api/v1/services/${serviceId}/metrics?window=${encodeURIComponent(window)}`,
      ),
  });
}
