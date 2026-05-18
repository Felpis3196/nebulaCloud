"use client";

import { useMemo, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { toast } from "sonner";
import { ArrowRight, Loader2 } from "lucide-react";
import { useTranslations } from "next-intl";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ApiError } from "@/lib/api-client";
import { useAuthStore } from "@/stores/auth-store";

type FormValues = {
  email: string;
  displayName?: string;
  password: string;
  confirm: string;
};

export default function RegisterPage() {
  const t = useTranslations("auth");
  const tCommon = useTranslations("common");
  const router = useRouter();
  const registerUser = useAuthStore((s) => s.register);

  const schema = useMemo(
    () =>
      z
        .object({
          email: z.string().email(t("validation.email")),
          displayName: z
            .string()
            .max(64, t("validation.tooLong"))
            .optional()
            .or(z.literal("")),
          password: z.string().min(12, t("validation.passwordMin")),
          confirm: z.string(),
        })
        .refine((d) => d.password === d.confirm, {
          message: t("validation.passwordMismatch"),
          path: ["confirm"],
        }),
    [t],
  );

  const [submitting, setSubmitting] = useState(false);
  const {
    register,
    handleSubmit,
    formState: { errors },
    setError,
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { email: "", displayName: "", password: "", confirm: "" },
  });

  async function onSubmit(values: FormValues) {
    setSubmitting(true);
    try {
      await registerUser(
        values.email,
        values.password,
        values.displayName?.trim() || undefined,
      );
      toast.success(t("accountCreated"));
      router.push("/dashboard");
    } catch (err) {
      const message =
        err instanceof ApiError ? err.message : tCommon("unexpectedError");
      setError("root", { message });
      toast.error(message);
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="space-y-8">
      <div className="space-y-2">
        <h1 className="text-2xl font-semibold tracking-tight">{t("createWorkspace")}</h1>
        <p className="text-sm text-muted-foreground">{t("registerSubtitle")}</p>
      </div>

      <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
        <div className="space-y-2">
          <Label htmlFor="email">{t("email")}</Label>
          <Input
            id="email"
            type="email"
            autoComplete="email"
            placeholder={t("emailPlaceholder")}
            aria-invalid={!!errors.email}
            {...register("email")}
          />
          {errors.email && (
            <p className="text-xs text-destructive">{errors.email.message}</p>
          )}
        </div>

        <div className="space-y-2">
          <Label htmlFor="displayName">{t("displayNameOptional")}</Label>
          <Input
            id="displayName"
            type="text"
            autoComplete="name"
            placeholder={t("displayNamePlaceholder")}
            {...register("displayName")}
          />
        </div>

        <div className="space-y-2">
          <Label htmlFor="password">{t("password")}</Label>
          <Input
            id="password"
            type="password"
            autoComplete="new-password"
            placeholder={t("passwordMinPlaceholder")}
            aria-invalid={!!errors.password}
            {...register("password")}
          />
          {errors.password && (
            <p className="text-xs text-destructive">{errors.password.message}</p>
          )}
        </div>

        <div className="space-y-2">
          <Label htmlFor="confirm">{t("confirmPassword")}</Label>
          <Input
            id="confirm"
            type="password"
            autoComplete="new-password"
            aria-invalid={!!errors.confirm}
            {...register("confirm")}
          />
          {errors.confirm && (
            <p className="text-xs text-destructive">{errors.confirm.message}</p>
          )}
        </div>

        {errors.root && (
          <div className="rounded-md border border-destructive/40 bg-destructive/10 px-3 py-2 text-xs text-destructive">
            {errors.root.message}
          </div>
        )}

        <Button type="submit" variant="gradient" size="lg" className="w-full" disabled={submitting}>
          {submitting ? (
            <>
              <Loader2 className="animate-spin" />
              {t("creatingAccount")}
            </>
          ) : (
            <>
              {t("createAccountBtn")}
              <ArrowRight />
            </>
          )}
        </Button>
      </form>

      <p className="text-center text-xs text-muted-foreground">{t("terms")}</p>

      <p className="text-center text-sm text-muted-foreground">
        {t("alreadyHave")}{" "}
        <Link
          href="/login"
          className="font-medium text-foreground underline-offset-4 hover:underline"
        >
          {t("signIn")}
        </Link>
      </p>
    </div>
  );
}
