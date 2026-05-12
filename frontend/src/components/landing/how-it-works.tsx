import { CircleCheck, GitMerge, Hammer, Rocket } from "lucide-react";

const STEPS = [
  {
    icon: GitMerge,
    title: "Connect a repo",
    body: "Install the GitHub App and pick the repository. Branch, environment, and runtime live in one config.",
  },
  {
    icon: Hammer,
    title: "Auto build",
    body: "Workers detect your stack and build a tiny OCI image with Cloud Native Buildpacks (or your Dockerfile).",
  },
  {
    icon: Rocket,
    title: "Auto deploy",
    body: "Runtime agent rolls out a new container with healthcheck and zero-downtime cutover behind Traefik.",
  },
  {
    icon: CircleCheck,
    title: "Observe",
    body: "Logs, metrics, build history, audit trail — all in the dashboard, scoped to your org and roles.",
  },
];

export function HowItWorks() {
  return (
    <section id="how" className="border-b border-border/40 py-20 sm:py-28">
      <div className="container">
        <div className="mx-auto max-w-2xl text-center">
          <p className="text-sm font-medium uppercase tracking-[0.2em] text-primary">
            How it works
          </p>
          <h2 className="mt-3 text-balance text-3xl font-semibold tracking-tight sm:text-4xl">
            From <span className="font-mono text-foreground">git push</span> to{" "}
            <span className="text-gradient">live</span> in four steps.
          </h2>
        </div>

        <div className="relative mx-auto mt-14 grid max-w-5xl gap-4 md:grid-cols-2 lg:grid-cols-4">
          <div
            aria-hidden
            className="absolute left-0 right-0 top-12 hidden h-px bg-gradient-to-r from-transparent via-border to-transparent lg:block"
          />
          {STEPS.map((step, i) => (
            <div
              key={step.title}
              className="relative flex flex-col items-start gap-3 rounded-xl border border-border/60 bg-card/40 p-5 backdrop-blur-md"
            >
              <div className="flex w-full items-center justify-between">
                <div className="flex h-8 w-8 items-center justify-center rounded-md bg-primary/10 text-primary ring-1 ring-primary/20">
                  <step.icon className="h-4 w-4" />
                </div>
                <span className="font-mono text-xs text-muted-foreground">
                  {String(i + 1).padStart(2, "0")}
                </span>
              </div>
              <h3 className="text-base font-semibold tracking-tight">{step.title}</h3>
              <p className="text-sm leading-relaxed text-muted-foreground">{step.body}</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
