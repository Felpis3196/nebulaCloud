"use client";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

export const TIMERANGES = [
  { id: "15m", label: "15m" },
  { id: "1h", label: "1h" },
  { id: "6h", label: "6h" },
  { id: "24h", label: "24h" },
  { id: "7d", label: "7d" },
] as const;

export type TimerangeId = (typeof TIMERANGES)[number]["id"];

export function TimerangePicker({
  value,
  onChange,
}: {
  value: TimerangeId;
  onChange: (next: TimerangeId) => void;
}) {
  return (
    <div className="inline-flex rounded-md border border-border/60 bg-card/40 p-0.5">
      {TIMERANGES.map((r) => (
        <Button
          key={r.id}
          size="sm"
          variant="ghost"
          className={cn(
            "h-7 rounded px-2.5 text-xs",
            value === r.id && "bg-secondary/80 text-foreground",
          )}
          onClick={() => onChange(r.id)}
        >
          {r.label}
        </Button>
      ))}
    </div>
  );
}
