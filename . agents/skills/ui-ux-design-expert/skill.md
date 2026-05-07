---
skill_id: SKILL-002
version: 1.0.0
status: active
priority: existing
group: Kiến trúc & Thiết kế
created_at: 2026-04-24
---

# SKILL-002 · UI/UX Design

## Mô tả

Phân tích yêu cầu → thiết kế giao diện, đảm bảo consistency theo Design System, tối ưu trải nghiệm người dùng.

## Agents sử dụng

- `ui-schema-generator-agent`
- `ui-renderer-agent`

---

## Năng lực cốt lõi

### 1. Requirement Analysis for UI

```
UI Requirement Extraction Process:
1. Identify user roles (actors from KG)
2. Map user goals → screens (1 primary action = 1 screen)
3. Define information hierarchy per screen
4. Identify critical user flows (happy path + error states)
5. Document edge cases and empty states
```

### 2. Design System (Atomic Design)

```
Atomic Design Hierarchy:
├── Atoms: Button, Input, Label, Icon, Badge
├── Molecules: Form Field, Search Bar, Card Header, Alert
├── Organisms: Navigation, Data Table, Form Section, Modal
├── Templates: Page Layout, Dashboard Layout, Form Page
└── Pages: Specific instances with real data

Design Tokens:
├── Colors: Primary, Secondary, Neutral, Semantic (Success/Warning/Error)
├── Typography: Font family, sizes (xs/sm/md/lg/xl/2xl), weights (400/600/700)
├── Spacing: 4px base unit (4/8/12/16/24/32/48/64/96)
├── Borders: radius (sm/md/lg/full), width (1px/2px)
├── Shadows: sm/md/lg/xl
└── Motion: duration (fast:150ms, normal:250ms, slow:400ms), easing
```

### 3. Visual Hierarchy

```
7 Principles:
1. Size    → Larger = more important
2. Color   → High contrast = primary action
3. Contrast → Dark on light for readability
4. Spacing  → Whitespace groups related elements
5. Position → Top-left = highest attention
6. Repetition → Consistent patterns = learnable
7. Alignment → Grid alignment = professional
```

### 4. Micro-interactions

```
Animation Guidelines:
- Hover: 150ms ease-out (instant feel)
- State change (loading/success): 250ms ease-in-out
- Modal open: 300ms ease-out (slide up + fade)
- Page transition: 200ms fade
- Skeleton loading: infinite 1.5s pulse animation

Feedback Patterns:
- Success: Green checkmark + toast (3s auto-dismiss)
- Error: Red alert inline, persistent until resolved
- Loading: Skeleton screen for initial load, spinner for actions
- Empty state: Illustration + CTA to add first item
```

### 5. Responsive Design

```css
/* Breakpoints chuẩn */
:root {
  --breakpoint-sm: 640px;   /* Mobile landscape */
  --breakpoint-md: 768px;   /* Tablet */
  --breakpoint-lg: 1024px;  /* Desktop */
  --breakpoint-xl: 1280px;  /* Wide desktop */
  --breakpoint-2xl: 1536px; /* Ultra-wide */
}

/* Mobile-first approach */
.container {
  padding: 16px;           /* Mobile */
}
@media (min-width: 768px) {
  .container { padding: 24px; }  /* Tablet */
}
@media (min-width: 1024px) {
  .container { padding: 32px; }  /* Desktop */
}
```

### 6. Accessibility (WCAG 2.1 AA)

```
Minimum Requirements:
├── Color contrast: 4.5:1 for text, 3:1 for large text
├── Focus indicators: visible on all interactive elements
├── Alt text: required for all images
├── Form labels: associated with inputs (not just placeholder)
├── Error messages: announced to screen readers
├── Keyboard navigation: all features accessible via keyboard
└── ARIA labels: for icon-only buttons

Testing:
- axe DevTools for automated checks
- Manual testing with VoiceOver/NVDA
- Keyboard-only navigation test
```

---

## Checklist

- [ ] Design tokens định nghĩa trong CSS variables / design system
- [ ] Mobile-first responsive design
- [ ] WCAG 2.1 AA contrast ratios
- [ ] Empty states và error states đã có design
- [ ] Loading states (skeleton screens) được thiết kế
- [ ] Focus indicators visible
- [ ] Micro-animations theo timing guidelines
