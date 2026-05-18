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
import { useTranslations } from "next-intl";

const webhookURL = `${env.NEXT_PUBLIC_API_URL}/api/v1/webhooks/github`;

interface Props {
  /** Required to PATCH repo_url / default_branch on the project. */
  projectId?: string;
  trigger?: React.ReactNode;
}

export function ConnectRepoDialog({ projectId, trigger }: Props) {
  const t = useTranslations("projects.connectRepo");
  const tCommon = useTranslations("common");
  const qc = useQueryClient();
  const [open, setOpen] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [repo, setRepo] = useState("");
  const [branch, setBranch] = useState("main");
  const [installationId, setInstallationId] = useState("");

  async function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (!projectId) {
      toast.error(t("needProject"));
      return;
    }
    setSubmitting(true);
    try {
      const trimmedInst = installationId.trim();
      if (trimmedInst !== "") {
        const n = Number.parseInt(trimmedInst, 10);
        if (!Number.isFinite(n) || n <= 0) {
          toast.error(t("invalidInstallation"));
          return;
        }
      }
      await api<Project>(`/api/v1/projects/${projectId}`, {
        method: "PATCH",
        body: {
          repo_url: repo.trim(),
          default_branch: branch.trim() || "main",
        },
      });
      if (trimmedInst !== "") {
        const n = Number.parseInt(trimmedInst, 10);
        await api<Project>(`/api/v1/projects/${projectId}/github-installation`, {
          method: "POST",
          body: { installation_id: n },
        });
      }
      await qc.invalidateQueries({ queryKey: ["project", projectId] });
      await qc.invalidateQueries({ queryKey: ["projects"] });
      setOpen(false);
      toast.success(trimmedInst ? t("savedWithInstall") : t("savedRepoOnly"));
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : t("saveFailed"));
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
          setInstallationId("");
        }
      }}
    >
      <DialogTrigger asChild>
        {trigger ?? (
          <Button variant="outline" size="sm" type="button" disabled={!projectId}>
            <Github className="h-4 w-4" /> {t("connectButton")}
          </Button>
        )}
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("dialogTitle")}</DialogTitle>
          <DialogDescription>{t("dialogDescription")}</DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="nebula-webhook-url">{t("webhookEndpoint")}</Label>
            <Input id="nebula-webhook-url" readOnly className="font-mono text-xs" value={webhookURL} />
            <p className="text-xs text-muted-foreground">{t("webhookHint")}</p>
          </div>
          <div className="space-y-2">
            <Label htmlFor="github-installation-id">{t("installationId")}</Label>
            <Input
              id="github-installation-id"
              inputMode="numeric"
              placeholder={t("installationPlaceholder")}
              value={installationId}
              onChange={(e) => setInstallationId(e.target.value)}
            />
            <p className="text-xs text-muted-foreground">{t("installationHintLong")}</p>
          </div>
          <div className="space-y-2">
            <Label htmlFor="repo">{t("repoUrl")}</Label>
            <Input
              id="repo"
              required
              placeholder={t("repoPlaceholder")}
              value={repo}
              onChange={(e) => setRepo(e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="branch">{t("defaultBranch")}</Label>
            <Input id="branch" value={branch} onChange={(e) => setBranch(e.target.value)} />
          </div>
          <DialogFooter>
            <DialogClose asChild>
              <Button type="button" variant="outline">
                {tCommon("cancel")}
              </Button>
            </DialogClose>
            <Button type="submit" variant="gradient" disabled={submitting || !projectId} className="inline-flex items-center gap-2">
              {submitting ? <Loader2 className="h-4 w-4 animate-spin" /> : null}
              {submitting ? t("saving") : tCommon("save")}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
