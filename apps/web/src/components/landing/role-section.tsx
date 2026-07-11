import { BadgeDollarSign, BriefcaseBusiness, ClipboardList, UsersRound } from "lucide-react";
import { SectionContainer } from "@/components/ui/section-container";

const roles = [
  {
    label: "For brands",
    title: "Stay close to the work that shapes campaign outcomes.",
    description:
      "Coordinate campaign expectations, understand submission progress, and retain a clean line from activity to performance context.",
    benefits: [
      { label: "Campaign coordination", icon: BriefcaseBusiness },
      { label: "Review-ready submissions", icon: ClipboardList },
      { label: "Performance context", icon: UsersRound },
    ],
  },
  {
    label: "For clippers",
    title: "Know what to make, what happens next, and what it is worth.",
    description:
      "Work from a clear brief, follow submission feedback, and see the information that informs expected and completed earnings.",
    benefits: [
      { label: "Clear creative direction", icon: ClipboardList },
      { label: "Visible review status", icon: UsersRound },
      { label: "Payout-aware records", icon: BadgeDollarSign },
    ],
  },
];

export function RoleSection() {
  return (
    <section id="roles" className="border-y border-border bg-surface-subtle py-16 sm:py-20">
      <SectionContainer>
        <div className="max-w-2xl">
          <p className="text-xs font-bold uppercase tracking-[0.12em] text-primary">
            Designed for both sides
          </p>
          <h2 className="mt-3 text-3xl font-bold tracking-[-0.02em] text-foreground-strong sm:text-4xl">
            Shared context without a one-size-fits-all workspace.
          </h2>
        </div>

        <div className="mt-10 grid gap-6 lg:grid-cols-2">
          {roles.map(({ label, title, description, benefits }) => (
            <article key={label} className="rounded-2xl border border-border bg-surface p-6 sm:p-8">
              <p className="text-sm font-bold text-primary">{label}</p>
              <h3 className="mt-3 max-w-lg text-2xl font-bold tracking-[-0.02em] text-foreground-strong">
                {title}
              </h3>
              <p className="mt-4 max-w-xl text-base leading-7 text-foreground-secondary">{description}</p>

              <ul className="mt-7 grid gap-3 sm:grid-cols-3">
                {benefits.map(({ label: benefit, icon: Icon }) => (
                  <li key={benefit} className="rounded-xl border border-border-subtle bg-surface-subtle p-4">
                    <Icon aria-hidden="true" className="text-primary" size={20} />
                    <span className="mt-3 block text-sm font-semibold text-foreground-strong">{benefit}</span>
                  </li>
                ))}
              </ul>
            </article>
          ))}
        </div>
      </SectionContainer>
    </section>
  );
}
