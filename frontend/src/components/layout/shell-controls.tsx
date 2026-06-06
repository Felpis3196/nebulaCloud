"use client";

import { LocaleSwitcher } from "@/components/layout/locale-switcher";
import { ThemeSwitcher } from "@/components/layout/theme-switcher";

/** Locale + theme controls shared by auth and other minimal shells. */
export function ShellControls() {
  return (
    <div className="flex items-center gap-1">
      <LocaleSwitcher />
      <ThemeSwitcher />
    </div>
  );
}
