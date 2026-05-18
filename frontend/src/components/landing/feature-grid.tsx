"use client";

import {
  GitBranch,
  Lock,
  ScrollText,
  ShieldCheck,
  TerminalSquare,
  Zap,
} from "lucide-react";
import { useTranslations } from "next-intl";

const FEATURE_KEYS = [
  { icon: GitBranch, titleKey: "pushTitle", bodyKey: "pushBody" },
  { icon: Zap, titleKey: "buildTitle", bodyKey: "buildBody" },
  { icon: ScrollText, titleKey: "logsTitle", bodyKey: "logsBody" },
  { icon: TerminalSquare, titleKey: "terminalTitle", bodyKey: "terminalBody" },
  { icon: Lock, titleKey: "secretsTitle", bodyKey: "secretsBody" },
  { icon: ShieldCheck, titleKey: "domainsTitle", bodyKey: "domainsBody" },
] as const;

export function FeatureGrid() {
  const t = useTranslations("landing.featureGrid");
  const tFeat = useTranslations("landing.featuresSection");

  return (
    <section id="features" className="border-b border-border/40 py-20 sm:py-28">
      <div className="container">
        <div className="mx-auto max-w-2xl text-center">
          <p className="text-sm font-medium uppercase tracking-[0.2em] text-primary">
            {t("platform")}
          </p>
          <h2 className="mt-3 text-balance text-3xl font-semibold tracking-tight sm:text-4xl">
            {t.rich("title", {
              highlight: () => (
                <span className="text-gradient">{t("titleHighlight")}</span>
              ),
            })}
          </h2>
          <p className="mt-4 text-balance text-muted-foreground">{t("subtitle")}</p>
        </div>

        <div className="mx-auto mt-14 grid max-w-6xl gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {FEATURE_KEYS.map((feat) => (
            <div
              key={feat.titleKey}
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
              <h3 className="text-base font-semibold tracking-tight">
                {tFeat(feat.titleKey)}
              </h3>
              <p className="mt-1.5 text-sm leading-relaxed text-muted-foreground">
                {tFeat(feat.bodyKey)}
              </p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
