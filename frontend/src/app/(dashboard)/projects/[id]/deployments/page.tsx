"use client";

import { useParams } from "next/navigation";
import { DeployTimeline } from "@/components/dashboard/deploy-timeline";
import { useDeployments } from "@/hooks/use-deployments";

export default function ProjectDeploymentsPage() {
  const params = useParams<{ id: string }>();
  const id = typeof params?.id === "string" ? params.id : "";
  const { data = [], isLoading } = useDeployments(id);

  if (isLoading) {
    return <p className="text-sm text-muted-foreground">Loading…</p>;
  }

  return <DeployTimeline deployments={data} limit={50} />;
}
