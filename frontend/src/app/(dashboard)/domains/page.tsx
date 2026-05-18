"use client";

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { useTranslations } from "next-intl";

export default function DomainsPage() {
  const t = useTranslations("dashboard.domains");
  return (
    <div className="space-y-6">
      <header>
        <h1 className="text-2xl font-semibold tracking-tight">{t("title")}</h1>
        <p className="text-sm text-muted-foreground">{t("subtitle")}</p>
      </header>

      <Card>
        <CardHeader>
          <CardTitle>{t("cardTitle")}</CardTitle>
          <CardDescription>{t("cardDesc")}</CardDescription>
        </CardHeader>
        <CardContent className="text-sm text-muted-foreground">{t("cardBody")}</CardContent>
      </Card>
    </div>
  );
}
