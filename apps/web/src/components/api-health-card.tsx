"use client";

import { useQuery } from "@tanstack/react-query";
import { Activity, Check, CircleAlert, LoaderCircle, RefreshCw } from "lucide-react";
import { getApiHealth, type ApiHealth } from "@/lib/health";

type ApiHealthViewProps =
  | { state: "checking"; health?: never; onRetry?: never }
  | { state: "connected"; health: ApiHealth; onRetry?: never }
  | { state: "unavailable"; health?: never; onRetry: () => void };

export function ApiHealthView({ state, health, onRetry }: ApiHealthViewProps) {
  const isConnected = state === "connected";

  return (
    <aside className="relative rounded-3xl border border-[#d9e4dd] bg-white p-6 shadow-[0_24px_70px_rgba(32,66,50,0.12)] sm:p-8">
      <div
        className="absolute -right-12 -top-12 size-40 rounded-full bg-[#e4f5eb] blur-3xl"
        aria-hidden="true"
      />
      <div className="relative">
        <div className="flex items-start justify-between gap-4">
          <div>
            <p className="text-xs font-bold uppercase tracking-[0.18em] text-[#6a7f74]">Live system check</p>
            <h2 className="mt-2 text-2xl font-bold tracking-[-0.03em]">Frontend ↔ API</h2>
          </div>
          <span
            className={`grid size-11 shrink-0 place-items-center rounded-xl ${
              isConnected ? "bg-[#e3f6ea] text-[#137a4b]" : "bg-[#f2f4f2] text-[#6a7f74]"
            }`}
          >
            <Activity aria-hidden="true" size={22} />
          </span>
        </div>

        {state === "checking" ? (
          <div className="mt-8 flex min-h-32 items-center justify-center rounded-2xl border border-dashed border-[#ccd9d1] bg-[#fafcfa]">
            <div className="flex items-center gap-3 text-sm font-semibold text-[#60746a]">
              <LoaderCircle aria-hidden="true" className="animate-spin" size={18} />
              Checking the API runtime…
            </div>
          </div>
        ) : null}

        {state === "unavailable" ? (
          <div className="mt-8 rounded-2xl border border-[#f0d6cf] bg-[#fff8f5] p-5">
            <div className="flex items-start gap-3">
              <CircleAlert aria-hidden="true" className="mt-0.5 shrink-0 text-[#b84c32]" size={20} />
              <div>
                <p className="font-bold text-[#7e321f]">API connection unavailable</p>
                <p className="mt-1 text-sm leading-6 text-[#8b5b4f]">
                  Start the API and its dependencies, then run this live check again.
                </p>
                <button
                  className="mt-4 inline-flex items-center gap-2 rounded-lg border border-[#e3b9ae] bg-white px-3 py-2 text-sm font-bold text-[#8b3e2a] transition hover:bg-[#fff3ee]"
                  onClick={onRetry}
                  type="button"
                >
                  <RefreshCw aria-hidden="true" size={15} />
                  Retry check
                </button>
              </div>
            </div>
          </div>
        ) : null}

        {state === "connected" ? (
          <div className="mt-8">
            <div className="flex items-center gap-2 rounded-xl bg-[#eaf7ef] px-4 py-3 text-sm font-bold text-[#11663f]">
              <span className="grid size-5 place-items-center rounded-full bg-[#137a4b] text-white">
                <Check aria-hidden="true" size={13} strokeWidth={3} />
              </span>
              API connection verified
            </div>
            <dl className="mt-5 grid grid-cols-2 gap-3">
              {Object.entries(health.checks).map(([name, check]) => (
                <div key={name} className="rounded-xl border border-[#e1e9e4] bg-[#fafcfa] p-4">
                  <dt className="text-xs font-bold uppercase tracking-[0.13em] text-[#71847a]">{name}</dt>
                  <dd className="mt-2 flex items-center gap-2 text-sm font-bold capitalize text-[#274436]">
                    <span className="size-2 rounded-full bg-[#1b9a61]" aria-hidden="true" />
                    {check.status}
                    <span className="font-medium text-[#84948c]">· {check.latency_ms} ms</span>
                  </dd>
                </div>
              ))}
            </dl>
            <p className="mt-5 font-mono text-xs text-[#71847a]">
              {health.service} · {health.version}
            </p>
          </div>
        ) : null}
      </div>
    </aside>
  );
}

export function ApiHealthCard() {
  const healthQuery = useQuery({
    queryKey: ["api-health"],
    queryFn: () => getApiHealth(),
    refetchInterval: 10_000,
  });

  if (healthQuery.isPending) {
    return <ApiHealthView state="checking" />;
  }

  if (healthQuery.isError) {
    return <ApiHealthView state="unavailable" onRetry={() => void healthQuery.refetch()} />;
  }

  return <ApiHealthView state="connected" health={healthQuery.data} />;
}
