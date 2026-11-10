export class APIError extends Error {
  constructor(
    public status: number,
    message: string,
    public code?: string,
  ) {
    super(message);
    this.name = "APIError";
  }
}
let refresh: Promise<boolean> | undefined;
async function refreshSession() {
  const attempt = async () => {
    const current = await fetch("/api/backend/auth/me", { cache: "no-store" });
    if (current.ok) return true;
    return (await fetch("/api/backend/auth/refresh", { method: "POST" })).ok;
  };
  if (typeof navigator !== "undefined" && navigator.locks)
    return navigator.locks.request("klipforge-session-refresh", attempt);
  return attempt();
}
export async function api<T>(path: string, init: RequestInit = {}, retry = true): Promise<T> {
  const response = await fetch(`/api/backend/${path}`, {
    ...init,
    cache: "no-store",
    headers: { "Content-Type": "application/json", ...init.headers },
  });
  if (response.status === 401 && retry && (!path.startsWith("auth/") || path === "auth/me")) {
    refresh ??= refreshSession().finally(() => {
      refresh = undefined;
    });
    if (await refresh) return api<T>(path, init, false);
  }
  const body = await response.json().catch(() => ({}));
  if (!response.ok)
    throw new APIError(
      response.status,
      body.error?.message || "The request could not be completed. Please try again.",
      body.error?.code,
    );
  return body as T;
}
export type User = {
  id: string;
  email: string;
  role: "BRAND" | "CLIPPER" | "ADMIN";
  display_name: string;
  is_active: boolean;
};
