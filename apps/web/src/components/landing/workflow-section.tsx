import { CheckCheck, FileCheck2, Send } from "lucide-react";
import { SectionContainer } from "@/components/ui/section-container";

const steps = [
  {
    number: "01",
    title: "Set the campaign context",
    description: "Define the brief, delivery expectations, and the signals that matter before work begins.",
    icon: Send,
  },
  {
    number: "02",
    title: "Keep submissions moving",
    description: "Give brands and clippers a shared view of delivery, review, and revision status.",
    icon: FileCheck2,
  },
  {
    number: "03",
    title: "Close the loop with evidence",
    description: "Connect campaign progress to performance context and records ready for payout operations.",
    icon: CheckCheck,
  },
];

const highlights = [
  "Role-aware campaign context",
  "Clear operational states",
  "Reporting that keeps its source",
];

export function WorkflowSection() {
  return (
    <section id="how-it-works" className="py-16 sm:py-20 lg:py-24">
      <SectionContainer>
        <div className="grid gap-10 lg:grid-cols-[minmax(0,0.85fr)_minmax(0,1.15fr)] lg:gap-16">
          <div className="max-w-xl">
            <p className="text-xs font-bold uppercase tracking-[0.12em] text-primary">How KlipForge works</p>
            <h2 className="mt-3 text-3xl font-bold tracking-[-0.02em] text-foreground-strong sm:text-4xl">
              Structure the work before chasing the result.
            </h2>
            <p className="mt-4 text-base leading-7 text-foreground-secondary">
              Every step is designed to retain the campaign context that teams need to act with confidence.
            </p>

            <div className="mt-8 border-l-2 border-primary-border pl-5">
              <p className="text-sm font-semibold text-foreground-strong">Platform highlights</p>
              <ul className="mt-3 space-y-3 text-sm leading-6 text-foreground-secondary">
                {highlights.map((highlight) => (
                  <li key={highlight}>{highlight}</li>
                ))}
              </ul>
            </div>
          </div>

          <ol className="grid gap-4">
            {steps.map(({ number, title, description, icon: Icon }) => (
              <li
                key={number}
                className="grid gap-4 rounded-2xl border border-border bg-surface p-5 sm:grid-cols-[auto_1fr] sm:items-start sm:p-6"
              >
                <div className="flex items-center gap-3 sm:flex-col sm:items-start">
                  <span className="font-mono text-sm font-bold text-primary">{number}</span>
                  <span className="grid size-10 place-items-center rounded-xl bg-surface-muted text-foreground-secondary">
                    <Icon aria-hidden="true" size={20} />
                  </span>
                </div>
                <div>
                  <h3 className="text-xl font-semibold tracking-[-0.01em] text-foreground-strong">{title}</h3>
                  <p className="mt-2 text-sm leading-6 text-foreground-secondary">{description}</p>
                </div>
              </li>
            ))}
          </ol>
        </div>
      </SectionContainer>
    </section>
  );
}
