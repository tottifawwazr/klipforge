import { Wordmark } from "@/components/brand/wordmark";
import { SectionContainer } from "@/components/ui/section-container";

const footerLinks = [
  { href: "#overview", label: "Platform" },
  { href: "#how-it-works", label: "How it works" },
  { href: "#roles", label: "For teams" },
  { href: "#analytics", label: "Analytics preview" },
];

export function SiteFooter() {
  return (
    <footer className="bg-canvas py-10">
      <SectionContainer className="flex flex-col gap-8 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <Wordmark href="#top" />
          <p className="mt-4 max-w-md text-sm leading-6 text-foreground-secondary">
            A dependable operating model for creator campaigns, from the brief to the payout-ready record.
          </p>
        </div>

        <nav className="flex flex-wrap gap-x-5 gap-y-3 text-sm font-semibold" aria-label="Footer navigation">
          {footerLinks.map((link) => (
            <a
              key={link.href}
              className="text-foreground-secondary transition-colors hover:text-foreground-strong"
              href={link.href}
            >
              {link.label}
            </a>
          ))}
        </nav>
      </SectionContainer>

      <SectionContainer className="mt-8 border-t border-border-subtle pt-6 text-sm text-foreground-muted">
        <p>Local development project · Phase 1 foundation</p>
      </SectionContainer>
    </footer>
  );
}
