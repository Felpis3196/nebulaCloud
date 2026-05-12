import Link from "next/link";
import { ArrowRight } from "lucide-react";
import { Button } from "@/components/ui/button";

export function LandingCTA() {
  return (
    <section className="relative overflow-hidden border-b border-border/40 py-20 sm:py-28">
      <div
        aria-hidden
        className="absolute inset-0 -z-10"
        style={{
          backgroundImage:
            "radial-gradient(60% 40% at 50% 100%, hsl(239 84% 60% / 0.18), transparent), radial-gradient(40% 30% at 80% 0%, hsl(305 80% 60% / 0.16), transparent)",
        }}
      />
      <div className="container">
        <div className="mx-auto flex max-w-3xl flex-col items-center gap-6 rounded-2xl border border-border/60 bg-card/50 p-10 text-center backdrop-blur-xl sm:p-14">
          <h2 className="text-balance text-3xl font-semibold tracking-tight sm:text-4xl">
            Ship your next service in <span className="text-gradient">five minutes.</span>
          </h2>
          <p className="max-w-xl text-balance text-muted-foreground">
            Spin up the full stack with one make-target. Connect a repo. Press deploy.
            Watch logs stream in real time.
          </p>
          <div className="flex flex-wrap items-center justify-center gap-3">
            <Button asChild size="xl" variant="gradient">
              <Link href="/register">
                Start deploying
                <ArrowRight />
              </Link>
            </Button>
            <Button asChild size="xl" variant="outline">
              <Link href="/login">Sign in</Link>
            </Button>
          </div>
        </div>
      </div>
    </section>
  );
}
