"use client";

import { useState } from "react";
import { Copy, KeyRound, Plus, ShieldCheck, Trash2 } from "lucide-react";
import { toast } from "sonner";
import { useTheme } from "next-themes";
import { useTranslations } from "next-intl";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Badge } from "@/components/ui/badge";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Separator } from "@/components/ui/separator";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { useAuth } from "@/hooks/use-auth";
import { relativeTime } from "@/lib/utils";

const MOCK_SESSIONS = [
  { id: "s1", agent: "Chrome on macOS · São Paulo, BR", current: true, last: minutesAgo(2) },
  { id: "s2", agent: "iPhone Safari · São Paulo, BR", current: false, last: hoursAgo(8) },
  { id: "s3", agent: "Firefox on Linux · Lisbon, PT", current: false, last: daysAgo(3) },
];

const MOCK_KEYS = [
  { id: "k1", name: "CI/CD pipeline", preview: "neb_•••••• 9a2c", scopes: ["deploy", "logs"], created: daysAgo(60) },
  { id: "k2", name: "Local CLI", preview: "neb_•••••• 7e1d", scopes: ["read"], created: daysAgo(12) },
];

export default function SettingsPage() {
  const t = useTranslations("settings");
  const tAuth = useTranslations("auth");
  const tCommon = useTranslations("common");
  const tTheme = useTranslations("theme");
  const { user } = useAuth();
  const { theme, setTheme } = useTheme();
  const [mfa, setMfa] = useState(user?.mfa_enabled ?? false);

  return (
    <div className="space-y-6">
      <header>
        <h1 className="text-2xl font-semibold tracking-tight">{t("title")}</h1>
        <p className="text-sm text-muted-foreground">{t("subtitle")}</p>
      </header>

      <div className="rounded-md border border-amber-500/30 bg-amber-500/5 px-3 py-2 text-xs text-muted-foreground">
        <span className="font-medium text-foreground">{t("placeholderBanner")}</span> {t("placeholderText")}
      </div>

      <Tabs defaultValue="profile">
        <TabsList>
          <TabsTrigger value="profile">{t("tabs.profile")}</TabsTrigger>
          <TabsTrigger value="security">{t("tabs.security")}</TabsTrigger>
          <TabsTrigger value="keys">{t("tabs.keys")}</TabsTrigger>
          <TabsTrigger value="appearance">{t("tabs.appearance")}</TabsTrigger>
        </TabsList>

        <TabsContent value="profile" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>{t("profile.title")}</CardTitle>
              <CardDescription>{t("profile.description")}</CardDescription>
            </CardHeader>
            <CardContent className="space-y-6">
              <div className="flex items-center gap-4">
                <Avatar className="h-16 w-16">
                  {user?.avatar_url && <AvatarImage src={user.avatar_url} alt="" />}
                  <AvatarFallback className="bg-primary/15 text-primary">
                    {(user?.display_name ?? user?.email ?? "?")[0]?.toUpperCase()}
                  </AvatarFallback>
                </Avatar>
                <div className="space-y-1">
                  <p className="text-sm font-medium">
                    {user?.display_name ?? tCommon("operator")}
                  </p>
                  <p className="text-xs text-muted-foreground">{user?.email}</p>
                </div>
                <Button variant="outline" size="sm" className="ml-auto">
                  {tCommon("upload")}
                </Button>
              </div>
              <div className="grid gap-4 sm:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="displayName">{t("profile.displayName")}</Label>
                  <Input id="displayName" defaultValue={user?.display_name ?? ""} />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="email">{tAuth("email")}</Label>
                  <Input id="email" type="email" defaultValue={user?.email ?? ""} />
                </div>
              </div>
              <div className="flex justify-end">
                <Button variant="gradient" size="sm">
                  {t("profile.saveProfile")}
                </Button>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="security" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>{t("security.changePassword")}</CardTitle>
              <CardDescription>{t("security.changePasswordDesc")}</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid gap-4 sm:grid-cols-3">
                <div className="space-y-2">
                  <Label htmlFor="current">{t("security.current")}</Label>
                  <Input id="current" type="password" />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="new">{t("security.new")}</Label>
                  <Input id="new" type="password" />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="confirm">{t("security.confirm")}</Label>
                  <Input id="confirm" type="password" />
                </div>
              </div>
              <div className="flex justify-end">
                <Button size="sm" variant="gradient">
                  {t("security.updatePassword")}
                </Button>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-start justify-between gap-4">
              <div>
                <CardTitle className="flex items-center gap-2">
                  {t("security.mfa")}
                  {mfa && <Badge variant="success">{tCommon("enabled")}</Badge>}
                </CardTitle>
                <CardDescription>{t("security.mfaDesc")}</CardDescription>
              </div>
              <Switch checked={mfa} onCheckedChange={setMfa} aria-label={t("security.toggleMfa")} />
            </CardHeader>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>{t("security.sessions")}</CardTitle>
              <CardDescription>{t("security.sessionsDesc")}</CardDescription>
            </CardHeader>
            <CardContent className="px-0 pb-0">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{tCommon("device")}</TableHead>
                    <TableHead>{tCommon("lastActive")}</TableHead>
                    <TableHead />
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {MOCK_SESSIONS.map((s) => (
                    <TableRow key={s.id}>
                      <TableCell>
                        <div className="flex items-center gap-2">
                          <ShieldCheck className="h-3.5 w-3.5 text-muted-foreground" />
                          {s.agent}
                          {s.current && (
                            <Badge variant="success" className="text-[10px]">
                              {tCommon("current")}
                            </Badge>
                          )}
                        </div>
                      </TableCell>
                      <TableCell className="text-xs text-muted-foreground">
                        {relativeTime(s.last)}
                      </TableCell>
                      <TableCell className="text-right">
                        <Button
                          variant="ghost"
                          size="sm"
                          disabled={s.current}
                          className="text-destructive hover:text-destructive"
                          onClick={() => toast.success(t("security.sessionRevoked"))}
                        >
                          {tCommon("revoke")}
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="keys" className="space-y-4">
          <Card>
            <CardHeader className="flex flex-row items-start justify-between gap-4">
              <div>
                <CardTitle>{t("keys.title")}</CardTitle>
                <CardDescription>{t("keys.description")}</CardDescription>
              </div>
              <Button
                size="sm"
                variant="gradient"
                onClick={() => toast.success(t("keys.keyCreated"))}
              >
                <Plus /> {t("keys.newKey")}
              </Button>
            </CardHeader>
            <CardContent className="px-0 pb-0">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{tCommon("name")}</TableHead>
                    <TableHead>{tCommon("token")}</TableHead>
                    <TableHead>{tCommon("scopes")}</TableHead>
                    <TableHead>{tCommon("created")}</TableHead>
                    <TableHead />
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {MOCK_KEYS.map((k) => (
                    <TableRow key={k.id}>
                      <TableCell>
                        <div className="flex items-center gap-2">
                          <KeyRound className="h-3.5 w-3.5 text-muted-foreground" />
                          {k.name}
                        </div>
                      </TableCell>
                      <TableCell>
                        <button
                          className="inline-flex items-center gap-2 rounded-md border border-border/60 bg-card/40 px-2 py-1 font-mono text-xs hover:bg-secondary/40"
                          onClick={() => toast.success(t("keys.keyCopied"))}
                        >
                          <span>{k.preview}</span>
                          <Copy className="h-3 w-3 text-muted-foreground" />
                        </button>
                      </TableCell>
                      <TableCell>
                        <div className="flex gap-1">
                          {k.scopes.map((s) => (
                            <Badge key={s} variant="muted">
                              {s}
                            </Badge>
                          ))}
                        </div>
                      </TableCell>
                      <TableCell className="text-xs text-muted-foreground">
                        {relativeTime(k.created)}
                      </TableCell>
                      <TableCell className="text-right">
                        <Button variant="ghost" size="icon" className="text-destructive">
                          <Trash2 />
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="appearance" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>{t("appearance.title")}</CardTitle>
              <CardDescription>{t("appearance.description")}</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="grid max-w-xl gap-3 sm:grid-cols-2">
                <ThemeOption
                  label={tTheme("dark")}
                  value="dark"
                  active={theme === "dark"}
                  onClick={() => setTheme("dark")}
                  activeLabel={tCommon("active")}
                />
                <ThemeOption
                  label={tTheme("light")}
                  value="light"
                  active={theme === "light"}
                  onClick={() => setTheme("light")}
                  activeLabel={tCommon("active")}
                />
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
      <Separator />
    </div>
  );
}

function ThemeOption({
  label,
  value,
  active,
  onClick,
  activeLabel,
}: {
  label: string;
  value: string;
  active: boolean;
  onClick: () => void;
  activeLabel: string;
}) {
  return (
    <button
      onClick={onClick}
      className={`group relative overflow-hidden rounded-xl border p-1.5 text-left transition-colors ${
        active ? "border-primary/60" : "border-border/60 hover:border-border"
      }`}
    >
      <div
        className={`flex h-32 w-full flex-col gap-1.5 rounded-lg p-3 ${
          value === "dark"
            ? "bg-[#0b0b10] text-white"
            : "bg-zinc-100 text-zinc-900"
        }`}
      >
        <div className="h-2 w-2/3 rounded-full bg-current/20" />
        <div className="h-2 w-1/2 rounded-full bg-current/15" />
        <div className="mt-auto flex gap-1.5">
          <div className="h-6 flex-1 rounded bg-current/10" />
          <div className="h-6 flex-1 rounded bg-current/10" />
        </div>
      </div>
      <div className="flex items-center justify-between px-2 pb-1 pt-2 text-sm">
        {label}
        {active && <Badge variant="success">{activeLabel}</Badge>}
      </div>
    </button>
  );
}

function minutesAgo(n: number) {
  return new Date(Date.now() - n * 60_000).toISOString();
}
function hoursAgo(n: number) {
  return new Date(Date.now() - n * 3_600_000).toISOString();
}
function daysAgo(n: number) {
  return new Date(Date.now() - n * 86_400_000).toISOString();
}
