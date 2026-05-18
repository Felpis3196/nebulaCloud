import { LOCALE_COOKIE, type Locale } from "@/i18n/config";

/** Persist locale in a cookie (read by `src/i18n/request.ts`). */
export function setLocaleCookie(locale: Locale) {
  document.cookie = `${LOCALE_COOKIE}=${locale};path=/;max-age=31536000;SameSite=Lax`;
}
