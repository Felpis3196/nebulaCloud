"use client";

import { useMemo, useState } from "react";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { TimerangePicker, type TimerangeId } from "@/components/metrics/timerange-picker";
import { MetricChart } from "@/components/metrics/metric-chart";
import { makeSeries } from "@/lib/mock-data";

interface Props {
  services: { id: string; name: string }[];
}

const POINTS_BY_RANGE: Record<TimerangeId, number> = {
  "15m": 15,
  "1h": 60,
  "6h": 72,
  "24h": 96,
  "7d": 84,
};

export function MetricsPanel({ services }: Props) {
  const [serviceId, setServiceId] = useState(services[0]?.id ?? "");
  const [range, setRange] = useState<TimerangeId>("1h");
  const points = POINTS_BY_RANGE[range];

  const series = useMemo(
    () => ({
      cpu: makeSeries("CPU", 35, 22, points),
      ram: makeSeries("RAM", 480, 90, points),
      net_in: makeSeries("Network in", 6.4, 2.6, points),
      net_out: makeSeries("Network out", 4.1, 1.8, points),
      requests: makeSeries("Requests", 220, 80, points),
    }),
    // re-run when range changes (serviceId would matter once we hit a real backend)
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [range, serviceId],
  );

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-3">
          <Select value={serviceId} onValueChange={setServiceId}>
            <SelectTrigger className="h-9 w-[220px]">
              <SelectValue placeholder="Service" />
            </SelectTrigger>
            <SelectContent>
              {services.map((s) => (
                <SelectItem key={s.id} value={s.id}>
                  {s.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <TimerangePicker value={range} onChange={setRange} />
      </div>

      <div className="grid gap-4 lg:grid-cols-2">
        <MetricChart
          title="CPU"
          unit="%"
          description="Average across replicas"
          series={series.cpu}
          color="hsl(239 84% 67%)"
        />
        <MetricChart
          title="Memory"
          unit=" MB"
          description="Resident set size, p95 across replicas"
          series={series.ram}
          color="hsl(305 80% 65%)"
        />
        <MetricChart
          title="Network in"
          unit=" MB/s"
          description="Inbound traffic to the service"
          series={series.net_in}
          color="hsl(217 91% 60%)"
        />
        <MetricChart
          title="Network out"
          unit=" MB/s"
          description="Outbound traffic from the service"
          series={series.net_out}
          color="hsl(152 76% 44%)"
        />
        <MetricChart
          title="Request rate"
          unit=" rpm"
          description="HTTP requests per minute"
          series={series.requests}
          color="hsl(38 92% 50%)"
        />
      </div>
    </div>
  );
}
