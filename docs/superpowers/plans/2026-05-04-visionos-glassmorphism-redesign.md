# VisionOS Glassmorphism Redesign — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Transform the "Electric Dark" design system into a visionOS-inspired spatial interface with multi-layer glass, floating panels, ambient light, and spring animations.

**Architecture:** All visual changes flow from two foundation files (`tailwind.config.js` and `index.css`) into the component layer. Components reference CSS classes defined in index.css, which in turn use tokens from tailwind.config.js. This means tasks must execute in order: config first, then CSS, then components.

**Tech Stack:** React 18, Tailwind CSS 3.4, TypeScript, Vite 5, Tauri v2

**Spec:** `docs/superpowers/specs/2026-05-04-visionos-glassmorphism-redesign.md`

---

### Task 1: Update Tailwind Config — Tokens & Animations

**Files:**
- Modify: `frontend/tailwind.config.js`

This task updates the design tokens in the Tailwind config. All downstream changes depend on these tokens being correct.

- [ ] **Step 1: Update surface colors**

Replace the `surface` color block in `theme.extend.colors`:

```js
surface: {
  DEFAULT: "#0A0C10",
  elevated: "#13161C",
  "elevated-2": "#1C1F26",
  "elevated-3": "#262A33",
},
```

- [ ] **Step 2: Add ambient light colors**

Add a new `ambient` color block after `glass`:

```js
ambient: {
  blue: "rgba(100, 150, 255, 0.08)",
  purple: "rgba(180, 130, 255, 0.06)",
  glow: "rgba(255, 255, 255, 0.02)",
},
```

- [ ] **Step 3: Update box shadows**

Replace the `boxShadow` block entirely:

```js
boxShadow: {
  "apple-sm": "0 1px 2px rgba(0,0,0,0.5), 0 0 1px rgba(255,255,255,0.1)",
  apple: "0 4px 12px rgba(0,0,0,0.6), 0 0 1px rgba(255,255,255,0.15)",
  "apple-md": "0 8px 24px rgba(0,0,0,0.7), 0 0 1px rgba(255,255,255,0.15)",
  "apple-lg": "0 16px 48px rgba(0,0,0,0.8), 0 0 1px rgba(255,255,255,0.2)",
  island:
    "0 0 0 1px rgba(255,255,255,0.06), 0 2px 4px rgba(0,0,0,0.3), 0 8px 24px rgba(0,0,0,0.4), 0 24px 60px rgba(0,0,0,0.5)",
  "island-hover":
    "0 0 0 1px rgba(255,255,255,0.12), 0 4px 8px rgba(0,0,0,0.35), 0 16px 32px rgba(0,0,0,0.45), 0 32px 72px rgba(0,0,0,0.55)",
  "island-glow":
    "inset 0 1px 0 rgba(255,255,255,0.08), inset 0 0 20px rgba(255,255,255,0.02)",
  "glow-blue": "0 0 20px rgba(47,129,247,0.3), 0 0 40px rgba(47,129,247,0.1)",
  "glow-green": "0 0 20px rgba(63,185,80,0.3), 0 0 40px rgba(63,185,80,0.1)",
  "glow-red": "0 0 20px rgba(248,81,73,0.3), 0 0 40px rgba(248,81,73,0.1)",
  "glow-purple": "0 0 20px rgba(163,113,247,0.3), 0 0 40px rgba(163,113,247,0.1)",
},
```

- [ ] **Step 4: Add spacing tokens**

Add a `spacing` section to `theme.extend`:

```js
spacing: {
  "apple-xs": "4px",
  "apple-sm": "8px",
  "apple-md": "16px",
  "apple-lg": "24px",
  "apple-xl": "32px",
  "apple-2xl": "48px",
},
```

- [ ] **Step 5: Add animation timing curves**

Add to `theme.extend.transitionTimingFunction`:

```js
transitionTimingFunction: {
  apple: "cubic-bezier(0.32, 0.72, 0, 1)",
  "apple-bounce": "cubic-bezier(0.34, 1.56, 0.64, 1)",
  "apple-ease": "cubic-bezier(0.25, 0.1, 0.25, 1)",
},
```

- [ ] **Step 6: Add new animations**

Replace the `animation` and `keyframes` blocks:

```js
animation: {
  "fade-in": "fadeIn 0.3s apple-ease",
  "slide-up": "slideUp 0.35s cubic-bezier(0.32, 0.72, 0, 1)",
  "slide-down": "slideDown 0.35s cubic-bezier(0.32, 0.72, 0, 1)",
  "scale-in": "scaleIn 0.3s cubic-bezier(0.32, 0.72, 0, 1)",
  "scale-out": "scaleOut 0.2s cubic-bezier(0.32, 0.72, 0, 1)",
  "slide-in-right": "slideInRight 0.35s cubic-bezier(0.32, 0.72, 0, 1)",
  "glow-pulse": "glowPulse 3s ease-in-out infinite",
  shimmer: "shimmer 2s infinite linear",
  "pulse-slow": "pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite",
},
keyframes: {
  fadeIn: {
    from: { opacity: "0" },
    to: { opacity: "1" },
  },
  slideUp: {
    from: { opacity: "0", transform: "translateY(12px) scale(0.98)" },
    to: { opacity: "1", transform: "translateY(0) scale(1)" },
  },
  slideDown: {
    from: { opacity: "0", transform: "translateY(-12px) scale(0.98)" },
    to: { opacity: "1", transform: "translateY(0) scale(1)" },
  },
  scaleIn: {
    from: { opacity: "0", transform: "scale(0.95) translateY(8px)" },
    to: { opacity: "1", transform: "scale(1) translateY(0)" },
  },
  scaleOut: {
    from: { opacity: "1", transform: "scale(1) translateY(0)" },
    to: { opacity: "0", transform: "scale(0.95) translateY(8px)" },
  },
  slideInRight: {
    from: { opacity: "0", transform: "translateX(20px)" },
    to: { opacity: "1", transform: "translateX(0)" },
  },
  glowPulse: {
    "0%, 100%": { opacity: "0.6" },
    "50%": { opacity: "1" },
  },
  shimmer: {
    "0%": { backgroundPosition: "-200% 0" },
    "100%": { backgroundPosition: "200% 0" },
  },
},
```

- [ ] **Step 7: Verify build compiles**

Run: `cd frontend && npx tsc --noEmit`
Expected: No errors (config is JS, but downstream CSS references must resolve)

- [ ] **Step 8: Commit**

```bash
git add frontend/tailwind.config.js
git commit -m "feat(design): update tailwind tokens for visionOS glassmorphism"
```

---

### Task 2: Update index.css — Glass Classes & Typography

**Files:**
- Modify: `frontend/src/index.css`

This task rewrites the component CSS classes to use the new visionOS design language.

- [ ] **Step 1: Update body typography**

In the `@layer base` block, change `body` line-height and add `leading-relaxed`:

```css
body {
  font-family: -apple-system, BlinkMacSystemFont, "SF Pro Text",
    "SF Pro Display", "Helvetica Neue", Helvetica, system-ui, sans-serif;
  background-color: #0A0C10;
  background-image:
    radial-gradient(circle at 50% -20%, rgba(47, 129, 247, 0.12), transparent 80%),
    radial-gradient(circle at 0% 0%, rgba(163, 113, 247, 0.04), transparent 40%);
  background-attachment: fixed;
  color: #F0F6FC;
  line-height: 1.625;
  letter-spacing: -0.022em;
}
```

- [ ] **Step 2: Replace liquid-glass classes with vision-glass**

Replace the entire `/* ===== Liquid Glass Base ===== */` section with:

```css
/* ===== Vision Glass ===== */
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
  transition: all 0.25s cubic-bezier(0.32, 0.72, 0, 1);
}

.vision-glass-hover:hover {
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

.vision-glass-strong {
  background: rgba(255, 255, 255, 0.06);
  backdrop-filter: saturate(200%) blur(80px);
  -webkit-backdrop-filter: saturate(200%) blur(80px);
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-top-color: rgba(255, 255, 255, 0.18);
  border-radius: 16px;
  box-shadow:
    inset 0 1px 0 rgba(255, 255, 255, 0.1),
    inset 0 0 30px rgba(255, 255, 255, 0.03),
    0 0 0 1px rgba(255, 255, 255, 0.08),
    0 4px 8px rgba(0, 0, 0, 0.35),
    0 12px 32px rgba(0, 0, 0, 0.45),
    0 28px 64px rgba(0, 0, 0, 0.55);
}

.vision-glass-subtle {
  background: rgba(255, 255, 255, 0.02);
  backdrop-filter: saturate(150%) blur(40px);
  -webkit-backdrop-filter: saturate(150%) blur(40px);
  border: 1px solid rgba(255, 255, 255, 0.05);
  border-radius: 16px;
}
```

- [ ] **Step 3: Update card-dark to use vision glass**

Replace the `/* ===== Card Dark ===== */` section:

```css
/* ===== Card Dark (vision) ===== */
.card-dark {
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
  transition: all 0.25s cubic-bezier(0.32, 0.72, 0, 1);
}

.card-dark:hover {
  background: rgba(255, 255, 255, 0.06);
  border-top-color: rgba(255, 255, 255, 0.2);
  transform: translateY(-2px);
  box-shadow:
    inset 0 1px 0 rgba(255, 255, 255, 0.12),
    inset 0 0 30px rgba(255, 255, 255, 0.03),
    0 0 0 1px rgba(255, 255, 255, 0.12),
    0 4px 8px rgba(0, 0, 0, 0.35),
    0 16px 32px rgba(0, 0, 0, 0.45),
    0 32px 72px rgba(0, 0, 0, 0.55);
}
```

- [ ] **Step 4: Update button styles**

Replace the `/* ===== Button Dark ===== */` section:

```css
/* ===== Button Dark (vision) ===== */
.btn-dark {
  @apply inline-flex items-center justify-center gap-1.5 font-medium rounded-apple-sm transition-all duration-200 active:scale-[0.98] disabled:opacity-40 disabled:cursor-not-allowed disabled:active:scale-100;
  transition-timing-function: cubic-bezier(0.32, 0.72, 0, 1);
}

.btn-dark-primary {
  @apply text-white;
  background: linear-gradient(180deg, #2F81F7 0%, #1A6ED8 100%);
  border: 1px solid rgba(255, 255, 255, 0.1);
  backdrop-filter: blur(10px);
  box-shadow:
    inset 0 1px 0 rgba(255, 255, 255, 0.2),
    0 2px 4px rgba(0, 0, 0, 0.3),
    0 0 15px rgba(47, 129, 247, 0.2);
}

.btn-dark-primary:hover {
  background: linear-gradient(180deg, #4493f8 0%, #2F81F7 100%);
  transform: translateY(-1px);
  box-shadow:
    inset 0 1px 0 rgba(255, 255, 255, 0.3),
    0 4px 12px rgba(47, 129, 247, 0.35),
    0 0 25px rgba(47, 129, 247, 0.25);
}

.btn-dark-secondary {
  @apply text-text-primary;
  background: rgba(255, 255, 255, 0.06);
  border: 1px solid rgba(255, 255, 255, 0.1);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.04);
}

.btn-dark-secondary:hover {
  background: rgba(255, 255, 255, 0.1);
  border-color: rgba(255, 255, 255, 0.18);
}

.btn-dark-ghost {
  @apply text-text-secondary bg-transparent;
}

.btn-dark-ghost:hover {
  @apply text-text-primary;
  background: rgba(255, 255, 255, 0.06);
}

.btn-dark-danger {
  @apply text-white;
  background: linear-gradient(180deg, #F85149 0%, #DA3633 100%);
  border: 1px solid rgba(255, 255, 255, 0.1);
  backdrop-filter: blur(10px);
}

.btn-dark-danger:hover {
  background: linear-gradient(180deg, #ff7b72 0%, #F85149 100%);
  transform: translateY(-1px);
  box-shadow: 0 4px 16px rgba(248, 81, 73, 0.4);
}
```

- [ ] **Step 5: Update input styles**

Replace the `/* ===== Input Dark ===== */` section:

```css
/* ===== Input Dark (vision) ===== */
.input-dark {
  @apply w-full px-3.5 py-2.5 rounded-apple-sm text-sm transition-all duration-150;
  background: rgba(0, 0, 0, 0.3);
  border: 1px solid rgba(255, 255, 255, 0.08);
  color: #F0F6FC;
  box-shadow: inset 0 2px 4px rgba(0, 0, 0, 0.5);
}

.input-dark::placeholder {
  @apply text-text-quaternary;
}

.input-dark:hover {
  border-color: rgba(255, 255, 255, 0.15);
}

.input-dark:focus {
  background: rgba(0, 0, 0, 0.35);
  border-color: #2F81F7;
  box-shadow:
    0 0 0 3px rgba(47, 129, 247, 0.15),
    inset 0 2px 4px rgba(0, 0, 0, 0.5);
  outline: none;
}
```

- [ ] **Step 6: Update nav glass**

Replace the `/* ===== Nav Glass Dark ===== */` section:

```css
/* ===== Nav Glass Dark (vision) ===== */
.nav-glass-dark {
  background: rgba(10, 12, 16, 0.7);
  backdrop-filter: saturate(200%) blur(30px);
  -webkit-backdrop-filter: saturate(200%) blur(30px);
  border-bottom: 1px solid transparent;
  background-image: linear-gradient(rgba(255, 255, 255, 0.06), rgba(255, 255, 255, 0.06));
  background-size: 100% 1px;
  background-repeat: no-repeat;
  background-position: bottom;
  box-shadow: 0 4px 24px rgba(0, 0, 0, 0.4);
}
```

- [ ] **Step 7: Verify build compiles**

Run: `cd frontend && npx tsc --noEmit`
Expected: No errors

- [ ] **Step 8: Commit**

```bash
git add frontend/src/index.css
git commit -m "feat(design): upgrade CSS classes to visionOS glassmorphism style"
```

---

### Task 3: Update Card Component

**Files:**
- Modify: `frontend/src/components/Card.tsx`

- [ ] **Step 1: Update Card to use vision-glass**

Replace the `baseClass` logic in `Card.tsx`:

```tsx
const baseClass = glass
  ? "vision-glass"
  : "bg-surface-elevated rounded-apple-xl border border-glass-border border-t-white/[0.05]";
```

Change `hoverCls` from `liquid-glass-hover` to `vision-glass-hover`:

```tsx
const hoverCls = hover
  ? "vision-glass-hover cursor-pointer"
  : "";
```

- [ ] **Step 2: Verify build compiles**

Run: `cd frontend && npx tsc --noEmit`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/Card.tsx
git commit -m "feat(design): update Card to use vision-glass"
```

---

### Task 4: Update Button Component

**Files:**
- Modify: `frontend/src/components/Button.tsx`

- [ ] **Step 1: Update Button sizes**

Change the `sizes` object to use `py-3` for `lg`:

```tsx
const sizes = {
  sm: "px-3 py-1.5 text-xs",
  md: "px-4 py-2 text-sm",
  lg: "px-5 py-3 text-sm",
};
```

- [ ] **Step 2: Verify build compiles**

Run: `cd frontend && npx tsc --noEmit`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/Button.tsx
git commit -m "feat(design): update Button sizes for visionOS"
```

---

### Task 5: Update Input Component

**Files:**
- Modify: `frontend/src/components/Input.tsx`

- [ ] **Step 1: Update Input to use translucent background and enhanced focus**

Replace the `inputBase` and `inputState` constants:

```tsx
const inputBase =
  "w-full bg-black/30 border rounded-apple-sm px-3 py-2 text-sm text-text-primary placeholder:text-text-quaternary transition-all duration-150 appearance-none shadow-[inset_0_2px_4px_rgba(0,0,0,0.5)]";

const inputState = error
  ? "border-brand-danger"
  : "border-white/[0.08] hover:border-white/[0.15] focus:border-brand-primary focus:shadow-[0_0_0_3px_rgba(47,129,247,0.15),inset_0_2px_4px_rgba(0,0,0,0.5)] focus:outline-none";
```

- [ ] **Step 2: Verify build compiles**

Run: `cd frontend && npx tsc --noEmit`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/Input.tsx
git commit -m "feat(design): update Input with translucent bg and enhanced focus ring"
```

---

### Task 6: Update Select Component

**Files:**
- Modify: `frontend/src/components/Select.tsx`

- [ ] **Step 1: Update Select to match Input styling**

Replace `selectBase` and `selectState`:

```tsx
const selectBase =
  "w-full bg-black/30 border rounded-apple-sm px-3 py-2 pr-9 text-sm text-text-primary transition-all duration-150 appearance-none shadow-[inset_0_2px_4px_rgba(0,0,0,0.5)]";

const selectState = error
  ? "border-brand-danger"
  : "border-white/[0.08] hover:border-white/[0.15] focus:border-brand-primary focus:shadow-[0_0_0_3px_rgba(47,129,247,0.15),inset_0_2px_4px_rgba(0,0,0,0.5)] focus:outline-none";
```

- [ ] **Step 2: Verify build compiles**

Run: `cd frontend && npx tsc --noEmit`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/Select.tsx
git commit -m "feat(design): update Select to match visionOS input style"
```

---

### Task 7: Update Table Component

**Files:**
- Modify: `frontend/src/components/Table.tsx`

- [ ] **Step 1: Update Table header and row styles**

Replace the `<thead>` `<tr>` className:

```tsx
<tr className="border-b border-white/[0.06] bg-white/[0.03]">
```

Replace the data row `<tr>` className:

```tsx
className={`
  border-b border-white/[0.04] relative
  transition-colors duration-150
  ${isClickable ? "cursor-pointer hover:bg-white/[0.06] hover:shadow-[inset_3px_0_0_0_#2F81F7]" : "hover:bg-white/[0.04]"}
`}
```

- [ ] **Step 2: Verify build compiles**

Run: `cd frontend && npx tsc --noEmit`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/Table.tsx
git commit -m "feat(design): update Table with visionOS header and row styles"
```

---

### Task 8: Update Modal Component

**Files:**
- Modify: `frontend/src/components/Modal.tsx`

- [ ] **Step 1: Update Modal with stronger glass and scale animation**

Replace the backdrop `<div>`:

```tsx
<div className="fixed inset-0 bg-black/60 backdrop-blur-xl" />
```

Replace the modal content `<div>`:

```tsx
<div
  ref={contentRef}
  tabIndex={-1}
  role="dialog"
  aria-modal="true"
  aria-labelledby={title ? "modal-title" : undefined}
  className={`relative z-10 w-full ${sizeMap[size]} vision-glass-strong rounded-[20px] animate-scale-in outline-none`}
>
```

- [ ] **Step 2: Verify build compiles**

Run: `cd frontend && npx tsc --noEmit`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/Modal.tsx
git commit -m "feat(design): update Modal with vision-glass-strong and scale animation"
```

---

### Task 9: Update Navbar Component

**Files:**
- Modify: `frontend/src/components/Navbar.tsx`

- [ ] **Step 1: Update active indicator to pill style**

Replace the active indicator `<span>` in both nav item maps (global and project). The current code:

```tsx
{isActive && (
  <span className="absolute bottom-0 left-1/2 -translate-x-1/2 w-4 h-[2px] bg-brand-primary rounded-full shadow-[0_0_8px_rgba(47,129,247,0.6)]" />
)}
```

Replace with pill background on the link itself. Update the Link className in both places:

For global nav items:
```tsx
<Link
  key={item.path}
  to={item.path}
  aria-current={isActive ? "page" : undefined}
  className={`relative px-3 py-1.5 text-[13px] font-medium rounded-full transition-all duration-200 whitespace-nowrap ${
    isActive
      ? "text-text-primary bg-white/[0.08]"
      : "text-text-tertiary hover:text-text-secondary hover:bg-white/[0.04]"
  }`}
>
  {item.label}
</Link>
```

For project nav items (same pattern):
```tsx
<Link
  key={item.path}
  to={item.path}
  aria-current={isActive ? "page" : undefined}
  className={`relative px-3 py-1.5 text-[13px] font-medium rounded-full transition-all duration-200 whitespace-nowrap ${
    isActive
      ? "text-text-primary bg-white/[0.08]"
      : "text-text-tertiary hover:text-text-secondary hover:bg-white/[0.04]"
  }`}
>
  {item.label}
</Link>
```

Remove the `<span>` indicator elements from both.

- [ ] **Step 2: Verify build compiles**

Run: `cd frontend && npx tsc --noEmit`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/Navbar.tsx
git commit -m "feat(design): update Navbar with pill indicators and enhanced glass"
```

---

### Task 10: Update Badge, Skeleton, EmptyState

**Files:**
- Modify: `frontend/src/components/Badge.tsx`
- Modify: `frontend/src/components/Skeleton.tsx`
- Modify: `frontend/src/components/EmptyState.tsx`

- [ ] **Step 1: Update Badge border radius**

In `Badge.tsx`, change `rounded-full` to `rounded-lg` for a more visionOS feel (keep `rounded-full` only for dot variant). Update the `<span>` className:

```tsx
<span
  className={`inline-flex items-center gap-1 rounded-lg font-medium border backdrop-blur-sm ${styles.bg} ${styles.text} ${styles.border} ${styles.glow || ""} ${sizeCls} ${className}`}
  {...props}
>
```

- [ ] **Step 2: Update Skeleton to use vision-glass**

In `Skeleton.tsx`, update `SkeletonCard` to use `vision-glass` instead of `liquid-glass`:

```tsx
export function SkeletonCard({ lines = 3 }: { lines?: number }) {
  return (
    <div className="vision-glass p-5 space-y-3">
      <Skeleton className="h-4 w-1/3" />
      {Array.from({ length: lines }).map((_, i) => (
        <Skeleton key={i} className="h-3 w-full" />
      ))}
    </div>
  );
}
```

- [ ] **Step 3: Update EmptyState**

In `EmptyState.tsx`, update the icon container to use vision styling:

```tsx
<div className="w-12 h-12 rounded-apple bg-white/[0.04] flex items-center justify-center text-text-quaternary mb-4 border border-white/[0.08]">
```

Update title to use the new typography hierarchy:

```tsx
<h3 className="text-sm font-medium text-text-secondary mb-1">{title}</h3>
```

- [ ] **Step 4: Verify build compiles**

Run: `cd frontend && npx tsc --noEmit`
Expected: No errors

- [ ] **Step 5: Commit**

```bash
git add frontend/src/components/Badge.tsx frontend/src/components/Skeleton.tsx frontend/src/components/EmptyState.tsx
git commit -m "feat(design): update Badge, Skeleton, EmptyState for visionOS consistency"
```

---

### Task 11: Update Toast Component

**Files:**
- Modify: `frontend/src/components/Toast.tsx`

- [ ] **Step 1: Update Toast to use vision glass**

Replace the toast `<div>` className and inline style:

```tsx
<div
  key={t.id}
  className={`pointer-events-auto flex items-center gap-2.5 px-4 py-2.5 rounded-2xl border ${bgMap[t.type]} animate-slide-down`}
  style={{
    background: "rgba(18, 18, 26, 0.85)",
    backdropFilter: "saturate(200%) blur(60px)",
    WebkitBackdropFilter: "saturate(200%) blur(60px)",
    boxShadow:
      "inset 0 1px 0 rgba(255,255,255,0.08), 0 4px 12px rgba(0,0,0,0.4), 0 16px 40px rgba(0,0,0,0.3)",
  }}
>
```

- [ ] **Step 2: Verify build compiles**

Run: `cd frontend && npx tsc --noEmit`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/Toast.tsx
git commit -m "feat(design): update Toast with vision glass styling"
```

---

### Task 12: Visual Verification

**Files:** None (read-only verification)

- [ ] **Step 1: Start dev server**

Run: `cd frontend && npm run dev`

- [ ] **Step 2: Open in browser and verify**

Open `http://localhost:5173` and verify:
- Cards have floating glass effect with multi-layer shadows
- Buttons have glass feel on secondary, hover lift on primary
- Inputs have translucent background with recessed feel
- Modal uses stronger glass and scale-in animation
- Navbar has pill-style active indicators
- Typography has light page titles, medium body text
- Overall depth and spatial layering feels visionOS-inspired

- [ ] **Step 3: Fix any visual issues**

If any component looks off, adjust the CSS classes in index.css or component files.

- [ ] **Step 4: Final commit if fixes needed**

```bash
git add -A
git commit -m "fix(design): polish visionOS glassmorphism after visual review"
```
