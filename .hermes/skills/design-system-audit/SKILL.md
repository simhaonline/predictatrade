---
name: design-system-audit
description: "Audit UI for accessibility, consistency, and slop."
---

# design-system-audit

Use when reviewing Predict-A-Trade UI for quality, accessibility, and consistency.

## Audit Checklist

### Accessibility (WCAG 2.1 AA)
1. All interactive elements keyboard accessible (tabindex, role)
2. Focus states visible on all controls
3. Color contrast >= 4.5:1 for text, 3:1 for large text
4. Form inputs have associated labels
5. Images have alt text (or aria-hidden if decorative)
6. prefers-reduced-motion respected for animations
7. Touch targets >= 44px on mobile
8. No color-only information (use icons/text too)

### Honest States (AGENTS.md requirement)
[x] LIVE — green pulse, data timestamp visible
[x] DELAYED — amber, shows lag duration
[x] STALE — red, last data age shown
[x] DEGRADED — warning icon, which features affected
[x] DISCONNECTED — red, reconnect countdown
[x] UNAVAILABLE — grey, "data unavailable" not fake data
[x] INITIALIZING — spinner with message
[x] MAINTENANCE — banner with expected duration
[x] ERROR — error code + retry action
[x] LOADING — skeleton or spinner, not blank
[x] EMPTY — helpful message, not "No data"

### Consistency
1. Same component used for same pattern (don't reimplement)
2. Spacing scale matches tailwind.config.ts
3. Typography scale consistent (headings, body, captions)
4. Color tokens from theme, not hardcoded hex
5. Error messages follow same pattern
6. Button variants consistent (primary, secondary, ghost, danger)

### Anti-Slop Check
1. No fake/generated live data
2. No decorative stats without real source
3. No icon-topper cards without purpose
4. No glassmorphism without depth system
5. No centered-hero for Monitor surfaces
6. No rainbow palettes

## Verification
- Run: cd frontend && npx tsc --noEmit
- Run: cd frontend && npm test
- Run: cd frontend && npx eslint .
- Check: npx @google/design.md lint DESIGN.md (if exists)
- Visual: playwright screenshot tests
