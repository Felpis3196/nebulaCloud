"use client";

import { Github } from "lucide-react";
import { useTranslations } from "next-intl";
import { Button } from "@/components/ui/button";
import { githubOAuthStartUrl } from "@/lib/github-oauth";

interface Props {
  returnTo: string;
  variant?: "outline" | "secondary" | "ghost";
  className?: string;
}

export function GitHubSignInButton({ returnTo, variant = "outline", className }: Props) {
  const t = useTranslations("auth");

  return (
    <Button
      type="button"
      variant={variant}
      size="lg"
      className={className ?? "w-full"}
      onClick={() => {
        window.location.href = githubOAuthStartUrl(returnTo);
      }}
    >
      <Github className="h-4 w-4" />
      {t("continueWithGitHub")}
    </Button>
  );
}
