"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { KeyRound, Loader2, Plus, Trash2 } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { useServices } from "@/hooks/use-services";
import { useEnvVars } from "@/hooks/use-env-vars";
import { api, ApiError } from "@/lib/api-client";
import type { EnvVar } from "@/types/api";
import { relativeTime } from "@/lib/utils";

export default function ProjectEnvPage() {
  const params = useParams<{ id: string }>();
  const projectId = typeof params?.id === "string" ? params.id : "";
  const qc = useQueryClient();
  const { data: services = [], isPending: svcLoading } = useServices(projectId || undefined);
  const [serviceId, setServiceId] = useState("");

  useEffect(() => {
    if (!serviceId && services.length > 0) {
      setServiceId(services[0]!.id);
    }
  }, [services, serviceId]);

  const { data: vars = [], isPending: envLoading } = useEnvVars(serviceId || undefined);

  const addMutation = useMutation({
    mutationFn: async (payload: { key: string; value: string; is_secret: boolean }) => {
      await api<EnvVar>(`/api/v1/services/${serviceId}/env-vars`, {
        method: "POST",
        body: payload,
      });
    },
    onSuccess: async () => {
      await qc.invalidateQueries({ queryKey: ["env-vars", serviceId] });
      toast.success("Variable saved.");
      setNewKey("");
      setNewVal("");
      setNewSecret(false);
    },
    onError: (e: Error) => {
      toast.error(e instanceof ApiError ? e.message : "Could not add variable");
    },
  });

  const deleteMutation = useMutation({
    mutationFn: async (key: string) => {
      await api(`/api/v1/services/${serviceId}/env-vars/${encodeURIComponent(key)}`, {
        method: "DELETE",
      });
    },
    onSuccess: async () => {
      await qc.invalidateQueries({ queryKey: ["env-vars", serviceId] });
      toast.success("Variable removed.");
    },
    onError: (e: Error) => {
      toast.error(e instanceof ApiError ? e.message : "Could not delete variable");
    },
  });

  const [newKey, setNewKey] = useState("");
  const [newVal, setNewVal] = useState("");
  const [newSecret, setNewSecret] = useState(false);

  function addVar(e: React.FormEvent) {
    e.preventDefault();
    if (!serviceId) return;
    const k = newKey.trim();
    if (!k) {
      toast.error("Key is required.");
      return;
    }
    addMutation.mutate({ key: k, value: newVal, is_secret: newSecret });
  }

  const loading = svcLoading || (serviceId !== "" && envLoading);

  return (
    <Card>
      <CardHeader className="flex flex-row flex-wrap items-start justify-between gap-4">
        <div>
          <CardTitle>Environment variables</CardTitle>
          <CardDescription>
            Sealed at rest with AES-256-GCM. Scoped per service; pick a service to edit its env.
          </CardDescription>
        </div>
      </CardHeader>
      <CardContent className="space-y-6 px-0 pb-0">
        <div className="flex flex-wrap items-end gap-3 px-6">
          <div className="space-y-2">
            <Label>Service</Label>
            {svcLoading ? (
              <p className="text-sm text-muted-foreground">Loading services…</p>
            ) : services.length === 0 ? (
              <p className="text-sm text-muted-foreground">Add a service from the overview first.</p>
            ) : (
              <Select value={serviceId} onValueChange={setServiceId}>
                <SelectTrigger className="w-[220px]">
                  <SelectValue placeholder="Service" />
                </SelectTrigger>
                <SelectContent>
                  {services.map((s) => (
                    <SelectItem key={s.id} value={s.id}>
                      {s.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            )}
          </div>
        </div>

        {serviceId && (
          <>
            <form onSubmit={addVar} className="grid gap-3 border-y border-border/60 bg-muted/20 px-6 py-4 sm:grid-cols-[1fr_1fr_auto_auto] sm:items-end">
              <div className="space-y-2">
                <Label htmlFor="env-key">Key</Label>
                <Input
                  id="env-key"
                  className="font-mono text-sm"
                  value={newKey}
                  onChange={(e) => setNewKey(e.target.value)}
                  placeholder="DATABASE_URL"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="env-val">Value</Label>
                <Input
                  id="env-val"
                  className="font-mono text-sm"
                  type={newSecret ? "password" : "text"}
                  value={newVal}
                  onChange={(e) => setNewVal(e.target.value)}
                  placeholder="value"
                />
              </div>
              <div className="flex items-center gap-2 pb-2">
                <Switch id="env-secret" checked={newSecret} onCheckedChange={setNewSecret} />
                <Label htmlFor="env-secret" className="text-sm font-normal">
                  Secret
                </Label>
              </div>
              <Button type="submit" size="sm" variant="gradient" disabled={addMutation.isPending}>
                {addMutation.isPending ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <>
                    <Plus className="h-4 w-4" /> Add
                  </>
                )}
              </Button>
            </form>

            {loading ? (
              <p className="px-6 text-sm text-muted-foreground">Loading variables…</p>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Key</TableHead>
                    <TableHead>Value</TableHead>
                    <TableHead>Type</TableHead>
                    <TableHead>Updated</TableHead>
                    <TableHead className="w-[72px]" />
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {vars.map((v) => (
                    <TableRow key={v.id}>
                      <TableCell>
                        <div className="flex items-center gap-2 font-mono text-sm">
                          <KeyRound className="h-3 w-3 text-muted-foreground" />
                          {v.key}
                        </div>
                      </TableCell>
                      <TableCell className="font-mono text-sm text-muted-foreground">
                        {v.is_secret ? `••••••••${v.preview ?? ""}` : (v.preview ?? "—")}
                      </TableCell>
                      <TableCell>
                        {v.is_secret ? (
                          <Badge variant="warning">secret</Badge>
                        ) : (
                          <Badge variant="muted">plain</Badge>
                        )}
                      </TableCell>
                      <TableCell className="text-xs text-muted-foreground">
                        {relativeTime(v.updated_at)}
                      </TableCell>
                      <TableCell className="text-right">
                        <Button
                          variant="ghost"
                          size="icon"
                          type="button"
                          aria-label="Remove variable"
                          disabled={deleteMutation.isPending}
                          onClick={() => deleteMutation.mutate(v.key)}
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
            {vars.length === 0 && !loading && (
              <p className="px-6 pb-4 text-sm text-muted-foreground">No variables for this service yet.</p>
            )}
          </>
        )}
      </CardContent>
    </Card>
  );
}
