"use client";

import Link from "next/link";
import { Cloud, Github } from "lucide-react";
import { useTranslations } from "next-intl";
import { env } from "@/lib/env";
import { githubBlob } from "@/lib/github-urls";

export function LandingFooter() {
  const t = useTranslations("landing");

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
            <p className="max-w-sm text-sm text-muted-foreground">{t("footerTagline")}</p>
          </div>
          <FooterColumn title={t("product")}>
            <FooterLink href="#features">{t("features")}</FooterLink>
            <FooterLink href="#how">{t("howItWorks")}</FooterLink>
            <FooterLink href="/login">{t("signIn")}</FooterLink>
          </FooterColumn>
          <FooterColumn title={t("resources")}>
            <FooterLink href={env.NEXT_PUBLIC_GITHUB_URL} external>
              <Github className="h-3 w-3" />
              {t("sourceCode")}
            </FooterLink>
            <FooterLink href={githubBlob("README.md")} external>
              {t("readme")}
            </FooterLink>
            <FooterLink href={githubBlob("docs/ARCHITECTURE.md")} external>
              {t("architecture")}
            </FooterLink>
            <FooterLink href={githubBlob("docs/SECURITY.md")} external>
              {t("security")}
            </FooterLink>
            <FooterLink href={githubBlob("docs/CONTRIBUTING.md")} external>
              {t("contributing")}
            </FooterLink>
          </FooterColumn>
        </div>

        <div className="mt-10 flex flex-col items-start justify-between gap-3 border-t border-border/40 pt-6 sm:flex-row sm:items-center">
          <p className="text-xs text-muted-foreground" suppressHydrationWarning>
            {t("copyright", { year: new Date().getFullYear() })}
          </p>
          <p className="font-mono text-xs text-muted-foreground">{t("version")}</p>
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
