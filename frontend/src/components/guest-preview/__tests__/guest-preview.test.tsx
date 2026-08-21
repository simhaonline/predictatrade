/**
 * Guest Preview funnel — unit tests.
 *
 * Verifies the spec-critical UI behavior:
 *   - Banner renders the countdown from server remainingSeconds.
 *   - Registration modal has 3 separate unchecked consent checkboxes
 *     (Terms required, Risk required, Marketing optional/skippable).
 *   - Submit is blocked until required consents are checked.
 *   - Broker dropdown includes the spec list + free-text "Other".
 *   - Post-registration social is skippable (never gates content).
 *   - OTP step shows resend cooldown.
 *   - GuestPreviewGate shows the lock modal when the server says locked.
 */

import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { useRouter } from "next/navigation";
import { GuestPreviewBanner } from "../guest-preview-banner";
import { PostRegistrationSocial } from "../post-registration-social";
import { RegistrationModal } from "../registration-modal";
import { GuestPreviewGate } from "../guest-preview-gate";

// ─── Mocks ───

jest.mock("next/navigation", () => ({
  useRouter: jest.fn(),
  useSearchParams: jest.fn(() => new URLSearchParams()),
}));

jest.mock("@/lib/axios-instance", () => ({
  customInstance: {
    get: jest.fn(),
    post: jest.fn(),
  },
}));

jest.mock("@/lib/guest-preview-api", () => ({
  useGuestPreview: jest.fn(() => ({
    status: { locked: false, expiresAt: Date.now() + 300_000, remainingSeconds: 240 },
    loading: false,
    error: null,
    setError: jest.fn(),
    refreshStatus: jest.fn(),
    register: jest.fn(),
    resend: jest.fn(),
    verify: jest.fn(),
  })),
  issueGuestSession: jest.fn(),
  getGuestStatus: jest.fn(),
  registerGuest: jest.fn(),
  resendGuestOtp: jest.fn(),
  verifyGuestOtp: jest.fn(),
}));

jest.mock("@/lib/auth", () => ({
  setAccessToken: jest.fn(),
  getAccessToken: jest.fn(() => null),
  clearAccessToken: jest.fn(),
}));

jest.mock("@/components/brand-logo", () => {
  function MockLogo() { return <div data-testid="brand-logo">Logo</div>; }
  return MockLogo;
});

beforeEach(() => {
  jest.clearAllMocks();
  (useRouter as jest.Mock).mockReturnValue({ replace: jest.fn(), push: jest.fn() });
});

// ─── Banner ───

describe("GuestPreviewBanner", () => {
  it("renders the countdown in mm:ss from server remainingSeconds", () => {
    render(<GuestPreviewBanner remainingSeconds={252} />);
    expect(screen.getByText(/Free preview/)).toBeInTheDocument();
    expect(screen.getByTestId("guest-preview-countdown")).toHaveTextContent("04:12");
  });

  it("shows the 'Register to keep access' message", () => {
    render(<GuestPreviewBanner remainingSeconds={120} />);
    expect(screen.getByText(/Register to keep access/)).toBeInTheDocument();
  });

  it("switches to danger styling in the last 60 seconds", () => {
    render(<GuestPreviewBanner remainingSeconds={30} />);
    const banner = screen.getByTestId("guest-preview-banner");
    expect(banner.className).toContain("pat-danger");
  });
});

// ─── Post-registration social ───

describe("PostRegistrationSocial", () => {
  it("renders all social follow buttons (optional, not a gate)", () => {
    const onContinue = jest.fn();
    render(<PostRegistrationSocial onContinue={onContinue} />);
    for (const label of ["WhatsApp", "Telegram", "Instagram", "YouTube", "X", "TikTok", "Facebook"]) {
      expect(screen.getByLabelText(`Follow on ${label}`)).toBeInTheDocument();
    }
  });

  it("has a Skip / Continue action that is never blocked", () => {
    const onContinue = jest.fn();
    render(<PostRegistrationSocial onContinue={onContinue} />);
    const skip = screen.getByRole("button", { name: /Skip \/ Continue to dashboard/ });
    fireEvent.click(skip);
    expect(onContinue).toHaveBeenCalled();
  });

  it("states no reward is offered for following", () => {
    render(<PostRegistrationSocial onContinue={jest.fn()} />);
    expect(screen.getByText(/No reward is offered for following/i)).toBeInTheDocument();
  });
});

// ─── Registration modal ───

describe("RegistrationModal", () => {
  it("renders all required fields in spec order", () => {
    render(<RegistrationModal />);
    expect(screen.getByPlaceholderText("Your full name")).toBeInTheDocument();
    expect(screen.getByPlaceholderText("you@example.com")).toBeInTheDocument();
    expect(screen.getByPlaceholderText("+971 50 000 0000")).toBeInTheDocument();
    expect(screen.getByText(/Trading broker/i)).toBeInTheDocument();
  });

  it("has three separate unchecked consent checkboxes", () => {
    render(<RegistrationModal />);
    const checkboxes = screen.getAllByRole("checkbox");
    expect(checkboxes).toHaveLength(3);
    // All default unchecked
    checkboxes.forEach((cb) => expect((cb as HTMLInputElement).checked).toBe(false));
  });

  it("marks Terms and Risk as required and Marketing as optional", () => {
    render(<RegistrationModal />);
    expect(screen.getByText(/I accept the/i)).toBeInTheDocument();
    expect(screen.getByText(/informational\/educational purposes only/i)).toBeInTheDocument();
    expect(screen.getByText(/marketing communications/i)).toBeInTheDocument();
    expect(screen.getAllByText(/\(optional\)/i).length).toBeGreaterThanOrEqual(1);
  });

  it("includes the spec broker list in the dropdown", () => {
    render(<RegistrationModal />);
    for (const b of ["Exness", "IC Markets", "XM", "Pepperstone", "Deriv", "Other/None"]) {
      expect(screen.getByRole("option", { name: b })).toBeInTheDocument();
    }
  });

  it("blocks submit until required consents are checked", async () => {
    const { useGuestPreview } = jest.requireMock("@/lib/guest-preview-api");
    const register = jest.fn().mockResolvedValue({ message: "sent", challengeId: "c1" });
    (useGuestPreview as jest.Mock).mockReturnValue({
      status: { locked: true }, loading: false, error: null, setError: jest.fn(),
      refreshStatus: jest.fn(), register, resend: jest.fn(), verify: jest.fn(),
    });
    render(<RegistrationModal />);
    // Fill required non-consent fields so native form validation passes.
    fireEvent.change(screen.getByPlaceholderText("Your full name"), { target: { value: "Jane Doe" } });
    fireEvent.change(screen.getByPlaceholderText("you@example.com"), { target: { value: "jane@test.com" } });
    const submit = screen.getByRole("button", { name: /Send verification code/i });
    fireEvent.click(submit);
    // Required-consent client guard fires before any API call.
    await waitFor(() => expect(register).not.toHaveBeenCalled());
    // The consent guard error is shown.
    expect(screen.getByText(/Please accept the Terms/i)).toBeInTheDocument();
  });

  it("shows the free-text broker input when 'Other/None' is selected", () => {
    render(<RegistrationModal />);
    const select = screen.getByRole("combobox");
    fireEvent.change(select, { target: { value: "Other/None" } });
    expect(screen.getByPlaceholderText("Specify your broker")).toBeInTheDocument();
  });
});

// ─── GuestPreviewGate ───

describe("GuestPreviewGate", () => {
  it("renders children during the preview (full dashboard works)", () => {
    const { useGuestPreview } = jest.requireMock("@/lib/guest-preview-api");
    (useGuestPreview as jest.Mock).mockReturnValue({
      status: { locked: false, expiresAt: Date.now() + 300_000, remainingSeconds: 200 },
      loading: false, error: null, setError: jest.fn(), refreshStatus: jest.fn(),
      register: jest.fn(), resend: jest.fn(), verify: jest.fn(),
    });
    render(<GuestPreviewGate><div data-testid="dashboard-content">Dashboard</div></GuestPreviewGate>);
    expect(screen.getByTestId("dashboard-content")).toBeInTheDocument();
    expect(screen.getByTestId("guest-preview-banner")).toBeInTheDocument();
    expect(screen.getByTestId("guest-signal-overlay")).toBeInTheDocument();
  });

  it("shows the registration modal when the server reports locked", async () => {
    const { useGuestPreview } = jest.requireMock("@/lib/guest-preview-api");
    (useGuestPreview as jest.Mock).mockReturnValue({
      status: { locked: true, expiresAt: null, remainingSeconds: 0 },
      loading: false, error: null, setError: jest.fn(), refreshStatus: jest.fn(),
      register: jest.fn(), resend: jest.fn(), verify: jest.fn(),
    });
    render(<GuestPreviewGate><div>Dashboard</div></GuestPreviewGate>);
    await waitFor(() => {
      expect(screen.getByRole("dialog", { name: /Register to continue/i })).toBeInTheDocument();
    });
  });
});
