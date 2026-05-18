"use client";

import { useTranslations } from "next-intl";
import { cn } from "@/lib/utils";
import type { DeploymentStatus, ServiceStatus, SslStatus } from "@/types/api";

type AnyStatus = DeploymentStatus | ServiceStatus | SslStatus;

const TONE_CLASSES: Record<string, { dot: string; ring: string; text: string }> = {
  running: { dot: "bg-success", ring: "bg-success/15", text: "text-success" },
  issued: { dot: "bg-success", ring: "bg-success/15", text: "text-success" },
  building: { dot: "bg-warning", ring: "bg-warning/15", text: "text-warning" },
  pushing: { dot: "bg-warning", ring: "bg-warning/15", text: "text-warning" },
  deploying: { dot: "bg-info", ring: "bg-info/15", text: "text-info" },
  queued: { dot: "bg-muted-foreground", ring: "bg-muted/40", text: "text-muted-foreground" },
  pending: { dot: "bg-warning", ring: "bg-warning/15", text: "text-warning" },
  failed: { dot: "bg-destructive", ring: "bg-destructive/15", text: "text-destructive" },
  canceled: { dot: "bg-muted-foreground", ring: "bg-muted/40", text: "text-muted-foreground" },
  rolled_back: { dot: "bg-info", ring: "bg-info/15", text: "text-info" },
  idle: { dot: "bg-muted-foreground/60", ring: "bg-muted/40", text: "text-muted-foreground" },
  stopped: { dot: "bg-muted-foreground/60", ring: "bg-muted/40", text: "text-muted-foreground" },
  disabled: { dot: "bg-muted-foreground/60", ring: "bg-muted/40", text: "text-muted-foreground" },
};

const PULSING = new Set(["building", "pushing", "deploying", "queued", "pending"]);

const STATUS_KEYS = [
  "running",
  "issued",
  "building",
  "pushing",
  "deploying",
  "queued",
  "pending",
  "failed",
  "canceled",
  "rolled_back",
  "idle",
  "stopped",
  "disabled",
] as const;

type StatusKey = (typeof STATUS_KEYS)[number];

function isStatusKey(value: string): value is StatusKey {
  return (STATUS_KEYS as readonly string[]).includes(value);
}

export function StatusPill({
  status,
  className,
}: {
  status: AnyStatus | string;
  className?: string;
}) {
  const t = useTranslations("status");
  const tone = TONE_CLASSES[status] ?? TONE_CLASSES.idle!;
  const label = isStatusKey(status) ? t(status) : t("idle");
  const animate = PULSING.has(status);

  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs font-medium",
        tone.ring,
        tone.text,
        className,
      )}
    >
      <span className="relative flex h-1.5 w-1.5">
        <span
          className={cn(
            "absolute inline-flex h-full w-full rounded-full opacity-75",
            tone.dot,
            animate && "animate-ping",
          )}
        />
        <span className={cn("relative inline-flex h-1.5 w-1.5 rounded-full", tone.dot)} />
      </span>
      {label}
    </span>
  );
}
