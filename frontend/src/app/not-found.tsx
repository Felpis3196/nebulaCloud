"use client";

import Link from "next/link";
import { ArrowLeft, Compass } from "lucide-react";
import { useTranslations } from "next-intl";
import { Button } from "@/components/ui/button";

export default function NotFound() {
  const t = useTranslations("notFound");
  const tCommon = useTranslations("common");

  return (
    <div className="flex min-h-screen flex-col items-center justify-center px-6 text-center">
      <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-primary/10 text-primary ring-1 ring-primary/20">
        <Compass className="h-5 w-5" />
      </div>
      <p className="mt-4 font-mono text-xs uppercase tracking-[0.2em] text-muted-foreground">
        {t("code")}
      </p>
      <h1 className="mt-2 text-balance text-3xl font-semibold tracking-tight sm:text-4xl">
        {t("title")}
      </h1>
      <p className="mt-3 max-w-md text-balance text-sm text-muted-foreground">{t("description")}</p>
      <div className="mt-6 flex flex-wrap items-center justify-center gap-3">
        <Button asChild variant="outline">
          <Link href="/">
            <ArrowLeft />
            {tCommon("home")}
          </Link>
        </Button>
        <Button asChild variant="gradient">
          <Link href="/dashboard">{t("goDashboard")}</Link>
        </Button>
      </div>
    </div>
  );
}
