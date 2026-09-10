/**
 * Branded email template tests — every platform email must render inside the
 * PAT shell: logo header, dark brand tokens, footer with copyright, and (for
 * marketing) the unsubscribe block.
 */
import { renderBrandedEmail, textFooter } from './email-template';

describe('renderBrandedEmail', () => {
  const base = {
    title: 'Test title',
    bodyHtml: '<p>Hello world</p>',
    bodyText: 'Hello world',
  };

  it('includes the logo image in a header band', () => {
    const html = renderBrandedEmail(base);
    expect(html).toContain('predict-a-trade_horizontal_white.png');
    expect(html).toContain('alt="Predict-A-Trade"');
  });

  it('uses PAT dark brand tokens', () => {
    const html = renderBrandedEmail(base);
    expect(html).toContain('#0A111D'); // page bg
    expect(html).toContain('#101A2C'); // card bg
    expect(html).toContain('#F2F5F9'); // primary text
    // primary blue appears in the footer unsubscribe link
    const withUnsub = renderBrandedEmail({ ...base, unsubscribeUrl: 'https://platform.predictatrade.com/unsubscribe' });
    expect(withUnsub).toContain('#2362EA');
  });

  it('renders the title and body content', () => {
    const html = renderBrandedEmail(base);
    expect(html).toContain('Test title');
    expect(html).toContain('Hello world');
  });

  it('always includes copyright footer with current year + disclaimer', () => {
    const html = renderBrandedEmail(base);
    const year = new Date().getUTCFullYear();
    expect(html).toContain(`© ${year} Predict-A-Trade. All rights reserved.`);
    expect(html).toContain('nothing in this email is financial advice');
  });

  it('includes support + platform links in footer', () => {
    const html = renderBrandedEmail(base);
    expect(html).toContain('support@predictatrade.com');
    expect(html).toContain('platform.predictatrade.com');
  });

  it('adds unsubscribe block only when unsubscribeUrl is provided', () => {
    const without = renderBrandedEmail(base);
    expect(without).not.toContain('Unsubscribe</a>');
    const withUnsub = renderBrandedEmail({ ...base, unsubscribeUrl: 'https://platform.predictatrade.com/unsubscribe?token=x' });
    expect(withUnsub).toContain('Unsubscribe</a>');
  });

  it('includes hidden preheader for inbox preview when provided', () => {
    const html = renderBrandedEmail({ ...base, preheader: 'You have 1 new signal' });
    expect(html).toContain('display:none');
    expect(html).toContain('You have 1 new signal');
  });

  it('renders footerNote when provided', () => {
    const html = renderBrandedEmail({ ...base, footerNote: 'Because you are a client.' });
    expect(html).toContain('Because you are a client.');
  });

  it('textFooter carries copyright, platform and disclaimer lines', () => {
    const t = textFooter('https://platform.predictatrade.com/unsubscribe');
    expect(t).toContain('© ' + new Date().getUTCFullYear() + ' Predict-A-Trade');
    expect(t).toContain('support@predictatrade.com');
    expect(t).toContain('Unsubscribe from marketing emails');
  });
});