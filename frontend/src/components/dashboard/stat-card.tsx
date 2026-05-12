import type { LucideIcon } from "lucide-react";
import { ArrowDownRight, ArrowUpRight } from "lucide-react";
import { cn } from "@/lib/utils";
import { Card } from "@/components/ui/card";

interface StatCardProps {
  label: string;
  value: string;
  hint?: string;
  delta?: { direction: "up" | "down" | "flat"; value: string };
  icon?: LucideIcon;
  accent?: boolean;
}

export function StatCard({ label, value, hint, delta, icon: Icon, accent }: StatCardProps) {
  return (
    <Card className={cn("relative overflow-hidden p-5", accent && "border-primary/40")}>
      {accent && (
        <div
          aria-hidden
          className="absolute inset-0 -z-10 opacity-60"
          style={{
            backgroundImage:
              "radial-gradient(120% 80% at 0% 0%, hsl(239 84% 60% / 0.16), transparent 60%)",
          }}
        />
      )}
      <div className="flex items-start justify-between gap-4">
        <div className="space-y-1.5">
          <p className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
            {label}
          </p>
          <p className="text-2xl font-semibold tracking-tight tabular-nums">{value}</p>
          <div className="flex items-center gap-2 text-xs text-muted-foreground">
            {delta && (
              <span
                className={cn(
                  "inline-flex items-center gap-0.5 font-medium",
                  delta.direction === "up" && "text-success",
                  delta.direction === "down" && "text-destructive",
                )}
              >
                {delta.direction === "up" ? (
                  <ArrowUpRight className="h-3 w-3" />
                ) : delta.direction === "down" ? (
                  <ArrowDownRight className="h-3 w-3" />
                ) : null}
                {delta.value}
              </span>
            )}
            {hint && <span>{hint}</span>}
          </div>
        </div>
        {Icon && (
          <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary ring-1 ring-primary/20">
            <Icon className="h-4 w-4" />
          </div>
        )}
      </div>
    </Card>
  );
}
