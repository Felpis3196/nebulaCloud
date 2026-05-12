import Link from "next/link";
import { ArrowRight, GitBranch, Github } from "lucide-react";
import type { Project } from "@/types/api";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { relativeTime } from "@/lib/utils";

export function ProjectCard({ project }: { project: Project }) {
  return (
    <Link href={`/projects/${project.id}`} className="group block">
      <Card className="h-full transition-all hover:-translate-y-px hover:border-primary/40">
        <CardHeader className="flex flex-row items-start justify-between gap-3">
          <div className="min-w-0 space-y-1.5">
            <h3 className="truncate text-base font-semibold tracking-tight">
              {project.name}
            </h3>
            <p className="line-clamp-2 text-sm text-muted-foreground">
              {project.description ?? "No description"}
            </p>
          </div>
          <Badge variant="muted" className="shrink-0">
            {project.services_count} svc
          </Badge>
        </CardHeader>
        <CardContent className="space-y-3">
          {project.repo_url && (
            <div className="flex items-center gap-2 text-xs text-muted-foreground">
              <Github className="h-3 w-3" />
              <span className="truncate font-mono">
                {project.repo_url.replace(/^https?:\/\/(www\.)?github\.com\//, "")}
              </span>
            </div>
          )}
          <div className="flex items-center justify-between text-xs text-muted-foreground">
            <span className="inline-flex items-center gap-1.5">
              <GitBranch className="h-3 w-3" />
              <span className="font-mono">{project.default_branch}</span>
            </span>
            <span>updated {relativeTime(project.updated_at)}</span>
          </div>
          <div className="flex items-center text-xs font-medium text-primary opacity-0 transition-opacity group-hover:opacity-100">
            View project <ArrowRight className="ml-1 h-3 w-3" />
          </div>
        </CardContent>
      </Card>
    </Link>
  );
}
