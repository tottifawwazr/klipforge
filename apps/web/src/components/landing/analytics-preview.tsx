import { BarChart3, TrendingUp } from "lucide-react";
import { SectionContainer } from "@/components/ui/section-container";

const metrics = [
  { label: "Qualified submissions", value: "48" },
  { label: "Review-ready clips", value: "31" },
  { label: "Tracked views", value: "1.2M" },
];

export function AnalyticsPreview() {
  return (
    <section id="analytics" className="py-16 sm:py-20 lg:py-24">
      <SectionContainer className="grid gap-10 lg:grid-cols-[minmax(0,0.82fr)_minmax(0,1.18fr)] lg:items-center lg:gap-16">
        <div className="max-w-xl">
          <p className="text-xs font-bold uppercase tracking-[0.12em] text-primary">
            Campaign analytics preview
          </p>
          <h2 className="mt-3 text-3xl font-bold tracking-[-0.02em] text-foreground-strong sm:text-4xl">
            Read the progress behind the campaign, not just the headline number.
          </h2>
          <p className="mt-4 text-base leading-7 text-foreground-secondary">
            KlipForge is designed to keep delivery and performance context together, so campaign decisions
            have a clearer operational trail.
          </p>

          <div className="mt-8 flex items-start gap-3 rounded-xl border border-primary-border bg-primary-subtle p-4 text-sm leading-6 text-primary-on-subtle">
            <TrendingUp aria-hidden="true" className="mt-0.5 shrink-0" size={20} />
            <p>
              The preview illustrates the reporting structure. Live campaign analytics arrive with the product
              workflow, not as simulated data.
            </p>
          </div>
        </div>

        <figure className="rounded-2xl border border-border bg-surface p-5 shadow-[var(--kf-shadow-card)] sm:p-6">
          <div className="flex flex-wrap items-start justify-between gap-4">
            <div>
              <p className="text-sm font-semibold text-foreground-strong">Campaign performance overview</p>
              <p className="mt-1 text-sm text-foreground-muted">Illustrative 30-day reporting layout</p>
            </div>
            <span className="grid size-10 place-items-center rounded-xl bg-primary-subtle text-primary">
              <BarChart3 aria-hidden="true" size={20} />
            </span>
          </div>

          <dl className="mt-6 grid grid-cols-3 gap-3">
            {metrics.map(({ label, value }) => (
              <div
                key={label}
                className="rounded-xl border border-border-subtle bg-surface-subtle p-3 sm:p-4"
              >
                <dt className="text-xs font-medium leading-5 text-foreground-muted">{label}</dt>
                <dd className="mt-2 text-xl font-bold tracking-[-0.02em] text-foreground-strong tabular-nums sm:text-2xl">
                  {value}
                </dd>
              </div>
            ))}
          </dl>

          <div className="mt-6 rounded-xl border border-border-subtle bg-surface-subtle p-4">
            <svg
              className="h-44 w-full"
              viewBox="0 0 560 176"
              role="img"
              aria-labelledby="analytics-chart-title analytics-chart-description"
            >
              <title id="analytics-chart-title">Illustrative campaign performance trend</title>
              <desc id="analytics-chart-description">
                A single green trend line rises over four weekly reporting periods, with a dashed reference
                line.
              </desc>
              <line x1="20" x2="540" y1="144" y2="144" className="stroke-border" strokeWidth="1" />
              <line
                x1="20"
                x2="540"
                y1="96"
                y2="96"
                className="stroke-border-subtle"
                strokeDasharray="4 6"
                strokeWidth="1"
              />
              <line
                x1="20"
                x2="540"
                y1="48"
                y2="48"
                className="stroke-border-subtle"
                strokeDasharray="4 6"
                strokeWidth="1"
              />
              <polyline
                className="fill-none stroke-primary"
                points="20,132 94,118 168,122 242,94 316,102 390,68 464,76 540,36"
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth="4"
              />
              {[20, 168, 316, 464, 540].map((cx) => (
                <circle
                  key={cx}
                  className="fill-primary"
                  cx={cx}
                  cy={cx === 20 ? 132 : cx === 168 ? 122 : cx === 316 ? 102 : cx === 464 ? 76 : 36}
                  r="4"
                />
              ))}
            </svg>
            <div className="mt-2 flex justify-between text-xs font-medium text-foreground-muted">
              <span>Week 1</span>
              <span>Week 2</span>
              <span>Week 3</span>
              <span>Week 4</span>
            </div>
          </div>

          <figcaption className="mt-4 text-sm leading-6 text-foreground-secondary">
            Example reporting keeps delivery signals, review readiness, and measured reach in one readable
            view.
          </figcaption>
        </figure>
      </SectionContainer>
    </section>
  );
}
