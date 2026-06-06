"use client";

import { useMemo, useState } from "react";
import { GitCommit, Search } from "lucide-react";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { StatusPill } from "@/components/dashboard/status-pill";
import { DeploymentDetailDrawer } from "@/components/deployments/deployment-detail-drawer";
import type { Deployment, DeploymentStatus } from "@/types/api";
import { formatDuration, relativeTime, shortSha } from "@/lib/utils";
import { useTranslations } from "next-intl";

const STATUS_VALUES = [
  "all",
  "running",
  "building",
  "deploying",
  "queued",
  "failed",
  "rolled_back",
] as const;

export function DeploymentsTable({ deployments }: { deployments: Deployment[] }) {
  const t = useTranslations("deployments.table");
  const tStatus = useTranslations("status");
  const tCommon = useTranslations("common");
  const [status, setStatus] = useState<DeploymentStatus | "all">("all");
  const [project, setProject] = useState<string>("all");
  const [query, setQuery] = useState("");
  const [selected, setSelected] = useState<Deployment | null>(null);

  const projects = Array.from(new Set(deployments.map((d) => d.project_name)));

  const rows = useMemo(() => {
    return deployments.filter((d) => {
      if (status !== "all" && d.status !== status) return false;
      if (project !== "all" && d.project_name !== project) return false;
      if (query) {
        const q = query.toLowerCase();
        const text = `${d.project_name} ${d.service_name} ${d.commit_message ?? ""} ${d.commit_sha ?? ""}`;
        if (!text.toLowerCase().includes(q)) return false;
      }
      return true;
    });
  }, [deployments, status, project, query]);

  return (
    <>
      <Card>
        <CardHeader className="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <CardTitle>{t("title")}</CardTitle>
            <CardDescription>{t("description")}</CardDescription>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <div className="relative">
              <Search className="pointer-events-none absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
              <Input
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                placeholder={t("search")}
                className="h-9 w-56 pl-8 text-sm"
              />
            </div>
            <Select value={status} onValueChange={(v) => setStatus(v as DeploymentStatus | "all")}>
              <SelectTrigger className="h-9 w-[170px] text-sm">
                <SelectValue placeholder={t("statusPlaceholder")} />
              </SelectTrigger>
              <SelectContent>
                {STATUS_VALUES.map((value) => (
                  <SelectItem key={value} value={value}>
                    {value === "all" ? t("allStatuses") : tStatus(value)}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Select value={project} onValueChange={setProject}>
              <SelectTrigger className="h-9 w-[170px] text-sm">
                <SelectValue placeholder={t("projectPlaceholder")} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">{t("allProjects")}</SelectItem>
                {projects.map((p) => (
                  <SelectItem key={p} value={p}>
                    {p}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </CardHeader>
        <CardContent className="px-0 pb-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("service")}</TableHead>
                <TableHead>{t("commit")}</TableHead>
                <TableHead>{tCommon("status")}</TableHead>
                <TableHead>{t("trigger")}</TableHead>
                <TableHead>{t("duration")}</TableHead>
                <TableHead>{t("when")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((d) => (
                <TableRow
                  key={d.id}
                  onClick={() => setSelected(d)}
                  className="cursor-pointer"
                >
                  <TableCell>
                    <div className="flex flex-col">
                      <span className="font-medium">{d.project_name}</span>
                      <span className="text-xs text-muted-foreground">{d.service_name}</span>
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-2">
                      <GitCommit className="h-3 w-3 text-muted-foreground" />
                      <span className="font-mono text-xs">{shortSha(d.commit_sha)}</span>
                    </div>
                    <p className="mt-0.5 max-w-[280px] truncate text-xs text-muted-foreground">
                      {d.commit_message}
                    </p>
                  </TableCell>
                  <TableCell>
                    <div className="space-y-1">
                      <StatusPill status={d.status} />
                      {d.status === "failed" && d.error_message && (
                        <p className="max-w-[220px] truncate text-[11px] text-destructive" title={d.error_message}>
                          {d.error_message}
                        </p>
                      )}
                    </div>
                  </TableCell>
                  <TableCell>
                    <Badge variant="muted">{d.trigger}</Badge>
                  </TableCell>
                  <TableCell className="font-mono text-xs">
                    {formatDuration(d.duration_ms ?? 0)}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {relativeTime(d.created_at)}
                  </TableCell>
                </TableRow>
              ))}
              {rows.length === 0 && (
                <TableRow>
                  <TableCell colSpan={6} className="py-12 text-center text-sm text-muted-foreground">
                    {t("noMatch")}
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      <DeploymentDetailDrawer
        deployment={selected}
        open={selected !== null}
        onOpenChange={(o) => !o && setSelected(null)}
      />
    </>
  );
}
