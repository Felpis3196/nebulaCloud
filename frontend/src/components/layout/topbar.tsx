import { Bell, ExternalLink } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Breadcrumbs } from "@/components/layout/breadcrumbs";
import { CommandPalette } from "@/components/layout/command-palette";
import { ThemeToggle } from "@/components/layout/theme-toggle";
import { UserMenu } from "@/components/layout/user-menu";

export function Topbar() {
  return (
    <header className="sticky top-0 z-30 flex h-14 shrink-0 items-center gap-3 border-b border-border/60 bg-background/70 px-6 backdrop-blur-md">
      <div className="flex flex-1 items-center gap-3">
        <Breadcrumbs />
      </div>
      <div className="flex items-center gap-1">
        <CommandPalette />
        <Tooltip>
          <TooltipTrigger asChild>
            <Button variant="ghost" size="icon" aria-label="Documentation" asChild>
              <a href="https://github.com/nebulacloud/nebula" target="_blank" rel="noreferrer">
                <ExternalLink />
              </a>
            </Button>
          </TooltipTrigger>
          <TooltipContent>Documentation</TooltipContent>
        </Tooltip>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button variant="ghost" size="icon" aria-label="Notifications">
              <Bell />
            </Button>
          </TooltipTrigger>
          <TooltipContent>Notifications</TooltipContent>
        </Tooltip>
        <ThemeToggle />
        <div className="ml-2 h-6 w-px bg-border/60" />
        <UserMenu />
      </div>
    </header>
  );
}
