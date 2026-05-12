import { cn } from "@/lib/utils";
import type { DeploymentStatus, ServiceStatus, SslStatus } from "@/types/api";

type AnyStatus = DeploymentStatus | ServiceStatus | SslStatus;

const TONES: Record<string, { dot: string; ring: string; text: string; label: string }> = {
  running: { dot: "bg-success", ring: "bg-success/15", text: "text-success", label: "Running" },
  issued: { dot: "bg-success", ring: "bg-success/15", text: "text-success", label: "Issued" },
  building: { dot: "bg-warning", ring: "bg-warning/15", text: "text-warning", label: "Building" },
  pushing: { dot: "bg-warning", ring: "bg-warning/15", text: "text-warning", label: "Pushing" },
  deploying: { dot: "bg-info", ring: "bg-info/15", text: "text-info", label: "Deploying" },
  queued: { dot: "bg-muted-foreground", ring: "bg-muted/40", text: "text-muted-foreground", label: "Queued" },
  pending: { dot: "bg-warning", ring: "bg-warning/15", text: "text-warning", label: "Pending" },
  failed: { dot: "bg-destructive", ring: "bg-destructive/15", text: "text-destructive", label: "Failed" },
  canceled: { dot: "bg-muted-foreground", ring: "bg-muted/40", text: "text-muted-foreground", label: "Canceled" },
  rolled_back: { dot: "bg-info", ring: "bg-info/15", text: "text-info", label: "Rolled back" },
  idle: { dot: "bg-muted-foreground/60", ring: "bg-muted/40", text: "text-muted-foreground", label: "Idle" },
  stopped: { dot: "bg-muted-foreground/60", ring: "bg-muted/40", text: "text-muted-foreground", label: "Stopped" },
  disabled: { dot: "bg-muted-foreground/60", ring: "bg-muted/40", text: "text-muted-foreground", label: "Disabled" },
};

const PULSING = new Set(["building", "pushing", "deploying", "queued", "pending"]);

export function StatusPill({
  status,
  className,
}: {
  status: AnyStatus | string;
  className?: string;
}) {
  const tone = TONES[status] ?? TONES.idle!;
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
      {tone.label}
    </span>
  );
}
