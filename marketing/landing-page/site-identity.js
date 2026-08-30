(() => {
  const update = () => {
    document.querySelectorAll('[href="mailto:hello@predict-a-trade.com"]').forEach(link => { link.href = 'mailto:predictatrade@gmail.com'; });
    const walker = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT);
    const nodes = [];
    while (walker.nextNode()) nodes.push(walker.currentNode);
    nodes.forEach(node => {
      node.nodeValue = node.nodeValue.replaceAll('hello@predict-a-trade.com', 'predictatrade@gmail.com').replaceAll('Simha Online', 'Simha FinTech').replaceAll('PAT / ', '').replaceAll('Forward with context', 'Make decisions with context');
    });
    document.querySelectorAll('[href="https://simhaonline.com"]').forEach(link => { link.href = 'https://predictatrade.com/'; });
    document.querySelectorAll('.hero-visual img').forEach(image => {
      image.src = '/media/predict-a-trade-static-hero.webp';
      image.alt = 'Abstract gold-market signal and analytical context';
    });
    document.querySelectorAll('.module-strip').forEach(node => node.remove());
    const footerNav = document.querySelector('.footer-nav');
    if (footerNav && !footerNav.querySelector('a[href="/features/"]')) {
      const link = document.createElement('a');
      link.href = '/features/';
      link.textContent = 'Capabilities';
      footerNav.appendChild(link);
    }
    if (footerNav && !footerNav.parentNode.querySelector('.footer-socials')) {
      const socials = document.createElement('nav');
      socials.className = 'footer-socials';
      socials.setAttribute('aria-label', 'Social media');
      socials.innerHTML = '<a href="https://www.instagram.com/predictatrade" target="_blank" rel="noopener noreferrer" aria-label="Predict-A-Trade on Instagram"><svg viewBox="0 0 24 24" aria-hidden="true"><rect x="3" y="3" width="18" height="18" rx="5"></rect><circle cx="12" cy="12" r="4"></circle><circle cx="17.5" cy="6.5" r="1"></circle></svg></a><a href="https://www.facebook.com/predictatrade" target="_blank" rel="noopener noreferrer" aria-label="Predict-A-Trade on Facebook"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M14 21v-8h2.7l.4-3H14V8.1c0-.9.3-1.6 1.7-1.6h1.8V3.8c-.3 0-1.3-.1-2.4-.1-2.5 0-4.2 1.5-4.2 4.3V10H8v3h2.9v8H14Z"></path></svg></a><a href="https://www.google.com/search?q=%40predictatrade" target="_blank" rel="noopener noreferrer" aria-label="Predict-A-Trade on Google"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M21.6 12.2c0-.7-.1-1.3-.2-1.9H12v3.6h5.4a4.6 4.6 0 0 1-2 3v2.5h3.2c1.9-1.8 3-4.4 3-7.2Z"></path><path d="M12 22c2.7 0 5-.9 6.6-2.5l-3.2-2.5c-.9.6-2 .9-3.4.9-2.6 0-4.8-1.8-5.6-4.2H3.1v2.6A10 10 0 0 0 12 22Z"></path><path d="M6.4 13.7a6 6 0 0 1 0-3.4V7.7H3.1a10 10 0 0 0 0 8.6l3.3-2.6Z"></path><path d="M12 6.1c1.5 0 2.8.5 3.8 1.5l2.8-2.8C17 3.2 14.7 2 12 2a10 10 0 0 0-8.9 5.7l3.3 2.6C7.2 7.9 9.4 6.1 12 6.1Z"></path></svg></a><a href="https://x.com/predictatrade" target="_blank" rel="noopener noreferrer" aria-label="Predict-A-Trade on X / Twitter"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M18.9 2H22l-6.8 7.8L23.2 22h-6.3l-4.9-6.4L6.4 22H3.3l7.3-8.4L2.8 2h6.4l4.4 5.8L18.9 2Zm-1.1 17.5h1.7L8.3 4.4H6.5l11.3 15.1Z"></path></svg></a>';
      footerNav.parentNode.insertBefore(socials, footerNav.nextSibling);
    } else if (footerNav && footerNav.parentNode.querySelector('.footer-socials')) {
      const socials = footerNav.parentNode.querySelector('.footer-socials');
      if (socials.previousElementSibling !== footerNav) footerNav.parentNode.insertBefore(socials, footerNav.nextSibling);
    }
    const menuLinks = document.querySelector('.menu-panel-links');
    const menuPanel = document.querySelector('.menu-panel');
    if (menuPanel) {
      menuPanel.classList.add('editorial-menu');
      if (!menuPanel.dataset.editorialGuard) {
        menuPanel.dataset.editorialGuard = 'true';
        const keepEditorialMenu = new MutationObserver(() => {
          if (!menuPanel.classList.contains('editorial-menu')) menuPanel.classList.add('editorial-menu');
        });
        keepEditorialMenu.observe(menuPanel, {attributes: true, attributeFilter: ['class']});
      }
    }
    if (menuLinks && !menuLinks.dataset.editorialMenu) {
      menuLinks.dataset.editorialMenu = 'true';
      const dashboard = menuLinks.querySelector('a[href^="https://live.predictatrade.com"]');
      const login = menuLinks.querySelector('a[href^="https://platform.predictatrade.com"]');
      if (dashboard) {
        dashboard.classList.add('menu-dashboard');
        const label = document.createElement('div');
        label.className = 'menu-product-label';
        label.textContent = 'Product';
        menuLinks.insertBefore(label, dashboard);
      }
      [...menuLinks.querySelectorAll(':scope > a')].slice(0, 4).forEach((item, index) => {
        const number = document.createElement('span');
        number.className = 'menu-index';
        number.textContent = String(index + 1).padStart(2, '0');
        item.insertBefore(number, item.firstChild);
      });
      if (login) {
        login.textContent = 'Sign in';
        login.classList.add('menu-sign-in');
        const create = login.cloneNode(true);
        create.textContent = 'Create account';
        create.classList.remove('menu-sign-in');
        create.classList.add('menu-create-account');
        const accountActions = document.createElement('div');
        accountActions.className = 'menu-account-actions';
        login.parentNode.insertBefore(accountActions, login);
        accountActions.append(login, create);
      }
    }
    const capabilities = document.querySelector('#public-capabilities');
    const siteFooter = document.querySelector('.site-footer');
    if (capabilities && siteFooter && siteFooter.parentNode !== capabilities.parentNode) {
      siteFooter.parentNode.insertBefore(capabilities, siteFooter);
    }
    if (capabilities && siteFooter) capabilities.classList.add('is-ready');
    const newsletter = document.querySelector('#newsletter-signup');
    if (newsletter && siteFooter && siteFooter.parentNode !== newsletter.parentNode) {
      siteFooter.parentNode.insertBefore(newsletter, siteFooter);
    }
    if (newsletter && siteFooter) newsletter.classList.add('is-ready');
    const newsletterForm = document.querySelector('#newsletter-form');
    if (newsletterForm && !newsletterForm.dataset.bound) {
      newsletterForm.dataset.bound = 'true';
      newsletterForm.addEventListener('submit', event => {
        event.preventDefault();
        const email = document.querySelector('#newsletter-email').value.trim();
        const status = document.querySelector('#newsletter-status');
        status.textContent = 'Sending your request…';
        fetch('/api/contact.php', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({kind: 'newsletter', email, consent: true})})
          .then(response => response.json().then(result => ({ok: response.ok && result.ok, message: result.message})))
          .then(result => { status.textContent = result.ok ? result.message : result.message || 'Unable to send your request.'; if (result.ok) newsletterForm.reset(); })
          .catch(() => { status.textContent = 'Unable to send your request. Please email predictatrade@gmail.com.'; });
      });
    }
    const contactPanel = document.querySelector('.contact-panel');
    if (contactPanel && !contactPanel.querySelector('.contact-form')) {
      const oldButton = contactPanel.querySelector('a[href^="mailto:"]');
      if (oldButton) oldButton.remove();
      const form = document.createElement('form');
      form.className = 'contact-form';
      form.innerHTML = '<input name="name" type="text" placeholder="Your name" autocomplete="name" required><input name="email" type="email" placeholder="Your email" autocomplete="email" required><textarea name="message" placeholder="How can we help?" rows="4" required></textarea><label><input name="consent" type="checkbox" required> I agree to be contacted about this message.</label><input name="website" type="text" tabindex="-1" autocomplete="off" aria-hidden="true" class="contact-honeypot"><button type="submit">Send message</button><p class="contact-status" role="status" aria-live="polite"></p>';
      form.addEventListener('submit', event => {
        event.preventDefault();
        const formData = new FormData(form);
        const status = form.querySelector('.contact-status');
        status.textContent = 'Sending your message…';
        fetch('/api/contact.php', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({kind: 'contact', name: formData.get('name'), email: formData.get('email'), message: formData.get('message'), consent: formData.get('consent') === 'on', website: formData.get('website')})})
          .then(response => response.json().then(result => ({ok: response.ok && result.ok, message: result.message})))
          .then(result => { status.textContent = result.ok ? result.message : result.message || 'Unable to send your message.'; if (result.ok) form.reset(); })
          .catch(() => { status.textContent = 'Unable to send your message. Please email predictatrade@gmail.com.'; });
      });
      contactPanel.appendChild(form);
    }
    const footerBrand = document.querySelector('.footer-brand');
    if (footerBrand && !footerBrand.dataset.rebranded) {
      footerBrand.dataset.rebranded = 'true';
      footerBrand.replaceChildren();
      const lockup = document.createElement('img');
      lockup.src = '/media/predict-a-trade_primary-vertical.png';
      lockup.alt = 'Predict-A-Trade';
      lockup.className = 'footer-brand-lockup';
      footerBrand.append(lockup);
    }
    const scrollTop = document.querySelector('#scroll-top');
    if (scrollTop && !scrollTop.dataset.bound) {
      scrollTop.dataset.bound = 'true';
      const toggleScrollTop = () => scrollTop.classList.toggle('is-visible', window.scrollY > 420);
      window.addEventListener('scroll', toggleScrollTop, {passive: true});
      scrollTop.addEventListener('click', () => {
        window.scrollTo({top: 0, behavior: window.matchMedia('(prefers-reduced-motion: reduce)').matches ? 'auto' : 'smooth'});
      });
      toggleScrollTop();
    }
  };
  new MutationObserver(update).observe(document.documentElement, {childList: true, subtree: true});
  update();
})();
