import type { ReactNode } from "react";
import { AuthArtPanel } from "@/components/auth/auth-art-panel";
import { ShellControls } from "@/components/layout/shell-controls";

export default function AuthLayout({ children }: { children: ReactNode }) {
  return (
    <div className="grid min-h-screen lg:grid-cols-[1fr_minmax(420px,520px)]">
      <AuthArtPanel />
      <div className="flex min-h-screen flex-col">
        <div className="flex justify-end gap-1 p-4">
          <ShellControls />
        </div>
        <div className="flex flex-1 items-center justify-center p-6 sm:p-12">
          <div className="w-full max-w-sm">{children}</div>
        </div>
      </div>
    </div>
  );
}
