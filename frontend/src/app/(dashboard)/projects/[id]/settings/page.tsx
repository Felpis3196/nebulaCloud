"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { useProject } from "@/hooks/use-project";
import { api, ApiError } from "@/lib/api-client";
import type { Project } from "@/types/api";

export default function ProjectSettingsPage() {
  const params = useParams<{ id: string }>();
  const id = typeof params?.id === "string" ? params.id : "";
  const qc = useQueryClient();
  const { data: project, isLoading } = useProject(id || undefined);

  const [name, setName] = useState("");
  const [branch, setBranch] = useState("");
  const [desc, setDesc] = useState("");
  const [repoUrl, setRepoUrl] = useState("");
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (!project) return;
    setName(project.name);
    setBranch(project.default_branch);
    setDesc(project.description ?? "");
    setRepoUrl(project.repo_url ?? "");
  }, [project]);

  async function save(e: React.FormEvent) {
    e.preventDefault();
    if (!id) return;
    setSaving(true);
    try {
      const body: Record<string, string | undefined> = {
        name: name.trim(),
        default_branch: branch.trim() || "main",
      };
      const d = desc.trim();
      body.description = d === "" ? undefined : d;
      const r = repoUrl.trim();
      body.repo_url = r === "" ? "" : r;

      await api<Project>(`/api/v1/projects/${id}`, { method: "PATCH", body });
      await qc.invalidateQueries({ queryKey: ["project", id] });
      await qc.invalidateQueries({ queryKey: ["projects"] });
      toast.success("Project updated.");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Save failed");
    } finally {
      setSaving(false);
    }
  }

  if (isLoading || !project) {
    return <p className="text-sm text-muted-foreground">Loading…</p>;
  }

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader>
          <CardTitle>General</CardTitle>
          <CardDescription>
            Update display name, description, repository URL, and default branch. Slug stays fixed
            after creation.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={save} className="space-y-4">
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="name">Name</Label>
                <Input id="name" value={name} onChange={(e) => setName(e.target.value)} required />
              </div>
              <div className="space-y-2">
                <Label htmlFor="branch">Default branch</Label>
                <Input id="branch" value={branch} onChange={(e) => setBranch(e.target.value)} />
              </div>
            </div>
            <div className="space-y-2">
              <Label htmlFor="repo">Repository URL</Label>
              <Input
                id="repo"
                value={repoUrl}
                onChange={(e) => setRepoUrl(e.target.value)}
                placeholder="https://github.com/org/repo"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="desc">Description</Label>
              <Input id="desc" value={desc} onChange={(e) => setDesc(e.target.value)} />
            </div>
            <div className="flex justify-end">
              <Button type="submit" variant="gradient" size="sm" disabled={saving}>
                {saving ? "Saving…" : "Save changes"}
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>

      <Card className="border-destructive/40">
        <CardHeader>
          <CardTitle className="text-destructive">Danger zone</CardTitle>
          <CardDescription>
            Delete this project and all of its services, deployments, and audit history. API support
            is not implemented yet.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Button variant="destructive" size="sm" type="button" disabled title="Coming in a later release">
            <Trash2 className="h-4 w-4" /> Delete project
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}
