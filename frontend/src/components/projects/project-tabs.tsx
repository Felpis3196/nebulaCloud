"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useTranslations } from "next-intl";
import { cn } from "@/lib/utils";

const TAB_SLUGS = ["", "deployments", "env", "logs", "metrics", "domains", "settings"] as const;
const TAB_KEYS = [
  "overview",
  "deployments",
  "environment",
  "logs",
  "metrics",
  "domains",
  "settings",
] as const;

export function ProjectTabs({ projectId }: { projectId: string }) {
  const t = useTranslations("projects.tabs");
  const pathname = usePathname();
  const base = `/projects/${projectId}`;

  return (
    <div className="border-b border-border/60">
      <nav className="-mb-px flex flex-wrap gap-1">
        {TAB_SLUGS.map((slug, i) => {
          const href = slug ? `${base}/${slug}` : base;
          const active =
            slug === "" ? pathname === base : pathname === href || pathname.startsWith(`${href}/`);
          return (
            <Link
              key={slug}
              href={href}
              className={cn(
                "relative px-3 py-2.5 text-sm font-medium transition-colors",
                active
                  ? "text-foreground"
                  : "text-muted-foreground hover:text-foreground",
              )}
            >
              {t(TAB_KEYS[i]!)}
              {active && (
                <span className="absolute inset-x-2 -bottom-px h-0.5 rounded-full bg-primary" />
              )}
            </Link>
          );
        })}
      </nav>
    </div>
  );
}
