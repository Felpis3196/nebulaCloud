"use client";

import Link from "next/link";
import { Cloud, Github } from "lucide-react";
import { Button } from "@/components/ui/button";
import { env } from "@/lib/env";

export function LandingNav() {
  return (
    <header className="sticky top-0 z-30 border-b border-border/40 bg-background/70 backdrop-blur-md">
      <div className="container flex h-14 items-center justify-between">
        <Link href="/" className="flex items-center gap-2">
          <div className="flex h-7 w-7 items-center justify-center rounded-md gradient-primary">
            <Cloud className="h-4 w-4 text-white" />
          </div>
          <span className="text-sm font-semibold tracking-tight">NebulaCloud</span>
        </Link>

        <nav className="hidden items-center gap-6 text-sm text-muted-foreground md:flex">
          <Link href="#features" className="transition-colors hover:text-foreground">
            Features
          </Link>
          <Link href="#how" className="transition-colors hover:text-foreground">
            How it works
          </Link>
          <a
            href={env.NEXT_PUBLIC_GITHUB_URL}
            target="_blank"
            rel="noreferrer"
            className="transition-colors hover:text-foreground"
          >
            GitHub
          </a>
        </nav>

        <div className="flex items-center gap-2">
          <Button asChild variant="ghost" size="sm">
            <a href={env.NEXT_PUBLIC_GITHUB_URL} target="_blank" rel="noreferrer">
              <Github className="h-4 w-4" />
              <span className="hidden sm:inline">Star on GitHub</span>
            </a>
          </Button>
          <Button asChild size="sm" variant="ghost">
            <Link href="/login">Sign in</Link>
          </Button>
          <Button asChild size="sm" variant="gradient">
            <Link href="/register">Get started</Link>
          </Button>
        </div>
      </div>
    </header>
  );
}
