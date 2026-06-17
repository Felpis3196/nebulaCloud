"use client";

import { useMemo } from "react";
import {
  AlertCircle,
  CheckCircle2,
  Circle,
  GitCommit,
  Loader2,
  RotateCcw,
  Terminal,
  XCircle,
} from "lucide-react";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { StatusPill } from "@/components/dashboard/status-pill";
import { DeploymentLogViewer } from "@/components/deployments/deployment-log-viewer";
import { useDeploymentBuildLogs } from "@/hooks/use-deployment-build-logs";
import { useDeployment } from "@/hooks/use-deployment";
import { useDeploymentLogStream } from "@/hooks/use-deployment-log-stream";
import type { BuildLogLine, Deployment, DeploymentStatus } from "@/types/api";
import { formatDuration, relativeTime, shortSha } from "@/lib/utils";
import { useTranslations } from "next-intl";
import { cn } from "@/lib/utils";

interface Props {
  deploymentId: string | null;
  initialDeployment?: Deployment | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

const PIPELINE_STEPS = [
  { key: "webhookReceived" as const, phase: 0 },
  { key: "sourceCloned" as const, phase: 1 },
  { key: "stackDetected" as const, phase: 2 },
  { key: "imageBuilt" as const, phase: 3 },
  { key: "imagePushed" as const, phase: 4 },
  { key: "replicasRolled" as const, phase: 5 },
  { key: "healthchecks" as const, phase: 6 },
] as const;

type StepVisual = "pending" | "running" | "done" | "error";

function failedStepIndex(error?: string): number {
  const e = (error ?? "").toLowerCase();
  if (e.includes("docker pull") || e.includes("docker run") || e.includes("deploy")) {
    return 5;
  }
  if (e.includes("docker push") || e.includes("push")) return 4;
  if (e.includes("docker build") || e.includes("pack build") || e.includes("building")) return 3;
  if (e.includes("detect") || e.includes("stack")) return 2;
  if (e.includes("clone") || e.includes("git")) return 1;
  return 3;
}

function activePhase(status: DeploymentStatus): number {
  switch (status) {
    case "queued":
      return 0;
    case "building":
      return 3;
    case "pushing":
      return 4;
    case "deploying":
      return 5;
    case "running":
      return 7;
    default:
      return 0;
  }
}

function stepVisual(
  stepPhase: number,
  status: DeploymentStatus,
  errorMessage?: string,
): StepVisual {
  if (status === "failed") {
    const failAt = failedStepIndex(errorMessage);
    if (stepPhase < failAt) return "done";
    if (stepPhase === failAt) return "error";
    return "pending";
  }
  const current = activePhase(status);
  if (stepPhase < current) return "done";
  if (stepPhase === current) return "running";
  return "pending";
}

function StepIcon({ state }: { state: StepVisual }) {
  if (state === "done") return <CheckCircle2 className="h-4 w-4 shrink-0 text-success" />;
  if (state === "error") return <XCircle className="h-4 w-4 shrink-0 text-destructive" />;
  if (state === "running") return <Loader2 className="h-4 w-4 shrink-0 animate-spin text-primary" />;
  return <Circle className="h-4 w-4 shrink-0 text-muted-foreground/50" />;
}

function mergeLogLines(history: BuildLogLine[], live: BuildLogLine[]): BuildLogLine[] {
  const seen = new Set<string>();
  const out: BuildLogLine[] = [];
  for (const line of [...history, ...live]) {
    const key = `${line.ts ?? ""}|${line.message}`;
    if (seen.has(key)) continue;
    seen.add(key);
    out.push(line);
  }
  return out;
}

export function DeploymentDetailDrawer({
  deploymentId,
  initialDeployment,
  open,
  onOpenChange,
}: Props) {
  const t = useTranslations("deployments.drawer");
  const { data: polled } = useDeployment(open ? deploymentId ?? undefined : undefined, open);
  const deployment = polled ?? initialDeployment ?? null;
  const depId = deployment?.id;
  const depStatus = deployment?.status;
  const pollLogs =
    !!deployment &&
    !!depStatus &&
    ["queued", "building", "pushing", "deploying", "failed"].includes(depStatus);
  const { data: history = [], isLoading: logsLoading, refetch } = useDeploymentBuildLogs(
    open ? depId : undefined,
    open && pollLogs,
  );
  const { liveLines, connected, isLive } = useDeploymentLogStream(
    open ? deployment?.service_id : undefined,
    open ? depId : undefined,
    deployment?.status,
  );

  const allLines = useMemo(() => mergeLogLines(history, liveLines), [history, liveLines]);

  if (!deployment) return null;

  const status = deployment.status;
  const isFailed = status === "failed";

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="right" className="flex w-full flex-col gap-0 sm:max-w-xl">
        <SheetHeader className="flex flex-row items-start justify-between gap-4 pb-4">
          <div className="space-y-1">
            <SheetTitle className="flex items-center gap-2">
              {deployment.project_name}
              <span className="text-muted-foreground">/</span>
              <span className="text-muted-foreground">{deployment.service_name}</span>
            </SheetTitle>
            <SheetDescription className="flex flex-wrap items-center gap-2 pt-1">
              <StatusPill status={status} />
              <Badge variant="muted" className="font-mono">
                {shortSha(deployment.commit_sha)}
              </Badge>
              <span className="text-xs">{relativeTime(deployment.created_at)}</span>
              <span className="text-xs">· {formatDuration(deployment.duration_ms ?? 0)}</span>
            </SheetDescription>
          </div>
        </SheetHeader>

        <div className="flex flex-1 flex-col gap-6 overflow-y-auto pr-1">
          {isFailed && (deployment.error_hint || deployment.error_message) && (
            <div className="flex gap-3 rounded-md border border-destructive/40 bg-destructive/10 px-3 py-3 text-sm">
              <AlertCircle className="mt-0.5 h-4 w-4 shrink-0 text-destructive" />
              <div className="min-w-0 space-y-2">
                <p className="font-medium text-destructive">{t("deployFailed")}</p>
                {deployment.error_hint && (
                  <p className="text-sm leading-relaxed text-amber-200/90">{deployment.error_hint}</p>
                )}
                {deployment.error_message && (
                  <p className="whitespace-pre-wrap break-words font-mono text-xs text-destructive/90">
                    {deployment.error_message}
                  </p>
                )}
              </div>
            </div>
          )}

          <section className="space-y-2">
            <div className="flex items-center gap-2 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
              <GitCommit className="h-3 w-3" /> {t("commitSection")}
            </div>
            <p className="text-sm">{deployment.commit_message || "—"}</p>
            <div className="flex items-center gap-2 text-xs text-muted-foreground">
              <span className="font-mono">{deployment.ref ?? "—"}</span>
              <span>·</span>
              <span>{deployment.triggered_by?.email ?? "—"}</span>
            </div>
          </section>

          <Separator />

          <section className="space-y-3">
            <h3 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
              {t("pipeline")}
            </h3>
            <ol className="space-y-2">
              {PIPELINE_STEPS.map((step, i) => {
                const visual = stepVisual(step.phase, status, deployment.error_message);
                return (
                  <li
                    key={step.key}
                    className={cn(
                      "flex items-center gap-3 rounded-md border px-3 py-2 text-sm",
                      visual === "error"
                        ? "border-destructive/40 bg-destructive/5"
                        : "border-border/60 bg-card/40",
                    )}
                  >
                    <StepIcon state={visual} />
                    <span className="font-mono text-xs text-muted-foreground">
                      {String(i + 1).padStart(2, "0")}
                    </span>
                    <span className="flex-1">{t(step.key)}</span>
                  </li>
                );
              })}
            </ol>
          </section>

          <section className="space-y-3">
            <div className="flex items-center justify-between gap-2">
              <h3 className="flex items-center gap-2 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                <Terminal className="h-3 w-3" /> {t("buildLogPreview")}
              </h3>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                className="h-7 text-xs"
                onClick={() => void refetch()}
              >
                {t("refreshLogs")}
              </Button>
            </div>
            <DeploymentLogViewer
              lines={allLines}
              loading={logsLoading && allLines.length === 0}
              live={isLive && connected}
              emptyHint={
                isFailed
                  ? t("noLogsCheckWorkers")
                  : isLive
                    ? t("waitingForLogs")
                    : t("noLogsYet")
              }
            />
          </section>

          {(status === "running" || status === "deploying") &&
            deployment.route_host && (
              <section className="space-y-2">
                <h3 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                  {t("publicUrl")}
                </h3>
                <p className="font-mono text-sm">
                  <a
                    href={`http://${deployment.route_host}`}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-primary underline-offset-4 hover:underline"
                  >
                    http://{deployment.route_host}
                  </a>
                </p>
                {deployment.listen_port != null && deployment.listen_port > 0 && (
                  <p className="text-xs text-muted-foreground">
                    {t("listenPort", { port: deployment.listen_port })}
                  </p>
                )}
                <p className="text-xs text-muted-foreground">{t("url404Hint")}</p>
              </section>
            )}

          {deployment.image_ref && (
            <section className="space-y-3">
              <h3 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                {t("image")}
              </h3>
              <code className="block break-all rounded-md border border-border/60 bg-card/40 px-3 py-2 font-mono text-xs">
                {deployment.image_ref}
              </code>
            </section>
          )}
        </div>

        <div className="flex justify-end gap-2 border-t border-border/60 pt-4">
          <Button variant="outline" size="sm" disabled>
            {t("rollback")}
            <RotateCcw className="ml-1 h-3.5 w-3.5" />
          </Button>
        </div>
      </SheetContent>
    </Sheet>
  );
}
