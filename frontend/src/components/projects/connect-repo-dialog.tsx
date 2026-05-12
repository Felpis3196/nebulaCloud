"use client";

import { useState } from "react";
import { Github, Loader2 } from "lucide-react";
import { useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { api, ApiError } from "@/lib/api-client";
import { env } from "@/lib/env";
import type { Project } from "@/types/api";

const webhookURL = `${env.NEXT_PUBLIC_API_URL}/api/v1/webhooks/github`;

interface Props {
  /** Required to PATCH repo_url / default_branch on the project. */
  projectId?: string;
  trigger?: React.ReactNode;
}

export function ConnectRepoDialog({ projectId, trigger }: Props) {
  const qc = useQueryClient();
  const [open, setOpen] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [repo, setRepo] = useState("");
  const [branch, setBranch] = useState("main");

  async function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (!projectId) {
      toast.error("Open this dialog from a project to save the repository URL.");
      return;
    }
    setSubmitting(true);
    try {
      await api<Project>(`/api/v1/projects/${projectId}`, {
        method: "PATCH",
        body: {
          repo_url: repo.trim(),
          default_branch: branch.trim() || "main",
        },
      });
      await qc.invalidateQueries({ queryKey: ["project", projectId] });
      await qc.invalidateQueries({ queryKey: ["projects"] });
      setOpen(false);
      toast.success("Repository saved. Add the webhook URL in GitHub if you want automatic deploys.");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Could not save repository");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        setOpen(next);
        if (next) {
          setRepo("");
          setBranch("main");
        }
      }}
    >
      <DialogTrigger asChild>
        {trigger ?? (
          <Button variant="outline" size="sm" type="button" disabled={!projectId}>
            <Github className="h-4 w-4" /> Connect repository
          </Button>
        )}
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Connect a repository</DialogTitle>
          <DialogDescription>
            Saves the clone URL and default branch on this project. Configure a GitHub webhook
            below to trigger deploys on push (Phase 3 will automate this).
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="nebula-webhook-url">Webhook endpoint (GitHub → Nebula)</Label>
            <Input id="nebula-webhook-url" readOnly className="font-mono text-xs" value={webhookURL} />
            <p className="text-xs text-muted-foreground">
              Use Content type <span className="font-mono">application/json</span>, secret from{" "}
              <span className="font-mono">NEBULA_GITHUB_APP_WEBHOOK_SECRET</span>, events:{" "}
              <span className="font-mono">push</span>.
            </p>
          </div>
          <div className="space-y-2">
            <Label htmlFor="repo">Repository URL</Label>
            <Input
              id="repo"
              required
              placeholder="https://github.com/your-org/your-repo"
              value={repo}
              onChange={(e) => setRepo(e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="branch">Default branch</Label>
            <Input id="branch" value={branch} onChange={(e) => setBranch(e.target.value)} />
          </div>
          <DialogFooter>
            <DialogClose asChild>
              <Button type="button" variant="outline">
                Cancel
              </Button>
            </DialogClose>
            <Button type="submit" variant="gradient" disabled={submitting || !projectId} className="inline-flex items-center gap-2">
              {submitting ? <Loader2 className="h-4 w-4 animate-spin" /> : null}
              Save
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
