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

const STATUSES: { value: DeploymentStatus | "all"; label: string }[] = [
  { value: "all", label: "All statuses" },
  { value: "running", label: "Running" },
  { value: "building", label: "Building" },
  { value: "deploying", label: "Deploying" },
  { value: "queued", label: "Queued" },
  { value: "failed", label: "Failed" },
  { value: "rolled_back", label: "Rolled back" },
];

export function DeploymentsTable({ deployments }: { deployments: Deployment[] }) {
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
            <CardTitle>Deployments</CardTitle>
            <CardDescription>
              Every deploy across your workspace, newest first. Click a row to inspect its pipeline.
            </CardDescription>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <div className="relative">
              <Search className="pointer-events-none absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
              <Input
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                placeholder="Search commit, service..."
                className="h-9 w-56 pl-8 text-sm"
              />
            </div>
            <Select value={status} onValueChange={(v) => setStatus(v as DeploymentStatus | "all")}>
              <SelectTrigger className="h-9 w-[170px] text-sm">
                <SelectValue placeholder="Status" />
              </SelectTrigger>
              <SelectContent>
                {STATUSES.map((s) => (
                  <SelectItem key={s.value} value={s.value}>
                    {s.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Select value={project} onValueChange={setProject}>
              <SelectTrigger className="h-9 w-[170px] text-sm">
                <SelectValue placeholder="Project" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All projects</SelectItem>
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
                <TableHead>Service</TableHead>
                <TableHead>Commit</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Trigger</TableHead>
                <TableHead>Duration</TableHead>
                <TableHead>When</TableHead>
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
                    <StatusPill status={d.status} />
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
                    No deployments match your filters.
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
