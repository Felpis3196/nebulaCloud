"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  Activity,
  BarChart3,
  Cloud,
  FolderGit2,
  Globe,
  LayoutDashboard,
  Settings,
  Terminal,
} from "lucide-react";
import { useTranslations } from "next-intl";
import { cn } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";

interface NavItem {
  href: string;
  labelKey: "overview" | "projects" | "deployments" | "logs" | "metrics" | "domains";
  icon: React.ComponentType<{ className?: string }>;
  badge?: string;
}

const NAV_PRIMARY: NavItem[] = [
  { href: "/dashboard", labelKey: "overview", icon: LayoutDashboard },
  { href: "/projects", labelKey: "projects", icon: FolderGit2 },
  { href: "/deployments", labelKey: "deployments", icon: Cloud },
];

const NAV_INFRA: NavItem[] = [
  { href: "/logs", labelKey: "logs", icon: Terminal },
  { href: "/metrics", labelKey: "metrics", icon: BarChart3 },
  { href: "/domains", labelKey: "domains", icon: Globe },
];

export function Sidebar() {
  const t = useTranslations("nav");
  const tCommon = useTranslations("common");
  const pathname = usePathname();
  const isActive = (href: string) =>
    pathname === href || pathname.startsWith(`${href}/`);

  return (
    <aside className="hidden w-60 shrink-0 flex-col border-r border-border/60 bg-card/40 backdrop-blur-md lg:flex">
      <div className="flex h-14 items-center gap-2 border-b border-border/60 px-5">
        <div className="flex h-7 w-7 items-center justify-center rounded-md gradient-primary">
          <Cloud className="h-4 w-4 text-white" />
        </div>
        <span className="text-sm font-semibold tracking-tight">NebulaCloud</span>
        <Badge variant="muted" className="ml-auto px-1.5 py-0 text-[10px]">
          {tCommon("beta")}
        </Badge>
      </div>

      <nav className="flex-1 space-y-6 overflow-y-auto px-3 py-5">
        <SectionLabel>{t("workspace")}</SectionLabel>
        <ul className="space-y-0.5">
          {NAV_PRIMARY.map((item) => (
            <NavRow
              key={item.href}
              {...item}
              label={t(item.labelKey)}
              active={isActive(item.href)}
            />
          ))}
        </ul>

        <SectionLabel>{t("operate")}</SectionLabel>
        <ul className="space-y-0.5">
          {NAV_INFRA.map((item) => (
            <NavRow
              key={item.href}
              {...item}
              label={t(item.labelKey)}
              active={isActive(item.href)}
            />
          ))}
        </ul>
      </nav>

      <div className="border-t border-border/60 p-3">
        <Link
          href="/settings"
          className={cn(
            "flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors",
            isActive("/settings")
              ? "bg-secondary/60 text-foreground"
              : "text-muted-foreground hover:bg-secondary/40 hover:text-foreground",
          )}
        >
          <Settings className="h-4 w-4" />
          {t("settings")}
        </Link>
        <div className="mt-3 flex items-center gap-2 rounded-md bg-secondary/30 px-3 py-2 text-xs text-muted-foreground">
          <Activity className="h-3 w-3 text-success" />
          {t("allSystemsNormal")}
        </div>
      </div>
    </aside>
  );
}

function SectionLabel({ children }: { children: React.ReactNode }) {
  return (
    <p className="mb-2 px-3 text-[10px] font-semibold uppercase tracking-[0.16em] text-muted-foreground/80">
      {children}
    </p>
  );
}

function NavRow({
  href,
  label,
  icon: Icon,
  badge,
  active,
}: Omit<NavItem, "labelKey"> & { label: string; active: boolean }) {
  return (
    <li>
      <Link
        href={href}
        className={cn(
          "group flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors",
          active
            ? "bg-primary/10 text-foreground"
            : "text-muted-foreground hover:bg-secondary/40 hover:text-foreground",
        )}
      >
        <Icon
          className={cn(
            "h-4 w-4 shrink-0 transition-colors",
            active ? "text-primary" : "text-muted-foreground group-hover:text-foreground",
          )}
        />
        <span className="flex-1 truncate">{label}</span>
        {badge && (
          <Badge variant="muted" className="px-1.5 py-0 text-[10px]">
            {badge}
          </Badge>
        )}
        {active && <span className="ml-auto h-1 w-1 rounded-full bg-primary" />}
      </Link>
    </li>
  );
}
