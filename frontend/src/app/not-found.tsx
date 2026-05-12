import Link from "next/link";
import { ArrowLeft, Compass } from "lucide-react";
import { Button } from "@/components/ui/button";

export default function NotFound() {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center px-6 text-center">
      <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-primary/10 text-primary ring-1 ring-primary/20">
        <Compass className="h-5 w-5" />
      </div>
      <p className="mt-4 font-mono text-xs uppercase tracking-[0.2em] text-muted-foreground">
        404 — Lost in nebula
      </p>
      <h1 className="mt-2 text-balance text-3xl font-semibold tracking-tight sm:text-4xl">
        That page doesn't exist.
      </h1>
      <p className="mt-3 max-w-md text-balance text-sm text-muted-foreground">
        The route you tried to reach has drifted off the manifest. Head back to your
        dashboard or jump to the project list.
      </p>
      <div className="mt-6 flex flex-wrap items-center justify-center gap-3">
        <Button asChild variant="outline">
          <Link href="/">
            <ArrowLeft />
            Home
          </Link>
        </Button>
        <Button asChild variant="gradient">
          <Link href="/dashboard">Go to dashboard</Link>
        </Button>
      </div>
    </div>
  );
}
