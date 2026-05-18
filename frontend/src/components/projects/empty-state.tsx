"use client";

import { FolderPlus } from "lucide-react";
import { useTranslations } from "next-intl";

export function ProjectsEmptyState() {
  const t = useTranslations("dashboard.projects");

  return (
    <div className="relative overflow-hidden rounded-xl border border-dashed border-border/80 bg-card/30 p-10 text-center">
      <div
        aria-hidden
        className="absolute inset-0 -z-10"
        style={{
          backgroundImage:
            "radial-gradient(40% 50% at 50% 0%, hsl(239 84% 60% / 0.16), transparent)",
        }}
      />
      <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-primary/10 text-primary ring-1 ring-primary/20">
        <FolderPlus className="h-5 w-5" />
      </div>
      <h2 className="mt-4 text-lg font-semibold tracking-tight">{t("emptyTitle")}</h2>
      <p className="mx-auto mt-1.5 max-w-md text-sm text-muted-foreground">
        {t.rich("emptyDesc", {
          connect: (chunks) => (
            <span className="font-medium text-foreground">{chunks}</span>
          ),
        })}
      </p>
    </div>
  );
}
