# VisionOS Glassmorphism Redesign

Date: 2026-05-04
Scope: Design system + core components (Button, Card, Input, Table, Modal, Navbar)
Approach: Glassmorphism full upgrade — multi-layer glass, floating panels, ambient light, bold typography

## Goals

Transform the existing "Electric Dark" design system into a visionOS-inspired spatial interface with:
- Floating glass panels that feel suspended in space
- Multi-layer depth through shadows and ambient light
- Bold typography contrast (light titles, medium body)
- Smooth spring-based micro-interactions

## 1. Color System

### Surface Hierarchy (warm shift)

| Token | Current | New | Purpose |
|-------|---------|-----|---------|
| `surface` | `#0B0E14` | `#0A0C10` | Deepest background |
| `surface-elevated` | `#161B22` | `#13161C` | Primary panel surface |
| `surface-elevated-2` | `#21262D` | `#1C1F26` | Mid-layer panels |
| `surface-elevated-3` | `#30363D` | `#262A33` | Top-layer floating islands |

### Ambient Light Colors (new)

| Token | Value | Use |
|-------|-------|-----|
| `ambient-blue` | `rgba(100, 150, 255, 0.08)` | Panel edge glow |
| `ambient-purple` | `rgba(180, 130, 255, 0.06)` | Background decorative light |
| `ambient-glow` | `rgba(255, 255, 255, 0.02)` | Panel inner top highlight |

## 2. Shadow System

### Island Shadows (3-layer depth)

```css
shadow-island:
  0 0 0 1px rgba(255,255,255,0.06),     /* edge light */
  0 2px 4px rgba(0,0,0,0.3),             /* near shadow */
  0 8px 24px rgba(0,0,0,0.4),            /* mid shadow */
  0 24px 60px rgba(0,0,0,0.5);           /* far shadow */

shadow-island-hover:
  0 0 0 1px rgba(255,255,255,0.12),
  0 4px 8px rgba(0,0,0,0.35),
  0 16px 32px rgba(0,0,0,0.45),
  0 32px 72px rgba(0,0,0,0.55);
```

### Island Inner Glow (visionOS signature)

```css
shadow-island-glow:
  inset 0 1px 0 rgba(255,255,255,0.08),    /* top refraction */
  inset 0 0 20px rgba(255,255,255,0.02);    /* inner ambient */
```

## 3. Glass Effects

### vision-glass (replaces liquid-glass as default)

```css
.vision-glass {
  background: rgba(255, 255, 255, 0.04);
  backdrop-filter: saturate(200%) blur(60px);
  -webkit-backdrop-filter: saturate(200%) blur(60px);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-top-color: rgba(255, 255, 255, 0.15);
  border-radius: 16px;
  box-shadow:
    inset 0 1px 0 rgba(255, 255, 255, 0.08),
    inset 0 0 20px rgba(255, 255, 255, 0.02),
    0 0 0 1px rgba(255, 255, 255, 0.06),
    0 2px 4px rgba(0, 0, 0, 0.3),
    0 8px 24px rgba(0, 0, 0, 0.4),
    0 24px 60px rgba(0, 0, 0, 0.5);
}

.vision-glass:hover {
  background: rgba(255, 255, 255, 0.06);
  border-top-color: rgba(255, 255, 255, 0.2);
  transform: translateY(-1px);
  box-shadow:
    inset 0 1px 0 rgba(255, 255, 255, 0.12),
    inset 0 0 30px rgba(255, 255, 255, 0.03),
    0 0 0 1px rgba(255, 255, 255, 0.12),
    0 4px 8px rgba(0, 0, 0, 0.35),
    0 16px 32px rgba(0, 0, 0, 0.45),
    0 32px 72px rgba(0, 0, 0, 0.55);
}
```

### Glass variants

- `vision-glass-strong` — higher opacity (0.06 bg), stronger blur (80px), for modals/overlays
- `vision-glass-subtle` — lower opacity (0.02 bg), reduced blur (40px), for nested elements

## 4. Typography

### Hierarchy

| Role | Size | Weight | Tracking | Color |
|------|------|--------|----------|-------|
| Page Title | `text-3xl` (30px) | `font-light` (300) | `tracking-tight` | `text-text-primary` |
| Section Title | `text-lg` (18px) | `font-medium` (500) | normal | `text-text-primary` |
| Card Title | `text-base` (16px) | `font-semibold` (600) | normal | `text-text-primary` |
| Label | `text-xs` (12px) | `font-medium` (500) | `tracking-wider` | `text-text-secondary` |
| Body | `text-sm` (14px) | `font-normal` (400) | normal | `text-text-primary` |
| Secondary | `text-sm` (14px) | `font-normal` (400) | normal | `text-text-secondary` |
| Caption | `text-xs` (12px) | `font-normal` (400) | normal | `text-text-tertiary` |
| Mono/Code | `text-xs` (12px) | `font-mono` | normal | `text-text-secondary` |

### Body line-height

Change from `1.47059` to `1.625` (`leading-relaxed`) for more breathing room.

## 5. Spacing Tokens

| Token | Value | Use |
|-------|-------|-----|
| `apple-xs` | `4px` | Tight element spacing |
| `apple-sm` | `8px` | Component internal spacing |
| `apple-md` | `16px` | Card padding |
| `apple-lg` | `24px` | Section spacing |
| `apple-xl` | `32px` | Page area spacing |
| `apple-2xl` | `48px` | Large area separation |

Page content area: `px-8` (up from current `px-5`).

## 6. Component Upgrades

### Button

**Primary:**
- Add `backdrop-filter: blur(10px)` for glass feel
- Hover: `translateY(-1px)` + enhanced glow
- Active: `scale(0.98)` (softer than current 0.97)
- Transition: `all 200ms cubic-bezier(0.32, 0.72, 0, 1)`

**Secondary:**
- Change from solid `#21262D` to glass: `rgba(255,255,255,0.06)` bg + subtle border
- Hover: increase to `rgba(255,255,255,0.1)`

**Ghost:**
- Hover: glass background `rgba(255,255,255,0.06)`

**All:**
- Size `lg`: `py-3` (more touch-friendly)
- Transition duration: `200ms`

### Card

- Default glass: `vision-glass` class
- Border radius: `16px` (up from `12px`)
- Hover: `translateY(-2px)` + enhanced shadow (no blue glow — pure white light enhancement)
- Remove `card-dark` blue glow hover behavior

### Input

- Background: `rgba(0,0,0,0.3)` (translucent, not solid)
- Stronger inset shadow for "recessed" feel
- Focus: ambient light ring `box-shadow: 0 0 0 3px rgba(47,129,247,0.15), inset 0 2px 4px rgba(0,0,0,0.5)`
- Softer placeholder color

### Table

- Header: `rgba(255,255,255,0.03)` background
- Row hover: glass effect with left accent border
- Dividers: gradient lines (fading from edges)
- Selected row: 2px blue left indicator

### Modal

- Backdrop: `backdrop-blur-xl` (stronger)
- Panel: `vision-glass-strong` effect
- Border radius: `20px`
- Appear animation: `scale(0.95) + translateY(8px) → scale(1) + translateY(0)` (visionOS scale-pop)
- Disappear animation: reverse

### Navbar

- Increase blur, reduce background opacity
- Bottom border: gradient fade (not hard line)
- Active indicator: pill background instead of bottom bar

## 7. Animation System

### Timing Curves

| Curve | Value | Use |
|-------|-------|-----|
| `apple-spring` | `cubic-bezier(0.32, 0.72, 0, 1)` | Displacement |
| `apple-bounce` | `cubic-bezier(0.34, 1.56, 0.64, 1)` | Micro-bounce (hover) |
| `apple-ease` | `cubic-bezier(0.25, 0.1, 0.25, 1)` | Color transitions |

### New Animations

| Name | Effect | Use |
|------|--------|-----|
| `scale-in` | `scale(0.95)+opacity(0) → scale(1)+opacity(1)` | Modal/Popover appear |
| `scale-out` | reverse of scale-in | Disappear |
| `slide-in-right` | `translateX(20px)+opacity(0) → 0` | Side panels |
| `glow-pulse` | Ambient light intensity oscillation | Active state indicator |

### Duration Standards

| Type | Duration |
|------|----------|
| Color/background | `150ms` |
| Displacement/scale | `250ms` |
| Layout changes | `350ms` |
| Appear/disappear | `300ms` |

## 8. Files to Modify

| File | Changes |
|------|---------|
| `tailwind.config.js` | Colors, shadows, border-radius, animations, spacing tokens |
| `src/index.css` | Glass classes, button classes, input class, nav class, typography base, new animations |
| `src/components/Button.tsx` | Update variant classes, add glass feel |
| `src/components/Card.tsx` | Switch to vision-glass, update border-radius |
| `src/components/Input.tsx` | Translucent bg, enhanced focus ring |
| `src/components/Table.tsx` | Header bg, row hover, gradient dividers |
| `src/components/Modal.tsx` | Stronger glass, scale animation, larger radius |
| `src/components/Navbar.tsx` | Enhanced glass, pill indicators, gradient border |
| `src/components/Badge.tsx` | Minor polish (consistent with new tokens) |
| `src/components/Select.tsx` | Match input styling |
| `src/components/Skeleton.tsx` | Update shimmer to use new surface colors |
| `src/components/EmptyState.tsx` | Update text colors to new hierarchy |

## 9. Performance Considerations

- `backdrop-filter: blur(60px)` is GPU-accelerated on modern hardware
- Limit glass effects to top-level panels; nested elements use solid backgrounds
- Use `will-change: transform` on elements that animate hover lift
- Test on Tauri's WebView — Chromium-based, should handle backdrop-filter well

## 10. Out of Scope

- Page layout restructure (future work)
- New components
- Dark/light mode toggle (dark only)
- Animation for page transitions (react-router level)
