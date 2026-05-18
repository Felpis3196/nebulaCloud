"use client";

import { useEffect, useMemo, useState } from "react";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { TimerangePicker, type TimerangeId } from "@/components/metrics/timerange-picker";
import { MetricChart } from "@/components/metrics/metric-chart";
import { makeSeries } from "@/lib/mock-data";
import { useMetrics } from "@/hooks/use-metrics";
import { useTranslations } from "next-intl";

interface Props {
  services: { id: string; name: string }[];
  /** demo = synthetic charts; live = GET /services/:id/metrics (MVP scaffold). */
  variant?: "demo" | "live";
}

const POINTS_BY_RANGE: Record<TimerangeId, number> = {
  "15m": 15,
  "1h": 60,
  "6h": 72,
  "24h": 96,
  "7d": 84,
};

const duration: Record<TimerangeId, string> = {
  "15m": "15m",
  "1h": "60m",
  "6h": "6h",
  "24h": "24h",
  "7d": "168h",
};

const palette = ["hsl(239 84% 67%)", "hsl(305 80% 65%)", "hsl(217 91% 60%)", "hsl(142 71% 45%)", "hsl(38 92% 50%)"];

export function MetricsPanel({ services, variant = "demo" }: Props) {
  const t = useTranslations("dashboard.metrics");
  const tCommon = useTranslations("common");
  const [serviceId, setServiceId] = useState(services[0]?.id ?? "");
  const [range, setRange] = useState<TimerangeId>("1h");
  const points = POINTS_BY_RANGE[range];
  const win = duration[range] ?? "60m";

  useEffect(() => {
    if (services.length === 0) return;
    if (!serviceId || !services.some((s) => s.id === serviceId)) {
      setServiceId(services[0]!.id);
    }
  }, [services, serviceId]);

  const { data: liveMetrics = [], isPending } = useMetrics(
    variant === "live" ? serviceId : undefined,
    win,
  );

  const demoSeries = useMemo(
    () => ({
      cpu: makeSeries("CPU", 35, 22, points),
      ram: makeSeries("RAM", 480, 90, points),
      net_in: makeSeries("Network in", 6.4, 2.6, points),
      net_out: makeSeries("Network out", 4.1, 1.8, points),
      requests: makeSeries("Requests", 220, 80, points),
    }),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [range, serviceId],
  );

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-3">
          <Select value={serviceId} onValueChange={setServiceId} disabled={services.length === 0}>
            <SelectTrigger className="h-9 w-[220px]">
              <SelectValue placeholder={tCommon("service")} />
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

      {variant === "live" ? (
        <div className="space-y-4">
          {isPending ? (
            <p className="text-sm text-muted-foreground">{t("loadingMetrics")}</p>
          ) : liveMetrics.length === 0 ? (
            <p className="text-sm text-muted-foreground">{t("noSeriesReturned")}</p>
          ) : (
            <div className="grid gap-4 lg:grid-cols-2">
              {liveMetrics.map((series, i) => (
                <MetricChart
                  key={series.name}
                  title={series.name}
                  unit={series.unit}
                  description={t("fromApi")}
                  series={series}
                  color={palette[i % palette.length]}
                />
              ))}
            </div>
          )}
        </div>
      ) : (
        <div className="grid gap-4 lg:grid-cols-2">
          <MetricChart
            title="CPU"
            unit="%"
            description={t("avgReplicas")}
            series={demoSeries.cpu}
            color="hsl(239 84% 67%)"
          />
          <MetricChart
            title={t("memory")}
            unit=" MB"
            description={t("memoryDesc")}
            series={demoSeries.ram}
            color="hsl(305 80% 65%)"
          />
          <MetricChart
            title={t("networkIn")}
            unit=" MB/s"
            description={t("networkInDesc")}
            series={demoSeries.net_in}
            color="hsl(217 91% 60%)"
          />
          <MetricChart
            title={t("networkOut")}
            unit=" MB/s"
            description={t("networkOutDesc")}
            series={demoSeries.net_out}
            color="hsl(152 76% 44%)"
          />
          <MetricChart
            title={t("requestRate")}
            unit=" rpm"
            description={t("requestRateDesc")}
            series={demoSeries.requests}
            color="hsl(38 92% 50%)"
          />
        </div>
      )}
    </div>
  );
}
