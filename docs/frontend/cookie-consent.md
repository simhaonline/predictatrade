# Cookie Consent Implementation

## Model

Versioned consent stored in localStorage (`pat_cookie_consent`):
```json
{
  "version": "1",
  "necessary": true,
  "preferences": true,
  "analytics": false,
  "marketing": false,
  "updatedAt": "..."
}
```

## Categories

1. **Strictly Necessary**: auth cookies (pat_refresh_token, pat_access_token) — always active, cannot be disabled
2. **Preferences**: theme, accessibility settings — active by default
3. **Analytics**: not currently in use — off by default
4. **Marketing**: not currently in use — off by default

## Banner

Shows on first visit with exact copy:
> We use cookies to enhance site navigation, personalise content and ads, and analyse site usage. You can change your cookie settings at any time. For more information, please see our Cookie Policy.

Buttons: Reject All, Cookie Settings, Allow All Cookies

## Settings Modal

Granular toggle for each category. Save Preferences button persists consent.

## Persistence

- Consented: banner doesn't show again
- Footer "Cookie Settings" link reopens settings modal
- Version mismatch triggers re-consent
