"use client";

import { ExternalLink, Server } from "lucide-react";
import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { useQueryClient } from "@tanstack/react-query";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { StatusPill } from "@/components/dashboard/status-pill";
import { Button } from "@/components/ui/button";
import { relativeTime } from "@/lib/utils";
import { useProject } from "@/hooks/use-project";
import { useServices } from "@/hooks/use-services";
import { api, ApiError } from "@/lib/api-client";
import { toast } from "sonner";
import { useTranslations } from "next-intl";

export default function ProjectOverviewPage() {
  const t = useTranslations("projects.detail");
  const tCommon = useTranslations("common");
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const qc = useQueryClient();
  const id = typeof params?.id === "string" ? params.id : "";

  const { data: project, isLoading } = useProject(id);
  const { data: services = [] } = useServices(id);

  if (isLoading || !project) {
    return <div className="text-sm text-muted-foreground">{tCommon("loading")}</div>;
  }

  return (
    <div className="grid gap-4 lg:grid-cols-3">
      <Card className="lg:col-span-2">
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle>{t("services")}</CardTitle>
          <Button
            size="sm"
            variant="secondary"
            onClick={async () => {
              if (!id) return;
              const suffix =
                typeof crypto !== "undefined" && "randomUUID" in crypto
                  ? crypto.randomUUID().replace(/-/g, "").slice(0, 8)
                  : String(Date.now());
              const slug = `web-${suffix}`;
              try {
                await api(`/api/v1/projects/${id}/services`, {
                  method: "POST",
                  body: { slug, name: "Web", type: "web" },
                });
                await qc.invalidateQueries({ queryKey: ["services", id] });
                await qc.invalidateQueries({ queryKey: ["project", id] });
                await qc.invalidateQueries({ queryKey: ["projects"] });
                toast.success(t("serviceCreated", { slug }));
              } catch (e) {
                toast.error(e instanceof ApiError ? e.message : t("createServiceFailed"));
              }
            }}
          >
            {t("addWebService")}
          </Button>
        </CardHeader>
        <CardContent className="px-0 pb-0">
          <ul className="divide-y divide-border/60">
            {services.map((s) => (
              <li
                key={s.id}
                className="grid grid-cols-[auto_1fr_auto] items-center gap-3 px-5 py-3"
              >
                <div className="flex h-8 w-8 items-center justify-center rounded-md bg-primary/10 text-primary ring-1 ring-primary/20">
                  <Server className="h-3.5 w-3.5" />
                </div>
                <div className="min-w-0">
                  <div className="flex items-center gap-2 text-sm font-medium">
                    {s.name}
                    <span className="text-xs font-normal text-muted-foreground">{s.type}</span>
                  </div>
                  <div className="flex items-center gap-3 text-xs text-muted-foreground">
                    <span className="font-mono">{s.current_image ?? "—"}</span>
                    <span>{t("replicas", { count: s.replicas })}</span>
                    <span>{t("updatedAgo", { time: relativeTime(s.updated_at) })}</span>
                  </div>
                </div>
                <div className="flex flex-col items-end gap-2">
                  {s.url && (
                    <a
                      href={s.url}
                      target="_blank"
                      rel="noreferrer"
                      className="inline-flex items-center gap-1 text-xs text-muted-foreground transition-colors hover:text-foreground"
                    >
                      {t("open")} <ExternalLink className="h-3 w-3" />
                    </a>
                  )}
                  <Button size="sm" variant="outline" asChild className="h-7 px-2 text-[11px]">
                    <Link href={`/deployments?s=${encodeURIComponent(s.id)}`}>{t("deployLog")}</Link>
                  </Button>
                  <Button
                    size="sm"
                    variant="default"
                    className="h-7 px-2 text-[11px]"
                    onClick={async () => {
                      await api(`/api/v1/services/${s.id}/deployments`, {
                        method: "POST",
                        body: {},
                      });
                      await qc.invalidateQueries({ queryKey: ["deployments"] });
                      router.push(`/projects/${id}/deployments`);
                    }}
                  >
                    {t("deploy")}
                  </Button>
                  <StatusPill status={s.status} />
                </div>
              </li>
            ))}
          </ul>
          {services.length === 0 && (
            <p className="px-5 pb-5 text-sm text-muted-foreground">{t("noServices")}</p>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{t("quickStats")}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3 text-sm">
          <Row label={t("created")} value={relativeTime(project.created_at)} />
          <Row label={t("lastUpdated")} value={relativeTime(project.updated_at)} />
          <Row label={t("defaultBranch")} value={project.default_branch} mono />
          <Row label={t("services")} value={String(project.services_count)} />
        </CardContent>
      </Card>
    </div>
  );
}

function Row({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="flex items-center justify-between gap-4">
      <span className="text-muted-foreground">{label}</span>
      <span className={mono ? "font-mono text-xs" : "text-right"}>{value}</span>
    </div>
  );
}
