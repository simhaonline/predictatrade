/**
 * Email template — the single branded HTML shell every platform email renders in.
 *
 * Brand tokens mirror frontend/src/styles/globals.css (dark theme):
 *   page bg   #0A111D  (hsl 220 47% 8%)
 *   card bg   #101A2C  (hsl 219 46% 12%)
 *   primary   #2362EA  (hsl 221 83% 53%)
 *   text      #F2F5F9  (hsl 216 33% 97%)
 *   secondary #B9C2CE
 *   muted     #77828F
 *   border    #24375A  (hsl 216 31% 24%)
 *
 * Logo is served from the platform CDN URL (frontend /public PNG variants —
 * email clients do not render SVG). White horizontal wordmark on the dark
 * header band.
 */

const BRAND = {
  pageBg: '#0A111D',
  cardBg: '#101A2C',
  primary: '#2362EA',
  primaryHover: '#1D4FC4',
  text: '#F2F5F9',
  textSecondary: '#B9C2CE',
  textMuted: '#77828F',
  border: '#24375A',
  success: '#22C55E',
  warning: '#E8A33D',
  danger: '#EF4444',
};

export interface EmailTemplateInput {
  /** Preheader shown next to the subject in inbox lists (hidden in body). */
  preheader?: string;
  /** Card headline under the logo. */
  title: string;
  /** Main HTML content of the card (paragraphs, buttons, code blocks). */
  bodyHtml: string;
  /** Plain-text fallback (used for text/plain part; footer appended here too). */
  bodyText: string;
  /** Unsubscribe link — included in footer for all types (CAN-SPAM hygiene). */
  unsubscribeUrl?: string;
  /** Footer note (e.g. why the user got this email). */
  footerNote?: string;
}

const LOGO_URL =
  process.env.EMAIL_LOGO_URL || 'https://platform.predictatrade.com/predict-a-trade_horizontal_white.png';
const APP_URL = process.env.APP_FRONTEND_URL || 'https://platform.predictatrade.com';
const CURRENT_YEAR = new Date().getUTCFullYear();

/** Plain-text footer shared by every send. */
export function textFooter(unsubscribeUrl?: string): string {
  const lines = [
    '— Predict-A-Trade',
    '',
    `Platform: ${APP_URL}`,
    `Support: support@predictatrade.com`,
    `© ${CURRENT_YEAR} Predict-A-Trade. All rights reserved.`,
    'Predict-A-Trade is a market analysis platform; nothing in this email is financial advice.',
  ];
  if (unsubscribeUrl) lines.push(`Unsubscribe from marketing emails: ${unsubscribeUrl}`);
  return lines.join('\n');
}

/** Builds the full branded HTML email. */
export function renderBrandedEmail(input: EmailTemplateInput): string {
  const preheader = input.preheader
    ? `<div style="display:none;max-height:0;overflow:hidden;mso-hide:all;">${input.preheader}</div>`
    : '';

  const unsubscribeBlock = input.unsubscribeUrl
    ? `<p style="margin:6px 0 0;color:${BRAND.textMuted};font-size:11px;">
         Don't want marketing emails?
         <a href="${input.unsubscribeUrl}" style="color:${BRAND.primary};text-decoration:none;">Unsubscribe</a>.
       </p>`
    : '';

  return `<!DOCTYPE html>
<html lang="en" xmlns="http://www.w3.org/1999/xhtml">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <meta name="color-scheme" content="dark" />
  <title>Predict-A-Trade</title>
</head>
<body style="margin:0;padding:0;background:${BRAND.pageBg};">
  ${preheader}
  <!-- outer wrapper -->
  <table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="background:${BRAND.pageBg};">
    <tr>
      <td align="center" style="padding:32px 12px;">
        <!-- card -->
        <table role="presentation" width="560" cellpadding="0" cellspacing="0" border="0"
               style="max-width:560px;width:100%;background:${BRAND.cardBg};border:1px solid ${BRAND.border};border-radius:12px;overflow:hidden;">
          <!-- header: logo band -->
          <tr>
            <td align="center" style="background:#0D1524;padding:28px 32px 22px;border-bottom:1px solid ${BRAND.border};">
              <a href="${APP_URL}" target="_blank" style="text-decoration:none;">
                <img src="${LOGO_URL}" alt="Predict-A-Trade" width="264" height="66"
                     style="display:block;border:0;outline:none;max-width:264px;height:auto;" />
              </a>
            </td>
          </tr>
          <!-- title -->
          <tr>
            <td style="padding:28px 32px 0;">
              <h1 style="margin:0;color:${BRAND.text};font-family:Inter,-apple-system,'Segoe UI',Roboto,sans-serif;font-size:20px;font-weight:600;line-height:1.3;">
                ${input.title}
              </h1>
            </td>
          </tr>
          <!-- body -->
          <tr>
            <td style="padding:12px 32px 8px;">
              <div style="color:${BRAND.textSecondary};font-family:Inter,-apple-system,'Segoe UI',Roboto,sans-serif;font-size:14px;line-height:1.65;">
                ${input.bodyHtml}
              </div>
            </td>
          </tr>
          <!-- footer -->
          <tr>
            <td style="padding:20px 32px 0;">
              <hr style="border:none;border-top:1px solid ${BRAND.border};margin:0 0 16px;" />
              ${input.footerNote ? `<p style="margin:0 0 6px;color:${BRAND.textMuted};font-size:11px;line-height:1.5;">${input.footerNote}</p>` : ''}
              <p style="margin:0;color:${BRAND.textMuted};font-size:11px;line-height:1.6;">
                <a href="${APP_URL}" style="color:${BRAND.textSecondary};text-decoration:none;">${APP_URL.replace('https://', '')}</a>
                · support@predictatrade.com
              </p>
              <p style="margin:6px 0 0;color:${BRAND.textMuted};font-size:11px;line-height:1.6;">
                © ${CURRENT_YEAR} Predict-A-Trade. All rights reserved.<br />
                Predict-A-Trade is a market analysis platform; nothing in this email is financial advice.
              </p>
              ${unsubscribeBlock}
            </td>
          </tr>
          <!-- spacer bottom -->
          <tr><td style="padding:20px;">&nbsp;</td></tr>
        </table>
        <!-- card end -->
      </td>
    </tr>
  </table>
</body>
</html>`;
}