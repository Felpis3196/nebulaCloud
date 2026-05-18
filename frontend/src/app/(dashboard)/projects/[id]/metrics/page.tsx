"use client";

import { useState } from "react";
import { useParams } from "next/navigation";
import { MetricChart } from "@/components/metrics/metric-chart";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { TimerangePicker, type TimerangeId } from "@/components/metrics/timerange-picker";
import { useServices } from "@/hooks/use-services";
import { useMetrics } from "@/hooks/use-metrics";
import { useTranslations } from "next-intl";

const duration: Record<TimerangeId, string> = {
  "15m": "15m",
  "1h": "60m",
  "6h": "6h",
  "24h": "24h",
  "7d": "168h",
};

export default function ProjectMetricsPage() {
  const t = useTranslations("dashboard.metrics");
  const tCommon = useTranslations("common");
  const params = useParams<{ id: string }>();
  const id = typeof params?.id === "string" ? params.id : "";
  const { data: services = [], isPending: servicesLoading } = useServices(id);
  const [serviceId, setServiceId] = useState("");
  const [range, setRange] = useState<TimerangeId>("1h");

  const resolved = serviceId || services[0]?.id || "";
  const win = duration[range] ?? "60m";

  const { data: metrics = [], isPending: metricsLoading } = useMetrics(resolved, win);

  const palette = ["hsl(239 84% 67%)", "hsl(305 80% 65%)", "hsl(217 91% 60%)", "hsl(142 71% 45%)"];

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        {servicesLoading ? (
          <p className="text-sm text-muted-foreground">{t("loadingServices")}</p>
        ) : services.length > 0 ? (
          <Select value={resolved} onValueChange={setServiceId}>
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
        ) : (
          <p className="text-sm text-muted-foreground">{t("noService")}</p>
        )}
        <TimerangePicker value={range} onChange={setRange} />
      </div>

      {metricsLoading && resolved ? (
        <p className="text-sm text-muted-foreground">{t("loadingMetrics")}</p>
      ) : metrics.length === 0 ? (
        <p className="text-sm text-muted-foreground">
          {resolved ? t("noData") : t("noService")}
        </p>
      ) : (
        <div className="grid gap-4 lg:grid-cols-2">
          {metrics.map((series, idx) => (
            <MetricChart
              key={series.name}
              title={series.name}
              unit={series.unit}
              series={series}
              color={palette[idx % palette.length]}
              description=""
            />
          ))}
        </div>
      )}
    </div>
  );
}
