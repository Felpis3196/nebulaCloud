import {
  GitBranch,
  Lock,
  ScrollText,
  ShieldCheck,
  TerminalSquare,
  Zap,
} from "lucide-react";

const FEATURES = [
  {
    icon: GitBranch,
    title: "Push to deploy",
    body: "Connect a GitHub repo, push to your branch, and NebulaCloud builds, ships, and rolls it out behind HTTPS.",
  },
  {
    icon: Zap,
    title: "Smart builds",
    body: "Cloud Native Buildpacks auto-detect Node, Python, Go and .NET stacks. Bring a Dockerfile if you'd rather opt out.",
  },
  {
    icon: ScrollText,
    title: "Realtime logs",
    body: "Stream stdout from every container directly into your dashboard, with structured fields and correlation IDs.",
  },
  {
    icon: TerminalSquare,
    title: "Web terminal",
    body: "Open a shell inside any running container. SSH-grade ergonomics from the browser, scoped by RBAC.",
  },
  {
    icon: Lock,
    title: "Encrypted secrets",
    body: "Environment variables sealed with AES-256-GCM at rest. Pluggable for KMS / Vault when you need it.",
  },
  {
    icon: ShieldCheck,
    title: "Custom domains + ACME",
    body: "Bring your own hostname, NebulaCloud provisions Let's Encrypt certificates and rotates them automatically.",
  },
];

export function FeatureGrid() {
  return (
    <section id="features" className="border-b border-border/40 py-20 sm:py-28">
      <div className="container">
        <div className="mx-auto max-w-2xl text-center">
          <p className="text-sm font-medium uppercase tracking-[0.2em] text-primary">
            Platform
          </p>
          <h2 className="mt-3 text-balance text-3xl font-semibold tracking-tight sm:text-4xl">
            Everything between <span className="text-gradient">git push</span> and a live URL.
          </h2>
          <p className="mt-4 text-balance text-muted-foreground">
            Build pipeline, runtime, ingress, observability — composed into a single
            cohesive experience instead of a stitched-together stack.
          </p>
        </div>

        <div className="mx-auto mt-14 grid max-w-6xl gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {FEATURES.map((feat) => (
            <div
              key={feat.title}
              className="group relative rounded-xl border border-border/60 bg-card/40 p-5 transition-all hover:-translate-y-px hover:border-border hover:bg-card/70"
            >
              <div
                aria-hidden
                className="absolute -inset-px -z-10 rounded-xl opacity-0 blur-md transition-opacity group-hover:opacity-100"
                style={{
                  background:
                    "linear-gradient(120deg, hsl(239 84% 60% / 0.18), hsl(305 80% 60% / 0.14))",
                }}
              />
              <div className="mb-4 inline-flex h-9 w-9 items-center justify-center rounded-lg bg-primary/10 text-primary ring-1 ring-primary/20">
                <feat.icon className="h-4 w-4" />
              </div>
              <h3 className="text-base font-semibold tracking-tight">{feat.title}</h3>
              <p className="mt-1.5 text-sm leading-relaxed text-muted-foreground">
                {feat.body}
              </p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
