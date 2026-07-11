import { ArrowRight, Menu } from "lucide-react";
import { Wordmark } from "@/components/brand/wordmark";
import { ButtonLink } from "@/components/ui/button-link";
import { SectionContainer } from "@/components/ui/section-container";

const navigation = [
  { href: "#overview", label: "Platform" },
  { href: "#how-it-works", label: "How it works" },
  { href: "#roles", label: "For teams" },
  { href: "#analytics", label: "Analytics" },
];

export function SiteHeader() {
  return (
    <header className="border-b border-border bg-canvas">
      <SectionContainer className="flex min-h-16 items-center justify-between gap-4">
        <Wordmark href="#top" />

        <nav className="hidden items-center gap-1 md:flex" aria-label="Primary navigation">
          {navigation.map((item) => (
            <a
              key={item.href}
              className="inline-flex min-h-11 items-center rounded-xl px-3 text-sm font-semibold text-foreground-secondary transition-colors hover:bg-surface-muted hover:text-foreground-strong"
              href={item.href}
            >
              {item.label}
            </a>
          ))}
        </nav>

        <div className="hidden items-center gap-3 md:flex">
          <ButtonLink href="#health-check" variant="secondary">
            Live status
          </ButtonLink>
          <ButtonLink href="#get-started">
            Explore KlipForge
            <ArrowRight aria-hidden="true" size={20} />
          </ButtonLink>
        </div>

        <details className="relative md:hidden">
          <summary className="flex size-11 cursor-pointer list-none items-center justify-center rounded-xl border border-border bg-surface text-foreground-secondary transition-colors hover:border-border-strong hover:bg-surface-subtle [&::-webkit-details-marker]:hidden">
            <Menu aria-hidden="true" size={20} />
            <span className="sr-only">Open navigation menu</span>
          </summary>

          <nav
            className="absolute right-0 top-[calc(100%+0.5rem)] z-30 flex w-64 flex-col rounded-2xl border border-border bg-surface p-2 shadow-[var(--kf-shadow-raised)]"
            aria-label="Mobile navigation"
          >
            {navigation.map((item) => (
              <a
                key={item.href}
                className="inline-flex min-h-11 items-center rounded-xl px-3 text-sm font-semibold text-foreground-secondary transition-colors hover:bg-surface-muted hover:text-foreground-strong"
                href={item.href}
              >
                {item.label}
              </a>
            ))}
            <div className="mt-2 border-t border-border-subtle pt-2">
              <ButtonLink className="w-full" href="#get-started">
                Explore KlipForge
                <ArrowRight aria-hidden="true" size={20} />
              </ButtonLink>
            </div>
          </nav>
        </details>
      </SectionContainer>
    </header>
  );
}
