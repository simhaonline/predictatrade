(() => {
  const key = 'pat-cookie-preferences-v1';
  const form = document.querySelector('[data-cookie-form]');
  if (!form) return;
  const status = form.querySelector('[data-cookie-status]');
  const stored = JSON.parse(localStorage.getItem(key) || '{}');
  for (const name of ['analytics', 'preferences', 'marketing']) {
    const input = form.elements[name];
    if (input) input.checked = stored[name] === true;
  }
  form.addEventListener('submit', (event) => {
    event.preventDefault();
    const preferences = {
      necessary: true,
      analytics: form.elements.analytics.checked,
      preferences: form.elements.preferences.checked,
      marketing: form.elements.marketing.checked,
      updatedAt: new Date().toISOString()
    };
    localStorage.setItem(key, JSON.stringify(preferences));
    window.dispatchEvent(new CustomEvent('pat-cookie-preferences-saved'));
    status.textContent = 'Your cookie preferences have been saved.';
  });
})();
