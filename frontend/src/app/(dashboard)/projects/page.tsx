"use client";

import { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useQueryClient } from "@tanstack/react-query";
import { ProjectCard } from "@/components/projects/project-card";
import { ProjectsEmptyState } from "@/components/projects/empty-state";
import { ProjectsNoProjectsCta } from "@/components/projects/projects-no-projects-cta";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useSyncSelectedOrganization } from "@/hooks/use-sync-selected-organization";
import { useOrganizations } from "@/hooks/use-organizations";
import { useProjects } from "@/hooks/use-projects";
import { useOrganizationStore } from "@/stores/org-store";
import { api, ApiError } from "@/lib/api-client";
import { ensurePersonalOrganization } from "@/lib/ensure-organization";
import type { Organization, Project } from "@/types/api";
import { useTranslations } from "next-intl";
import { toast } from "sonner";

export default function ProjectsPage() {
  const t = useTranslations("dashboard.projects");
  const tNav = useTranslations("nav");
  const tCommon = useTranslations("common");
  const qc = useQueryClient();
  const router = useRouter();
  const { data: orgs = [], isLoading: orgLoading } = useOrganizations();
  const selected = useOrganizationStore((s) => s.selectedOrganizationId);
  const setOrg = useOrganizationStore((s) => s.setSelectedOrganizationId);

  useSyncSelectedOrganization();

  const {
    data: projects = [],
    isLoading: projLoading,
    refetch,
  } = useProjects(selected);

  const [creatingOrg, setCreatingOrg] = useState(false);
  const [orgSlug, setOrgSlug] = useState("");
  const [orgName, setOrgName] = useState("");
  const [bootstrapBusy, setBootstrapBusy] = useState(false);

  const [creatingProject, setCreatingProject] = useState(false);
  const [projectSlug, setProjectSlug] = useState("");
  const [projectName, setProjectName] = useState("");
  const [projectBusy, setProjectBusy] = useState(false);

  async function refreshOrgs() {
    await qc.invalidateQueries({ queryKey: ["organizations"] });
    const list = await qc.fetchQuery({
      queryKey: ["organizations"],
      queryFn: () => api<Organization[]>("/api/v1/organizations"),
    });
    return list;
  }

  async function bootstrapOrg() {
    setBootstrapBusy(true);
    try {
      const list = await ensurePersonalOrganization();
      await qc.invalidateQueries({ queryKey: ["organizations"] });
      if (list.length > 0) {
        setOrg(list[0]!.id);
        toast.success(t("orgReady"));
      }
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : t("createOrgFailed"));
    } finally {
      setBootstrapBusy(false);
    }
  }

  async function createOrgFromForm(e: React.FormEvent) {
    e.preventDefault();
    try {
      const o = await api<Organization>("/api/v1/organizations", {
        method: "POST",
        body: { slug: orgSlug.trim(), name: orgName.trim() },
      });
      setOrgSlug("");
      setOrgName("");
      setCreatingOrg(false);
      setOrg(o.id);
      await refreshOrgs();
      toast.success(t("orgReady"));
    } catch (err) {
      if (err instanceof ApiError && err.status === 409) {
        const list = await refreshOrgs();
        if (list.length > 0) {
          setOrg(list[0]!.id);
          toast.message(t("orgResynced"));
        }
        return;
      }
      toast.error(err instanceof ApiError ? err.message : t("createOrgFailed"));
    }
  }

  async function createDemoProject() {
    if (!selected) {
      toast.error(t("selectOrgFirst"));
      return;
    }
    try {
      const suffix =
        typeof crypto !== "undefined" && "randomUUID" in crypto
          ? crypto.randomUUID().replace(/-/g, "").slice(0, 6)
          : String(Date.now());
      await api(`/api/v1/organizations/${selected}/projects`, {
        method: "POST",
        body: { slug: `demo-${suffix}`, name: "Demo", default_branch: "main" },
      });
      await refetch();
      toast.success(t("projectCreated"));
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : t("createProjectFailed"));
    }
  }

  async function createProjectFromForm(e: React.FormEvent) {
    e.preventDefault();
    if (!selected) {
      toast.error(t("selectOrgFirst"));
      return;
    }
    setProjectBusy(true);
    try {
      const project = await api<Project>(`/api/v1/organizations/${selected}/projects`, {
        method: "POST",
        body: {
          slug: projectSlug.trim(),
          name: projectName.trim() || projectSlug.trim(),
          default_branch: "main",
        },
      });
      if (!project?.id) {
        toast.error(t("createProjectFailed"));
        return;
      }
      qc.setQueryData(["project", project.id], project);
      await qc.invalidateQueries({ queryKey: ["projects"] });
      setProjectSlug("");
      setProjectName("");
      setCreatingProject(false);
      await refetch();
      toast.success(t("projectCreated"));
      router.push(`/projects/${project.id}?connect=1`);
    } catch (err) {
      if (err instanceof ApiError && err.message.toLowerCase().includes("membership")) {
        toast.error(t("membershipError"));
        const list = await refreshOrgs();
        if (list.length > 0) setOrg(list[0]!.id);
        return;
      }
      toast.error(err instanceof ApiError ? err.message : t("createProjectFailed"));
    } finally {
      setProjectBusy(false);
    }
  }

  const effectiveSelected =
    selected && orgs.some((o) => o.id === selected) ? selected : null;

  return (
    <div className="space-y-6">
      <header className="flex flex-wrap items-end justify-between gap-3">
        <div className="space-y-3">
          <h1 className="text-2xl font-semibold tracking-tight">{t("title")}</h1>
          <p className="text-sm text-muted-foreground">{t("subtitle")}</p>
          {!orgLoading && orgs.length > 0 && (
            <Select
              value={effectiveSelected ?? undefined}
              onValueChange={(v) => setOrg(v)}
            >
              <SelectTrigger className="w-64">
                <SelectValue placeholder={t("organization")} />
              </SelectTrigger>
              <SelectContent>
                {orgs.map((o) => (
                  <SelectItem key={o.id} value={o.id}>
                    {o.name} ({o.slug})
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          )}
        </div>
        <div className="flex flex-wrap gap-2">
          <Button variant="outline" asChild>
            <Link href="/deployments">{tNav("deployments")}</Link>
          </Button>
          {orgs.length === 0 ? (
            <Button onClick={() => void bootstrapOrg()} disabled={bootstrapBusy}>
              {bootstrapBusy ? t("creatingOrg") : t("bootstrap")}
            </Button>
          ) : (
            <>
              {!creatingOrg ? (
                <Button variant="secondary" onClick={() => setCreatingOrg(true)}>
                  {t("newOrg")}
                </Button>
              ) : (
                <form onSubmit={createOrgFromForm} className="flex flex-wrap gap-2">
                  <Input placeholder={t("slug")} value={orgSlug} onChange={(e) => setOrgSlug(e.target.value)} required />
                  <Input placeholder={t("orgName")} value={orgName} onChange={(e) => setOrgName(e.target.value)} required />
                  <Button type="submit">{tCommon("save")}</Button>
                </form>
              )}
              {!creatingProject ? (
                <Button variant="gradient" disabled={!effectiveSelected} onClick={() => setCreatingProject(true)}>
                  {t("newProject")}
                </Button>
              ) : (
                <form onSubmit={createProjectFromForm} className="flex flex-wrap gap-2">
                  <Input
                    placeholder={t("projectSlug")}
                    value={projectSlug}
                    onChange={(e) => setProjectSlug(e.target.value)}
                    required
                  />
                  <Input
                    placeholder={t("projectName")}
                    value={projectName}
                    onChange={(e) => setProjectName(e.target.value)}
                  />
                  <Button type="submit" disabled={projectBusy || !effectiveSelected}>
                    {projectBusy ? t("creatingProject") : tCommon("save")}
                  </Button>
                  <Button type="button" variant="outline" onClick={() => setCreatingProject(false)}>
                    {tCommon("cancel")}
                  </Button>
                </form>
              )}
              <Button variant="outline" disabled={!effectiveSelected} onClick={() => void createDemoProject()}>
                {t("demoProject")}
              </Button>
            </>
          )}
        </div>
      </header>

      {orgLoading || projLoading ? (
        <div className="text-sm text-muted-foreground">{tCommon("loading")}</div>
      ) : !effectiveSelected ? (
        <ProjectsEmptyState />
      ) : projects.length === 0 ? (
        <ProjectsNoProjectsCta organizationId={effectiveSelected} onProjectCreated={() => void refetch()} />
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
          {projects.map((p) => (
            <ProjectCard key={p.id} project={p} />
          ))}
        </div>
      )}
    </div>
  );
}
