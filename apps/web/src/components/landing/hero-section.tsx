import { ArrowRight, CheckCircle2, ShieldCheck } from "lucide-react";
import { ApiHealthCard } from "@/components/api-health-card";
import { ButtonLink } from "@/components/ui/button-link";
import { SectionContainer } from "@/components/ui/section-container";

const assurances = ["Clear campaign ownership", "Submission traceability", "Payout-ready reporting"];

export function HeroSection() {
  return (
    <section id="top" className="overflow-hidden bg-canvas pb-20 pt-14 sm:pb-24 sm:pt-20 lg:pb-28">
      <SectionContainer className="grid gap-12 lg:grid-cols-[minmax(0,1.08fr)_minmax(24rem,0.92fr)] lg:items-center">
        <div className="max-w-2xl">
          <p className="inline-flex items-center gap-2 rounded-full border border-primary-border bg-primary-subtle px-3 py-1.5 text-sm font-semibold text-primary-on-subtle">
            <ShieldCheck aria-hidden="true" size={16} />
            Creator campaign operations, made accountable
          </p>

          <h1 className="mt-6 max-w-xl text-balance text-4xl font-bold tracking-[-0.035em] text-foreground-strong sm:text-5xl sm:leading-[1.08]">
            Build momentum from every short-form campaign.
          </h1>

          <p className="mt-6 max-w-xl text-pretty text-base leading-7 text-foreground-secondary sm:text-lg sm:leading-8">
            KlipForge gives brands and clippers one dependable workspace for campaign coordination, submission
            review, performance context, and payout-ready reporting.
          </p>

          <div className="mt-8 flex flex-col gap-3 sm:flex-row">
            <ButtonLink href="#how-it-works" size="large">
              See how it works
              <ArrowRight aria-hidden="true" size={20} />
            </ButtonLink>
            <ButtonLink href="#health-check" size="large" variant="secondary">
              View live system status
            </ButtonLink>
          </div>

          <ul
            className="mt-8 grid gap-3 text-sm font-semibold text-foreground-secondary sm:grid-cols-3"
            aria-label="KlipForge principles"
          >
            {assurances.map((assurance) => (
              <li key={assurance} className="flex items-center gap-2">
                <CheckCircle2 aria-hidden="true" className="shrink-0 text-primary" size={18} />
                {assurance}
              </li>
            ))}
          </ul>
        </div>

        <div id="health-check" className="scroll-mt-6">
          <ApiHealthCard />
        </div>
      </SectionContainer>
    </section>
  );
}
