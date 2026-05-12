"use client";

import { useState } from "react";
import { Copy, KeyRound, Plus, ShieldCheck, Trash2 } from "lucide-react";
import { toast } from "sonner";
import { useTheme } from "next-themes";

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
  const { user } = useAuth();
  const { theme, setTheme } = useTheme();
  const [mfa, setMfa] = useState(user?.mfa_enabled ?? false);

  return (
    <div className="space-y-6">
      <header>
        <h1 className="text-2xl font-semibold tracking-tight">Settings</h1>
        <p className="text-sm text-muted-foreground">
          Manage your profile, security, API keys, and dashboard appearance.
        </p>
      </header>

      <Tabs defaultValue="profile">
        <TabsList>
          <TabsTrigger value="profile">Profile</TabsTrigger>
          <TabsTrigger value="security">Security</TabsTrigger>
          <TabsTrigger value="keys">API keys</TabsTrigger>
          <TabsTrigger value="appearance">Appearance</TabsTrigger>
        </TabsList>

        <TabsContent value="profile" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Profile</CardTitle>
              <CardDescription>How you appear inside NebulaCloud.</CardDescription>
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
                    {user?.display_name ?? "Operator"}
                  </p>
                  <p className="text-xs text-muted-foreground">{user?.email}</p>
                </div>
                <Button variant="outline" size="sm" className="ml-auto">
                  Upload
                </Button>
              </div>
              <div className="grid gap-4 sm:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="displayName">Display name</Label>
                  <Input id="displayName" defaultValue={user?.display_name ?? ""} />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="email">Email</Label>
                  <Input id="email" type="email" defaultValue={user?.email ?? ""} />
                </div>
              </div>
              <div className="flex justify-end">
                <Button variant="gradient" size="sm">
                  Save profile
                </Button>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="security" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Change password</CardTitle>
              <CardDescription>
                Argon2id hashed server-side, with rotation logged in your audit trail.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid gap-4 sm:grid-cols-3">
                <div className="space-y-2">
                  <Label htmlFor="current">Current</Label>
                  <Input id="current" type="password" />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="new">New</Label>
                  <Input id="new" type="password" />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="confirm">Confirm</Label>
                  <Input id="confirm" type="password" />
                </div>
              </div>
              <div className="flex justify-end">
                <Button size="sm" variant="gradient">
                  Update password
                </Button>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-start justify-between gap-4">
              <div>
                <CardTitle className="flex items-center gap-2">
                  Two-factor authentication
                  {mfa && <Badge variant="success">enabled</Badge>}
                </CardTitle>
                <CardDescription>
                  Time-based one-time passwords. Phase 9 ships hardware key support.
                </CardDescription>
              </div>
              <Switch checked={mfa} onCheckedChange={setMfa} aria-label="Toggle MFA" />
            </CardHeader>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Active sessions</CardTitle>
              <CardDescription>Sign out other devices if you suspect anything.</CardDescription>
            </CardHeader>
            <CardContent className="px-0 pb-0">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Device</TableHead>
                    <TableHead>Last active</TableHead>
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
                              current
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
                          onClick={() => toast.success("Session revoked")}
                        >
                          Revoke
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
                <CardTitle>API keys</CardTitle>
                <CardDescription>
                  Use API keys for CI/CD integrations. Treat them like passwords.
                </CardDescription>
              </div>
              <Button
                size="sm"
                variant="gradient"
                onClick={() => toast.success("API key created. Copy it before closing.")}
              >
                <Plus /> New key
              </Button>
            </CardHeader>
            <CardContent className="px-0 pb-0">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>Token</TableHead>
                    <TableHead>Scopes</TableHead>
                    <TableHead>Created</TableHead>
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
                          onClick={() => toast.success("Key copied")}
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
              <CardTitle>Theme</CardTitle>
              <CardDescription>Dark is the default — light theme is also supported.</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="grid max-w-xl gap-3 sm:grid-cols-2">
                <ThemeOption
                  label="Dark"
                  value="dark"
                  active={theme === "dark"}
                  onClick={() => setTheme("dark")}
                />
                <ThemeOption
                  label="Light"
                  value="light"
                  active={theme === "light"}
                  onClick={() => setTheme("light")}
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
}: {
  label: string;
  value: string;
  active: boolean;
  onClick: () => void;
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
        {active && <Badge variant="success">active</Badge>}
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
