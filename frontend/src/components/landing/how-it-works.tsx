"use client";

import { CircleCheck, GitMerge, Hammer, Rocket } from "lucide-react";
import { useTranslations } from "next-intl";

const STEP_ICONS = [GitMerge, Hammer, Rocket, CircleCheck] as const;
const STEP_KEYS = ["connect", "build", "deploy", "observe"] as const;

export function HowItWorks() {
  const t = useTranslations("landing.howSection");

  return (
    <section id="how" className="border-b border-border/40 py-20 sm:py-28">
      <div className="container">
        <div className="mx-auto max-w-2xl text-center">
          <p className="text-sm font-medium uppercase tracking-[0.2em] text-primary">
            {t("label")}
          </p>
          <h2 className="mt-3 text-balance text-3xl font-semibold tracking-tight sm:text-4xl">
            {t("titlePrefix")}{" "}
            <span className="font-mono text-foreground">{t("titleMono")}</span> {t("titleMid")}{" "}
            <span className="text-gradient">{t("titleLive")}</span> {t("titleSuffix")}
          </h2>
        </div>

        <div className="relative mx-auto mt-14 grid max-w-5xl gap-4 md:grid-cols-2 lg:grid-cols-4">
          <div
            aria-hidden
            className="absolute left-0 right-0 top-12 hidden h-px bg-gradient-to-r from-transparent via-border to-transparent lg:block"
          />
          {STEP_KEYS.map((key, i) => {
            const Icon = STEP_ICONS[i]!;
            return (
              <div
                key={key}
                className="relative flex flex-col items-start gap-3 rounded-xl border border-border/60 bg-card/40 p-5 backdrop-blur-md"
              >
                <div className="flex w-full items-center justify-between">
                  <div className="flex h-8 w-8 items-center justify-center rounded-md bg-primary/10 text-primary ring-1 ring-primary/20">
                    <Icon className="h-4 w-4" />
                  </div>
                  <span className="font-mono text-xs text-muted-foreground">
                    {String(i + 1).padStart(2, "0")}
                  </span>
                </div>
                <h3 className="text-base font-semibold tracking-tight">
                  {t(`${key}Title`)}
                </h3>
                <p className="text-sm leading-relaxed text-muted-foreground">
                  {t(`${key}Body`)}
                </p>
              </div>
            );
          })}
        </div>
      </div>
    </section>
  );
}
