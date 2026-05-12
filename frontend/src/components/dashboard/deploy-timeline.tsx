import Link from "next/link";
import { ArrowRight, GitCommit } from "lucide-react";
import type { Deployment } from "@/types/api";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { StatusPill } from "@/components/dashboard/status-pill";
import { Button } from "@/components/ui/button";
import { cn, formatDuration, relativeTime, shortSha } from "@/lib/utils";

interface Props {
  deployments: Deployment[];
  limit?: number;
}

export function DeployTimeline({ deployments, limit = 8 }: Props) {
  const list = deployments.slice(0, limit);
  return (
    <Card>
      <CardHeader className="flex flex-row items-start justify-between gap-4">
        <div>
          <CardTitle>Recent deployments</CardTitle>
          <CardDescription>Latest activity across every service.</CardDescription>
        </div>
        <Button asChild variant="ghost" size="sm">
          <Link href="/deployments">
            All
            <ArrowRight />
          </Link>
        </Button>
      </CardHeader>
      <CardContent className="px-0 pb-0">
        <ul className="divide-y divide-border/60">
          {list.map((d) => (
            <li
              key={d.id}
              className="grid grid-cols-[auto_1fr_auto] items-center gap-3 px-5 py-3 transition-colors hover:bg-secondary/20"
            >
              <Dot status={d.status} />
              <div className="min-w-0">
                <div className="flex items-center gap-2 text-sm">
                  <span className="truncate font-medium">{d.project_name}</span>
                  <span className="text-muted-foreground">/</span>
                  <span className="truncate text-muted-foreground">{d.service_name}</span>
                </div>
                <div className="mt-0.5 flex items-center gap-2 text-xs text-muted-foreground">
                  <GitCommit className="h-3 w-3" />
                  <span className="font-mono">{shortSha(d.commit_sha)}</span>
                  <span className="truncate">{d.commit_message}</span>
                </div>
              </div>
              <div className="flex flex-col items-end gap-1.5">
                <StatusPill status={d.status} />
                <span className="text-[11px] text-muted-foreground">
                  {relativeTime(d.created_at)} · {formatDuration(d.duration_ms ?? 0)}
                </span>
              </div>
            </li>
          ))}
        </ul>
      </CardContent>
    </Card>
  );
}

function Dot({ status }: { status: Deployment["status"] }) {
  const color =
    status === "running"
      ? "bg-success"
      : status === "failed"
        ? "bg-destructive"
        : status === "rolled_back"
          ? "bg-info"
          : status === "queued" || status === "canceled"
            ? "bg-muted-foreground/60"
            : "bg-warning";
  return (
    <span className="relative flex h-2.5 w-2.5">
      <span
        className={cn("absolute inline-flex h-full w-full rounded-full opacity-50", color)}
      />
      <span className={cn("relative inline-flex h-2.5 w-2.5 rounded-full", color)} />
    </span>
  );
}
