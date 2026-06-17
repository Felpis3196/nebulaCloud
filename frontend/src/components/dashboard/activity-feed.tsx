"use client";

import { AlertTriangle, Loader2, Rocket } from "lucide-react";
import Link from "next/link";
import { useTranslations } from "next-intl";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { relativeTime } from "@/lib/utils";
import type { Deployment, DeploymentStatus } from "@/types/api";

const ACTIVE: DeploymentStatus[] = ["queued", "building", "pushing", "deploying"];

interface Props {
  deployments: Deployment[];
  loading?: boolean;
  limit?: number;
}

export function ActivityFeed({ deployments, loading = false, limit = 10 }: Props) {
  const t = useTranslations("dashboard.activity");
  const items = deployments.slice(0, limit);

  return (
    <Card className="h-full">
      <CardHeader>
        <CardTitle>{t("title")}</CardTitle>
        <CardDescription>
          {t("description")}{" "}
          <Link href="/deployments" className="underline underline-offset-2">
            {t("deploymentsLink")}
          </Link>
          .
        </CardDescription>
      </CardHeader>
      <CardContent className="px-0 pb-2">
        {loading ? (
          <div className="flex items-center justify-center gap-2 px-5 py-12 text-sm text-muted-foreground">
            <Loader2 className="h-4 w-4 animate-spin" />
            {t("loading")}
          </div>
        ) : items.length === 0 ? (
          <div className="flex flex-col items-center gap-2 px-5 py-12 text-center">
            <Rocket className="h-8 w-8 text-muted-foreground/50" />
            <p className="max-w-xs text-sm text-muted-foreground">{t("empty")}</p>
          </div>
        ) : (
          <ul className="space-y-0.5">
            {items.map((d) => (
              <ActivityRow key={d.id} deployment={d} />
            ))}
          </ul>
        )}
      </CardContent>
    </Card>
  );
}

function ActivityRow({ deployment: d }: { deployment: Deployment }) {
  const t = useTranslations("dashboard.activity");
  const { icon: Icon, iconClass } = statusVisual(d.status);
  const actor = d.triggered_by?.email;

  let message: string;
  if (d.status === "running") {
    message = t("deployed", { project: d.project_name, service: d.service_name });
  } else if (d.status === "failed") {
    message = t("failed", { project: d.project_name, service: d.service_name });
  } else if (ACTIVE.includes(d.status)) {
    message = t("inProgress", { project: d.project_name, service: d.service_name });
  } else {
    message = t("other", { project: d.project_name, service: d.service_name, status: d.status });
  }

  return (
    <li className="flex items-start gap-3 px-5 py-2.5">
      <div
        className={`mt-0.5 flex h-7 w-7 shrink-0 items-center justify-center rounded-full ${iconClass}`}
      >
        <Icon className={`h-3.5 w-3.5 ${ACTIVE.includes(d.status) ? "animate-spin" : ""}`} />
      </div>
      <div className="min-w-0 flex-1">
        <p className="text-sm leading-tight">
          <span className="font-medium text-foreground">{message}</span>
          {actor && (
            <span className="text-muted-foreground">
              {" "}
              · {actor}
            </span>
          )}
        </p>
        <p className="mt-0.5 text-[11px] text-muted-foreground">
          {relativeTime(d.created_at)}
          {d.trigger ? ` · ${d.trigger}` : ""}
        </p>
      </div>
    </li>
  );
}

function statusVisual(status: DeploymentStatus): {
  icon: typeof Rocket;
  iconClass: string;
} {
  if (status === "running") {
    return { icon: Rocket, iconClass: "bg-success/15 text-success" };
  }
  if (status === "failed") {
    return { icon: AlertTriangle, iconClass: "bg-destructive/15 text-destructive" };
  }
  if (ACTIVE.includes(status)) {
    return { icon: Loader2, iconClass: "bg-warning/15 text-warning" };
  }
  return { icon: Rocket, iconClass: "bg-muted/40 text-muted-foreground" };
}
