"use client";

import { Bell, ExternalLink } from "lucide-react";
import { useTranslations } from "next-intl";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Breadcrumbs } from "@/components/layout/breadcrumbs";
import { CommandPalette } from "@/components/layout/command-palette";
import { LocaleSwitcher } from "@/components/layout/locale-switcher";
import { ThemeSwitcher } from "@/components/layout/theme-switcher";
import { UserMenu } from "@/components/layout/user-menu";

export function Topbar() {
  const t = useTranslations("nav");

  return (
    <header className="sticky top-0 z-30 flex h-14 shrink-0 items-center gap-3 border-b border-border/60 bg-background/70 px-6 backdrop-blur-md">
      <div className="flex flex-1 items-center gap-3">
        <Breadcrumbs />
      </div>
      <div className="flex items-center gap-1">
        <CommandPalette />
        <Tooltip>
          <TooltipTrigger asChild>
            <Button variant="ghost" size="icon" aria-label={t("documentation")} asChild>
              <a href="https://github.com/nebulacloud/nebula" target="_blank" rel="noreferrer">
                <ExternalLink />
              </a>
            </Button>
          </TooltipTrigger>
          <TooltipContent>{t("documentation")}</TooltipContent>
        </Tooltip>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button variant="ghost" size="icon" aria-label={t("notifications")}>
              <Bell />
            </Button>
          </TooltipTrigger>
          <TooltipContent>{t("notifications")}</TooltipContent>
        </Tooltip>
        <LocaleSwitcher />
        <ThemeSwitcher />
        <div className="ml-2 h-6 w-px bg-border/60" />
        <UserMenu />
      </div>
    </header>
  );
}
