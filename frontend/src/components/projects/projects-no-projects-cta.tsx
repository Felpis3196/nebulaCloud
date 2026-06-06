"use client";

import { Github, Rocket } from "lucide-react";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useTranslations } from "next-intl";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { api, ApiError } from "@/lib/api-client";
import type { Project } from "@/types/api";

interface Props {
  organizationId: string;
  onProjectCreated?: () => void;
}

export function ProjectsNoProjectsCta({ organizationId, onProjectCreated }: Props) {
  const t = useTranslations("dashboard.projects");
  const router = useRouter();
  const qc = useQueryClient();
  const [busy, setBusy] = useState(false);

  async function createAndConnect() {
    setBusy(true);
    try {
      const suffix =
        typeof crypto !== "undefined" && "randomUUID" in crypto
          ? crypto.randomUUID().replace(/-/g, "").slice(0, 8)
          : String(Date.now());
      const slug = `app-${suffix}`;
      const project = await api<Project>(`/api/v1/organizations/${organizationId}/projects`, {
        method: "POST",
        body: { slug, name: "My app", default_branch: "main" },
      });
      if (!project?.id) {
        toast.error(t("createProjectFailed"));
        return;
      }
      qc.setQueryData(["project", project.id], project);
      await qc.invalidateQueries({ queryKey: ["projects"] });
      await onProjectCreated?.();
      router.push(`/projects/${project.id}?connect=1`);
    } catch (e) {
      if (e instanceof ApiError && e.message.toLowerCase().includes("membership")) {
        toast.error(t("membershipError"));
        return;
      }
      toast.error(e instanceof ApiError ? e.message : t("createProjectFailed"));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="rounded-xl border border-dashed border-border/80 bg-card/30 p-10 text-center">
      <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-primary/10 text-primary ring-1 ring-primary/20">
        <Github className="h-5 w-5" />
      </div>
      <h2 className="mt-4 text-lg font-semibold tracking-tight">{t("noProjectsTitle")}</h2>
      <p className="mx-auto mt-1.5 max-w-md text-sm text-muted-foreground">{t("noProjectsCtaDesc")}</p>
      <div className="mt-6 flex flex-wrap items-center justify-center gap-2">
        <Button variant="gradient" disabled={busy} onClick={() => void createAndConnect()}>
          <Rocket className="h-4 w-4" />
          {busy ? t("creatingProject") : t("createAndConnect")}
        </Button>
      </div>
    </div>
  );
}
