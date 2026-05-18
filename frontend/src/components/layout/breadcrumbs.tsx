"use client";

import { Fragment } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { ChevronRight } from "lucide-react";
import { useTranslations } from "next-intl";

const SEGMENT_KEYS = [
  "dashboard",
  "projects",
  "deployments",
  "logs",
  "metrics",
  "domains",
  "settings",
  "env",
] as const;

type SegmentKey = (typeof SEGMENT_KEYS)[number];

function isSegmentKey(slug: string): slug is SegmentKey {
  return (SEGMENT_KEYS as readonly string[]).includes(slug);
}

export function Breadcrumbs() {
  const t = useTranslations("nav");
  const pathname = usePathname();
  const segments = pathname.split("/").filter(Boolean);

  if (segments.length === 0) return null;

  function humanise(slug: string) {
    if (slug === "dashboard") return t("overview");
    if (isSegmentKey(slug)) return t(slug === "env" ? "environment" : slug);
    return slug.replace(/-/g, " ");
  }

  return (
    <nav aria-label={t("breadcrumb")} className="flex items-center gap-1.5 text-sm">
      {segments.map((seg, i) => {
        const href = "/" + segments.slice(0, i + 1).join("/");
        const isLast = i === segments.length - 1;
        const label = humanise(seg);
        return (
          <Fragment key={href}>
            {i > 0 && <ChevronRight className="h-3.5 w-3.5 text-muted-foreground/50" />}
            {isLast ? (
              <span className="font-medium text-foreground">{label}</span>
            ) : (
              <Link
                href={href}
                className="text-muted-foreground transition-colors hover:text-foreground"
              >
                {label}
              </Link>
            )}
          </Fragment>
        );
      })}
    </nav>
  );
}
