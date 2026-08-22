# Predict-A-Trade authentication UI

A responsive front-end implementation of the Predict-A-Trade login and account-creation experience, styled to match the public website's editorial fintech identity.

## Pages

- `index.html` — primary sign-in page
- `login.html` — sign-in alias for routing compatibility
- `signup.html` — two-step account-creation flow

## Included

- Predict-A-Trade brand imagery and locally bundled fonts
- Palette aligned with the public website's cream, ink and signal-blue tokens
- Desktop split-screen and focused mobile layouts
- Viewport-fitted composition with no page scrolling, including compact short-screen rules
- Accessible labels, keyboard focus states and inline validation
- Password visibility controls and strength requirements
- Responsive risk/legal footer with live legal-policy links
- Reduced-motion support
- Interactive loading, progress and success states

## Preview locally

```bash
cd predict-auth
python -m http.server 4173 --bind 0.0.0.0
```

Then open `http://localhost:4173/`. The create-account link opens `signup.html`.

## Integration notes

The forms intentionally simulate successful submission. Replace the submit handlers in `app.js` with your authentication and registration API calls. Also connect the password-recovery link to the production reset route.

The UI does not claim that an account has been created until a real backend is connected; the prototype success message makes that distinction explicit.
