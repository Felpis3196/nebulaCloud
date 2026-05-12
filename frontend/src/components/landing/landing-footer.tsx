import Link from "next/link";
import { Cloud, Github } from "lucide-react";
import { env } from "@/lib/env";

export function LandingFooter() {
  return (
    <footer className="border-t border-border/40">
      <div className="container py-12">
        <div className="grid gap-8 sm:grid-cols-4">
          <div className="space-y-3 sm:col-span-2">
            <Link href="/" className="inline-flex items-center gap-2">
              <div className="flex h-7 w-7 items-center justify-center rounded-md gradient-primary">
                <Cloud className="h-4 w-4 text-white" />
              </div>
              <span className="text-sm font-semibold tracking-tight">NebulaCloud</span>
            </Link>
            <p className="max-w-sm text-sm text-muted-foreground">
              Self-hosted PaaS for shipping containerised applications straight from Git.
              Built as a portfolio-grade reference architecture.
            </p>
          </div>
          <FooterColumn title="Product">
            <FooterLink href="#features">Features</FooterLink>
            <FooterLink href="#how">How it works</FooterLink>
            <FooterLink href="/login">Sign in</FooterLink>
          </FooterColumn>
          <FooterColumn title="Resources">
            <FooterLink href={env.NEXT_PUBLIC_GITHUB_URL} external>
              <Github className="h-3 w-3" />
              GitHub
            </FooterLink>
            <FooterLink href={`${env.NEXT_PUBLIC_GITHUB_URL}/blob/main/docs/ARCHITECTURE.md`} external>
              Architecture
            </FooterLink>
            <FooterLink href={`${env.NEXT_PUBLIC_GITHUB_URL}/blob/main/docs/SECURITY.md`} external>
              Security
            </FooterLink>
          </FooterColumn>
        </div>

        <div className="mt-10 flex flex-col items-start justify-between gap-3 border-t border-border/40 pt-6 sm:flex-row sm:items-center">
          <p className="text-xs text-muted-foreground">
            © {new Date().getFullYear()} NebulaCloud — MIT licensed.
          </p>
          <p className="font-mono text-xs text-muted-foreground">
            v0.1.0 · Phase 1 (identity)
          </p>
        </div>
      </div>
    </footer>
  );
}

function FooterColumn({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div>
      <h3 className="text-xs font-semibold uppercase tracking-[0.16em] text-muted-foreground">
        {title}
      </h3>
      <ul className="mt-3 space-y-2 text-sm">{children}</ul>
    </div>
  );
}

function FooterLink({
  href,
  external,
  children,
}: {
  href: string;
  external?: boolean;
  children: React.ReactNode;
}) {
  const className =
    "inline-flex items-center gap-1.5 text-muted-foreground transition-colors hover:text-foreground";
  if (external) {
    return (
      <li>
        <a href={href} target="_blank" rel="noreferrer" className={className}>
          {children}
        </a>
      </li>
    );
  }
  return (
    <li>
      <Link href={href} className={className}>
        {children}
      </Link>
    </li>
  );
}
