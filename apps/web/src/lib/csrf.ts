/** Match the browser origin against the public Host header, not Next's normalized internal URL. */
export function isSameOrigin(origin: string | null, host: string | null): boolean {
  if (!origin || !host) return false;
  try {
    const url = new URL(origin);
    return ["http:", "https:"].includes(url.protocol) && url.origin === origin && url.host === host;
  } catch {
    return false;
  }
}
