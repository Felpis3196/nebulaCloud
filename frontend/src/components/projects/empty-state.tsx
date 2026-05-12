import { FolderPlus } from "lucide-react";

export function ProjectsEmptyState() {
  return (
    <div className="relative overflow-hidden rounded-xl border border-dashed border-border/80 bg-card/30 p-10 text-center">
      <div
        aria-hidden
        className="absolute inset-0 -z-10"
        style={{
          backgroundImage:
            "radial-gradient(40% 50% at 50% 0%, hsl(239 84% 60% / 0.16), transparent)",
        }}
      />
      <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-primary/10 text-primary ring-1 ring-primary/20">
        <FolderPlus className="h-5 w-5" />
      </div>
      <h2 className="mt-4 text-lg font-semibold tracking-tight">No organization selected</h2>
      <p className="mx-auto mt-1.5 max-w-md text-sm text-muted-foreground">
        Select an organization above, or create one. Then add a project and use{" "}
        <span className="font-medium text-foreground">Connect repository</span> from the project
        header to link a GitHub repo and webhook URL.
      </p>
    </div>
  );
}
