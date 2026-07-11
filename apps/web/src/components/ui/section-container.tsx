import type { HTMLAttributes } from "react";

type SectionContainerProps = HTMLAttributes<HTMLDivElement>;

export function SectionContainer({ className = "", ...props }: SectionContainerProps) {
  return <div className={`mx-auto w-full max-w-7xl px-4 sm:px-6 lg:px-10 ${className}`} {...props} />;
}
