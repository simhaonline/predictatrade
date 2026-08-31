"use client";
import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import StatusBadge from "@/components/ui/status-badge";
import {
  IconDownload, IconBrandWindows, IconDeviceDesktop,
  IconClipboard, IconCheck, IconTerminal2, IconRefresh,
  IconFingerprint, IconShieldCheck, IconLicense,
} from "@tabler/icons-react";

interface DeviceActivation {
  id?: string;
  client_type: string;
  terminal_build?: string;
  ea_version?: string;
  broker_name?: string;
  broker_server?: string;
  mt_account_login?: string;
  installation_id?: string;
  fingerprint_hash?: string;
  activated_at?: string;
  terminal_connected?: boolean;
}

interface UserDevice {
  id: string;
  device_name: string;
  hostname: string;
  os: string;
  agent_version: string;
  status: string;
  security_state: string;
  registered_at: string;
  last_seen_at: string | null;
  installation_id: string;
  fingerprint_hash: string | null;
  license_key: string | null;
  license_status: string | null;
  client_type?: string;
  connection_status?: string;
  max_devices?: number;
  max_mt_accounts?: number;
  activations: DeviceActivation[] | null;
}

type InstallStep = "download" | "install" | "connect" | "mt4" | "mt5" | "verify";

export default function UserMtClientPage() {
  const [copiedKey, setCopiedKey] = useState(false);

  const { data: devices, isLoading } = useQuery<UserDevice[]>({
    queryKey: ["user-mt-devices"],
    queryFn: async () => (await customInstance.get("/licensing/devices")).data,
    refetchInterval: 5000,
  });

  // Per-user terminal status — derived from the user's OWN registered devices
  // NOT global agent status (which shows ALL agents on the server)
  const { data: agentsStatus, refetch: refetchAgents } = useQuery<{
    agents_connected: number;
    agents_online: boolean;
    snapshot_count: number;
    mt4_connected: number;
    mt5_connected: number;
    backend_reachable: boolean;
  }>({
    queryKey: ["user-mt-agents"],
    queryFn: async () => {
      // Terminal link state lives on each device's `activations` array, keyed by
      // client_type (MT4/MT5) with `terminal_connected` reflecting the live link —
      // NOT on the device object itself.
      const userDevices = devices || [];
      const onlineDevices = userDevices.filter((d) => d.status === "ONLINE" || d.connection_status === "ONLINE");
      const acts = userDevices.flatMap((d) => (d.activations || []) as DeviceActivation[]);
      const mt4Connected = acts.filter((a) => a.client_type === "MT4" && a.terminal_connected).length;
      const mt5Connected = acts.filter((a) => a.client_type === "MT5" && a.terminal_connected).length;
      return {
        agents_connected: onlineDevices.length,
        agents_online: onlineDevices.length > 0,
        snapshot_count: 0,
        mt4_connected: mt4Connected,
        mt5_connected: mt5Connected,
        backend_reachable: true,
      };
    },
    refetchInterval: 5000,
  });
  const { data: licenses } = useQuery<{ license_key: string; status: string; max_devices: number; max_mt_accounts: number }[]>({
    queryKey: ["user-mt-licenses"],
    queryFn: async () => (await customInstance.get("/licensing/licenses")).data,
  });

  const licenseKey = licenses?.[0]?.license_key || "—";
  const copyKey = () => {
    if (licenseKey !== "—") {
      navigator.clipboard.writeText(licenseKey);
      setCopiedKey(true);
      setTimeout(() => setCopiedKey(false), 2000);
    }
  };

  const downloadFiles = [
    { name: "Windows Agent (Installer)", file: "https://downloads.predictatrade.com/windows-agent/install.ps1", desc: "Automated installer — run in PowerShell as Administrator", size: "9.2 MB", icon: IconBrandWindows, type: "exe", primary: true },
    { name: "MT5 Expert Advisor (Compiled)", file: "/downloads/Predict-A-Trade.ex5", desc: "Pre-compiled EA for MetaTrader 5 — ready to use, no compilation needed", size: "128 KB", icon: IconTerminal2, type: "ex5" },
    { name: "MT4 Expert Advisor (Compiled)", file: "/downloads/PredictATrade.ex4", desc: "Pre-compiled EA for MetaTrader 4 — ready to use, no compilation needed", size: "104 KB", icon: IconTerminal2, type: "ex4" },
  ];

  const installSteps: { id: InstallStep; label: string }[] = [
    { id: "download", label: "1. Download" },
    { id: "install", label: "2. Install Agent" },
    { id: "connect", label: "3. Enter License" },
    { id: "mt4", label: "4. Install MT4 EA" },
    { id: "mt5", label: "5. Install MT5 EA" },
    { id: "verify", label: "6. Verify" },
  ];

  // Full installation steps shown inline on the dashboard (no toggling required).
  const guideSteps: Record<InstallStep, { title: string; steps: string[] }> = {
    download: {
      title: "Install the Windows Agent",
      steps: [
        "Open PowerShell as Administrator (Right-click Start → Windows PowerShell (Admin)).",
        "Copy and paste this command: irm https://downloads.predictatrade.com/windows-agent/install.ps1 | iex",
        "Press Enter. The installer will download and install the agent automatically.",
        "If Windows Defender prompts, click 'Allow' — the agent is safe and verified.",
        "The agent installs as a Windows Service (pat-agent) and starts automatically.",
        "Download the MT5 (.ex5) or MT4 (.ex4) Expert Advisor from the buttons above.",
      ],
    },
    install: {
      title: "Verify the Windows Agent",
      steps: [
        "Open Task Manager → Services tab → look for 'pat-agent' (should be Running).",
        "Open your browser and go to http://127.0.0.1:9000 — should show agent status page.",
        "If the service is not running, open PowerShell as Admin and run:",
        "  irm https://downloads.predictatrade.com/windows-agent/install.ps1 | iex",
        "The agent automatically connects to the Predict-A-Trade signal engine.",
        "No license key needed in the agent — the EA handles license validation.",
      ],
    },
    connect: {
      title: "Copy Your License Key",
      steps: [
        "Copy your license key from the box above (click the copy icon).",
        "You will need this key when setting up the MetaTrader EA.",
        "Your license determines which strategies you can receive:",
        "  FREE: Standard Scalping only",
        "  STANDARD: Standard Scalping + Standard Swing",
        "  PRO: All scalping + swing strategies",
        "  ELITE: All 5 strategies including EQFE and Ultra Scalping",
        "Your license is tied to your account — strategy selection is automatic.",
      ],
    },
    mt4: {
      title: "Install MT4 Expert Advisor (Compiled .ex4)",
      steps: [
        "Download PredictATrade.ex4 from the button above (pre-compiled — no MetaEditor needed).",
        "Open MetaTrader 4 → File → Open Data Folder → MQL4 → Experts.",
        "Copy PredictATrade.ex4 into the Experts folder.",
        "In MT4, open Navigator (Ctrl+N) → Right-click 'Expert Advisors' → 'Refresh'.",
        "Drag 'PredictATrade' onto an XAUUSD chart.",
        "In the EA inputs, paste your License Key into the LicenseKey field.",
        "Check 'Allow live trading' → OK.",
        "Enable the 'AutoTrading' button at the top (should turn green).",
        "The EA will connect to the Windows Agent and start receiving signals.",
      ],
    },
    mt5: {
      title: "Install MT5 Expert Advisor (Compiled .ex5)",
      steps: [
        "Download Predict-A-Trade.ex5 from the button above (pre-compiled — no MetaEditor needed).",
        "Open MetaTrader 5 → File → Open Data Folder → MQL5 → Experts.",
        "Copy Predict-A-Trade.ex5 into the Experts folder.",
        "In MT5, open Navigator → Right-click 'Expert Advisors' → 'Refresh'.",
        "Drag 'Predict-A-Trade' onto an XAUUSD chart.",
        "In the EA inputs, paste your License Key into the LicenseKey field.",
        "Check 'Allow Algo Trading' → OK.",
        "Enable the 'Algo Trading' button at the top (should turn green).",
        "The EA will connect to the Windows Agent and start receiving signals.",
      ],
    },
    verify: {
      title: "Verify Your Connection",
      steps: [
        "Check http://127.0.0.1:9000 — the Client Agent dashboard should show your CLIENT (EA) connection and a live SIGNAL DELIVERY → EA status.",
        "The Windows Agent sends live market data (candles) to the engine — the client dashboard at http://127.0.0.1:9000 shows the CLIENT (EA) connection and live SIGNAL DELIVERY → EA status.",
        "In MT4/MT5, check the Experts tab — should show 'License validated — ACTIVE'.",
        "In MT4/MT5, check the Journal tab — should show 'License strategies from server: ...'",
        "Your terminal status should appear as 'Online' in the dashboard above.",
        "The EA will only execute strategies allowed by your license plan.",
        "Stop-loss is enforced by the server — trades without SL are automatically closed.",
        "To update: just re-run the PowerShell install command and replace the .ex4/.ex5 file.",
      ],
    },
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">MetaTrader Client</h1>
        <p className="text-sm text-pat-text-secondary mt-1">
          Download the Windows Agent and MQL Expert Advisors. Manage your registered devices and terminals.
        </p>
      </div>

      {/* License info */}
      <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-5">
        <div className="flex items-center justify-between mb-3">
          <h2 className="text-sm font-semibold text-pat-text-primary flex items-center gap-2">
            <IconLicense size={16} className="text-pat-success" /> Your License
          </h2>
          <StatusBadge status={licenses?.[0]?.status || "INACTIVE"} size="sm" />
        </div>
        <div className="flex items-center gap-2 mb-3">
          <div className="flex-1 rounded-lg border border-pat-border bg-pat-bg-surface-secondary/30 px-3 py-2 font-mono text-sm text-pat-text-primary">
            {licenseKey}
          </div>
          <button onClick={copyKey} className="flex items-center gap-1.5 px-3 py-2 rounded-lg border border-pat-border bg-pat-bg-surface-secondary text-xs text-pat-text-secondary hover:text-pat-text-primary transition-colors">
            {copiedKey ? <IconCheck size={14} className="text-pat-success" /> : <IconClipboard size={14} />}
            {copiedKey ? "Copied" : "Copy"}
          </button>
        </div>
        <div className="flex flex-wrap gap-3 text-xs">
          <span className="text-pat-text-muted">Max devices: <span className="text-pat-text-secondary">{licenses?.[0]?.max_devices || "—"}</span></span>
          <span className="text-pat-text-muted">Max MT accounts: <span className="text-pat-text-secondary">{licenses?.[0]?.max_mt_accounts || "—"}</span></span>
        </div>
      </div>

      {/* Client Agent — Connection & Signal Delivery (no server connection shown) */}
      <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-4">
        <div className="flex items-center justify-between flex-wrap gap-3 mb-3">
          <div className="flex items-center gap-2">
            <IconDeviceDesktop size={18} className="text-pat-info" />
            <h2 className="text-sm font-semibold text-pat-text-primary">Client Agent — Connection &amp; Signal Delivery</h2>
          </div>
          <button
            onClick={() => refetchAgents()}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-pat-border bg-pat-bg-surface-secondary text-xs text-pat-text-secondary hover:text-pat-text-primary transition-colors"
          >
            <IconRefresh size={14} /> Recheck
          </button>
        </div>

        <div className="flex items-center gap-2 mb-3 text-xs text-pat-text-muted">
          <span className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-lg text-[11px] font-medium ${
            (agentsStatus?.agents_connected ?? 0) > 0
              ? "bg-pat-success/10 text-pat-success border border-pat-success/20"
              : "bg-pat-danger/10 text-pat-danger border border-pat-danger/20"
          }`}>
            <span className={`inline-block h-1.5 w-1.5 rounded-full ${(agentsStatus?.agents_connected ?? 0) > 0 ? "bg-pat-success animate-pulse" : "bg-pat-danger"}`} />
            Client Agent: {(agentsStatus?.agents_connected ?? 0) > 0 ? "LIVE" : "OFFLINE"}
          </span>
          <span>
            {agentsStatus?.agents_connected ?? 0} client agent(s)
          </span>
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <TerminalLiveness
            label="MT4 Terminal"
            connected={(agentsStatus?.mt4_connected ?? 0) > 0}
            detail={`${agentsStatus?.mt4_connected ?? 0} connected`}
          />
          <TerminalLiveness
            label="MT5 Terminal"
            connected={(agentsStatus?.mt5_connected ?? 0) > 0}
            detail={`${agentsStatus?.mt5_connected ?? 0} connected`}
          />
        </div>

        <div className="mt-3 text-[11px] text-pat-text-muted leading-relaxed">
          This page shows your <strong>Client Agent</strong> (execution) connection. Live
          <strong> signal delivery</strong> (signals → EA) and <strong>candle delivery</strong> (agent
          → engine) are reported on the local Windows Agent dashboard:
          Client <code>http://127.0.0.1:9000</code>.
        </div>
      </div>

      {/* Registered devices with terminal details */}
      <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-5">
        <h2 className="text-sm font-semibold text-pat-text-primary mb-4">Your Registered Devices & Terminals</h2>
        {isLoading && <div className="text-sm text-pat-text-secondary">Loading...</div>}
        {!isLoading && (!devices || devices.length === 0) && (
          <div className="flex items-center gap-2 py-6">
            <IconDeviceDesktop size={20} className="text-pat-text-muted" />
            <div className="text-sm text-pat-text-muted">No devices registered yet. Install the Windows Agent to connect.</div>
          </div>
        )}
        <div className="space-y-4">
          {devices?.map((d) => (
            <div key={d.id} className="rounded-lg border border-pat-border/60 bg-pat-bg-surface-secondary/20 p-4">
              {/* Device header */}
              <div className="flex items-center justify-between mb-3">
                <div className="flex items-center gap-3">
                  <div className="flex items-center justify-center w-9 h-9 rounded-lg bg-pat-info/10">
                    <IconBrandWindows size={18} className="text-pat-info" />
                  </div>
                  <div>
                    <div className="text-sm font-medium text-pat-text-primary">{d.device_name}</div>
                    <div className="text-[10px] text-pat-text-muted">
                      {d.os} | Host: {d.hostname} | Agent: {d.agent_version}
                    </div>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <span className={`inline-flex items-center gap-1 px-2 py-0.5 rounded text-[10px] font-medium ${
                    d.status === "ONLINE"
                      ? "bg-pat-success/10 text-pat-success border border-pat-success/20"
                      : "bg-pat-danger/10 text-pat-danger border border-pat-danger/20"
                  }`}>
                    <span className={`inline-block h-1.5 w-1.5 rounded-full ${d.status === "ONLINE" ? "bg-pat-success animate-pulse" : "bg-pat-danger"}`} />
                    {d.status === "ONLINE" ? "Online" : "Offline"}
                  </span>
                </div>
              </div>
              {/* Hardware fingerprint */}
              <div className="flex items-center gap-2 mb-3 text-[10px]">
                <IconFingerprint size={12} className="text-pat-text-muted" />
                <span className="text-pat-text-muted">Hardware ID:</span>
                <span className="font-mono text-pat-text-secondary">{d.fingerprint_hash ? d.fingerprint_hash.slice(0, 20) + "..." : "Not captured yet"}</span>
                <span className="px-1.5 py-0.5 rounded bg-pat-bg-surface-secondary text-pat-text-muted">{d.security_state}</span>
              </div>
              {/* Terminal activations */}
              {d.activations && d.activations.length > 0 && (
                <div className="space-y-2">
                  <div className="text-[10px] text-pat-text-muted uppercase">Connected Terminals</div>
                  {d.activations.map((a, i) => (
                    <div key={i} className="rounded-md bg-pat-bg-surface/50 px-3 py-2 flex items-center justify-between border-l-2 border-pat-success/40">
                      <div className="flex items-center gap-2">
                        <IconTerminal2 size={14} className="text-pat-text-secondary" />
                        <div>
                          <div className="text-xs font-medium text-pat-text-primary">
                            {a.client_type} — {a.broker_name || "Unknown broker"}
                          </div>
                          <div className="text-[10px] text-pat-text-muted">
                            Account: {a.mt_account_login} | Build: {a.terminal_build} | EA: {a.ea_version}
                            {a.broker_server && ` | Server: ${a.broker_server}`}
                          </div>
                        </div>
                      </div>
                      <div className="text-[10px] text-pat-text-muted">
                        {a.activated_at ? new Date(a.activated_at).toLocaleDateString() : "—"}
                      </div>
                    </div>
                  ))}
                </div>
              )}
              {/* Last seen */}
              <div className="text-[10px] text-pat-text-muted mt-2 pt-2 border-t border-pat-border/30">
                Registered: {d.registered_at ? new Date(d.registered_at).toLocaleString() : "—"} | Last seen: {d.last_seen_at ? new Date(d.last_seen_at).toLocaleString() : "—"}
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Download section */}
      <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-5">
        <h2 className="text-sm font-semibold text-pat-text-primary mb-4">Download Files</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
          {downloadFiles.map((file) => (
            <a key={file.file} href={file.file} download className={`flex items-center gap-3 rounded-lg border p-4 transition-all hover:border-pat-border/80 ${file.primary ? "border-pat-success/30 bg-pat-success/5" : "border-pat-border/60 bg-pat-bg-surface-secondary/30"}`}>
              <div className={`flex items-center justify-center w-10 h-10 rounded-lg ${file.primary ? "bg-pat-success/10" : "bg-pat-bg-surface-secondary"}`}>
                <file.icon size={20} className={file.primary ? "text-pat-success" : "text-pat-text-secondary"} />
              </div>
              <div className="flex-1 min-w-0">
                <div className="text-sm font-medium text-pat-text-primary">{file.name}</div>
                <div className="text-xs text-pat-text-muted truncate">{file.desc}</div>
                <div className="text-[10px] text-pat-text-muted mt-0.5">
                  <span className="px-1.5 py-0.5 rounded bg-pat-bg-surface-secondary uppercase">{file.type}</span>
                  <span className="ml-1.5">{file.size}</span>
                </div>
              </div>
              <IconDownload size={18} className="text-pat-text-muted shrink-0" />
            </a>
          ))}
        </div>
      </div>

      {/* Installation guide — always visible step list */}
      <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-5">
        <h2 className="text-sm font-semibold text-pat-text-primary mb-4">Installation Guide</h2>
        <ol className="space-y-4">
          {installSteps.map((step, i) => {
            const g = guideSteps[step.id];
            return (
              <li key={step.id} className="flex gap-3">
                <span className="flex items-center justify-center w-7 h-7 shrink-0 rounded-full bg-pat-success/15 text-pat-success text-xs font-bold">
                  {i + 1}
                </span>
                <div className="flex-1 min-w-0">
                  <div className="text-sm font-medium text-pat-text-primary mb-1.5">{g.title}</div>
                  <ul className="space-y-1">
                    {g.steps.map((s, j) => (
                      <li key={j} className="flex items-start gap-2 text-xs text-pat-text-secondary">
                        <span className="mt-1.5 w-1 h-1 rounded-full bg-pat-text-muted shrink-0" />
                        <span className="leading-relaxed">{s}</span>
                      </li>
                    ))}
                  </ul>
                </div>
              </li>
            );
          })}
        </ol>
      </div>

      {/* Risk protection info */}
      <div className="rounded-xl border border-pat-warning/20 bg-pat-warning/5 p-5">
        <h2 className="text-sm font-semibold text-pat-text-primary mb-2 flex items-center gap-2">
          <IconShieldCheck size={16} className="text-pat-warning" /> Hardware-Bound License Protection
        </h2>
        <div className="text-xs text-pat-text-muted space-y-1">
          <div>• Your license is bound to your hardware fingerprint (CPU ID, motherboard serial, disk serial).</div>
          <div>• No other machine can use your license key — prevents license sharing.</div>
          <div>• Max {licenses?.[0]?.max_devices || 2} devices and {licenses?.[0]?.max_mt_accounts || 2} MT accounts per license.</div>
          <div>• Capital protection: 5% daily loss limit, 1% per-trade risk, partial TP (50/30/20).</div>
          <div>• Swap and slippage protection per strategy.</div>
        </div>
      </div>
    </div>
  );
}

function TerminalLiveness({ label, connected, detail }: { label: string; connected: boolean; detail: string }) {
  return (
    <div className={`flex items-center justify-between rounded-lg border p-3 ${
      connected ? "border-pat-success/30 bg-pat-success/5" : "border-pat-border bg-pat-bg-surface-secondary/30"
    }`}>
      <div className="flex items-center gap-2.5">
        <span className={`flex items-center justify-center w-8 h-8 rounded-lg ${connected ? "bg-pat-success/10" : "bg-pat-danger/10"}`}>
          <IconTerminal2 size={16} className={connected ? "text-pat-success" : "text-pat-danger"} />
        </span>
        <div>
          <div className="text-sm font-medium text-pat-text-primary">{label}</div>
          <div className="text-[10px] text-pat-text-muted">{detail}</div>
        </div>
      </div>
      <span className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-lg text-[11px] font-medium ${
        connected
          ? "bg-pat-success/10 text-pat-success border border-pat-success/20"
          : "bg-pat-danger/10 text-pat-danger border border-pat-danger/20"
      }`}>
        <span className={`inline-block h-1.5 w-1.5 rounded-full ${connected ? "bg-pat-success animate-pulse" : "bg-pat-danger"}`} />
        {connected ? "CONNECTED" : "OFFLINE"}
      </span>
    </div>
  );
}
