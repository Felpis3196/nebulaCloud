"use client";

import { useEffect, useRef, useState } from "react";
import { connectStream } from "@/lib/ws";
import type { BuildLogLine, DeploymentStatus } from "@/types/api";

const ACTIVE: DeploymentStatus[] = ["queued", "building", "pushing", "deploying"];

function isActive(status: DeploymentStatus) {
  return ACTIVE.includes(status);
}

function parseLine(raw: string): BuildLogLine | null {
  try {
    const data = JSON.parse(raw) as BuildLogLine;
    if (data?.message) return data;
  } catch {
    if (raw.trim()) return { message: raw.trim(), level: "info" };
  }
  return null;
}

export function useDeploymentLogStream(
  serviceId: string | undefined,
  deploymentId: string | undefined,
  status: DeploymentStatus | undefined,
) {
  const [liveLines, setLiveLines] = useState<BuildLogLine[]>([]);
  const [connected, setConnected] = useState(false);
  const seen = useRef(new Set<string>());

  useEffect(() => {
    setLiveLines([]);
    seen.current.clear();
    setConnected(false);
  }, [deploymentId]);

  useEffect(() => {
    if (!serviceId || !deploymentId || !status || !isActive(status)) {
      return;
    }

    const path = `/api/v1/services/${serviceId}/logs/stream?deployment_id=${encodeURIComponent(deploymentId)}`;
    const ws = connectStream(path);

    ws.onopen = () => setConnected(true);
    ws.onclose = () => setConnected(false);
    ws.onerror = () => setConnected(false);
    ws.onmessage = (ev) => {
      const line = parseLine(String(ev.data));
      if (!line) return;
      const key = `${line.ts ?? ""}|${line.message}`;
      if (seen.current.has(key)) return;
      seen.current.add(key);
      setLiveLines((prev) => [...prev, line]);
    };

    return () => {
      ws.close();
    };
  }, [serviceId, deploymentId, status]);

  return { liveLines, connected, isLive: !!status && isActive(status) };
}
