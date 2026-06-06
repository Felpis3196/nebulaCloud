import type { ReactNode } from "react";
import { Sidebar } from "@/components/layout/sidebar";
import { Topbar } from "@/components/layout/topbar";
import { AuthBootstrap } from "@/components/layout/auth-bootstrap";
import { OrgSyncBootstrap } from "@/components/layout/org-sync-bootstrap";

export default function DashboardLayout({ children }: { children: ReactNode }) {
  return (
    <div className="flex min-h-screen">
      <AuthBootstrap />
      <OrgSyncBootstrap />
      <Sidebar />
      <div className="flex min-w-0 flex-1 flex-col">
        <Topbar />
        <main className="flex-1 overflow-x-hidden">
          <div className="mx-auto w-full max-w-screen-2xl p-6 lg:p-8">{children}</div>
        </main>
      </div>
    </div>
  );
}
