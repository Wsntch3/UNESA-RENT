---
version: "1.0"
name: cycle-inspired-playful-geometric
status: reference-derived
reference: "8 user-supplied screenshots of the Cycle marketing website, 2026-11-09"
description: >-
  A playful, geometric SaaS marketing design language built on a warm ivory
  canvas, punchy cobalt blue, saturated candy-color accents, confident tightly
  set sans-serif headlines, thin ink outlines, compact white modular cards,
  pill-shaped calls to action, and whimsical 2D/3D illustrations. Its defining
  rhythm alternates generous editorial whitespace with dense interactive
  showcases, asymmetrical feature grids, outlined accordion panels, and a
  full-bleed cobalt conversion band.

# Color values marked "sampled" below are exact recurring screenshot pixels.
# Other values, component dimensions, and behavior are design recommendations
# inferred from static desktop screenshots, not verified production CSS.
colors:
  primary: "#2A44E8"                # sampled: cobalt blue / principal brand action
  primary-deep: "#2945DD"           # sampled: slightly deeper blue inside illustrations
  primary-ink: "#2037BE"            # proposed: high-emphasis blue on pale surfaces
  primary-soft: "#E9EDFF"           # proposed: restrained selection tint
  ink: "#1A1A1A"                    # sampled: dominant dark type
  ink-soft: "#2E2E2E"               # sampled: dark outlines / secondary title color
  body: "#5F5F5F"                   # approximate: descriptive text
  body-muted: "#7D7D7D"             # approximate: captions / tertiary text
  on-primary: "#FFFFFF"
  on-dark: "#FFFFFF"
  canvas: "#FFFAF5"                 # sampled: warm ivory, the signature page backdrop
  canvas-white: "#FFFFFF"           # sampled: feature cards and white sections
  surface-subtle: "#F5F3F3"         # sampled: quiet panel surfaces
  surface-tinted: "#FFFBF7"         # sampled: near-ivory transitional shade
  surface-disabled: "#F0F0F0"       # proposed
  border-ink: "#2E2E2E"             # sampled: near-black thin outlines
  border-muted: "#D8D8D8"           # proposed: recessed / secondary dividers
  yellow: "#FED230"                 # sampled: headline highlight, pencil, circles
  yellow-panel: "#FFD43A"           # sampled: feature-showcase illustration panel
  mint: "#28E2A1"                   # sampled: geometric decorative shapes
  green-panel: "#2CC878"            # sampled: feedback lifecycle card panel
  green-deep: "#028053"             # sampled: hero illustration green
  cyan: "#22D9F2"                   # sampled: hero geometric panel
  lavender: "#F3BAFD"               # sampled: circle and tile accents
  orange: "#FF8A00"                 # proposed: occasional illustrative accent
  white-alpha: "rgba(255, 255, 255, 0.9)"
  focus-ring: "#2A44E8"

typography:
  font-family:
    primary: "Inter, Arial, Helvetica, system-ui, sans-serif"
    note: "Exact original face is not verifiable from screenshots; Inter is a recommended substitute."
  hero-display:
    fontFamily: "{typography.font-family.primary}"
    fontSize: "44px"
    fontWeight: 700
    lineHeight: 1.02
    letterSpacing: "-1.8px"
  section-display:
    fontFamily: "{typography.font-family.primary}"
    fontSize: "40px"
    fontWeight: 700
    lineHeight: 1.07
    letterSpacing: "-1.4px"
  section-heading:
    fontFamily: "{typography.font-family.primary}"
    fontSize: "34px"
    fontWeight: 700
    lineHeight: 1.1
    letterSpacing: "-1.1px"
  card-heading-xl:
    fontFamily: "{typography.font-family.primary}"
    fontSize: "24px"
    fontWeight: 700
    lineHeight: 1.18
    letterSpacing: "-0.6px"
  card-heading:
    fontFamily: "{typography.font-family.primary}"
    fontSize: "18px"
    fontWeight: 700
    lineHeight: 1.25
    letterSpacing: "-0.35px"
  feature-title:
    fontFamily: "{typography.font-family.primary}"
    fontSize: "15px"
    fontWeight: 600
    lineHeight: 1.3
    letterSpacing: "-0.15px"
  body-large:
    fontFamily: "{typography.font-family.primary}"
    fontSize: "16px"
    fontWeight: 400
    lineHeight: 1.45
    letterSpacing: "0px"
  body:
    fontFamily: "{typography.font-family.primary}"
    fontSize: "14px"
    fontWeight: 400
    lineHeight: 1.45
    letterSpacing: "0px"
  caption:
    fontFamily: "{typography.font-family.primary}"
    fontSize: "12px"
    fontWeight: 400
    lineHeight: 1.4
    letterSpacing: "0px"
  micro:
    fontFamily: "{typography.font-family.primary}"
    fontSize: "10px"
    fontWeight: 400
    lineHeight: 1.4
    letterSpacing: "0px"
  button:
    fontFamily: "{typography.font-family.primary}"
    fontSize: "14px"
    fontWeight: 600
    lineHeight: 1.15
    letterSpacing: "-0.1px"
  navigation:
    fontFamily: "{typography.font-family.primary}"
    fontSize: "12px"
    fontWeight: 600
    lineHeight: 1.3
    letterSpacing: "-0.1px"

spacing:
  none: "0px"
  xxs: "4px"
  xs: "8px"
  sm: "12px"
  md: "16px"
  lg: "24px"
  xl: "32px"
  xxl: "48px"
  3xl: "64px"
  section: "96px"
  section-large: "120px"

rounded:
  none: "0px"
  xs: "4px"                        # hard-edged feature card / image frame
  sm: "8px"                        # screenshots, smaller cards
  md: "12px"                       # accordion container
  lg: "20px"                       # rare floating illustrated object
  pill: "9999px"                   # CTA buttons / segmented controls
  circle: "50%"                    # balls, icon controls, decorative orbs

borders:
  hairline: "1px solid {colors.border-ink}"
  subtle: "1px solid {colors.border-muted}"
  selected: "2px solid {colors.primary}"

layout:
  max-page-width: "1152px"
  max-content-width: "960px"
  max-feature-width: "780px"
  max-copy-width: "560px"
  max-narrow-copy-width: "420px"
  desktop-gutter: "32px"
  tablet-gutter: "24px"
  mobile-gutter: "20px"
  card-gap: "12px"
  section-gap: "96px"

motion:
  duration-fast: "150ms"
  duration-base: "250ms"
  duration-slow: "400ms"
  easing: "cubic-bezier(0.2, 0.8, 0.2, 1)"
  press-scale: 0.97

components:
  site-nav:
    backgroundColor: "{colors.canvas}"
    textColor: "{colors.ink}"
    typography: "{typography.navigation}"
    minHeight: "64px"
    padding: "12px 0"
  button-primary:
    backgroundColor: "{colors.primary}"
    textColor: "{colors.on-primary}"
    typography: "{typography.button}"
    rounded: "{rounded.pill}"
    padding: "12px 22px"
    minHeight: "44px"
  button-outline:
    backgroundColor: "transparent"
    textColor: "{colors.primary}"
    border: "1px solid {colors.primary}"
    typography: "{typography.button}"
    rounded: "{rounded.pill}"
    padding: "11px 20px"
    minHeight: "44px"
  button-on-blue:
    backgroundColor: "{colors.canvas-white}"
    textColor: "{colors.primary}"
    typography: "{typography.button}"
    rounded: "{rounded.pill}"
    padding: "10px 22px"
    minHeight: "40px"
  segmented-control:
    backgroundColor: "{colors.canvas-white}"
    textColor: "{colors.ink}"
    activeBackground: "{colors.primary}"
    activeText: "{colors.on-primary}"
    rounded: "{rounded.pill}"
    padding: "3px"
  feature-outline-card:
    backgroundColor: "{colors.canvas-white}"
    border: "{borders.hairline}"
    rounded: "{rounded.xs}"
    padding: "16px"
  feature-illustration-tile:
    backgroundColor: "{colors.primary}"
    rounded: "{rounded.xs}"
    padding: "24px"
  feedback-accordion:
    backgroundColor: "{colors.canvas-white}"
    border: "{borders.hairline}"
    rounded: "{rounded.md}"
    padding: "20px"
  feedback-accordion-heading:
    typography: "{typography.card-heading-xl}"
    textColor: "{colors.ink}"
  integration-icon-tile:
    backgroundColor: "{colors.canvas-white}"
    border: "{borders.subtle}"
    rounded: "{rounded.md}"
    size: "40px"
  footer-conversion-band:
    backgroundColor: "{colors.primary}"
    textColor: "{colors.on-primary}"
    padding: "96px 24px 110px"
  footer-bottom-bar:
    backgroundColor: "{colors.canvas-white}"
    textColor: "{colors.ink}"
    rounded: "{rounded.md}"
    padding: "24px 32px"
---

# DESIGN.md — Cycle-Inspired Playful Geometric SaaS

## Overview

The reference is an **editorial SaaS landing page with a playful, almost toy-like visual personality**. It borrows the visual discipline of a minimalist product site — large negative space, clear sans-serif typography, and restrained navigation — then breaks that restraint with bold cobalt blue, expressive yellow underlines, saturated graphic tiles, small 3D characters, and geometric shapes.

The page does **not** treat every surface as a generic rounded dashboard card. Large portions of the layout are open warm-ivory space. Product demonstrations are concentrated into compact, thinly outlined modules. The brand's sense of fun comes from **illustration, composition, and high-chroma color**, not decorative UI chrome.

> **Evidence boundary:** This document is derived exclusively from eight supplied desktop screenshots. Recurring dominant color pixels were sampled from those images. Font family, spacing, breakpoints, animation, interaction states, and some measurements are practical recommendations because the original site's CSS and behavior were not provided. Recreate the *system*, not the reference company's logos, mascot, text, or proprietary graphics.

### Signature characteristics

- Warm off-white (`{colors.canvas}`) as the default page background; white (`{colors.canvas-white}`) for elevated content *without* heavy shadow.
- Strong cobalt (`{colors.primary}`) as the only dominant CTA/brand color; yellow, mint, cyan, and lavender are supporting *illustration* colors.
- Large, centered headlines with tight line height and negative tracking; short supporting paragraphs in muted gray.
- Light, technical **1px near-black borders** around cards and accordion panels.
- Two geometric vocabularies: **strict rectilinear content panels** and **fully rounded pill CTAs / circular ornaments**.
- A compact **asymmetrical bento grid** for feature explanations, not an endlessly repeated three-card grid.
- High-saturation flat panels paired with dimensional 3D stickers, googly eyes, abstract shapes, and thin connector lines.
- White-space-driven sections between dense UI showcases; a full-bleed cobalt CTA band at the bottom.

## Color System

### Sampled brand palette

| Token | Hex | Main use | Confidence |
|---|---|---|---|
| `{colors.primary}` | `#2A44E8` | Brand cobalt, hero buttons, footer band, accents | Sampled |
| `{colors.primary-deep}` | `#2945DD` | Alternate cobalt within feature illustration panels | Sampled |
| `{colors.canvas}` | `#FFFAF5` | Warm ivory page canvas | Sampled |
| `{colors.canvas-white}` | `#FFFFFF` | Cards, accordion, white-background sections | Sampled |
| `{colors.ink}` | `#1A1A1A` | Headlines and high-priority text | Sampled |
| `{colors.ink-soft}` | `#2E2E2E` | Fine outlines and supporting dark UI | Sampled |
| `{colors.yellow}` | `#FED230` | Highlight behind/under important words, geometric objects | Sampled |
| `{colors.yellow-panel}` | `#FFD43A` | Illustration blocks and inset elements | Sampled |
| `{colors.mint}` | `#28E2A1` | Decorative green geometric accent | Sampled |
| `{colors.green-panel}` | `#2CC878` | Feature illustration background | Sampled |
| `{colors.green-deep}` | `#028053` | Deep green hero illustration shape | Sampled |
| `{colors.cyan}` | `#22D9F2` | Decorative geometric vertical block | Sampled |
| `{colors.lavender}` | `#F3BAFD` | Oversized circle, tile, and sticker accents | Sampled |
| `{colors.surface-subtle}` | `#F5F3F3` | Soft in-card content area | Sampled |

Supporting values such as `{colors.body}`, `{colors.border-muted}`, `{colors.primary-soft}`, `{colors.orange}`, and `{colors.focus-ring}` are **implementation tokens**, not verified original CSS colors.

### Color allocation

- **65–80% neutral field:** warm ivory and white do most of the work. The reference is intentionally light-dominant.
- **10–25% saturated hero/panel color:** cobalt blue appears as a strong block, usually in a single section or illustration at a time.
- **Small playful accents:** yellow, mint, lavender, and cyan are localized to illustrations and underlines. They should not compete equally with the CTA blue.
- **Text remains neutral:** primary dark text on ivory/white, white on cobalt, secondary copy in gray. Do not use rainbow-colored paragraph text.

### Application rules

1. Use `{colors.primary}` for the main interactive action and selected states.
2. Use `{colors.yellow}` as a **visual highlighter**, especially a rectangular stripe behind the bottom of a word, rather than as a long block of low-contrast yellow text.
3. Use `{colors.mint}`, `{colors.cyan}`, and `{colors.lavender}` mainly for scene composition, illustrations, and colorful demo panels.
4. Keep large white cards on the ivory canvas to create contrast without relying on box shadows.
5. Never use yellow-on-white or mint-on-white for paragraph text; maintain legible contrast.
6. Avoid generic gradients on the main page. Gradual shading can live *inside 3D illustrations or screenshots* where it creates material depth.

## Typography

### Font family

**Recommended:** `Inter, Arial, Helvetica, system-ui, sans-serif`.

The screenshots show a modern geometric/grotesk sans with a friendly, compact headline rhythm. **The original font cannot be identified reliably from static screenshots**, so the design uses Inter as a practical, easily available approximation. Replace it only if the actual brand font is licensed and available.

### Type hierarchy

| Token | Desktop size | Weight | Line height | Tracking | Typical role |
|---|---:|---:|---:|---:|---|
| `{typography.hero-display}` | 44px | 700 | 1.02 | -1.8px | Centered hero message |
| `{typography.section-display}` | 40px | 700 | 1.07 | -1.4px | Product lifecycle, integrations |
| `{typography.section-heading}` | 34px | 700 | 1.10 | -1.1px | Two-column feature headline |
| `{typography.card-heading-xl}` | 24px | 700 | 1.18 | -0.6px | Accordion panel heading |
| `{typography.card-heading}` | 18px | 700 | 1.25 | -0.35px | Bento grid section title |
| `{typography.feature-title}` | 15px | 600 | 1.30 | -0.15px | Caption beneath preview tiles |
| `{typography.body-large}` | 16px | 400 | 1.45 | 0 | Hero and introductory subcopy |
| `{typography.body}` | 14px | 400 | 1.45 | 0 | Body paragraphs, feature descriptions |
| `{typography.caption}` | 12px | 400 | 1.40 | 0 | Feature metadata, supplementary descriptions |
| `{typography.micro}` | 10px | 400 | 1.40 | 0 | Footer legal / compact annotations |
| `{typography.button}` | 14px | 600 | 1.15 | -0.1px | Pills and CTAs |

**These sizes are implementation recommendations** calibrated for the screenshot proportions, not pixel-perfect measurements of source CSS.

### Typography principles

- **Bold and compact headlines.** Use 700 weight with slightly tightened tracking; avoid elegant serif or ultra-light display styles.
- **Keep headline line breaks deliberate.** Many reference headlines are two lines with a concise phrase: text should feel poster-like, not like a paragraph.
- **Use color on one meaningful word only.** E.g., a cobalt emphasis in a section heading or a yellow highlight on one word in the hero.
- **Supporting text is quieter.** Gray subcopy, constrained to roughly 380–560px, should not be wider than the heading block.
- **Cards use smaller type.** Preview tiles have dense 12–15px details, offset by illustrations and their larger headings.
- **Avoid all caps** for large headings; sentence case is key to the approachable voice.

### Highlighted-word pattern

A bold heading may contain a single **flat yellow underscore/highlight**, as seen in the hero and closing CTA. Implement with a background stripe, pseudo-element, or `linear-gradient` used *only as a sharp-edged highlight mask* (not as a decorative multi-color gradient):

```css
.highlight-word {
  background: linear-gradient(
    transparent 69%,
    var(--color-yellow) 69%
  );
  box-decoration-break: clone;
  -webkit-box-decoration-break: clone;
}
```

The visible yellow stripe should sit mainly under the bottom portion of the letters and should not reduce text contrast.

## Layout

### Page containers

| Container | Suggested maximum | Use |
|---|---:|---|
| Page shell | 1152px | Navigation, hero composition, wide decorative geometry |
| Content shell | 960px | General section content |
| Feature shell | 780px | Centered accordion and process showcase |
| Copy shell | 560px | Heading + introductory paragraph |
| Narrow copy | 420px | Centered subtitle or two-column explanatory copy |

The screenshots repeatedly concentrate **functional content in the center** and leave broad margins for breathing room and decorative overlap. Do not stretch feature cards to the full width just because the viewport is large.

### Spacing rhythm

- Base spatial rhythm: **8px** with half-steps at 4px and 12px.
- Desktop section separation: typically **96–120px**.
- Hero headline → body: approximately **16–24px**.
- Body → CTA cluster: approximately **24–32px**.
- Dense card spacing: typically **8–12px** between boxes.
- Inner card padding: **16–24px**, depending on information density.
- Left/right desktop margins: at least **32px**; mobile margins: **20px**.
- Avoid placing large blocks of colored UI directly adjacent unless they are a purposeful unified diagram.

### Composition patterns

**A. Centered editorial introduction**

- Eyebrow or wordmark, optional.
- One large centered heading, typically 2–3 short lines.
- Muted descriptive text below, narrow measure.
- One or two pill CTAs.
- Generous space before the next visualization.

**B. Asymmetric split feature**

- Copy block on the left and a product UI screenshot or illustration on the right.
- Screenshot sits in a thin rectangular frame, with minimal decoration.
- Heading may color one entire word cobalt.
- The text and image remain vertically aligned to a shared center line.

**C. Bento grid of delightful details**

- Thin outline around each grid item.
- Different column spans and row heights: wide top row, split middle row, mixed columns toward the bottom.
- Blend plain copy with expressive stickers, tiny interface diagrams, pin notes, keyboard keys, and theme swatches.
- Use near-zero shadow, small corner radius, and a 8–12px gutter.

**D. Single large process accordion**

- A section heading centered above.
- One expanded, illustrated panel plus compressed labeled rows beneath it.
- The panel acts as a narrative tour of the workflow, not a generic FAQ list.

**E. Full-bleed conversion section**

- Cobalt panel takes full viewport width.
- White centered headline and a white pill with blue label.
- Peripheral abstract geometric ornaments interrupt the section edges without obscuring content.
- A distinct white footer strip visually caps the bottom.

## Shape, Borders & Depth

### Radius scale

| Token | Value | Use |
|---|---:|---|
| `{rounded.none}` | 0px | Geometric blocks, full-width color bands |
| `{rounded.xs}` | 4px | Compact feature tiles, preview frames |
| `{rounded.sm}` | 8px | Larger in-card graphic rectangles |
| `{rounded.md}` | 12px | Accordion wrapper, white footer strip |
| `{rounded.lg}` | 20px | Occasional dimensional graphic, *not* all cards |
| `{rounded.pill}` | 9999px | Primary CTAs, outline CTAs, selection toggles |
| `{rounded.circle}` | 50% | Social icons, decorative circles, floating controls |

### Borders

- Default interactive demo / feature card: **1px solid `{colors.border-ink}`**.
- Border is visible but fine, producing a drawn-ink technical quality.
- Use near-black border rather than pale-gray SaaS-card borders on key showcase cards.
- Adjacent accordion rows should feel like a **single outlined stack**: share boundaries, avoid doubled 2px seams, and round only the true outer corners.
- Use rounded white circular icon tiles for integrations, with finer secondary border where necessary.

### Shadow philosophy

- Page layout: **flat**. Do not blanket every card with `box-shadow`.
- Visual depth belongs primarily to **3D stickers, rendered hands, pinheads, playful buttons, and object illustrations**.
- A very restrained localized object shadow is acceptable for tactile elements, but not as the default card treatment.
- Thin linework + color contrast should provide component separation.

## Illustration & Graphic Language

The visual identity mixes **minimal 2D geometry** and **soft rendered 3D objects**. The tension between these two styles is intentional.

### 2D toolkit

- Solid circles, half-circles, triangles, crosses, diamonds, rectangles, and overlaid outlined loops.
- Thin black or gray strokes, typically 1px at desktop screenshot scale.
- Thin wires and curved connectors linking source icons to a central blue graphic.
- Bright primary-color panels; little to no texture.
- Objects may intentionally crop off the edges of hero and footer sections.

### 3D toolkit

- Small floating eyeballs, character heads, hands, plugs, pencil ends, pushpins, keyboard keys.
- Rounded, slightly glossy material; soft object-only shadows.
- Whimsical proportions and playful orientation, but no unrelated stock illustrations.
- Treat 3D objects as **accent actors**, not the page's main background texture.

### Graphic composition rules

1. Each section gets at most **one dominant playful visual idea**.
2. Keep 3D illustrations around the edges of copy, rather than running behind paragraph text.
3. Favor a single illustrative narrative — e.g., sources converge into a hub — over an arbitrary constellation of icons.
4. Prefer transparent PNG/WebP/3D renders or clean vector SVG for decorations.
5. Product screenshots should stay legible and rectangular; decorative objects should not destroy screenshot clarity.
6. Icons from external tools or vendors are **content assets**, not replacements for the design system's core icon library.
7. Do not copy the original brand logo or mascot when implementing a distinct product.

## Components

### 1. Site navigation — `site-nav`

- Minimal top navigation on `{colors.canvas}`.
- Left: wordmark or small brand logo.
- Right: compact link(s) followed by `{components.button-primary}`.
- Content constrained to the page shell; wide blank margins are acceptable.
- Do not add a heavy nav border, a floating glass panel, or oversized active tabs.
- Mobile: collapse secondary links into a menu while preserving the main CTA when space allows.

### 2. Primary CTA — `button-primary`

- Cobalt background, white label, full pill radius.
- Suggested min-height **44px**; horizontal padding **22px**.
- Semibold compact text, centered.
- Default placement: main hero, primary navigation CTA, action areas.
- Press state: `transform: scale(0.97)`.
- Focus state: visible 2px cobalt outline with 3px offset on light backgrounds.
- Hover effect (implementation proposal): slightly darker cobalt, with a restrained 150ms transition.

### 3. Secondary CTA — `button-outline`

- Transparent background, cobalt border and text, full pill radius.
- Often includes a small leading circular play icon for video/demo actions.
- Visually subordinate to `{components.button-primary}` but still easy to identify as clickable.
- Maintain a 44px minimum interactive height.

### 4. White CTA on cobalt — `button-on-blue`

- White pill with cobalt text on a full cobalt section.
- Seen in the closing conversion band; do not invert into a yellow pill by default.
- Use subtle pressed feedback without heavy shadows.

### 5. Feature outline cards — `feature-outline-card`

- White surface, 1px ink border, 4px or 8px radius.
- Mix title, 1–3 lines of explanation, and a decorative visual or screenshot.
- Content blocks may be horizontally divided instead of using identical top-image / bottom-text arrangements.
- Permit some cards to span two columns or two rows.
- Keep the border geometry consistent across the grid.

**Reference-inspired variants:** `wide-upload-card`, `quick-preview-card`, `pinned-doc-card`, `search-feature-card`, `theme-feature-card`, and `keyboard-shortcut-card`.

### 6. Feature illustration tiles — `feature-illustration-tile`

- Large vivid rectangle within a white information panel.
- Cobalt, green, and yellow panels can each describe a step of a workflow.
- Rounded corners remain restrained (about 4px).
- Tiny external app icons, lines, lightweight inset UI, and toy-like 3D cues establish hierarchy.
- Below each tile: a semibold 15px title and a brief 12px description.

### 7. Feedback lifecycle accordion — `feedback-accordion`

- White background, near-black outline, 12px outer radius.
- Expanded section contains a 24px heading, description, and a horizontally scrollable or clipped showcase of three preview panels.
- Collapsed rows contain large bold labels, such as workflow stages, with a small right-aligned decorative symbol/status icon.
- Maintain shared borders and a clean vertical stack.
- For implementation, use **actual accordion semantics** (`button`, `aria-expanded`, `aria-controls`, appropriate associated panel IDs).
- Interaction recommendation: only one expanded section at a time; animate height/opacities gently, and respect `prefers-reduced-motion`.
- This exact behavior was **not confirmed** in screenshots; it is a suitable accessible implementation.

### 8. Segmented before/after toggle — `segmented-control`

- Small horizontal pill container with a thin border.
- Inactive option: white surface, dark text.
- Active option: solid cobalt pill, white text.
- Used to compare *without* and *with* product states in the central diagram.
- Selected state must have both color and accessible state semantics (`aria-pressed` / radio group), not color alone.

### 9. Integration icon matrix — `integration-icon-tile`

- Compact rows and columns of app icon tiles.
- Approximate size: 40–44px with a light border and 12px rounding.
- Balanced 4-column groups for distinct categories, e.g., incoming sources and collaboration tools.
- Each group has one bold label under the grid.
- Keep logo spacing consistent and use monochrome/colored marks according to actual asset requirements.
- If there are many integrations, do not scale icons down indefinitely; wrap or scroll at narrow widths.

### 10. Product screenshot frame — `product-screenshot-frame`

- Rectangular screenshot on ivory or white, narrow dark border, small radius (4px–8px).
- No oversized drop shadow or colorful fake browser chrome.
- Can float within a split layout with plenty of whitespace.
- The screenshot should remain the focal point while the heading stays outside it.

### 11. Closing CTA — `footer-conversion-band`

- Full-bleed background `{colors.primary}`.
- Large white centered heading, with one yellow-highlighted word.
- White pill `{components.button-on-blue}` directly underneath.
- Decorative lines/circles/diamonds/cyan or lavender tiles crop from the sides; they are non-interactive.
- Preserve a text-safe central column (roughly 480–620px).

### 12. Footer utility strip — `footer-bottom-bar`

- White panel with gently rounded top corners, sitting flush with the lower cobalt section.
- Small copyright on left; a short cluster of thin outlined circular social links on right.
- Keep labels small, but links must still have accessible names and comfortable touch targets.

### Inputs and forms (proposed expansion)

Input controls were not shown clearly in the screenshots. For a consistent extension:

- Field surface: white, 1px near-black outline, 8px radius, 44px minimum height.
- Field text: `{typography.body}` and `{colors.ink}`.
- Label: 12–14px semibold, placed above the input.
- Focus: 2px cobalt outline with offset; error: a clearly labeled message plus a distinct error indicator.
- Do not rely on pastel color alone to communicate success/error.
- Keep form controls calmer than hero illustrations.

## Page Architecture & Section Rhythm

The reference pages can be interpreted as the following reusable *marketing architecture*:

| Order | Section | Layout | Key visual technique |
|---:|---|---|---|
| 01 | Hero | Centered headline + CTAs | Peripheral 2D/3D playful geometry |
| 02 | Product hub diagram | Centered copy + wide visual | Source icons converge into a cobalt central mark |
| 03 | Integration ecosystem | Centered heading + grouped icon grids | Dense tiles inside expansive ivory whitespace |
| 04 | Lifecycle / methodology | Centered heading + large accordion | One expanded workflow panel with colored preview tiles |
| 05 | Customer-focused feature | Left text / right screenshot | Quiet editorial whitespace + one blue-highlighted word |
| 06 | Delightful details | Asymmetric card grid | Monochrome outlines plus playful inset illustrations |
| 07 | Final CTA | Full-width cobalt band | White headline, yellow underline, white pill, edge ornaments |
| 08 | Footer | White utility strip | Small legal text + circular social controls |

This order is a recommended reading sequence of the supplied crops, not proof of the exact original DOM or every intervening section.

## Motion & Interaction

Static imagery cannot confirm animation. The following motion language is intentionally *subtle* so it supports rather than overwhelms the playful graphics.

| Interaction | Recommended behavior | Duration |
|---|---|---|
| Primary CTA press | Scale to 0.97 | 150ms |
| Button hover | Small fill/tint change | 150ms |
| Accordion expand | Height + opacity progression | 250–400ms |
| Carousel next/prev | Horizontal slide, no exaggerated bounce | 250–400ms |
| Feature tile focus | Visible cobalt focus ring | Immediate |
| Decorative illustration | Optional slow float of 2–5px | 3–5s, if implemented |

Do not animate all ornaments simultaneously. A page with many concurrent looped movements quickly loses the reference's calm editorial quality. Under `prefers-reduced-motion: reduce`, remove nonessential floating motion and use near-instant transitions.

## Responsive Behavior

### Breakpoints (implementation proposals)

| Breakpoint | Width | Layout change |
|---|---|---|
| Wide desktop | ≥ 1280px | Full page shell; split sections and multi-column illustrations |
| Desktop | 1024–1279px | Slightly reduced margins; split layouts preserved |
| Tablet | 768–1023px | Reduce display size; feature grid 2 columns; some split sections stack |
| Large phone | 480–767px | Single-column editorial stack; cards span full width; accordion showcase scrolls |
| Small phone | ≤ 479px | Narrow gutters; 28–34px headings; one CTA per row if necessary |

**The supplied images are desktop only**; these breakpoints are engineering defaults, not source-observed behavior.

### Responsive composition rules

- **Hero:** keep center alignment, but move or hide peripheral decorations when they threaten the text safe area.
- **Product hub:** on mobile, simplify connector lines and reorganize source/destination icons into a compact flow rather than squeezing a desktop diagram.
- **Integration matrix:** keep icons recognizable at 40px or larger; wrap into smaller grids.
- **Asymmetric bento:** 2-column on tablet, 1-column on mobile; avoid awkward fixed-height empty cards.
- **Accordion preview:** on mobile, use snap-based horizontal scrolling inside the expanded panel, with visible affordance that more cards exist.
- **Split screenshot section:** switch to text-above-image, while preserving screenshot aspect ratio and readable detail.
- **Footer CTA:** scale ornaments down and crop beyond edges; do not overlap the headline, button, or footer links.
- **Touch targets:** aim for at least 44 × 44px for all actions; maintain keyboard navigation and focus visibility.

## Accessibility & Content Rules

- Treat graphic confetti, background lines, and purely decorative geometry as `aria-hidden="true"`.
- Use meaningful alt text for informative product screenshots; empty alt text for purely decorative 3D stickers.
- Keep dark body text on ivory/white and white text on cobalt; check all small-text contrast against WCAG AA.
- A yellow-highlighted word should remain dark text; do not invert to white text on yellow.
- Use real buttons and links, not clickable `<div>` containers.
- Every carousel arrow and accordion control requires an accessible name.
- Keep paragraph copy short: generally 2–4 lines per explanatory component at desktop width.
- Use playful microcopy sparingly, grounded in the actual product's workflow. Visual fun should not obscure practical meaning.
- Honor reduced motion and avoid autoplay that changes crucial information before it can be read.

## Do's and Don'ts

### Do

1. **Do** use `{colors.canvas}` as the broad background, then layer purposeful `{colors.canvas-white}` modules over it.
2. **Do** make cobalt the recurring brand/CTA signal.
3. **Do** keep heading typography bold, compact, and editorial with tight line breaks.
4. **Do** surround feature modules with thin dark outlines and small radii.
5. **Do** mix large quiet areas with focused, colorful demonstrations.
6. **Do** build bento arrangements with uneven spans and varied illustration/copy balance.
7. **Do** use yellow accent as a highlight under one phrase instead of an arbitrary badge.
8. **Do** pair friendly dimensional objects with clean geometric linework.
9. **Do** give interactive UI clear states, semantic markup, and mobile behavior.
10. **Do** let the final full-blue section stand out as a strong conversion moment.

### Don't

1. **Don't** turn the whole layout into large rounded cards with soft gray shadows.
2. **Don't** cover the canvas with gradients, blurred blobs, neon glow, or glassmorphism.
3. **Don't** make every UI element a different accent color — accents mostly serve illustrations.
4. **Don't** use bold yellow or mint for small body text on light backgrounds.
5. **Don't** use generic three-across SaaS feature cards as the only layout pattern.
6. **Don't** force complex desktop diagrams to shrink uniformly on mobile.
7. **Don't** copy the brand's logo, characters, wording, or product screenshots into a different brand.
8. **Don't** let decorative objects block buttons, text, or keyboard focus.
9. **Don't** add constant large-scale animation to an already colorful composition.
10. **Don't** conflate observed screenshot choices with verified production behavior.

## Implementation Starter: CSS Custom Properties

The YAML frontmatter is the canonical design-token layer. A practical CSS mapping:

```css
:root {
  --color-primary: #2a44e8;
  --color-primary-deep: #2945dd;
  --color-canvas: #fffaf5;
  --color-white: #ffffff;
  --color-ink: #1a1a1a;
  --color-body: #5f5f5f;
  --color-border: #2e2e2e;
  --color-yellow: #fed230;
  --color-mint: #28e2a1;
  --color-green-panel: #2cc878;
  --color-cyan: #22d9f2;
  --color-lavender: #f3bafd;

  --font-sans: Inter, Arial, Helvetica, system-ui, sans-serif;
  --container-page: 1152px;
  --container-feature: 780px;
  --space-section: 96px;
  --radius-card: 4px;
  --radius-panel: 12px;
  --radius-pill: 9999px;
}

body {
  background: var(--color-canvas);
  color: var(--color-ink);
  font-family: var(--font-sans);
  -webkit-font-smoothing: antialiased;
}

.section-shell {
  width: min(calc(100% - 40px), var(--container-page));
  margin-inline: auto;
  padding-block: var(--space-section);
}

.ui-outline-card {
  background: var(--color-white);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-card);
}

.btn-primary {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 44px;
  padding: 12px 22px;
  border: 1px solid transparent;
  border-radius: var(--radius-pill);
  background: var(--color-primary);
  color: var(--color-white);
  font: 600 14px/1.15 var(--font-sans);
  text-decoration: none;
  transition: transform 150ms ease, background-color 150ms ease;
}

.btn-primary:active { transform: scale(0.97); }
.btn-primary:focus-visible {
  outline: 2px solid var(--color-primary);
  outline-offset: 3px;
}

@media (max-width: 767px) {
  :root { --space-section: 64px; }
  .section-shell { width: min(calc(100% - 40px), var(--container-page)); }
}

@media (prefers-reduced-motion: reduce) {
  *, *::before, *::after {
    scroll-behavior: auto !important;
    animation-duration: 0.01ms !important;
    transition-duration: 0.01ms !important;
  }
}
```

## Iteration Guide

1. Start new components by selecting a **known role** (editorial section, feature card, diagram, accordion, CTA band), not by inventing a new style for each block.
2. Reuse `{colors.*}`, `{typography.*}`, `{spacing.*}`, and `{rounded.*}` as the source of truth. In implementation, map these to CSS variables or a framework theme.
3. First establish **layout proportions, typography, and quiet space**; only then introduce illustration.
4. Add decorative motifs only if they reinforce the section's topic or visual rhythm.
5. Prefer component variants with consistent geometry instead of one-off arbitrary radii.
6. Make every new state work with keyboard, touch, focus visibility, and reduced-motion settings.
7. Compare desktop and mobile screenshots before declaring the component complete.
8. For a different product, keep the *visual grammar* but replace all references to Cycle branding and proprietary artwork with original assets.

## Known Gaps & Verification Checklist

The following **cannot be confirmed from the supplied still screenshots**:

- Exact original font family, font weights, responsive font-size values, and tracking.
- Exact padding and grid CSS (the browser zoom, image scaling, and full page structure are unknown).
- Hover, focus, active, loading, error, carousel, accordion, and segmented-control behavior.
- Tablet and mobile layouts, touch interactions, and breakpoint thresholds.
- Asset source formats, animation libraries, page transitions, and whether the 3D elements animate.
- Complete design of forms, validation states, account/dashboard UI, and dark mode.
- Every footer/legal link and section not visible in the screenshots.

**Recommended implementation validation:** capture the rendered site at widths 1440px, 1024px, 768px, 390px, and 320px; verify text wrapping, card seams, touch targets, contrast, overflow, focus states, reduced-motion behavior, and asset loading. Treat the sampled hex values as strong visual anchors and the other measurements as adjustable defaults.
