"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { ProjectCard } from "@/components/projects/project-card";
import { ProjectsEmptyState } from "@/components/projects/empty-state";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useOrganizations } from "@/hooks/use-organizations";
import { useProjects } from "@/hooks/use-projects";
import { useOrganizationStore } from "@/stores/org-store";
import { api } from "@/lib/api-client";
import type { Organization } from "@/types/api";
import { useTranslations } from "next-intl";

export default function ProjectsPage() {
  const t = useTranslations("dashboard.projects");
  const tNav = useTranslations("nav");
  const tCommon = useTranslations("common");
  const { data: orgs = [], isLoading: orgLoading } = useOrganizations();
  const selected = useOrganizationStore((s) => s.selectedOrganizationId);
  const setOrg = useOrganizationStore((s) => s.setSelectedOrganizationId);

  useEffect(() => {
    if (!selected && orgs.length > 0) {
      setOrg(orgs[0]!.id);
    }
  }, [orgs, selected, setOrg]);

  const {
    data: projects = [],
    isLoading: projLoading,
    refetch,
  } = useProjects(selected);

  const [creating, setCreating] = useState(false);
  const [slug, setSlug] = useState("");
  const [name, setName] = useState("");

  async function bootstrapOrg() {
    const o = await api<Organization>("/api/v1/organizations", {
      method: "POST",
      body: { slug: "default", name: "Default Org" },
    });
    setOrg(o.id);
  }

  async function createOrgFromForm(e: React.FormEvent) {
    e.preventDefault();
    const o = await api<Organization>("/api/v1/organizations", {
      method: "POST",
      body: { slug, name },
    });
    setSlug("");
    setName("");
    setCreating(false);
    setOrg(o.id);
  }

  async function createDemoProject() {
    if (!selected) return;
    await api(`/api/v1/organizations/${selected}/projects`, {
      method: "POST",
      body: { slug: "demo", name: "Demo", default_branch: "main" },
    });
    await refetch();
  }

  return (
    <div className="space-y-6">
      <header className="flex flex-wrap items-end justify-between gap-3">
        <div className="space-y-3">
          <h1 className="text-2xl font-semibold tracking-tight">{t("title")}</h1>
          <p className="text-sm text-muted-foreground">{t("subtitle")}</p>
          {!orgLoading && orgs.length > 0 && (
            <Select
              value={selected ?? undefined}
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
            <Button onClick={bootstrapOrg}>{t("bootstrap")}</Button>
          ) : (
            <>
              {!creating ? (
                <Button variant="secondary" onClick={() => setCreating(true)}>
                  {t("newOrg")}
                </Button>
              ) : (
                <form onSubmit={createOrgFromForm} className="flex flex-wrap gap-2">
                  <Input placeholder={t("slug")} value={slug} onChange={(e) => setSlug(e.target.value)} />
                  <Input placeholder={t("orgName")} value={name} onChange={(e) => setName(e.target.value)} />
                  <Button type="submit">{tCommon("save")}</Button>
                </form>
              )}
              <Button variant="outline" disabled={!selected} onClick={createDemoProject}>
                {t("demoProject")}
              </Button>
            </>
          )}
        </div>
      </header>

      {orgLoading || projLoading ? (
        <div className="text-sm text-muted-foreground">{tCommon("loading")}</div>
      ) : !selected ? (
        <ProjectsEmptyState />
      ) : projects.length === 0 ? (
        <div className="rounded-lg border border-dashed p-8 text-center text-sm text-muted-foreground">
          {t("noProjectsYet")}
        </div>
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
