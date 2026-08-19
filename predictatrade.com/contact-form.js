(() => {
  const mount = () => {
    const mailLink = document.querySelector('.contact-panel a[href^="mailto:"]');
    if (!mailLink || document.querySelector('.contact-form')) return;
    const form = document.createElement('form');
    form.className = 'contact-form';
    form.innerHTML = `<label>Name<input name="name" autocomplete="name" required maxlength="120"></label><label>Email<input name="email" type="email" autocomplete="email" required maxlength="254"></label><label>Message<textarea name="message" required maxlength="5000" rows="5"></textarea></label><button class="button-primary" type="submit">Send message</button><p class="contact-form-status" role="status" aria-live="polite"></p>`;
    mailLink.replaceWith(form);
    form.addEventListener('submit', async (event) => {
      event.preventDefault();
      const button = form.querySelector('button');
      const status = form.querySelector('.contact-form-status');
      button.disabled = true;
      status.textContent = 'Sending…';
      try {
        const response = await fetch('/api/contact', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(Object.fromEntries(new FormData(form))) });
        if (!response.ok) throw new Error('Request failed');
        form.reset();
        status.textContent = 'Thank you — your message has been sent.';
      } catch {
        status.textContent = 'The contact service is temporarily unavailable. Please try again shortly.';
      } finally {
        button.disabled = false;
      }
    });
  };
  window.addEventListener('load', mount);
})();
