# Predict-A-Trade — Legal Pack

Six markdown documents, drafted for **Simha Fintech LLC (Dubai, UAE mainland)**, covering **GCC · Europe (EEA/UK/CH) · Asia-Pacific · Americas**.

| File | Page it replaces |
|---|---|
| `01-terms-and-policies.md` | /legal/terms-and-policies/ |
| `02-privacy-policy.md` | /legal/privacy-policy/ |
| `03-complaints-procedure.md` | /legal/complaints/ |
| `04-cookie-policy.md` | /legal/cookie-policy/ |
| `05-cookie-settings.md` | /legal/cookie-settings/ |
| `06-legal-trust-center.md` | /legal/ |

---

## 1. What changed vs. the current pages

| | Current site | This pack |
|---|---|---|
| Length | ~120 words each | Full operating documents |
| Scope | "Static site, no accounts, no analytics" | Drafted as if accounts, subscriptions, payments, email marketing and analytics are live (per your instruction) |
| Jurisdiction | None stated | Named regimes, named regulators, named deadlines in every bloc |
| Regional rights | Absent | Consolidated "Your rights by region" section (Privacy §15) mirrored in the other documents |
| Regulatory position | Stated, correctly, as unlicensed | Same, plus a restricted-jurisdiction table and a sanctions clause |
| Transfers | Not addressed | Mechanism per corridor, incl. SCCs + TIA as backup to the DPF |
| Breach | Not addressed | 72-hour operating standard + authority list |
| Complaints | "No statutory deadlines asserted" | Real deadlines, plus every escalation body by region |

---

## 2. Before you publish — the punch list

### Must fix (the documents are inaccurate until you do)

1. **Fill every `[BRACKETED PLACEHOLDER]`.** Search for `[` across all six files. Do not publish a document with a placeholder in it — an unfilled `[REGISTERED ADDRESS]` in a privacy policy is an accuracy defect, not a style issue.
2. **Names of providers (Privacy §7.1).** Hosting, payment processor, email, analytics, support, monitoring, fonts, CDN. Then date the table and review it quarterly.
3. **Cookie table (Cookie Policy §2).** Generate it from an actual crawl of the live site and app. Do not carry over my example rows.
4. **Restricted jurisdictions (Terms §4.3).** Confirm with counsel, then enforce at sign-up, at payment, and by IP geofencing.
5. **Supported languages (Terms §1.6, §24).** If you serve UAE consumers, Arabic is the obvious gap. Quebec needs French. Brazil and Mexico need Portuguese/Spanish.
6. **Retention periods (Privacy §9).** Confirm the UAE tax/commercial retention requirement and each other applicable period.
7. **DPO decision (Privacy §14.1).** Decide and document. Under UAE PDPL a DPO is required for high-risk processing; under GDPR Art. 37 for certain processing. Write down the assessment either way.
8. **EU/UK Art. 27 representative (Privacy §14.2).** If you *target* the EEA/UK market — as opposed to merely being accessible there — you must appoint one. This is a common miss.

### Should fix

9. **Build the consent tool to actually gate scripts.** Cookie Settings §5 is only true if optional scripts do not *load* until consent is `true`. Test "Reject all" with a clean profile and watch the network tab.
10. **Honour GPC.** Twelve US states require it. Detection is a few lines; ignoring it is a finding.
11. **Build the rights-request workflow before you publish the rights.** You are promising 30-day (GDPR), 45-day (US states), 2-working-day acknowledgement and 10–20 day completion (Vietnam) timelines. The strictest applies.
12. **Click-to-cancel.** US consumers need a cancel path at least as easy as subscribe. Build it, don't just describe it.
13. **Version archive + 30-day change notice** (Privacy §17, Terms §1.5).
14. **Breach runbook** with the 72-hour clock and the authority list from Privacy §11.2 pre-populated.

---

## 3. Legal-review priorities

These are the areas where the law is genuinely unsettled or where I would not want you relying on my summary alone:

| Area | Why it needs counsel |
|---|---|
| **UAE PDPL Executive Regulations** | Sources conflict. Authoritative trackers (AWS, Chambers, DLA) state they are **not yet published in the Official Gazette** as a discrete instrument as of mid-2026; one secondary source claims a 2024 Cabinet Decision. My documents take the conservative position (not published; UAE Data Office guidance applies; 72-hour de facto standard). **Re-verify at publication and on each review.** |
| **Whether UAE PDPL applies at all to you** | It excludes DIFC/ADGM entities, government data, and health/credit/financial data already governed by specific UAE legislation. Confirm your categorisation. |
| **Art. 27 representative** | "Targeting" vs. "accessible" is a factual test with a large penalty attached to getting it wrong. |
| **Terms §4.3 restricted jurisdictions** | Determines whether you are distributing an unlicensed financial product in any given country. This is the single highest-risk clause in the pack. |
| **EU–US DPF** | *Latombe* was dismissed in Sept 2025 but is on appeal to the CJEU, and the EDPB asked the Commission to reassess in July 2026 after *Trump v. Slaughter*. My documents already back every transfer with SCCs + TIA so nothing depends on it — keep it that way. |
| **India DPDP** | Phased: Board live since Nov 2025, enforcement powers Nov 2026, **substantive obligations 13/14 May 2027**. I drafted to comply now. Confirm you want to absorb that cost early. |
| **Vietnam Decree 356** | The strictest response deadlines in the pack (2 working days to acknowledge). Confirm whether you have any Vietnamese users before committing to them operationally. |
| **Australia ADM transparency** | Required in privacy policies from **December 2026**. Already drafted (Privacy §6.4). |
| **California risk assessments** | Regs effective Jan 2026; first assessments due to the CPPA **1 April 2028**. Confirm scope. |
| **US state count** | 20 in effect as of 2026; Oklahoma (Jan 2027) and Alabama (2027) already enacted. Re-check each January. |

---

## 4. Standing calendar

| When | Do |
|---|---|
| **January** | Re-check the US state privacy list (new laws have taken effect every January) |
| **Quarterly** | Refresh the subprocessor/cookie tables; run a consent tool test |
| **April 2028** | California CPPA risk assessments due |
| **10 Dec 2026** | Australian ADM transparency and Children's Online Privacy Code |
| **13 Nov 2026** | India DPDP Board enforcement powers commence |
| **13 May 2027** | India DPDP substantive obligations in full |
| **1 Jan 2027** | UAE Child Digital Safety Law full compliance |
| **Annually** | Full review of all six documents; re-verify every citation in §15 of the Privacy Policy |

---

## 5. A note on what this is

This is a **drafting pack**, not legal advice, and I am not acting as your lawyer. The legal summaries in §15 of the Privacy Policy and §24 of the Terms are accurate to the best of my knowledge as at **August 2026**, but several of the regimes cited are moving fast and some statements (notably the UAE Executive Regulations, flagged above) rest on sources that disagree.

Have counsel admitted in **each bloc** review before publication. That is not boilerplate caution — the restricted-jurisdiction clause and the regulatory-status language determine whether this business can be distributed at all in several of the markets you named.
