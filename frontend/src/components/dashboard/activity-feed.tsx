"use client";

import {
  AlertTriangle,
  GitMerge,
  KeyRound,
  Rocket,
  ScrollText,
  ShieldCheck,
  UserPlus,
} from "lucide-react";
import Link from "next/link";
import { useTranslations } from "next-intl";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { relativeTime } from "@/lib/utils";

interface Activity {
  id: string;
  icon: typeof GitMerge;
  iconClass: string;
  title: React.ReactNode;
  ts: string;
}

const ACTIVITIES: Activity[] = [
  {
    id: "a1",
    icon: Rocket,
    iconClass: "bg-success/15 text-success",
    title: (
      <>
        <Bold>Ada Lovelace</Bold> deployed <Bold>Payments / API</Bold>
      </>
    ),
    ts: minutesAgo(8),
  },
  {
    id: "a2",
    icon: GitMerge,
    iconClass: "bg-info/15 text-info",
    title: (
      <>
        <Bold>storefront</Bold> webhook fired on <code className="font-mono">main</code>
      </>
    ),
    ts: minutesAgo(22),
  },
  {
    id: "a3",
    icon: KeyRound,
    iconClass: "bg-warning/15 text-warning",
    title: (
      <>
        Env var <code className="font-mono">STRIPE_SECRET_KEY</code> rotated by <Bold>Ada Lovelace</Bold>
      </>
    ),
    ts: minutesAgo(64),
  },
  {
    id: "a4",
    icon: ShieldCheck,
    iconClass: "bg-info/15 text-info",
    title: (
      <>
        TLS issued for <Bold>shop.acme.test</Bold>
      </>
    ),
    ts: minutesAgo(120),
  },
  {
    id: "a5",
    icon: AlertTriangle,
    iconClass: "bg-destructive/15 text-destructive",
    title: (
      <>
        <Bold>Analytics / Stream</Bold> build failed at step 6/9
      </>
    ),
    ts: minutesAgo(180),
  },
  {
    id: "a6",
    icon: UserPlus,
    iconClass: "bg-primary/15 text-primary",
    title: (
      <>
        <Bold>devops@acme.test</Bold> joined the workspace as <em>developer</em>
      </>
    ),
    ts: minutesAgo(420),
  },
  {
    id: "a7",
    icon: ScrollText,
    iconClass: "bg-muted/40 text-muted-foreground",
    title: (
      <>
        Audit retention rotated, 12,402 events archived
      </>
    ),
    ts: minutesAgo(720),
  },
];

export function ActivityFeed() {
  const t = useTranslations("dashboard.activity");

  return (
    <Card className="h-full">
      <CardHeader>
        <CardTitle>{t("title")}</CardTitle>
        <CardDescription>
          {t("description")}{" "}
          <Link href="/deployments" className="underline underline-offset-2">
            {t("deploymentsLink")}
          </Link>
          .
        </CardDescription>
      </CardHeader>
      <CardContent className="px-0 pb-2">
        <ul className="space-y-0.5">
          {ACTIVITIES.map((a) => (
            <li key={a.id} className="flex items-start gap-3 px-5 py-2.5">
              <div
                className={`mt-0.5 flex h-7 w-7 shrink-0 items-center justify-center rounded-full ${a.iconClass}`}
              >
                <a.icon className="h-3.5 w-3.5" />
              </div>
              <div className="min-w-0 flex-1">
                <p className="text-sm leading-tight">{a.title}</p>
                <p className="mt-0.5 text-[11px] text-muted-foreground">{relativeTime(a.ts)}</p>
              </div>
            </li>
          ))}
        </ul>
      </CardContent>
    </Card>
  );
}

function Bold({ children }: { children: React.ReactNode }) {
  return <span className="font-medium text-foreground">{children}</span>;
}

function minutesAgo(n: number) {
  return new Date(Date.now() - n * 60_000).toISOString();
}
