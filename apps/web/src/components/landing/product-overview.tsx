import { BarChart3, CircleCheckBig, ClipboardCheck, WalletCards } from "lucide-react";
import { SectionContainer } from "@/components/ui/section-container";

const capabilities = [
  {
    title: "Campaign command center",
    description: "Bring briefs, timelines, expectations, and creator activity into one operational view.",
    icon: ClipboardCheck,
  },
  {
    title: "Submission clarity",
    description:
      "Keep every clip, review decision, and requested revision connected to its campaign context.",
    icon: CircleCheckBig,
  },
  {
    title: "Performance context",
    description: "Turn activity into readable campaign signals before teams make their next decision.",
    icon: BarChart3,
  },
  {
    title: "Payout-ready records",
    description:
      "Make contribution, status, and financial context easy to inspect before payout operations begin.",
    icon: WalletCards,
  },
];

export function ProductOverview() {
  return (
    <section id="overview" className="border-y border-border bg-surface-subtle py-16 sm:py-20">
      <SectionContainer>
        <div className="max-w-2xl">
          <p className="text-xs font-bold uppercase tracking-[0.12em] text-primary">One operating picture</p>
          <h2 className="mt-3 text-3xl font-bold tracking-[-0.02em] text-foreground-strong sm:text-4xl">
            A platform built around accountable campaign work.
          </h2>
          <p className="mt-4 text-base leading-7 text-foreground-secondary">
            KlipForge is designed to make creator campaigns easier to coordinate without separating the work
            from its performance and payout context.
          </p>
        </div>

        <div className="mt-10 grid gap-4 sm:grid-cols-2 lg:grid-cols-4 lg:gap-6">
          {capabilities.map(({ title, description, icon: Icon }) => (
            <article key={title} className="rounded-2xl border border-border bg-surface p-5 sm:p-6">
              <span className="grid size-11 place-items-center rounded-xl bg-primary-subtle text-primary">
                <Icon aria-hidden="true" size={22} />
              </span>
              <h3 className="mt-5 text-xl font-semibold tracking-[-0.01em] text-foreground-strong">
                {title}
              </h3>
              <p className="mt-2 text-sm leading-6 text-foreground-secondary">{description}</p>
            </article>
          ))}
        </div>
      </SectionContainer>
    </section>
  );
}
