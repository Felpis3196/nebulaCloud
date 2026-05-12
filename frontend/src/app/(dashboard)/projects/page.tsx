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

export default function ProjectsPage() {
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
          <h1 className="text-2xl font-semibold tracking-tight">Projects</h1>
          <p className="text-sm text-muted-foreground">
            Group services that share a repo, secrets, and team.
          </p>
          {!orgLoading && orgs.length > 0 && (
            <Select
              value={selected ?? undefined}
              onValueChange={(v) => setOrg(v)}
            >
              <SelectTrigger className="w-64">
                <SelectValue placeholder="Organization" />
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
            <Link href="/deployments">Deployments</Link>
          </Button>
          {orgs.length === 0 ? (
            <Button onClick={bootstrapOrg}>Create default org</Button>
          ) : (
            <>
              {!creating ? (
                <Button variant="secondary" onClick={() => setCreating(true)}>
                  New org
                </Button>
              ) : (
                <form onSubmit={createOrgFromForm} className="flex flex-wrap gap-2">
                  <Input placeholder="slug" value={slug} onChange={(e) => setSlug(e.target.value)} />
                  <Input placeholder="name" value={name} onChange={(e) => setName(e.target.value)} />
                  <Button type="submit">Save</Button>
                </form>
              )}
              <Button variant="outline" disabled={!selected} onClick={createDemoProject}>
                New demo project
              </Button>
            </>
          )}
        </div>
      </header>

      {orgLoading || projLoading ? (
        <div className="text-sm text-muted-foreground">Loading…</div>
      ) : !selected ? (
        <ProjectsEmptyState />
      ) : projects.length === 0 ? (
        <div className="rounded-lg border border-dashed p-8 text-center text-sm text-muted-foreground">
          No projects yet — add a starter project above.
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
