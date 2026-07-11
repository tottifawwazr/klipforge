import { AnalyticsPreview } from "@/components/landing/analytics-preview";
import { FinalCta } from "@/components/landing/final-cta";
import { HeroSection } from "@/components/landing/hero-section";
import { ProductOverview } from "@/components/landing/product-overview";
import { RoleSection } from "@/components/landing/role-section";
import { SiteFooter } from "@/components/landing/site-footer";
import { SiteHeader } from "@/components/landing/site-header";
import { WorkflowSection } from "@/components/landing/workflow-section";

export default function Home() {
  return (
    <div className="min-h-dvh bg-canvas text-foreground">
      <SiteHeader />
      <main id="main-content">
        <HeroSection />
        <ProductOverview />
        <WorkflowSection />
        <RoleSection />
        <AnalyticsPreview />
        <FinalCta />
      </main>
      <SiteFooter />
    </div>
  );
}
