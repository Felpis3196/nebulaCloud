"use client";

import { Area, AreaChart, ResponsiveContainer, Tooltip } from "recharts";
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from "@/components/ui/card";
import type { MetricSeries } from "@/types/api";

interface Props {
  title: string;
  description?: string;
  series: MetricSeries;
  unit?: string;
  color?: string;
}

export function ResourceMiniChart({
  title,
  description,
  series,
  unit = "",
  color = "hsl(var(--primary))",
}: Props) {
  const data = series.points.map((p) => ({ ts: p.ts, value: Math.round(p.value * 100) / 100 }));
  const last = data[data.length - 1]?.value ?? 0;
  const max = Math.max(...data.map((d) => d.value));

  return (
    <Card>
      <CardHeader className="flex flex-row items-start justify-between gap-4 pb-2">
        <div>
          <CardTitle className="text-sm font-medium">{title}</CardTitle>
          {description && (
            <CardDescription className="mt-0.5 text-xs">{description}</CardDescription>
          )}
        </div>
        <div className="text-right">
          <div className="font-mono text-base font-semibold tabular-nums">
            {last.toFixed(1)}
            <span className="ml-0.5 text-xs text-muted-foreground">{unit}</span>
          </div>
          <p className="text-[10px] uppercase tracking-wider text-muted-foreground">
            peak {max.toFixed(1)}
            {unit}
          </p>
        </div>
      </CardHeader>
      <CardContent className="p-0">
        <div className="h-24 w-full">
          <ResponsiveContainer>
            <AreaChart data={data} margin={{ top: 4, right: 0, left: 0, bottom: 0 }}>
              <defs>
                <linearGradient id={`grad-${title}`} x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor={color} stopOpacity={0.4} />
                  <stop offset="100%" stopColor={color} stopOpacity={0} />
                </linearGradient>
              </defs>
              <Tooltip
                cursor={{ stroke: "hsl(var(--border))", strokeWidth: 1 }}
                contentStyle={{
                  background: "hsl(var(--popover))",
                  border: "1px solid hsl(var(--border))",
                  borderRadius: 8,
                  fontSize: 11,
                  padding: "4px 8px",
                }}
                labelFormatter={(ts: number) => new Date(ts).toLocaleTimeString()}
                formatter={(v: number) => [`${v}${unit}`, title]}
              />
              <Area
                type="monotone"
                dataKey="value"
                stroke={color}
                strokeWidth={1.5}
                fill={`url(#grad-${title})`}
                isAnimationActive={false}
              />
            </AreaChart>
          </ResponsiveContainer>
        </div>
      </CardContent>
    </Card>
  );
}
