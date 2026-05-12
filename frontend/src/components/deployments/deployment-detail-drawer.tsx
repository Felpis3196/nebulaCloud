"use client";

import { CheckCircle2, GitCommit, RotateCcw, Terminal } from "lucide-react";
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
import type { Deployment } from "@/types/api";
import { formatDuration, relativeTime, shortSha } from "@/lib/utils";

interface Props {
  deployment: Deployment | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

const PIPELINE = [
  { label: "Webhook received", duration: "120ms" },
  { label: "Source cloned", duration: "1.4s" },
  { label: "Stack detected", duration: "80ms" },
  { label: "Image built", duration: "32s" },
  { label: "Image pushed", duration: "4.1s" },
  { label: "Replicas rolled", duration: "9.6s" },
  { label: "Healthchecks", duration: "3.2s" },
];

export function DeploymentDetailDrawer({ deployment, open, onOpenChange }: Props) {
  if (!deployment) return null;

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
              <StatusPill status={deployment.status} />
              <Badge variant="muted" className="font-mono">
                {shortSha(deployment.commit_sha)}
              </Badge>
              <span className="text-xs">{relativeTime(deployment.created_at)}</span>
              <span className="text-xs">· {formatDuration(deployment.duration_ms ?? 0)}</span>
            </SheetDescription>
          </div>
        </SheetHeader>

        <div className="flex flex-1 flex-col gap-6 overflow-y-auto pr-1">
          <section className="space-y-2">
            <div className="flex items-center gap-2 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
              <GitCommit className="h-3 w-3" /> Commit
            </div>
            <p className="text-sm">{deployment.commit_message}</p>
            <div className="flex items-center gap-2 text-xs text-muted-foreground">
              <span className="font-mono">{deployment.ref}</span>
              <span>·</span>
              <span>{deployment.triggered_by?.email}</span>
            </div>
          </section>

          <Separator />

          <section className="space-y-3">
            <h3 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
              Pipeline
            </h3>
            <ol className="space-y-2">
              {PIPELINE.map((step, i) => (
                <li
                  key={step.label}
                  className="flex items-center gap-3 rounded-md border border-border/60 bg-card/40 px-3 py-2 text-sm"
                >
                  <CheckCircle2 className="h-4 w-4 shrink-0 text-success" />
                  <span className="font-mono text-xs text-muted-foreground">
                    {String(i + 1).padStart(2, "0")}
                  </span>
                  <span className="flex-1">{step.label}</span>
                  <span className="font-mono text-xs text-muted-foreground">{step.duration}</span>
                </li>
              ))}
            </ol>
          </section>

          <section className="space-y-3">
            <h3 className="flex items-center gap-2 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
              <Terminal className="h-3 w-3" /> Build log preview
            </h3>
            <div className="overflow-hidden rounded-md border border-border/60 bg-card/40 font-mono text-[12px]">
              {[
                "▸ Resolved Node 20.11 (paketo-buildpacks/nodejs)",
                "▸ Restored cache layer node_modules (sha256:8a3c2f0)",
                "▸ Installing dependencies  npm ci  --ignore-scripts",
                "✓ added 412 packages in 14s",
                "▸ Running build  npm run build",
                "✓ Compiled successfully in 9.2s",
                "▸ Exporting OCI image",
                "✓ Image pushed: registry/payments-api:8a3c2f0",
              ].map((line, i) => (
                <div
                  key={i}
                  className="whitespace-pre px-3 py-1 text-muted-foreground odd:bg-secondary/20"
                >
                  {line}
                </div>
              ))}
            </div>
          </section>

          <section className="space-y-3">
            <h3 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
              Image
            </h3>
            <code className="block break-all rounded-md border border-border/60 bg-card/40 px-3 py-2 font-mono text-xs">
              {deployment.image_ref}
            </code>
          </section>
        </div>

        <div className="flex justify-end gap-2 border-t border-border/60 pt-4">
          <Button variant="outline" size="sm">
            View logs
          </Button>
          <Button variant="gradient" size="sm">
            <RotateCcw />
            Rollback
          </Button>
        </div>
      </SheetContent>
    </Sheet>
  );
}
