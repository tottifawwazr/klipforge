import { z } from "zod";

const dependencyHealthSchema = z.object({
  status: z.enum(["up", "down"]),
  latency_ms: z.number().nonnegative(),
});

export const apiHealthSchema = z.object({
  status: z.enum(["ok", "degraded"]),
  service: z.literal("klipforge-api"),
  version: z.string().min(1),
  timestamp: z.string().datetime(),
  checks: z.object({
    postgres: dependencyHealthSchema,
    redis: dependencyHealthSchema,
  }),
});

const connectivityResponseSchema = z.discriminatedUnion("connected", [
  z.object({ connected: z.literal(true), api: apiHealthSchema }),
  z.object({ connected: z.literal(false), error: z.string().min(1) }),
]);

export type ApiHealth = z.infer<typeof apiHealthSchema>;

export async function getApiHealth(fetcher: typeof fetch = fetch): Promise<ApiHealth> {
  const response = await fetcher("/api/health", {
    cache: "no-store",
    headers: { Accept: "application/json" },
  });
  const payload: unknown = await response.json();
  const parsed = connectivityResponseSchema.safeParse(payload);

  if (!parsed.success) {
    throw new Error("The API connectivity response was invalid.");
  }

  if (!response.ok || !parsed.data.connected) {
    const message = parsed.data.connected ? "The API is unavailable." : parsed.data.error;
    throw new Error(message);
  }

  return parsed.data.api;
}
