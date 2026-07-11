# KlipForge design system

Status: Canonical source of truth  
Version: 1.0  
Last updated: 2026-07-11

## Authority and scope

This document is the normative source of truth for KlipForge visual design. It applies to the public landing page and all future brand, clipper, admin, analytics, campaign, moderation, and payout surfaces.

When implementation begins, Tailwind and CSS tokens must mirror this document. Shared components must consume semantic tokens instead of raw values. If existing Phase 1 code differs, treat the code as legacy until a separately scoped migration is approved; this document does not authorize redesigning working pages.

[`ui-guidelines.md`](./ui-guidelines.md) defines how to apply this system. It may not redefine token values. Page-level exceptions require a documented rationale in this file and may not weaken accessibility requirements.

## Design basis

This system was developed with the installed `ui-ux-pro-max` skill at `.codex/skills/ui-ux-pro-max`. The design-system search used these dials:

- Design variance: 3/10 — centered and minimal.
- Motion intensity: 2/10 — subtle.
- Visual density: 6/10 — standard, with controlled dashboard density.

The adopted recommendations are an operations-focused SaaS pattern, Swiss-style structured minimalism, an Inter UI type system, a disciplined grid, status-aware data presentation, restrained motion, and accessibility-first interaction.

The product brief overrides incompatible generated candidates. KlipForge therefore does not use creator-pink branding, generic purple themes, exaggerated display type, animated card lift, or decorative gradients.

## Product character

KlipForge should feel accountable, operational, calm, precise, and creator-aware.

- Use a warm-neutral light canvas, white surfaces, dark charcoal text, and one green brand accent.
- Establish hierarchy with typography, spacing, alignment, borders, and content order before adding effects.
- Keep public pages spacious and conversion-focused; keep dashboards moderately dense and scannable.
- Make campaign status, moderation decisions, analytics, and payout consequences explicit.
- Use consistent component shells across roles. Information architecture may change; visual rules do not.

Do not use:

- excessive or decorative gradients;
- generic AI purple or pink themes;
- glassmorphism as a card system;
- emojis as interface icons;
- headings larger than the documented scale;
- ornamental, blocking, or page-wide animation;
- colored glow shadows or 3D decoration;
- ad hoc card radii, borders, padding, or elevation;
- raw color values in page and component code.

## Token architecture

Tokens have three layers:

1. **Foundation tokens** define color, type, spacing, radius, shadow, motion, and layout values.
2. **Semantic tokens** describe intent such as canvas, foreground, primary, danger, and focus.
3. **Component tokens** map semantics to buttons, fields, cards, tables, charts, badges, and navigation.

Components must use semantic or component tokens. A raw value is allowed only inside the central token definition.

## Colors

Light mode is the version 1 baseline. Do not ship a partial dark theme. A future dark theme must remap every semantic surface, foreground, border, state, chart, and focus token and pass a separate contrast review.

### Core semantic palette

| Token                           | Value                   | Use                                           |
| ------------------------------- | ----------------------- | --------------------------------------------- |
| `--color-canvas`                | `#F7F9F7`               | Default page background                       |
| `--color-surface`               | `#FFFFFF`               | Cards, controls, menus, dialogs               |
| `--color-surface-subtle`        | `#FAFCFA`               | Table headers and nested data regions         |
| `--color-surface-muted`         | `#F1F5F2`               | Quiet or disabled regions                     |
| `--color-surface-inverse`       | `#13231C`               | Rare inverse regions                          |
| `--color-foreground-strong`     | `#10241B`               | Headings and primary metrics                  |
| `--color-foreground`            | `#13231C`               | Default body text                             |
| `--color-foreground-secondary`  | `#53695F`               | Secondary copy                                |
| `--color-foreground-muted`      | `#60746A`               | Metadata and supporting copy                  |
| `--color-foreground-inverse`    | `#FFFFFF`               | Text on primary or inverse surfaces           |
| `--color-border-subtle`         | `#E1E9E4`               | Internal dividers                             |
| `--color-border`                | `#D9E4DD`               | Cards and nonessential boundaries             |
| `--color-border-strong`         | `#B9C9BF`               | Emphasized card boundaries                    |
| `--color-control-border`        | `#82968B`               | Input/control boundary; at least 3:1 on white |
| `--color-control-border-hover`  | `#60746A`               | Hovered input/control boundary                |
| `--color-primary`               | `#137A4B`               | Primary action and brand accent               |
| `--color-primary-hover`         | `#0E6940`               | Primary hover                                 |
| `--color-primary-active`        | `#0A5735`               | Primary pressed state                         |
| `--color-primary-subtle`        | `#EAF7EF`               | Selected or active background                 |
| `--color-primary-subtle-active` | `#D7F0E1`               | Selected pressed background                   |
| `--color-primary-border`        | `#B9D9C7`               | Accent boundary                               |
| `--color-primary-on-subtle`     | `#11663F`               | Text on the primary tint                      |
| `--color-focus`                 | `#137A4B`               | Focus outline and ring                        |
| `--color-selection`             | `#B9E6CB`               | Text selection                                |
| `--color-overlay`               | `rgba(8, 20, 14, 0.56)` | Modal and sheet scrim                         |
| `--color-disabled-background`   | `#EDF1EE`               | Disabled controls                             |
| `--color-disabled-foreground`   | `#718078`               | Disabled content only, never body copy        |

### Semantic states

| State         | Strong/action | Tint      | Border    | Text on tint |
| ------------- | ------------- | --------- | --------- | ------------ |
| Success       | `#137A4B`     | `#EAF7EF` | `#B9D9C7` | `#11663F`    |
| Warning       | `#B45309`     | `#FFF8E1` | `#E8C978` | `#7A4D00`    |
| Danger        | `#B42318`     | `#FFF1F0` | `#F4B8B2` | `#7A271A`    |
| Danger hover  | `#912018`     | —         | —         | —            |
| Danger active | `#7A1B14`     | —         | —         | —            |
| Info          | `#1D4ED8`     | `#EFF6FF` | `#BFDBFE` | `#1E3A8A`    |
| Neutral       | `#53695F`     | `#F1F5F2` | `#D9E4DD` | `#324A3E`    |

Color must never be the only state cue. Pair it with text and, where useful, a Lucide icon or another non-color indicator.

### Approved contrast pairs

| Foreground/background             | Contrast |
| --------------------------------- | -------: |
| Foreground on canvas              |  15.45:1 |
| Strong foreground on white        |  16.28:1 |
| Secondary foreground on canvas    |   5.59:1 |
| Muted foreground on canvas        |   4.72:1 |
| White on primary                  |   5.37:1 |
| White on primary hover            |   6.75:1 |
| Primary-on-subtle on primary tint |   6.36:1 |
| White on danger                   |   6.57:1 |
| Warning text on warning tint      |   6.84:1 |
| Info text on info tint            |   9.52:1 |

Normal text requires at least 4.5:1. Large text, focus indicators, control boundaries, and meaningful graphics require at least 3:1 against adjacent colors.

### Chart palette

Use this order for Recharts series. These colors are for data encoding, not decorative UI accents.

| Series | Value     | Default role                                  |
| ------ | --------- | --------------------------------------------- |
| 1      | `#137A4B` | Primary campaign series                       |
| 2      | `#2563EB` | Comparison series                             |
| 3      | `#B45309` | Secondary comparison or warning-adjacent data |
| 4      | `#0F766E` | Additional positive/neutral series            |
| 5      | `#B42318` | Negative or exception series                  |
| 6      | `#1E3A5F` | Baseline or reference series                  |

Differentiate multiple series with direct labels, symbols, and solid/dashed/dotted strokes as well as color. Do not use red and green as the only pair.

## Typography

Use Inter for headings, body copy, labels, controls, and data. When implementation is authorized, load it through `next/font` with stable fallback behavior. Use only weights 400, 500, 600, and 700.

Fallback stack:

```css
Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif
```

Use the existing system mono stack only for code, identifiers, versions, and diagnostics. Monetary values, counts, percentages, durations, and table numerics use `font-variant-numeric: tabular-nums`; they do not switch to a monospace font.

| Role       | Size / line height | Weight |   Tracking | Use                                   |
| ---------- | -----------------: | -----: | ---------: | ------------------------------------- |
| Display    |        `56 / 60px` |    700 | `-0.035em` | Marketing hero only; absolute maximum |
| Heading 1  |        `40 / 48px` |    700 | `-0.025em` | Public page title                     |
| Heading 2  |        `32 / 40px` |    700 | `-0.020em` | App page title or major section       |
| Heading 3  |        `24 / 32px` |    700 | `-0.010em` | Panel or card group title             |
| Heading 4  |        `20 / 28px` |    600 | `-0.005em` | Card or empty-state title             |
| Metric     |        `32 / 40px` |    700 | `-0.020em` | Primary KPI value                     |
| Body large |        `18 / 28px` |    400 |        `0` | Introductory copy only                |
| Body       |        `16 / 24px` |    400 |        `0` | Default copy and forms                |
| Body small |        `14 / 20px` |    400 |        `0` | Dense UI and table content            |
| Label      |        `14 / 20px` |    600 |        `0` | Controls, navigation, fields          |
| Caption    |        `12 / 16px` |    500 |   `0.01em` | Metadata only                         |
| Overline   |        `12 / 16px` |    700 |   `0.12em` | Short uppercase section label only    |

Display type scales to `40/44px` below 768px, `48/52px` from 768px, and `56/60px` from 1280px. Application page titles use Heading 2, not Display. Do not use weights 800 or 900. Keep prose between 60 and 75 characters per line.

## Spacing and density

The base rhythm is 4px. Use 2px and 6px only for optical alignment, icons, or small badges.

| Token        |  Value | Token         |  Value |
| ------------ | -----: | ------------- | -----: |
| `--space-0`  |    `0` | `--space-0-5` |  `2px` |
| `--space-1`  |  `4px` | `--space-1-5` |  `6px` |
| `--space-2`  |  `8px` | `--space-3`   | `12px` |
| `--space-4`  | `16px` | `--space-5`   | `20px` |
| `--space-6`  | `24px` | `--space-8`   | `32px` |
| `--space-10` | `40px` | `--space-12`  | `48px` |
| `--space-16` | `64px` | `--space-20`  | `80px` |
| `--space-24` | `96px` |               |        |

Semantic spacing contracts:

- Interactive controls: 44px minimum height; large controls: 48px.
- Adjacent touch targets: at least 8px apart where practical.
- Card padding: 20px on compact screens, 24px from `md`.
- Card grid gap: 16px on compact screens, 24px from `md`.
- Dashboard section gap: 32px.
- Marketing section spacing: 64px compact, 80px desktop.
- Inline icon/text gap: 8px.

## Layout and responsive breakpoints

Use Tailwind's default breakpoints. Do not create one-off page breakpoints.

| Name  |      Width | Layout contract                                               |
| ----- | ---------: | ------------------------------------------------------------- |
| Base  |   `<640px` | One column, 16px gutter, mobile app bar                       |
| `sm`  |  `>=640px` | 24px gutter; actions may align horizontally                   |
| `md`  |  `>=768px` | 32px gutter; two columns where hierarchy permits              |
| `lg`  | `>=1024px` | 40px gutter; persistent dashboard sidebar                     |
| `xl`  | `>=1280px` | Full marketing type scale; preserve max widths                |
| `2xl` | `>=1536px` | Add outer whitespace; do not stretch content indiscriminately |

Use a 4-column grid below 768px, 8 columns from 768px, and 12 columns from 1024px. Grid gaps are 16px compact and 24px from `md`.

- Public container: 1280px maximum (`max-w-7xl`).
- Dashboard content: 1440px maximum, excluding the sidebar.
- Prose: 720px maximum.
- Primary forms: 640px maximum.
- Viewport layouts: use `min-height: 100dvh`, not fixed `100vh`.
- Horizontal page scrolling is prohibited. A table may own a labelled local scroll region as a last resort.

Required verification widths are 375px, 768px, 1024px, and 1440px, plus compact landscape and zoom/reflow checks.

## Radius, borders, shadows, and layers

### Radius

| Token           |    Value | Use                                             |
| --------------- | -------: | ----------------------------------------------- |
| `--radius-sm`   |    `6px` | Small tags and details                          |
| `--radius-md`   |    `8px` | Compact controls and menu items                 |
| `--radius-lg`   |   `12px` | Buttons, inputs, navigation items               |
| `--radius-xl`   |   `16px` | Standard cards and table shells                 |
| `--radius-2xl`  |   `24px` | One featured panel or dialog; not routine cards |
| `--radius-full` | `9999px` | Badges and avatars only                         |

Borders are 1px by default. Use 2px only for selected controls or focus treatment. Prefer a border to a shadow for routine surface separation.

### Shadows

| Token        | Value                                | Use                        |
| ------------ | ------------------------------------ | -------------------------- |
| `--shadow-0` | `none`                               | Default flat surface       |
| `--shadow-1` | `0 1px 2px rgba(16, 36, 27, 0.06)`   | Raised or interactive card |
| `--shadow-2` | `0 12px 32px rgba(16, 36, 27, 0.08)` | Dropdown or featured panel |
| `--shadow-3` | `0 24px 56px rgba(16, 36, 27, 0.12)` | Modal or dialog only       |

Do not use colored glow shadows. Do not translate or scale cards on hover.

### Layer scale

| Layer            | z-index |
| ---------------- | ------: |
| Base             |       0 |
| Raised           |      10 |
| Sticky           |      20 |
| Dropdown/popover |      30 |
| Overlay          |      40 |
| Modal/sheet      |      50 |
| Toast            |      60 |
| Tooltip          |      70 |

Components must not invent other z-index values.

## Icons

Lucide is the only product interface icon family.

- Inline icons: 16px.
- Controls and navigation: 20px.
- Feature icons: 24px.
- Empty-state icons: 32px.
- Default stroke: 2px; 2.5px is reserved for selected or confirmation glyphs.
- Decorative icons use `aria-hidden="true"`.
- Icon-only controls require a precise accessible name and a 44 by 44px target.
- Keep filled/outline treatment and stroke weight consistent at the same hierarchy.

Do not use arbitrary icon sizes, raster UI icons, mixed icon libraries, or emojis.

## Motion

| Token             |                        Value | Use                                      |
| ----------------- | ---------------------------: | ---------------------------------------- |
| `--duration-fast` |                      `150ms` | Hover, focus, pressed feedback           |
| `--duration-base` |                      `200ms` | Menus, disclosures, simple state changes |
| `--duration-slow` |                      `250ms` | Dialog and sheet entrance                |
| `--ease-standard` | `cubic-bezier(0.2, 0, 0, 1)` | Entering and state changes               |
| `--ease-exit`     | `cubic-bezier(0.4, 0, 1, 1)` | Exiting; finish faster than entrance     |

Motion must communicate state, hierarchy, or causality. Animate opacity, color, border-color, background-color, or transform. Avoid width, height, positional, scroll-jacking, parallax, bounce, and page-wide reveal animation. Motion must not block input.

`prefers-reduced-motion: reduce` must remove smooth scrolling and nonessential animation. Loading and status must remain understandable without motion.

## Buttons

All actions use semantic `<button>` or link elements. Only one primary action should appear per screen or bounded decision region.

Base button contract: 44px minimum height, Label typography, 12px radius, 16px horizontal padding, 8px content gap, and a 20px icon. Large buttons are 48px high with 20px horizontal padding. Icon-only buttons are 44 by 44px.

| Variant   | Default                                    | Hover                                | Active                |
| --------- | ------------------------------------------ | ------------------------------------ | --------------------- |
| Primary   | Primary background, white text, `shadow-1` | Primary hover                        | Primary active        |
| Secondary | Surface, foreground, control border        | Surface subtle, control-border-hover | Surface muted         |
| Tertiary  | Transparent, primary text                  | Primary subtle                       | Primary subtle active |
| Danger    | Danger background, white text              | Danger hover                         | Danger active         |
| Link      | Primary text                               | Underline + primary hover            | Primary active        |

Every enabled variant uses a 2px focus-visible outline in `--color-focus` with a 2px offset. Disabled buttons use disabled tokens, no shadow, semantic disabled behavior, and do not rely on opacity alone.

Loading buttons prevent repeat submission, keep their width and label, include a 16px spinner, and expose a polite progress message. Do not replace the label with an unlabeled spinner.

## Forms

- Use a visible Label 8px above every control. A placeholder is an example, never the only label.
- Text inputs and selects are at least 44px high, use 16px text on mobile, 12px horizontal padding, a white surface, the control border, and 12px radius.
- Textareas use the same treatment and begin at 112px high.
- Helper and error text use Body small and sit 6px below the control.
- Hover uses the control-border-hover token. Focus uses a primary border plus the standard focus outline.
- Invalid fields use the danger border and a specific recovery message associated with `aria-describedby`.
- Validate on blur and submit, not on every keystroke. Preserve input after failure and focus the first invalid field after submission.
- Checkbox and radio visuals are 20px, while the combined label/control target remains at least 44px.
- Read-only, disabled, loading, and unavailable states must look and behave differently.
- Use `fieldset` and `legend` for related controls, plus semantic input types, `inputMode`, and `autocomplete`.
- Payout and other financial actions show the currency, amount, recipient, and consequence before confirmation.

## Cards

Every card uses a white surface, a consistent border, and the same internal hierarchy. Do not nest cards more than one level.

| Variant            | Contract                                                                            |
| ------------------ | ----------------------------------------------------------------------------------- |
| Standard           | 16px radius, default border, `shadow-0`, 20px compact/24px desktop padding          |
| Metric             | Standard shell; 14px muted label, 32px tabular value, optional labelled delta       |
| Interactive        | Standard shell + `shadow-1`; strong hover border; one semantic link/button; no lift |
| Featured           | 24px radius, default border, `shadow-2`, 24/32px padding; at most one per view      |
| Nested data region | Subtle surface, subtle border, 12px radius, 16px padding, no heavy shadow           |

Use 16px between card header and action, 8px from title to description, and 20px before the footer. Align card actions consistently when cards appear in a grid. Glass/backdrop blur is not a card variant.

## Tables

- Place tables in a white surface with a 1px border and 16px outer radius. Do not round individual cells.
- Headers use the subtle surface, Label typography, muted text, and at least 44px height.
- Rows are at least 52px high and use Body small. Cells use 16px horizontal and 12px vertical padding with subtle horizontal dividers.
- Avoid full cell grids and zebra striping by default.
- Left-align labels. Right-align comparable currency, percentages, and counts; use tabular numerals and state units in headers.
- Sort buttons are keyboard-operable and expose `aria-sort`.
- Row action controls are 44 by 44px. Selected rows use a primary tint plus a non-color cue.
- Sticky headers remain opaque. Filter, sort, pagination, and selection state should survive navigation.
- Below 768px, prioritize columns, use a structured stacked-row view, or provide a labelled local scroll region as a last resort.
- Every table defines loading, first-use empty, filtered empty, error, and permission states.
- Bulk and destructive actions show the selection count and require confirmation when consequences are significant.

## Charts

Use Recharts through a shared chart wrapper. Do not repeat palette, axis, tooltip, or responsive behavior in individual features.

| Property             | Contract                                                             |
| -------------------- | -------------------------------------------------------------------- |
| Plot surface         | `--color-surface`                                                    |
| Grid                 | `--color-border-subtle`                                              |
| Axis and legend text | `--color-foreground-secondary`                                       |
| Series stroke        | 2px                                                                  |
| Point                | 4px visual with a larger keyboard/touch affordance                   |
| Height               | 240px compact, 320px default, 400px large                            |
| Tooltip              | White surface, default border, 12px radius, `shadow-2`, 12px padding |

Chart selection:

- Line: campaign, reach, view, conversion, or payout trend over time.
- Bar: category, campaign, creator, or leaderboard comparison.
- Stacked bar: composition over time.
- Bullet/progress: performance against a target; prefer this to a gauge.
- Area: one cumulative series only.
- Donut: at most five categories; use bars when precise comparison matters.

Every chart needs a visible title, time range and unit, locale-formatted tooltip values, nearby legend for multiple series, responsive tick reduction, a concise text summary, and an accessible table or data view. Interactive tooltips and marks cannot rely on hover alone.

Do not use 3D effects, chart gradients, decorative shadows, truncated axes, unexplained dual axes, or empty axes as loading states. Chart animation is off by default. If justified, limit initial opacity animation to 180ms and disable it for reduced motion.

## Badges and status

Badges are non-interactive labels: at least 24px high, Caption typography at weight 600, 8px horizontal and 2px vertical padding, 4px icon gap, full radius, and a 1px border.

Use neutral, success, warning, danger, or info state tokens. Always include explicit status text. Add a 16px icon when the state could be ambiguous. Do not use a bare colored dot.

Interactive filters are chips, not badges. Chips have a 44px target, 12px radius, keyboard-operable selected state, and `aria-pressed` or checkbox semantics.

## Navigation

### Public navigation

Use a 64px mobile and 72px desktop header with the brand on the left, concise navigation, and one primary CTA on the right. Below 768px, move links into a modal sheet opened by a labelled 44 by 44px button.

### Application navigation

At 1024px and above, use a persistent 256px sidebar and 64px utility header. Below 1024px, use a 56px sticky top bar and a modal navigation sheet no wider than 320px or 85vw.

Navigation items are 44px high with 12px radius, 12px horizontal padding, 8px gap, a 20px Lucide icon, and Label typography. An active item uses the primary tint, primary-on-subtle text, stronger weight, and a 3px inset indicator so selection is not color-only.

Use breadcrumbs at three or more hierarchy levels. Preserve route, scroll, filters, and input state on back navigation. Separate logout, delete, and account-danger actions spatially. Fixed navigation must reserve content space, and client-side route changes move focus to the page heading or main region.

Bottom navigation is allowed only when a role has five or fewer stable top-level destinations. Do not mix sidebar and bottom navigation at the same hierarchy level.

## Loading states

- Under 300ms: retain the current content and space to prevent indicator flicker.
- From 300ms to roughly 1 second: show a small spinner with plain-language status when structure is already known.
- Longer or structure-fetching operations: use geometry-matched skeletons.
- Skeletons use muted surfaces and a subtle opacity pulse; reduced motion gets a static skeleton. Do not use sweeping shimmer.
- Preserve final dimensions to prevent layout shift. Mark the affected region `aria-busy="true"` and use one nearby polite status message.
- Tables keep the header and show 5–8 row skeletons. Charts show title, legend, and plot placeholders rather than empty axes.
- Background polling must not repeatedly announce itself.

## Empty states

Distinguish first-use empty, filtered no-results, zero-data, permission denied, and unavailable states.

A standard empty panel has a maximum content width of 440px, a 32px Lucide icon, Heading 4 title, concise Body or Body small explanation, and at most one primary plus one tertiary action. Use 32px padding compact and 48px desktop.

State what is absent, why when known, and the next useful action. Filtered empty states offer **Clear filters**. Permission states explain the required role or contact path. Avoid giant illustrations, jokes, blame, or dead-end “No data” messages.

## Error states

Use the smallest level that fully explains and recovers from the failure:

1. Field error: danger boundary and a message below the field.
2. Section error: danger tint/border panel, 20px `CircleAlert`, cause, and Retry or recovery action.
3. Page/system error: retain navigation, explain the cause, include a safe reference ID when available, and provide Retry and safe-back actions.
4. Destructive confirmation: name the object and consequence; do not autofocus the danger action.

Errors state what happened and how to recover, preserve user input and filters, and never rely on red alone. Blocking errors may use `role="alert"`; routine notices use polite live regions. Timeouts and offline states include Retry. Persistent errors do not auto-dismiss.

Nonblocking success toasts use `aria-live="polite"`, do not steal focus, and normally dismiss after 3–5 seconds. A toast is not the only confirmation for a payout or other high-risk operation.

## Accessibility requirements

KlipForge targets WCAG 2.2 AA.

- Normal text contrast is at least 4.5:1; large text and meaningful UI graphics are at least 3:1.
- All functionality works by keyboard with logical DOM and focus order.
- Every interactive element has the documented `focus-visible` treatment; never remove focus indicators.
- Add a visible-on-focus skip link, semantic landmarks, one clear `h1`, and sequential headings.
- Touch and click targets are at least 44 by 44px, with 8px separation where practical.
- Do not rely on hover or color alone.
- Every input has a persistent label; helpers and errors are programmatically associated.
- Dialogs contain focus, close with Escape, offer a visible cancel/close route, and restore focus to their trigger.
- Decorative icons are hidden from assistive technology; icon-only actions have precise accessible names.
- Meaningful images use descriptive alternative text. Video uses captions or transcripts when applicable.
- Do not disable browser zoom. Content and actions must survive enlarged text and 400% reflow.
- Dynamic updates use intentional live regions without announcing routine polling.
- Tables expose headers and sort state. Charts include a text summary and tabular/data alternative.
- Test keyboard-only behavior, reduced motion, zoom/reflow, and at least one screen reader before release.

## Tailwind and CSS contract

When application implementation is authorized:

- Define all canonical values once in `apps/web/src/app/globals.css`.
- Expose semantic colors and fonts through Tailwind CSS v4 `@theme inline` aliases.
- Build shared variants for buttons, fields, cards, badges, tables, feedback, and chart wrappers.
- Prohibit raw hex values, arbitrary shadows, arbitrary radii, and one-off transitions in page components.
- Use `next/font` for Inter and preserve stable fallback metrics.
- Treat this document as the review baseline; do not infer new tokens from a single page.

## Governance

- Update this document before or in the same change as any token or component-contract change.
- Reuse an existing primitive or variant before adding a new one.
- Record approved deviations with scope, rationale, owner, and expiry date.
- Role-specific layouts may change information priority and navigation labels, but not the visual theme.
- Campaign, moderation, analytics, and payout workflows share status language and controls.
- Accessibility requirements are not overridable.

### Approved deviations

None.
