(() => {
  "use strict";

  const emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
  const eyeOpen = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" aria-hidden="true"><path d="M2.5 12s3.5-6 9.5-6 9.5 6 9.5 6-3.5 6-9.5 6S2.5 12 2.5 12Z"/><circle cx="12" cy="12" r="2.5"/></svg>';
  const eyeClosed = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" aria-hidden="true"><path d="m3 3 18 18M10.7 6.1A10.4 10.4 0 0 1 12 6c6 0 9.5 6 9.5 6a15.8 15.8 0 0 1-2.2 2.9M6.6 6.6C4 8.3 2.5 12 2.5 12s3.5 6 9.5 6c1.3 0 2.5-.3 3.5-.7M9.9 9.9a3 3 0 0 0 4.2 4.2"/></svg>';

  function setError(input, message) {
    if (!input) return false;
    const error = document.getElementById(`${input.id}-error`);
    input.setAttribute("aria-invalid", message ? "true" : "false");
    if (error) error.textContent = message;
    return !message;
  }

  function clearErrorOnInput(input) {
    input?.addEventListener("input", () => {
      if (input.getAttribute("aria-invalid") === "true") setError(input, "");
    });
  }

  document.querySelectorAll("input, select").forEach(clearErrorOnInput);

  document.querySelectorAll("[data-toggle-password]").forEach((button) => {
    button.addEventListener("click", () => {
      const input = document.getElementById(button.dataset.togglePassword);
      if (!input) return;
      const reveal = input.type === "password";
      input.type = reveal ? "text" : "password";
      button.setAttribute("aria-label", reveal ? "Hide password" : "Show password");
      button.innerHTML = reveal ? eyeClosed : eyeOpen;
      input.focus({ preventScroll: true });
    });
  });

  function showToast() {
    const toast = document.getElementById("toast");
    if (!toast) return;
    toast.classList.add("visible");
    window.clearTimeout(showToast.timer);
    showToast.timer = window.setTimeout(() => toast.classList.remove("visible"), 4200);
  }

  document.querySelectorAll("[data-demo-link]").forEach((link) => {
    link.addEventListener("click", (event) => {
      event.preventDefault();
      const toast = document.getElementById("toast");
      if (toast) {
        toast.querySelector("strong").textContent = "Password recovery";
        toast.querySelector("span").textContent = "Connect this link to your existing password-reset route.";
      }
      showToast();
    });
  });

  const loginForm = document.getElementById("login-form");
  if (loginForm) {
    loginForm.addEventListener("submit", (event) => {
      event.preventDefault();
      const email = document.getElementById("login-email");
      const password = document.getElementById("login-password");
      let valid = true;

      if (!email.value.trim()) valid = setError(email, "Enter your email address.") && valid;
      else if (!emailPattern.test(email.value.trim())) valid = setError(email, "Enter a valid email address.") && valid;
      else setError(email, "");

      if (!password.value) valid = setError(password, "Enter your password.") && valid;
      else setError(password, "");

      if (!valid) {
        loginForm.querySelector('[aria-invalid="true"]')?.focus();
        return;
      }

      const submit = document.getElementById("login-submit");
      submit.disabled = true;
      submit.classList.add("loading");
      submit.setAttribute("aria-busy", "true");
      submit.querySelector(".button-label").textContent = "Signing in";

      window.setTimeout(() => {
        submit.disabled = false;
        submit.classList.remove("loading");
        submit.removeAttribute("aria-busy");
        submit.querySelector(".button-label").textContent = "Sign in securely";
        const toast = document.getElementById("toast");
        if (toast) {
          toast.querySelector("strong").textContent = "Interface ready";
          toast.querySelector("span").textContent = "Validation passed. Connect this form to your authentication API.";
        }
        showToast();
      }, 900);
    });
  }

  const signupForm = document.getElementById("signup-form");
  if (!signupForm) return;

  const password = document.getElementById("signup-password");
  const confirmPassword = document.getElementById("confirm-password");
  const ruleChecks = {
    length: (value) => value.length >= 8,
    upper: (value) => /[A-Z]/.test(value),
    number: (value) => /\d/.test(value),
    special: (value) => /[^A-Za-z0-9]/.test(value)
  };

  function passwordIsStrong(value) {
    return Object.values(ruleChecks).every((check) => check(value));
  }

  password.addEventListener("input", () => {
    Object.entries(ruleChecks).forEach(([name, check]) => {
      document.querySelector(`[data-rule="${name}"]`)?.classList.toggle("met", check(password.value));
    });
    if (confirmPassword.value && confirmPassword.value === password.value) setError(confirmPassword, "");
  });

  function validateStepOne() {
    const firstName = document.getElementById("first-name");
    const lastName = document.getElementById("last-name");
    const email = document.getElementById("signup-email");
    let valid = true;

    if (!firstName.value.trim()) { setError(firstName, "Enter your first name."); valid = false; } else setError(firstName, "");
    if (!lastName.value.trim()) { setError(lastName, "Enter your last name."); valid = false; } else setError(lastName, "");

    if (!email.value.trim()) { setError(email, "Enter your email address."); valid = false; }
    else if (!emailPattern.test(email.value.trim())) { setError(email, "Enter a valid email address."); valid = false; }
    else setError(email, "");

    if (!password.value) { setError(password, "Create a password."); valid = false; }
    else if (!passwordIsStrong(password.value)) { setError(password, "Meet all four password requirements."); valid = false; }
    else setError(password, "");

    if (!confirmPassword.value) { setError(confirmPassword, "Repeat your password."); valid = false; }
    else if (confirmPassword.value !== password.value) { setError(confirmPassword, "The passwords do not match."); valid = false; }
    else setError(confirmPassword, "");

    if (!valid) document.getElementById("signup-step-one").querySelector('[aria-invalid="true"]')?.focus();
    return valid;
  }

  const stepOne = document.getElementById("signup-step-one");
  const stepTwo = document.getElementById("signup-step-two");
  const progressOne = document.getElementById("progress-one");
  const progressTwo = document.getElementById("progress-two");
  const stepMeta = document.getElementById("step-meta");
  const signupLede = document.getElementById("signup-lede");

  function goToStepTwo() {
    stepOne.classList.remove("active");
    stepTwo.classList.add("active");
    progressOne.classList.remove("active");
    progressOne.classList.add("complete");
    progressTwo.classList.add("active");
    stepMeta.textContent = "02 / 02";
    signupLede.textContent = "Confirm your preferences and review the account terms.";
    document.getElementById("step-two-title").focus({ preventScroll: true });
    document.querySelector(".auth-wrap").scrollIntoView({ behavior: "smooth", block: "start" });
  }

  function goToStepOne() {
    stepTwo.classList.remove("active");
    stepOne.classList.add("active");
    progressTwo.classList.remove("active");
    progressOne.classList.remove("complete");
    progressOne.classList.add("active");
    stepMeta.textContent = "01 / 02";
    signupLede.textContent = "Set up your account details. It only takes a moment.";
    document.getElementById("step-one-title").focus({ preventScroll: true });
  }

  document.getElementById("next-step").addEventListener("click", () => {
    if (validateStepOne()) goToStepTwo();
  });
  document.getElementById("previous-step").addEventListener("click", goToStepOne);

  signupForm.addEventListener("submit", (event) => {
    event.preventDefault();
    const country = document.getElementById("country");
    const terms = document.getElementById("terms-consent");
    const termsError = document.getElementById("terms-error");
    let valid = true;

    if (!country.value) { setError(country, "Choose your country or region."); valid = false; }
    else setError(country, "");

    if (!terms.checked) {
      terms.setAttribute("aria-invalid", "true");
      termsError.textContent = "Accept the Terms of Service and Privacy Policy to continue.";
      valid = false;
    } else {
      terms.setAttribute("aria-invalid", "false");
      termsError.textContent = "";
    }

    if (!valid) {
      if (!country.value) country.focus(); else terms.focus();
      return;
    }

    const submit = document.getElementById("signup-submit");
    submit.disabled = true;
    submit.classList.add("loading");
    submit.setAttribute("aria-busy", "true");
    submit.querySelector(".button-label").textContent = "Creating account";

    window.setTimeout(() => {
      signupForm.style.display = "none";
      document.querySelector(".progress").style.display = "none";
      document.getElementById("success-state").classList.add("visible");
      document.getElementById("signup-heading").innerHTML = "You’re <em>all set.</em>";
      signupLede.textContent = `A polished registration flow for ${document.getElementById("signup-email").value.trim()}.`;
      document.getElementById("success-state").querySelector("h2").focus?.();
    }, 950);
  });

  document.getElementById("terms-consent").addEventListener("change", (event) => {
    if (event.target.checked) {
      event.target.setAttribute("aria-invalid", "false");
      document.getElementById("terms-error").textContent = "";
    }
  });
})();
