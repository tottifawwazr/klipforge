# KlipForge UI guidelines

## Purpose and authority

[`design-system.md`](./design-system.md) is the canonical source for KlipForge tokens and component specifications. This guide defines how those rules are applied across product surfaces, responsive layouts, interaction states, and delivery checks.

When the documents conflict, `design-system.md` takes precedence. Do not redefine token values here or introduce raw colors, arbitrary spacing, radii, shadows, or motion values in page code. Propose visual changes in the design system first.

These guidelines describe future product behavior without claiming that Phase 2–6 features already exist. They do not authorize redesigning the current landing page or changing the backend.

## Implementation baseline

- Use Next.js App Router and Server Components by default. Add `"use client"` only where browser state, event handling, or Recharts requires it.
- Read the locally installed Next.js guidance before implementation, as required by `apps/web/AGENTS.md`.
- Use Tailwind utilities backed by canonical semantic tokens. Arbitrary values require an approved design-system exception.
- Use Lucide exclusively for product interface icons. Never use emojis as interface icons.
- Use Recharts through the shared chart contract. Do not create per-chart palettes.
- Reuse shared components and variants instead of recreating buttons, fields, cards, badges, tables, or feedback locally.
- Give each screen or decision region one visually dominant primary action.

## Product surfaces and roles

### Public landing page

Use the most spacious density, concise readable copy, strong proof points, and one primary conversion action per section. An operations-focused sequence should lead from value proposition to proof or live product evidence, explain the workflow, and finish with a clear CTA.

Decorative effects remain restrained. Do not use oversized display type, heavy gradients, glass panels, or animation that delays content.

### Brand workspace

Prioritize campaign health, budget, submissions requiring attention, and measurable outcomes. Campaign creation and management actions must be obvious without competing with analytics. Financial totals and deadlines use tabular figures and explicit units.

### Clipper workspace

Prioritize available campaigns, submission progress, revision requests, approval status, and expected or completed earnings. Make the next action clear. Earnings and payout status must never be hidden or communicated through color alone.

### Admin workspace

Favor operational clarity and moderate density. Surface queues, exceptions, risk, and audit context before decorative summaries. Bulk, moderation, and payout actions require explicit selection state and confirmation. Separate destructive controls from routine actions.

### Analytics

Lead with the question being answered, then the primary KPI, visualization, interpretation, and exact underlying values. Filters and time ranges must be visible, shareable when useful, and preserved across navigation.

### Campaign management

Keep lifecycle status, owner, budget, schedule, eligibility, deliverables, and primary next action near the page title. Separate configuration from live performance. Changes with participant or payout impact require a consequence summary.

### Submission moderation

Keep media, campaign requirements, policy context, creator history where authorized, decision controls, and rationale close together. Approve/reject/revision decisions must have explicit labels and keyboard access. Rejection requires a useful reason; high-impact or bulk decisions require confirmation.

### Payout management

Make amount, currency, recipient, campaign, status, calculation basis, date, and consequence unambiguous before confirmation. Use tabular figures. Never abbreviate a final confirmation amount, and never use a toast as the only record of a payout result.

Role-specific UI visibility is not authorization. Backend authorization remains authoritative. When a visible destination is unavailable, explain why and provide a valid recovery or contact path when possible.

## Navigation and page structure

- Public pages use a compact top navigation. Authenticated desktop workspaces use one persistent sidebar for top-level destinations and a top bar for context and utilities.
- Compact workspaces use a top bar and accessible navigation sheet. Do not combine sidebar, tabs, and bottom navigation at the same hierarchy level.
- Keep navigation placement and labels stable. Mark the current destination with more than color alone.
- Give every important workspace view a stable URL. Preserve filters, search, pagination, and selected tabs in URL parameters when users benefit from refreshing, sharing, or returning to the view.
- Use breadcrumbs at three or more levels. Back navigation should restore the prior filter, selection, input, and scroll state.
- Page order is: optional breadcrumb, title and concise context, primary action, high-value summary, then detailed content.
- Provide a skip link. On client-side route changes, move focus to the page heading or main region without disrupting pointer users.
- Dialogs are for bounded decisions, not primary navigation. They require a labelled title, Escape handling, focus containment, a visible cancel/close action, and focus return.
- Place dangerous account actions away from ordinary navigation items.

## Responsive behavior

Use the breakpoint names and values in `design-system.md`. Do not create page-specific breakpoints.

- Build mobile-first. Show the core task first, stack controls, collapse secondary detail, and place non-primary actions in a labelled overflow menu.
- Do not allow horizontal page scrolling. A wide table may use a labelled, keyboard-accessible local scroll region only when column prioritization or a structured list would lose important relationships.
- Filters may collapse into a sheet or disclosure on compact screens. Show the active-filter count and a clear-all action.
- Card grids become one column without changing reading order or removing essential actions.
- Charts use `ResponsiveContainer`, a stable minimum height, fewer ticks, shorter labels, and an appropriate compact representation. Do not shrink labels below the canonical minimum.
- Sticky headers, action bars, and navigation reserve content space. Prevent content and focus indicators from being hidden.
- Use `min-h-dvh` for viewport layouts. Keep long-form copy within the readable measure.
- Controls and content must remain usable with enlarged text and browser zoom.

Verify at 375px, 768px, 1024px, and 1440px, plus compact landscape and 400% zoom/reflow.

## Forms and actions

- Every input has a persistent visible label. Placeholder text is an example or hint, never the only label.
- Associate help and error text with the control. Mark required fields in text and semantics.
- Use the correct input type, `inputMode`, and `autocomplete`. Group related controls with `fieldset` and `legend` where appropriate.
- Validate after blur and again on submit. Do not show errors while a user is entering an untouched value.
- Place a specific error beside the affected field. For multiple failures, add an error summary with links and focus the first invalid field.
- Error messages state what happened and how to recover. Preserve valid input after failure.
- Async actions provide immediate pressed feedback, prevent duplicate submission, retain stable button width, and use a meaningful progress label.
- Distinguish read-only, disabled, loading, and unavailable states visually and semantically.
- Confirm irreversible or high-impact campaign, moderation, bulk, and payout actions. State the affected item, amount/count, consequence, and whether reversal is possible.
- Offer undo for safely reversible actions and warn before discarding unsaved work.
- Confirm consequential success near the changed content without interrupting the workflow.

## Tables and data-dense views

- Use native table structure for genuinely tabular data, including scoped headers and an accessible name or caption.
- Make sortable headers buttons and expose `aria-sort`. Give selection checkboxes row-specific accessible names.
- Left-align identifiers and labels. Align comparable numeric values consistently and use tabular numerals.
- Use shared locale-aware formatters for dates, times, percentages, counts, and currency.
- Identify payout currency explicitly; do not depend on a symbol when ambiguity is possible. Show timezone context near deadlines and time-sensitive records.
- Keep the primary row action visible and put secondary actions in a labelled menu. Icon-only actions require an accessible name and tooltip.
- On compact screens, prioritize columns or use a structured list/card representation. Preserve reading order, status, and all critical actions.
- Use server pagination or incremental loading for large datasets. Virtualize long client-rendered lists when roughly 50 or more rows create measurable interaction cost.
- Provide loading, filtered-empty, first-use-empty, error, and permission states inside the table region.

## Charts and analytics

- Choose line for trends, bar for comparison, stacked bar for composition, and a table for exact lookup. Avoid pie or donut charts beyond five categories.
- Use the canonical chart palette. Never encode meaning with red/green or color alone; add direct labels, symbols, line styles, or patterns.
- Label axes and units. Format tooltip values with the same locale rules as adjacent totals.
- Multiple series need a nearby legend. Directly label simple charts when that is clearer.
- Interactive chart content must not rely on hover alone. Use Recharts accessibility support where applicable and provide an equivalent control or data view.
- Every chart has a concise text summary and an accessible table or data view. Business-critical analytics should support CSV export when implemented.
- Reduce tick density on compact screens. Aggregate or sample datasets above roughly 1,000 points and provide drill-down instead of rendering every point.
- Provide chart-specific loading, empty, error, and retry states. Never show an empty axis frame for loading or no data.
- Animation is optional, may not delay comprehension, and is disabled for reduced motion.

## Loading, empty, success, and error behavior

### Loading

Use App Router `loading.tsx` and focused `Suspense` boundaries for meaningful route or section loading. Reserve final geometry to prevent layout shift. From roughly 300ms to 1 second, a small labelled spinner is appropriate when the structure is already known. For longer or structure-fetching work, use a geometry-matched skeleton, or determinate progress when progress is measurable.

Use `aria-busy` on the affected region. Do not replace an entire page when only one panel is refreshing, and do not repeatedly announce background polling.

### Empty

Distinguish first-use empty, filtered no-results, zero-data, unavailable, and permission-denied states. Explain why the region is empty and provide one relevant next action, such as creating a campaign or clearing filters. A Lucide icon may support the message but must not replace it.

### Success

Confirm consequential actions near the changed content. Toasts may confirm nonblocking actions, use `aria-live="polite"`, do not steal focus, and normally dismiss after 3–5 seconds.

### Error

Use specific plain-language messages that state the failure and recovery path. Keep Retry local to the failed region. App Router route failures should use `error.tsx` with retry/reset, and missing resources should use `not-found.tsx`.

Persistent or actionable errors do not auto-dismiss. Do not expose stack traces, secrets, sensitive authorization details, or internal identifiers other than a safe support/reference ID.

## Accessibility requirements

KlipForge targets WCAG 2.2 AA.

- Normal text requires at least 4.5:1 contrast; large text and meaningful UI graphics require at least 3:1.
- All functionality works with keyboard alone. Focus order follows reading order, and every interactive element has a visible `focus-visible` treatment.
- Use native HTML before ARIA. Controls expose their accessible name, role, state, and relationships.
- Decorative Lucide icons use `aria-hidden="true"`; icon-only buttons receive a specific accessible name.
- Touch targets are at least 44 by 44 CSS pixels, with 8px between adjacent targets where practical.
- Never use color as the sole signal for status, validation, selection, or chart series.
- Meaningful images have useful alternative text; decorative images use empty alternative text. Media workflows provide captions/transcripts when applicable.
- Use sequential headings, labelled landmarks, and one clear `h1`.
- Do not disable browser zoom. Body text remains at least 16px on mobile, and layouts retain content and actions during zoom/reflow.
- Dynamic updates use an appropriate live region without stealing focus. Routine background updates should not chatter.
- Respect `prefers-reduced-motion`; content remains understandable with animation removed.
- Dialog focus, route focus, validation focus, and focus restoration must work predictably.
- Charts provide a text summary and accessible data alternative; sortable tables expose `aria-sort`.

## Motion and performance

- Motion communicates state, hierarchy, or causality. Do not animate decoration merely to make a screen feel active.
- Use canonical durations and easing. Exit motion finishes faster than entrance motion.
- Animate transform and opacity for movement. Avoid dimensions or position properties that trigger layout.
- Motion is interruptible and never blocks input. Limit a view to one or two meaningful animated elements.
- Prefer Server Components, parallel data fetching, streaming, and granular Suspense boundaries. Avoid client waterfalls and unnecessarily large client bundles.
- Use `next/image` with reserved dimensions or aspect ratio and modern formats.
- Use `next/font` for stable font loading and preload only critical variants.
- Dynamically load heavy client-only analytics when below the fold or not immediately needed. Reserve chart dimensions and target CLS below 0.1.
- Debounce high-frequency filters where appropriate. Paginate, aggregate, or virtualize large datasets.

## Content and formatting conventions

- Use concise, direct, sentence-case interface copy.
- Button labels start with specific verbs such as **Create campaign**, **Approve submission**, or **Retry**. Avoid vague labels such as **OK** or **Submit** when a precise action exists.
- Use product terms consistently: brand, clipper, campaign, submission, moderation, payout, and admin.
- Status labels describe a state, not an action. Use one controlled vocabulary across surfaces.
- Pair destructive language with the exact object and consequence. Avoid blame, jokes, emojis, or technical implementation detail in errors.
- Use centralized `Intl` formatters and show timezone context when dates affect deadlines.
- Do not manually concatenate currency symbols, date fragments, or percentage signs.
- Prefer wrapping. When truncation is necessary, make the full value available through expansion or an accessible tooltip.

## QA and definition of done

A UI change is complete only when all relevant checks pass.

### Visual consistency

- [ ] Uses semantic tokens and shared variants; no unexplained raw or arbitrary values.
- [ ] Uses Lucide consistently, with no interface emojis or mixed icon families.
- [ ] Preserves the standard card, border, radius, shadow, and primary-action hierarchy.
- [ ] Adds no unapproved gradient, glass effect, oversized heading, or decorative animation.

### Behavior and states

- [ ] Covers default, hover, active, focus-visible, disabled, loading, success, empty, error, and permission states where relevant.
- [ ] Async controls prevent duplicate actions and preserve context after failure.
- [ ] Destructive, moderation, bulk, and payout workflows communicate consequences and use suitable confirmation.

### Responsive quality

- [ ] Verified at 375px, 768px, 1024px, and 1440px, plus compact landscape.
- [ ] Has no horizontal page overflow, obscured content, clipped focus ring, or unreachable action.
- [ ] Tables, filters, charts, navigation, and sticky controls adapt without losing essential information.
- [ ] Remains usable during 400% zoom/reflow.

### Accessibility

- [ ] Keyboard-only navigation and logical focus order pass a manual check.
- [ ] Focus management works for routes, dialogs, validation, and dismissed overlays.
- [ ] Contrast, accessible names, landmarks, headings, live updates, and target sizes meet this guide.
- [ ] Meaning is preserved without color and with reduced motion enabled.
- [ ] Charts provide a text summary and accessible data alternative.

### Performance and engineering

- [ ] Loading placeholders reserve layout space and introduce no avoidable layout shift.
- [ ] Client components and heavy dependencies are limited to the smallest necessary boundary.
- [ ] No console errors or hydration warnings occur.
- [ ] Formatting, linting, type checking, relevant tests, and the production build pass.
