import { BarChart3, CheckCircle2, Clapperboard, DatabaseZap, ServerCog, ShieldCheck } from "lucide-react";
import { ApiHealthCard } from "@/components/api-health-card";

const foundations = [
  {
    title: "Typed product surface",
    description:
      "Next.js App Router, strict TypeScript, Tailwind CSS, and a query layer ready for product workflows.",
    icon: Clapperboard,
  },
  {
    title: "Operational API core",
    description:
      "A versioned Go API with structured logs, request tracing, dependency checks, and graceful shutdown.",
    icon: ServerCog,
  },
  {
    title: "Durable infrastructure",
    description:
      "Health-checked PostgreSQL and Redis services provide the foundation for upcoming campaign operations.",
    icon: DatabaseZap,
  },
];

export default function Home() {
  return (
    <main className="min-h-screen overflow-hidden bg-[#f7f9f7] text-[#13231c]">
      <header className="mx-auto flex w-full max-w-7xl items-center justify-between px-6 py-6 lg:px-10">
        <a className="flex items-center gap-3" href="#top" aria-label="KlipForge home">
          <span className="grid size-10 place-items-center rounded-xl bg-[#137a4b] text-white shadow-sm">
            <Clapperboard aria-hidden="true" size={21} strokeWidth={2.2} />
          </span>
          <span className="text-lg font-bold tracking-[-0.02em]">KlipForge</span>
        </a>
        <div className="hidden items-center gap-2 rounded-full border border-[#d9e4dd] bg-white px-3 py-1.5 text-xs font-semibold text-[#466255] shadow-sm sm:flex">
          <ShieldCheck aria-hidden="true" className="text-[#137a4b]" size={15} />
          Phase 1 foundation
        </div>
      </header>

      <section
        id="top"
        className="mx-auto grid w-full max-w-7xl gap-12 px-6 pb-20 pt-12 lg:grid-cols-[1.1fr_0.9fr] lg:items-center lg:px-10 lg:pb-28 lg:pt-20"
      >
        <div>
          <div className="mb-6 inline-flex items-center gap-2 rounded-full border border-[#b9d9c7] bg-[#eaf7ef] px-3 py-1.5 text-sm font-semibold text-[#11663f]">
            <CheckCircle2 aria-hidden="true" size={16} />
            Creator operations, built to stay accountable
          </div>
          <h1 className="max-w-3xl text-balance text-5xl font-bold leading-[1.04] tracking-[-0.045em] text-[#10241b] sm:text-6xl lg:text-7xl">
            Turn short-form campaigns into measurable momentum.
          </h1>
          <p className="mt-7 max-w-2xl text-pretty text-lg leading-8 text-[#53695f] sm:text-xl">
            KlipForge is becoming one dependable workspace for brands and creators to coordinate campaigns,
            validate performance, and understand every payout.
          </p>
          <div className="mt-9 flex flex-col gap-3 sm:flex-row">
            <a
              className="inline-flex h-12 items-center justify-center rounded-xl bg-[#137a4b] px-6 text-sm font-bold text-white shadow-[0_10px_24px_rgba(19,122,75,0.18)] transition hover:bg-[#0e6940] focus:outline-none focus:ring-2 focus:ring-[#137a4b] focus:ring-offset-2"
              href="#foundation"
            >
              View the foundation
            </a>
            <span className="inline-flex h-12 items-center justify-center gap-2 rounded-xl border border-[#d9e4dd] bg-white px-5 text-sm font-semibold text-[#466255]">
              <BarChart3 aria-hidden="true" size={17} />
              Product workflows arrive in later phases
            </span>
          </div>
        </div>

        <ApiHealthCard />
      </section>

      <section id="foundation" className="border-y border-[#dfe8e2] bg-white/75">
        <div className="mx-auto w-full max-w-7xl px-6 py-16 lg:px-10 lg:py-20">
          <div className="max-w-2xl">
            <p className="text-sm font-bold uppercase tracking-[0.18em] text-[#137a4b]">Foundation status</p>
            <h2 className="mt-3 text-3xl font-bold tracking-[-0.035em] text-[#10241b] sm:text-4xl">
              A small first slice, wired end to end.
            </h2>
            <p className="mt-4 text-base leading-7 text-[#60746a]">
              This phase establishes the production-shaped runtime. Campaigns, identity, metrics, and payouts
              remain explicitly outside the current scope.
            </p>
          </div>

          <div className="mt-10 grid gap-5 md:grid-cols-3">
            {foundations.map(({ title, description, icon: Icon }) => (
              <article
                key={title}
                className="rounded-2xl border border-[#dfe8e2] bg-white p-6 shadow-[0_12px_35px_rgba(32,66,50,0.06)]"
              >
                <span className="grid size-11 place-items-center rounded-xl bg-[#eaf7ef] text-[#137a4b]">
                  <Icon aria-hidden="true" size={22} />
                </span>
                <h3 className="mt-5 text-lg font-bold tracking-[-0.02em]">{title}</h3>
                <p className="mt-2 text-sm leading-6 text-[#60746a]">{description}</p>
              </article>
            ))}
          </div>
        </div>
      </section>

      <footer className="mx-auto flex w-full max-w-7xl flex-col gap-3 px-6 py-8 text-sm text-[#60746a] sm:flex-row sm:items-center sm:justify-between lg:px-10">
        <span className="font-semibold text-[#324a3e]">KlipForge</span>
        <span>Original local-development portfolio project · Phase 1</span>
      </footer>
    </main>
  );
}
