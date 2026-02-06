# Investment Portfolio Manager - Design System

**Last Updated**: 2026-02-05
**Status**: Extracted from existing codebase

This design system documents the established patterns, tokens, and components used throughout the Investment Portfolio Manager frontend application.

---

## Design Direction

**Style**: Professional financial dashboard with clean, data-focused interface
**Depth Strategy**: Border-centric with subtle shadows for elevation
**Grid System**: 8px baseline grid
**Theme Support**: Light and dark modes
**Mobile Support**: Responsive design with 768px primary breakpoint

---

## Spacing

**Base Unit**: 8px
**Scale**: Multiples of 4px

```
--space-1: 4px      (0.25rem)
--space-2: 8px      (0.5rem)   ← Most common
--space-3: 12px     (0.75rem)
--space-4: 16px     (1rem)
--space-5: 20px     (1.25rem)  ← Very common
--space-6: 24px     (1.5rem)
--space-8: 32px     (2rem)
--space-10: 40px    (2.5rem)
```

**Usage Guidelines**:
- Component internal padding: 8px, 12px, 16px, 20px
- Card/container padding: 20px (mobile: 12-16px)
- Section gaps: 20px, 24px
- Page margins: 32px, 40px
- Stack spacing (flex/grid gap): 8px, 12px, 16px

---

## Border Radius

```
--radius-sm: 2px     (badges, small elements)
--radius-md: 4px     (default: buttons, inputs, cards)
--radius-lg: 8px     (containers, modals)
--radius-xl: 12px    (banners, large components)
--radius-full: 50%   (circular elements)
```

**Default**: 4px for most interactive elements

---

## Colors

### Semantic Colors

**Primary (Blue)**
```
--primary: #1976d2
--primary-light: #2196f3
--primary-dark: #0d47a1
--primary-bg: #e3f2fd (backgrounds)
```

**Success (Green)**
```
--success: #4caf50
--success-dark: #2e7d32
--success-light: #66bb6a
--success-bg: #e8f5e9
```

**Error (Red)**
```
--error: #f44336
--error-dark: #c62828
--error-light: #ef5350
--error-bg: #ffebee
```

**Warning (Orange/Yellow)**
```
--warning: #ffc107
--warning-dark: #f57c00
--warning-light: #ff9800
--warning-bg: #fff3cd
```

**Neutral (Gray)**
```
--gray-50: #f8f9fa
--gray-100: #f5f5f5
--gray-200: #eee
--gray-300: #ddd
--gray-400: #ccc
--gray-500: #999
--gray-600: #666
--gray-700: #495057
--gray-800: #343a40
--gray-900: #212529
```

### Functional Colors

**Action Colors** (defined in common.css):
```
--add-color: #4caf50       (success green)
--edit-color: #ffc107      (warning yellow)
--delete-color: #f44336    (error red)
--archive-color: #9e9e9e   (neutral gray)
--positive-color: #4caf50  (gains)
--negative-color: #f44336  (losses)
```

### Dark Mode

**Backgrounds**:
```
--dark-bg-primary: #1a1a1a
--dark-bg-secondary: #2d2d2d
--dark-bg-elevated: #343a40
--dark-bg-hover: #495057
```

**Text**:
```
--dark-text-primary: #f8f9fa
--dark-text-secondary: #e0e0e0
--dark-text-muted: #adb5bd
```

---

## Typography

### Font Sizes

```
--text-xs: 0.75rem (12px)    (badges, tooltips, small labels)
--text-sm: 0.8rem (13px)     (secondary text)
--text-base: 1rem (16px)     (body text, default)
--text-md: 0.9rem (14px)     (common labels)
--text-lg: 1.1rem (18px)     (section headers)
--text-xl: 1.2rem (19px)     (subsection headers)
--text-2xl: 1.5rem (24px)    (page/modal titles)
```

**Default Body**: 1rem (16px)

### Font Weights

```
--font-normal: 400   (body text)
--font-medium: 500   (labels, secondary emphasis)
--font-semibold: 600 (headings, strong emphasis)
```

**Note**: 700 (bold) is rarely used in this system

### Line Heights

```
--leading-tight: 1.2  (headings)
--leading-normal: 1.5 (body text)
--leading-relaxed: 1.6 (long-form content)
```

---

## Depth & Elevation

### Shadows

The system uses **subtle shadows** for elevation, not dramatic depth.

```
--shadow-sm: 0 1px 3px rgba(0, 0, 0, 0.1)      (subtle lift)
--shadow-md: 0 2px 4px rgba(0, 0, 0, 0.1)      (cards, default elevation)
--shadow-lg: 0 4px 6px rgba(0, 0, 0, 0.1)      (modals, elevated containers)
--shadow-xl: 0 4px 12px rgba(0, 0, 0, 0.15)    (hover states, focus)
--shadow-2xl: 0 10px 40px rgba(0, 0, 0, 0.3)   (overlays, critical alerts)
```

### Borders

**Primary depth strategy**: Borders over shadows

```
--border-default: 1px solid #ddd       (standard borders)
--border-medium: 2px solid             (emphasis, active states)
--border-heavy: 3px solid              (strong emphasis, badges)
--border-light: 1px solid #eee         (very subtle dividers)
```

**Border Colors**:
- Default: `#ddd`, `#dee2e6`
- Dark: `#ccc`
- Light: `#eee`
- Semantic: Use semantic color variables for focus/active

### Focus States

```
outline: 2px solid var(--primary)
box-shadow: 0 0 0 0.2rem rgba(0, 123, 255, 0.25)
```

---

## Component Patterns

### Button

**Sizes**:
```
Small:  height: 32-36px, padding: 0.375rem 0.75rem (6px 12px)
Medium: height: 38-44px, padding: 0.5rem 1rem (8px 16px)      ← Default
Large:  height: 44-52px, padding: 0.75rem 1.5rem (12px 24px)
```

**Border Radius**: 4px

**Variants**:
- **Primary**: Background `--primary`, white text
- **Success**: Background `--success`, white text
- **Danger**: Background `--error`, white text
- **Warning**: Background `--warning`, dark text
- **Secondary**: Background `--gray-500`, white text
- **Outline**: Border 1px, background transparent

**States**:
- Hover: Darken 10%, shadow: `--shadow-lg`
- Active: Darken 15%
- Disabled: Opacity 0.5, cursor not-allowed

**Transition**: 0.2s ease

---

### Card/Container

**Padding**: 20px (desktop), 12-16px (mobile)
**Border Radius**: 8px
**Border**: 1px solid `--gray-300` (optional)
**Shadow**: `--shadow-md` (0 2px 4px rgba(0,0,0,0.1))

**Hover State**: Shadow increases to `--shadow-lg`

**Gap Between Cards**: 12-20px

```css
.card {
  padding: 20px;
  border-radius: 8px;
  border: 1px solid var(--gray-300);
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.card:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}
```

---

### Input/Form Field

**Height**: 36-44px (aim for 44px for accessibility)
**Padding**: 8px
**Border**: 1px solid `--gray-300`
**Border Radius**: 4px
**Font Size**: 1rem

**States**:
- **Focus**: Border color `--primary`, shadow: `0 0 0 0.2rem rgba(0, 123, 255, 0.25)`
- **Error**: Border color `--error`, shadow with red tint
- **Disabled**: Background `--gray-100`, cursor not-allowed

```css
input, select, textarea {
  height: 44px;
  padding: 8px;
  border: 1px solid var(--gray-300);
  border-radius: 4px;
  font-size: 1rem;
}

input:focus {
  border-color: var(--primary);
  box-shadow: 0 0 0 0.2rem rgba(0, 123, 255, 0.25);
  outline: none;
}
```

---

### Modal

**Sizes**:
```
Small:  max-width: 400px
Medium: max-width: 600px   ← Default
Large:  max-width: 900px
XLarge: max-width: 1200px
```

**Padding**: 20px
**Border Radius**: 8px
**Shadow**: `--shadow-lg` (0 4px 6px rgba(0,0,0,0.1))
**Backdrop**: rgba(0, 0, 0, 0.5)

**Structure**:
- Header: 20px bottom margin
- Body: Standard spacing
- Footer: 20px top margin, buttons right-aligned

```css
.modal {
  max-width: 600px;
  padding: 20px;
  border-radius: 8px;
  box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
  background: white;
}
```

---

### Table

**Row Height**: 36-44px
**Cell Padding**: 0.75rem (12px)
**Border**: 1px solid `--gray-300`

**Header**:
- Background: `--gray-100` or `--gray-50`
- Font weight: 600
- Text align: left (numbers: right)

**Rows**:
- Hover: Background `--gray-50`
- Alternate (optional): Background `--gray-100` (zebra striping)

**Responsive**: Horizontal scroll on mobile

```css
table {
  width: 100%;
  border-collapse: collapse;
}

th, td {
  padding: 0.75rem;
  text-align: left;
  border: 1px solid var(--gray-300);
}

th {
  background: var(--gray-100);
  font-weight: 600;
}

tr:hover {
  background: var(--gray-50);
}
```

---

### Badge

**Padding**: 0.2rem 0.5rem (3px 8px)
**Border Radius**: 3px
**Font Size**: 0.75rem (12px)
**Font Weight**: 600
**Text Transform**: uppercase

**Variants**: Match semantic colors (primary, success, error, warning)

```css
.badge {
  display: inline-block;
  padding: 0.2rem 0.5rem;
  border-radius: 3px;
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
}

.badge-success {
  background: var(--success);
  color: white;
}
```

---

## Transitions & Animation

**Standard Transition**: `0.2s ease`
**Properties**: background-color, color, border-color, box-shadow, transform

**Hover/Focus Timing**: 200ms
**Modal/Toast Entry**: 300ms
**Loading Spinners**: 1s linear infinite

```css
button, input, .card {
  transition: all 0.2s ease;
}
```

---

## Responsive Breakpoints

```
--breakpoint-sm: 576px   (small devices)
--breakpoint-md: 768px   (tablets)     ← Primary breakpoint
--breakpoint-lg: 992px   (desktops)
--breakpoint-xl: 1200px  (large desktops)
```

**Mobile-First Approach**: Design for mobile, enhance for desktop

**Common Patterns**:
- Stack vertically on mobile (< 768px)
- Side-by-side on tablet+ (>= 768px)
- Reduce padding on mobile (12-16px vs 20px)

---

## Accessibility

### Focus Indicators

All interactive elements MUST have visible focus states:
```css
:focus {
  outline: 2px solid var(--primary);
  outline-offset: 2px;
}
```

### Color Contrast

- Text on light backgrounds: `--gray-900` or darker
- Text on dark backgrounds: `--gray-50` or lighter
- Minimum contrast ratio: 4.5:1 (WCAG AA)

### Touch Targets

- Minimum size: 44x44px (especially mobile)
- Spacing between: 8px minimum

---

## Usage Notes

### When to Use Shadows vs Borders

**Use Borders**:
- Defining component boundaries (cards, inputs, tables)
- Separating sections (dividers, headers)
- Visual structure and layout

**Use Shadows**:
- Elevating components above the surface (modals, dropdowns)
- Hover states to indicate interactivity
- Focus states combined with outline

**Do Not**:
- Mix heavy shadows with borders on the same element
- Use shadows for flat, inline components (badges, chips)

### Color Usage

**Primary Blue**: CTAs, links, focus states, primary actions
**Success Green**: Positive actions (add, save), gains, success messages
**Error Red**: Destructive actions (delete), losses, error messages
**Warning Orange**: Edit actions, warnings, caution states
**Gray**: Neutral actions (cancel, archive), disabled states

### Spacing Consistency

- Use the spacing scale exclusively (no random values like 17px, 23px)
- Prefer `0.5rem`, `1rem`, `1.5rem` over pixel values for maintainability
- Keep consistent gaps in flex/grid layouts (8px, 12px, 16px)

---

## Design System Compliance

To check if your code follows this system, audit for:

1. **Spacing violations**: Values not on the 4px/8px grid
2. **Color violations**: Colors not in the defined palette
3. **Shadow violations**: Shadows not matching the defined scale
4. **Border violations**: Border widths not 1px, 2px, or 3px
5. **Pattern drift**: Components not matching defined patterns

Run `/interface-design:audit` to scan for violations.

---

## Maintenance

**Review Cadence**: Quarterly or when adding major features
**Update Process**: Extract patterns from new code, update system.md
**Breaking Changes**: Version the system if making major changes

**Last Extraction**: 2026-02-05
**Files Analyzed**: 24 CSS files, 60+ component files
