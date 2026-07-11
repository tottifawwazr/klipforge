import { ArrowRight, CircleCheckBig } from "lucide-react";
import { ButtonLink } from "@/components/ui/button-link";
import { SectionContainer } from "@/components/ui/section-container";

export function FinalCta() {
  return (
    <section id="get-started" className="scroll-mt-6 border-y border-border bg-surface-subtle py-16 sm:py-20">
      <SectionContainer>
        <div className="rounded-2xl border border-border bg-surface px-6 py-10 shadow-[var(--kf-shadow-card)] sm:px-10 sm:py-12">
          <div className="max-w-2xl">
            <p className="inline-flex items-center gap-2 text-sm font-bold text-primary">
              <CircleCheckBig aria-hidden="true" size={18} />
              Built for accountable growth
            </p>
            <h2 className="mt-4 text-3xl font-bold tracking-[-0.02em] text-foreground-strong sm:text-4xl">
              See the operating model before the workflow ships.
            </h2>
            <p className="mt-4 text-base leading-7 text-foreground-secondary">
              KlipForge is currently building the campaign, analytics, moderation, and payout workflows
              described here. The live health check reflects the working Phase 1 foundation.
            </p>
          </div>

          <div className="mt-8 flex flex-col gap-3 sm:flex-row">
            <ButtonLink href="#how-it-works" size="large">
              Review the workflow
              <ArrowRight aria-hidden="true" size={20} />
            </ButtonLink>
            <ButtonLink href="#health-check" size="large" variant="secondary">
              Check the live connection
            </ButtonLink>
          </div>
        </div>
      </SectionContainer>
    </section>
  );
}
