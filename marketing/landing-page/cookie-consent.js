(() => {
  const key = 'pat-cookie-preferences-v1';
  if (localStorage.getItem(key)) return;
  const banner = document.createElement('aside');
  banner.className = 'cookie-banner';
  banner.setAttribute('aria-labelledby', 'cookie-banner-title');
  banner.innerHTML = '<div><strong id="cookie-banner-title">Cookie preferences</strong><p>This website uses necessary storage and does not currently load analytics or marketing scripts. You can accept, reject optional categories, or manage your choices.</p></div><div class="cookie-banner-actions"><button type="button" data-cookie-choice="reject">Reject Optional</button><button type="button" data-cookie-choice="settings">Cookie Settings</button><button type="button" data-cookie-choice="accept">Accept All</button></div>';
  document.body.appendChild(banner);
  window.addEventListener('pat-cookie-preferences-saved', () => banner.remove(), { once: true });
  const save = (optional) => { localStorage.setItem(key, JSON.stringify({ necessary: true, analytics: optional, preferences: optional, marketing: optional, updatedAt: new Date().toISOString() })); banner.remove(); };
  banner.querySelector('[data-cookie-choice="reject"]').addEventListener('click', () => save(false));
  banner.querySelector('[data-cookie-choice="accept"]').addEventListener('click', () => save(true));
  banner.querySelector('[data-cookie-choice="settings"]').addEventListener('click', () => { window.location.href = '/legal/cookie-settings/'; });
})();
