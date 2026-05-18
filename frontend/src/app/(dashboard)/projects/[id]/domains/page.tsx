"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

export default function ProjectDomainsPage() {
  const params = useParams<{ id: string }>();
  const id = typeof params?.id === "string" ? params.id : "";

  return (
    <div className="space-y-6">
      <header>
        <h1 className="text-2xl font-semibold tracking-tight">Domains</h1>
        <p className="text-sm text-muted-foreground">
          Per-project custom hostnames and certificates are not part of this MVP.
        </p>
      </header>

      <Card>
        <CardHeader>
          <CardTitle>Not implemented yet</CardTitle>
          <CardDescription>
            Earlier builds showed mock domain rows; those have been removed. Phase 7 will add API routes, DNS
            verification, and ACME issuance.
          </CardDescription>
        </CardHeader>
        <CardContent className="text-sm text-muted-foreground">
          Configure the default service URL from the{" "}
          <Link href={`/projects/${id}`} className="underline underline-offset-2">
            project overview
          </Link>{" "}
          after deploy — the runtime agent exposes{" "}
          <span className="font-mono">service.project.&lt;base&gt;</span> on HTTP (see Traefik labels).
        </CardContent>
      </Card>
    </div>
  );
}
