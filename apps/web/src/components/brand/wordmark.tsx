import Link from "next/link";
import { Clapperboard } from "lucide-react";

type WordmarkProps = {
  href?: string;
};

export function Wordmark({ href = "/" }: WordmarkProps) {
  return (
    <Link
      className="inline-flex min-h-11 items-center gap-3 rounded-xl"
      href={href}
      aria-label="KlipForge home"
    >
      <span className="grid size-10 place-items-center rounded-xl bg-primary text-foreground-inverse shadow-sm">
        <Clapperboard aria-hidden="true" size={21} strokeWidth={2.2} />
      </span>

      <span className="text-lg font-bold tracking-[-0.02em] text-foreground-strong">KlipForge</span>
    </Link>
  );
}
