"use client";

import Link from "next/link";
import { useTranslations } from "next-intl";
import { ArrowRight, GitCommit, Rocket } from "lucide-react";
import type { Deployment } from "@/types/api";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { StatusPill } from "@/components/dashboard/status-pill";
import { Button } from "@/components/ui/button";
import { cn, formatDuration, relativeTime, shortSha } from "@/lib/utils";

interface Props {
  deployments: Deployment[];
  limit?: number;
  loading?: boolean;
  timelineDescription?: string;
}

export function DeployTimeline({
  deployments,
  limit = 8,
  loading = false,
  timelineDescription,
}: Props) {
  const t = useTranslations("dashboard.deployTimeline");
  const list = deployments.slice(0, limit);

  return (
    <Card>
      <CardHeader className="flex flex-row items-start justify-between gap-4">
        <div>
          <CardTitle>{t("title")}</CardTitle>
          <CardDescription>
            {timelineDescription ?? t("defaultDesc")}
          </CardDescription>
        </div>
        <Button asChild variant="ghost" size="sm">
          <Link href="/deployments">
            {t("all")}
            <ArrowRight />
          </Link>
        </Button>
      </CardHeader>
      <CardContent className="px-0 pb-0">
        {loading ? (
          <p className="px-5 py-12 text-center text-sm text-muted-foreground">{t("loading")}</p>
        ) : list.length === 0 ? (
          <div className="flex flex-col items-center gap-3 px-5 py-12 text-center">
            <Rocket className="h-8 w-8 text-muted-foreground/50" />
            <p className="max-w-sm text-sm text-muted-foreground">{t("empty")}</p>
            <Button asChild variant="outline" size="sm">
              <Link href="/projects">{t("emptyAction")}</Link>
            </Button>
          </div>
        ) : (
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
        )}
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
