"use client";

import { useEffect, useRef } from "react";
import { Loader2 } from "lucide-react";
import { useTranslations } from "next-intl";
import type { BuildLogLine } from "@/types/api";
import { cn } from "@/lib/utils";

interface Props {
  lines: BuildLogLine[];
  loading?: boolean;
  emptyHint?: string;
  live?: boolean;
}

function levelClass(level?: string) {
  switch (level) {
    case "error":
      return "text-destructive";
    case "warn":
      return "text-amber-500";
    default:
      return "text-muted-foreground";
  }
}

export function DeploymentLogViewer({ lines, loading, emptyHint, live }: Props) {
  const t = useTranslations("deployments.drawer");
  const bottomRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [lines.length]);

  return (
    <div className="overflow-hidden rounded-md border border-border/60 bg-card/40 font-mono text-[12px]">
      {live && (
        <div className="flex items-center gap-2 border-b border-border/60 bg-secondary/30 px-3 py-1.5 text-[11px] text-muted-foreground">
          <span className="relative flex h-2 w-2">
            <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-success opacity-75" />
            <span className="relative inline-flex h-2 w-2 rounded-full bg-success" />
          </span>
          {t("liveLogs")}
        </div>
      )}
      <div className="max-h-64 overflow-y-auto">
        {loading && lines.length === 0 ? (
          <div className="flex items-center gap-2 px-3 py-4 text-muted-foreground">
            <Loader2 className="h-3.5 w-3.5 animate-spin" />
            {t("loadingLogs")}
          </div>
        ) : lines.length === 0 ? (
          <p className="px-3 py-4 text-muted-foreground">{emptyHint ?? t("noLogsYet")}</p>
        ) : (
          lines.map((line, i) => (
            <div
              key={`${line.ts ?? i}-${line.message.slice(0, 40)}`}
              className={cn(
                "whitespace-pre-wrap break-all px-3 py-1 odd:bg-secondary/20",
                levelClass(line.level),
              )}
            >
              {line.source ? (
                <span className="mr-2 text-[10px] uppercase opacity-60">{line.source}</span>
              ) : null}
              {line.message}
            </div>
          ))
        )}
        <div ref={bottomRef} />
      </div>
    </div>
  );
}
