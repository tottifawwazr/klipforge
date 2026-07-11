import Link from "next/link";
import type { ComponentProps } from "react";

type ButtonLinkProps = ComponentProps<typeof Link> & {
  variant?: "primary" | "secondary" | "quiet";
  size?: "default" | "large";
};

const variants = {
  primary:
    "bg-primary text-foreground-inverse shadow-[var(--kf-shadow-primary)] hover:bg-primary-hover active:bg-primary-active",

  secondary:
    "border border-border bg-surface text-foreground hover:border-border-strong hover:bg-surface-subtle active:bg-surface-muted",

  quiet: "text-foreground-secondary hover:bg-surface-muted hover:text-foreground-strong",
};

const sizes = {
  default: "min-h-11 px-4 text-sm",
  large: "min-h-12 px-6 text-sm",
};

export function ButtonLink({
  className = "",
  variant = "primary",
  size = "default",
  ...props
}: ButtonLinkProps) {
  return (
    <Link
      className={[
        "inline-flex items-center justify-center gap-2 rounded-xl",
        "font-semibold transition-colors",
        variants[variant],
        sizes[size],
        className,
      ].join(" ")}
      {...props}
    />
  );
}
