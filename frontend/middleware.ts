import { NextResponse, type NextRequest } from "next/server";

const COOKIE_ACCESS = "nebula_access";

const PROTECTED_PREFIXES = [
  "/projects",
  "/deployments",
  "/logs",
  "/metrics",
  "/domains",
  "/settings",
];

const AUTH_PAGES = ["/login", "/register"];

function isProtected(pathname: string): boolean {
  if (pathname === "/dashboard" || pathname.startsWith("/dashboard/")) return true;
  return PROTECTED_PREFIXES.some(
    (p) => pathname === p || pathname.startsWith(`${p}/`),
  );
}

/**
 * Lightweight auth gate.
 *
 *   - If user hits a protected route without `nebula_access`, redirect to /login?next=...
 *   - If user is already authenticated and visits /login or /register, push them to /
 */
export function middleware(req: NextRequest) {
  const { pathname, search } = req.nextUrl;
  const access = req.cookies.get(COOKIE_ACCESS)?.value;

  if (isProtected(pathname) && !access) {
    const url = req.nextUrl.clone();
    url.pathname = "/login";
    url.searchParams.set("next", pathname + search);
    return NextResponse.redirect(url);
  }

  if (AUTH_PAGES.includes(pathname) && access) {
    const url = req.nextUrl.clone();
    url.pathname = "/dashboard";
    url.search = "";
    return NextResponse.redirect(url);
  }

  return NextResponse.next();
}

export const config = {
  // Run on everything except Next internals & static assets.
  matcher: [
    "/((?!_next/static|_next/image|favicon.ico|fonts|images|api).*)",
  ],
};
