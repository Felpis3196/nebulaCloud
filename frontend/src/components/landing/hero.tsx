import Link from "next/link";
import { ArrowRight, Github } from "lucide-react";
import { Button } from "@/components/ui/button";
import { TerminalMockup } from "@/components/landing/terminal-mockup";
import { env } from "@/lib/env";

export function Hero() {
  return (
    <section className="relative overflow-hidden border-b border-border/40 pb-24 pt-20 sm:pb-32 sm:pt-28">
      {/* Background gradient */}
      <div
        aria-hidden
        className="absolute inset-0 -z-10"
        style={{
          backgroundImage:
            "radial-gradient(80% 50% at 50% 0%, hsl(239 84% 60% / 0.18), transparent), radial-gradient(60% 40% at 80% 30%, hsl(305 80% 60% / 0.14), transparent)",
        }}
      />
      <div
        aria-hidden
        className="absolute inset-x-0 top-0 -z-10 h-[600px] bg-grid-faint bg-grid mask-fade-b opacity-[0.04]"
      />

      <div className="container">
        <div className="mx-auto max-w-3xl text-center">
          <Link
            href={env.NEXT_PUBLIC_GITHUB_URL}
            target="_blank"
            className="mx-auto mb-6 inline-flex items-center gap-2 rounded-full border border-border/60 bg-card/40 px-3 py-1 text-xs text-muted-foreground backdrop-blur-md transition-colors hover:border-border hover:text-foreground"
          >
            <span className="rounded-full bg-success/15 px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wider text-success">
              New
            </span>
            Self-hosted, single binary, MIT-licensed
            <ArrowRight className="h-3 w-3" />
          </Link>

          <h1 className="text-balance text-4xl font-semibold tracking-tight sm:text-6xl">
            <span className="text-gradient">Deploy from Git</span>
            <br />
            <span className="text-foreground">in seconds.</span>
          </h1>

          <p className="mx-auto mt-6 max-w-2xl text-balance text-base leading-relaxed text-muted-foreground sm:text-lg">
            NebulaCloud is a self-hosted PaaS in the spirit of Railway, Render, and Heroku.
            Connect a repo, ship a container, get realtime logs, metrics, and a polished
            dashboard — all on infrastructure you control.
          </p>

          <div className="mt-8 flex flex-wrap items-center justify-center gap-3">
            <Button asChild size="xl" variant="gradient">
              <Link href="/register">
                Start deploying
                <ArrowRight />
              </Link>
            </Button>
            <Button asChild size="xl" variant="outline">
              <a href={env.NEXT_PUBLIC_GITHUB_URL} target="_blank" rel="noreferrer">
                <Github className="h-4 w-4" />
                View source
              </a>
            </Button>
          </div>

          <p className="mt-4 text-xs text-muted-foreground">
            Free to self-host. No credit card. Boots on Docker Compose in under a minute.
          </p>
        </div>

        <div className="mx-auto mt-16 max-w-4xl">
          <TerminalMockup />
        </div>
      </div>
    </section>
  );
}
