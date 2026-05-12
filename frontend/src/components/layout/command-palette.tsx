"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { useTheme } from "next-themes";
import {
  BarChart3,
  Cloud,
  FolderGit2,
  Globe,
  LayoutDashboard,
  LogOut,
  Moon,
  Settings,
  Sun,
  Terminal,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  CommandSeparator,
  CommandShortcut,
} from "@/components/ui/command";
import { useAuthStore } from "@/stores/auth-store";

/**
 * Global cmd-K palette. Lists every dashboard route plus quick actions.
 *
 * Phase 6 will extend with project/service/deployment search by hitting the
 * backend; the surface is ready for it.
 */
export function CommandPalette() {
  const [open, setOpen] = useState(false);
  const router = useRouter();
  const { setTheme, resolvedTheme } = useTheme();
  const logout = useAuthStore((s) => s.logout);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "k") {
        e.preventDefault();
        setOpen((o) => !o);
      }
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, []);

  function go(path: string) {
    return () => {
      setOpen(false);
      router.push(path);
    };
  }

  async function handleLogout() {
    setOpen(false);
    await logout();
    router.push("/login");
  }

  return (
    <>
      <Button
        variant="outline"
        size="sm"
        className="hidden h-8 gap-2 bg-secondary/30 px-2.5 text-muted-foreground md:inline-flex"
        onClick={() => setOpen(true)}
      >
        <span className="text-xs">Search...</span>
        <kbd className="ml-2 hidden rounded border border-border/60 bg-card/40 px-1.5 py-0.5 text-[10px] text-muted-foreground sm:inline">
          ⌘K
        </kbd>
      </Button>

      <CommandDialog open={open} onOpenChange={setOpen}>
        <CommandInput placeholder="Search projects, deployments, settings..." />
        <CommandList>
          <CommandEmpty>No results found.</CommandEmpty>
          <CommandGroup heading="Navigate">
            <CommandItem onSelect={go("/dashboard")}>
              <LayoutDashboard /> Overview
            </CommandItem>
            <CommandItem onSelect={go("/projects")}>
              <FolderGit2 /> Projects
            </CommandItem>
            <CommandItem onSelect={go("/deployments")}>
              <Cloud /> Deployments
            </CommandItem>
            <CommandItem onSelect={go("/logs")}>
              <Terminal /> Logs
            </CommandItem>
            <CommandItem onSelect={go("/metrics")}>
              <BarChart3 /> Metrics
            </CommandItem>
            <CommandItem onSelect={go("/domains")}>
              <Globe /> Domains
            </CommandItem>
          </CommandGroup>
          <CommandSeparator />
          <CommandGroup heading="Account">
            <CommandItem onSelect={go("/settings")}>
              <Settings /> Settings
            </CommandItem>
            <CommandItem
              onSelect={() => setTheme(resolvedTheme === "dark" ? "light" : "dark")}
            >
              {resolvedTheme === "dark" ? <Sun /> : <Moon />}
              Toggle theme
              <CommandShortcut>⇧⌘L</CommandShortcut>
            </CommandItem>
            <CommandItem onSelect={handleLogout}>
              <LogOut /> Sign out
            </CommandItem>
          </CommandGroup>
        </CommandList>
      </CommandDialog>
    </>
  );
}
