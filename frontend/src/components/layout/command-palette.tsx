"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { useTheme } from "next-themes";
import { useTranslations } from "next-intl";
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

export function CommandPalette() {
  const t = useTranslations("command");
  const tNav = useTranslations("nav");
  const tTheme = useTranslations("theme");
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
        <span className="text-xs">{t("search")}</span>
        <kbd className="ml-2 hidden rounded border border-border/60 bg-card/40 px-1.5 py-0.5 text-[10px] text-muted-foreground sm:inline">
          ⌘K
        </kbd>
      </Button>

      <CommandDialog open={open} onOpenChange={setOpen}>
        <CommandInput placeholder={t("placeholder")} />
        <CommandList>
          <CommandEmpty>{t("empty")}</CommandEmpty>
          <CommandGroup heading={t("navigate")}>
            <CommandItem onSelect={go("/dashboard")}>
              <LayoutDashboard /> {tNav("overview")}
            </CommandItem>
            <CommandItem onSelect={go("/projects")}>
              <FolderGit2 /> {tNav("projects")}
            </CommandItem>
            <CommandItem onSelect={go("/deployments")}>
              <Cloud /> {tNav("deployments")}
            </CommandItem>
            <CommandItem onSelect={go("/logs")}>
              <Terminal /> {tNav("logs")}
            </CommandItem>
            <CommandItem onSelect={go("/metrics")}>
              <BarChart3 /> {tNav("metrics")}
            </CommandItem>
            <CommandItem onSelect={go("/domains")}>
              <Globe /> {tNav("domains")}
            </CommandItem>
          </CommandGroup>
          <CommandSeparator />
          <CommandGroup heading={t("account")}>
            <CommandItem onSelect={go("/settings")}>
              <Settings /> {tNav("settings")}
            </CommandItem>
            <CommandItem
              onSelect={() => setTheme(resolvedTheme === "dark" ? "light" : "dark")}
            >
              {resolvedTheme === "dark" ? <Sun /> : <Moon />}
              {tTheme("toggle")}
              <CommandShortcut>⇧⌘L</CommandShortcut>
            </CommandItem>
            <CommandItem onSelect={handleLogout}>
              <LogOut /> {t("signOut")}
            </CommandItem>
          </CommandGroup>
        </CommandList>
      </CommandDialog>
    </>
  );
}
