# Cookie Settings

**Predict-A-Trade** — manage your preferences for this browser

> Change your choices at any time. Withdrawing consent is as easy as giving it. Your selection is stored in `pat_consent` and lasts `[12 months]`.

---

## Your choices

---

### Strictly necessary — **Always active**

Required to deliver the Service, keep you signed in, remember your cookie choice, balance traffic and protect against fraud and abuse. These cannot be switched off, because the Service cannot function without them. They do not require your consent.

**What they do:** session and authentication · security and CSRF protection · load balancing · storing this cookie choice · bot detection and rate limiting

`[ ] Always enabled`

---

### Preferences — `[ ] Off` `[ ] On`

Optional. Remembers your watchlists, alert thresholds, display settings, dashboard layout, language, currency and time zone, so you do not have to reconfigure on every visit.

**If you turn this off,** the Service still works — you may simply need to re-set your preferences each session.

---

### Analytics — `[ ] Off` `[ ] On`

Optional. Helps us understand which features are used, where the Service fails, and what to improve. We aggregate this data and truncate IP addresses where possible.

**Currently loaded:** `[NO ANALYTICS PROVIDER IS CURRENTLY LOADED — or name the provider]`

**If you turn this off,** nothing changes except that we learn less about how to improve the Service.

**In the EEA, UK, Switzerland, Brazil and India this stays off unless you switch it on.** We never enable it by scrolling, browsing or inactivity.

---

### Marketing — `[ ] Off` `[ ] On`

Optional. Would be used to measure campaigns and, if we ever advertise, to show relevant ads off-site.

**Currently loaded:** `NO MARKETING TECHNOLOGY IS CURRENTLY LOADED`

We do not run advertising, retargeting or cross-site tracking. If that ever changes, we will update the [Cookie Policy](https://predictatrade.com/legal/cookie-policy/) and ask for your consent **before** anything loads.

---

### `[ ] Global Privacy Control detected`

`[If your browser is sending a GPC signal, we show this and treat it as an opt-out of analytics and marketing automatically.]`

---

## Buttons

| | |
|---|---|
| **Accept all** | Enables preferences, analytics and marketing |
| **Reject all** | Enables strictly necessary only |
| **Save my choices** | Applies the toggles exactly as set |

**Reject all is presented with the same size, colour and prominence as Accept all.** No pre-ticked boxes. No nudging. No cookie wall — you can use the Service whichever you choose.

---

## What your choice does

- It applies to **this browser and device**. Other browsers or devices need their own setting.
- It is stored for `[12 months]`, after which we ask again so your choice stays current.
- **Withdrawing consent is always as easy as giving it** — revisit this page or use the footer link any time.
- Withdrawal stops future processing. It does not undo processing already lawfully carried out.
- Clearing your browser's cookies resets everything to **necessary only**.

---

## Browser-level controls

You can also manage cookies in your browser:

| Browser | Path |
|---|---|
| Chrome | Settings → Privacy and security → Third-party cookies |
| Edge | Settings → Cookies and site permissions |
| Firefox | Settings → Privacy & Security → Cookies and Site Data |
| Safari | Settings → Privacy → Manage Website Data |
| Opera | Settings → Advanced → Privacy & security → Cookies |

**Note:** blocking strictly necessary cookies will stop you being able to sign in.

---

## Related

- [Cookie Policy](https://predictatrade.com/legal/cookie-policy/) — the full list of cookies, providers and lifetimes
- [Privacy Policy](https://predictatrade.com/legal/privacy-policy/) — how we handle all personal data, and your rights by region
- [Complaints Procedure](https://predictatrade.com/legal/complaints/) — if you are unhappy with how we handled your choice
- `[PRIVACY EMAIL]` — questions

---

> **Implementation note.** The toggles above must be **wire-framed against real behaviour**, not just documented: optional scripts must not load, and optional cookies must not be written, until the corresponding consent flag is `true` in `pat_consent`. Test the "Reject all" path with a clean profile and confirm in devtools that no optional request fires. Regulators in the EEA specifically test for "reject" being honoured in fact rather than in form.
