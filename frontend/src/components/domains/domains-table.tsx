"use client";

import { useState } from "react";
import { Globe, Plus } from "lucide-react";
import { toast } from "sonner";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { StatusPill } from "@/components/dashboard/status-pill";
import { relativeTime } from "@/lib/utils";
import type { Domain } from "@/types/api";
import { useTranslations } from "next-intl";

export function DomainsTable({ domains }: { domains: Domain[] }) {
  const t = useTranslations("domains");
  const tCommon = useTranslations("common");
  const [open, setOpen] = useState(false);

  return (
    <Card>
      <CardHeader className="flex flex-row items-start justify-between gap-4">
        <div>
          <CardTitle>{t("cardTitle")}</CardTitle>
          <CardDescription>{t("cardDesc")}</CardDescription>
        </div>
        <Dialog open={open} onOpenChange={setOpen}>
          <DialogTrigger asChild>
            <Button size="sm" variant="gradient">
              <Plus /> {t("addDomain")}
            </Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>{t("dialogTitle")}</DialogTitle>
              <DialogDescription>{t("dialogDesc")}</DialogDescription>
            </DialogHeader>
            <form
              className="space-y-4"
              onSubmit={(e) => {
                e.preventDefault();
                setOpen(false);
                toast.success(t("verifyQueued"));
              }}
            >
              <div className="space-y-2">
                <Label htmlFor="hostname">{t("hostname")}</Label>
                <Input id="hostname" placeholder="app.acme.com" required />
              </div>
              <div className="rounded-md border border-border/60 bg-secondary/40 px-3 py-2 font-mono text-xs text-muted-foreground">
                {t("cnameHint")}
              </div>
              <DialogFooter>
                <DialogClose asChild>
                  <Button type="button" variant="outline">
                    {tCommon("cancel")}
                  </Button>
                </DialogClose>
                <Button type="submit" variant="gradient">
                  {t("addDomain")}
                </Button>
              </DialogFooter>
            </form>
          </DialogContent>
        </Dialog>
      </CardHeader>
      <CardContent className="px-0 pb-0">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t("hostnameCol")}</TableHead>
              <TableHead>{t("serviceCol")}</TableHead>
              <TableHead>{t("sslCol")}</TableHead>
              <TableHead>{t("lastVerified")}</TableHead>
              <TableHead />
            </TableRow>
          </TableHeader>
          <TableBody>
            {domains.map((d) => (
              <TableRow key={d.id}>
                <TableCell>
                  <div className="flex items-center gap-2">
                    <Globe className="h-3.5 w-3.5 text-muted-foreground" />
                    <span className="font-medium">{d.hostname}</span>
                    {d.is_primary && (
                      <Badge variant="muted" className="text-[10px]">
                        {t("primary")}
                      </Badge>
                    )}
                  </div>
                </TableCell>
                <TableCell className="text-sm text-muted-foreground">{d.service_name}</TableCell>
                <TableCell>
                  <StatusPill status={d.ssl_status} />
                </TableCell>
                <TableCell className="text-xs text-muted-foreground">
                  {d.verified_at ? relativeTime(d.verified_at) : "—"}
                </TableCell>
                <TableCell className="text-right">
                  <Button variant="ghost" size="sm">
                    {t("manage")}
                  </Button>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
}
