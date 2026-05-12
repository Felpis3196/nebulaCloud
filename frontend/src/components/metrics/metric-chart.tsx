"use client";

import { Area, AreaChart, CartesianGrid, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import type { MetricSeries } from "@/types/api";

interface Props {
  title: string;
  description?: string;
  series: MetricSeries;
  unit?: string;
  color?: string;
}

export function MetricChart({
  title,
  description,
  series,
  unit = "",
  color = "hsl(var(--primary))",
}: Props) {
  const data = series.points.map((p) => ({
    ts: p.ts,
    value: Math.round(p.value * 100) / 100,
  }));
  const last = data[data.length - 1]?.value ?? 0;
  const sum = data.reduce((acc, p) => acc + p.value, 0);
  const avg = data.length ? sum / data.length : 0;
  const max = Math.max(...data.map((d) => d.value));

  return (
    <Card>
      <CardHeader className="flex flex-row items-start justify-between gap-4 pb-2">
        <div>
          <CardTitle className="text-sm">{title}</CardTitle>
          {description && (
            <CardDescription className="mt-0.5 text-xs">{description}</CardDescription>
          )}
        </div>
        <div className="flex gap-4 text-xs text-muted-foreground">
          <Stat label="now" value={`${last.toFixed(1)}${unit}`} />
          <Stat label="avg" value={`${avg.toFixed(1)}${unit}`} />
          <Stat label="max" value={`${max.toFixed(1)}${unit}`} />
        </div>
      </CardHeader>
      <CardContent>
        <div className="h-44 w-full">
          <ResponsiveContainer>
            <AreaChart data={data} margin={{ top: 4, right: 8, left: -10, bottom: 0 }}>
              <defs>
                <linearGradient id={`m-${title}`} x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor={color} stopOpacity={0.4} />
                  <stop offset="100%" stopColor={color} stopOpacity={0} />
                </linearGradient>
              </defs>
              <CartesianGrid stroke="hsl(var(--border))" strokeDasharray="3 3" vertical={false} />
              <XAxis
                dataKey="ts"
                tickFormatter={(ts: number) =>
                  new Date(ts).toLocaleTimeString(undefined, { hour: "2-digit", minute: "2-digit" })
                }
                tick={{ fontSize: 10, fill: "hsl(var(--muted-foreground))" }}
                axisLine={false}
                tickLine={false}
                minTickGap={28}
              />
              <YAxis
                tick={{ fontSize: 10, fill: "hsl(var(--muted-foreground))" }}
                axisLine={false}
                tickLine={false}
                width={36}
                tickFormatter={(v: number) => `${v}${unit}`}
              />
              <Tooltip
                contentStyle={{
                  background: "hsl(var(--popover))",
                  border: "1px solid hsl(var(--border))",
                  borderRadius: 8,
                  fontSize: 11,
                }}
                labelFormatter={(ts: number) => new Date(ts).toLocaleString()}
                formatter={(v: number) => [`${v}${unit}`, title]}
              />
              <Area
                type="monotone"
                dataKey="value"
                stroke={color}
                strokeWidth={1.5}
                fill={`url(#m-${title})`}
                isAnimationActive={false}
              />
            </AreaChart>
          </ResponsiveContainer>
        </div>
      </CardContent>
    </Card>
  );
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div className="text-right">
      <div className="font-mono text-sm font-semibold tabular-nums text-foreground">{value}</div>
      <div className="text-[10px] uppercase tracking-wider">{label}</div>
    </div>
  );
}
