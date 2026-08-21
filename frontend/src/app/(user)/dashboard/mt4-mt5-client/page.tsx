"use client";
import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import StatusBadge from "@/components/ui/status-badge";
import {
  IconDownload, IconBrandWindows, IconDeviceDesktop,
  IconClipboard, IconCheck, IconTerminal2,
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
  max_devices?: number;
  max_mt_accounts?: number;
  activations: DeviceActivation[] | null;
}

type InstallStep = "download" | "install" | "connect" | "mt4" | "mt5" | "verify";

export default function UserMtClientPage() {
  const [activeStep, setActiveStep] = useState<InstallStep>("download");
  const [copiedKey, setCopiedKey] = useState(false);

  const { data: devices, isLoading } = useQuery<UserDevice[]>({
    queryKey: ["user-mt-devices"],
    queryFn: async () => (await customInstance.get("/licensing/devices")).data,
    refetchInterval: 5000,
  });

  // Go engine live agent connection status
  const { data: agentsStatus } = useQuery<{ agents_connected: number; master_node_connected: boolean; snapshot_count: number }>({
    queryKey: ["user-mt-agents"],
    queryFn: async () => (await customInstance.get("/agents/status")).data,
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
    { name: "Windows Agent (Installer)", file: "/downloads/PredictATrade-Agent-Setup.exe", desc: "Connects your MT4/MT5 terminal to the Predict-A-Trade signal engine", size: "9.2 MB", icon: IconBrandWindows, type: "exe", primary: true },
    { name: "MT4 Expert Advisor", file: "/downloads/PredictATrade_MT4.mq4", desc: "Signal receiver EA for MetaTrader 4", size: "42 KB", icon: IconTerminal2, type: "mql4" },
    { name: "MT5 Expert Advisor", file: "/downloads/PredictATrade_MT5.mq5", desc: "Signal receiver EA for MetaTrader 5", size: "43 KB", icon: IconTerminal2, type: "mql5" },
  ];

  const installSteps: { id: InstallStep; label: string }[] = [
    { id: "download", label: "1. Download" },
    { id: "install", label: "2. Install Agent" },
    { id: "connect", label: "3. Enter License" },
    { id: "mt4", label: "4. Install MT4 EA" },
    { id: "mt5", label: "5. Install MT5 EA" },
    { id: "verify", label: "6. Verify" },
  ];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">MT4/MT5 Client Setup</h1>
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

      {/* Live Connection Status */}
      <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-4">
        <div className="flex items-center justify-between flex-wrap gap-3">
          <div className="flex items-center gap-3">
            <div className={`flex items-center justify-center w-10 h-10 rounded-lg ${(agentsStatus?.agents_connected ?? 0) > 0 ? "bg-pat-success/10" : "bg-pat-danger/10"}`}>
              <svg className={`w-5 h-5 ${(agentsStatus?.agents_connected ?? 0) > 0 ? "text-pat-success" : "text-pat-danger"}`} fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M8.288 15.038a5.25 5.25 0 017.424 0M5.106 11.856c3.407-.304 6.728.897 9.336 3.504M1.924 8.674c5.327-.69 10.65.98 14.704 5.034m1.429-1.429l-1.429 1.429L15.2 9.278M12 18.75a.75.75 0 100-1.5.75.75 0 000 1.5z" />
              </svg>
            </div>
            <div>
              <div className="text-sm font-semibold text-pat-text-primary">Windows Agent Connection</div>
              <div className="text-xs text-pat-text-muted">
                {agentsStatus?.agents_connected ?? 0} agent(s) connected
                {agentsStatus?.master_node_connected ? " · Master Node: ONLINE" : " · Master Node: OFFLINE"}
                {" · "}{(agentsStatus?.snapshot_count ?? 0).toLocaleString()} snapshots
              </div>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <span className={`inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium ${
              (agentsStatus?.agents_connected ?? 0) > 0
                ? "bg-pat-success/10 text-pat-success border border-pat-success/20"
                : "bg-pat-danger/10 text-pat-danger border border-pat-danger/20"
            }`}>
              <span className={`inline-block h-2 w-2 rounded-full ${(agentsStatus?.agents_connected ?? 0) > 0 ? "bg-pat-success animate-pulse" : "bg-pat-danger"}`} />
              {(agentsStatus?.agents_connected ?? 0) > 0 ? "LIVE" : "OFFLINE"}
            </span>
          </div>
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

      {/* Installation guide */}
      <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-5">
        <h2 className="text-sm font-semibold text-pat-text-primary mb-4">Installation Guide</h2>
        <div className="flex flex-wrap gap-1.5 mb-4">
          {installSteps.map((step) => (
            <button key={step.id} onClick={() => setActiveStep(step.id)}
              className={`px-3 py-1.5 text-xs font-medium rounded-lg transition-all ${activeStep === step.id ? "bg-pat-success/15 text-pat-success border border-pat-success/30" : "text-pat-text-muted hover:text-pat-text-secondary border border-transparent"}`}>
              {step.label}
            </button>
          ))}
        </div>
        <div className="rounded-lg bg-pat-bg-surface-secondary/20 p-4">
          {activeStep === "download" && <Steps title="Download the Windows Agent" steps={[
            "Click the Download button for the Windows Agent (Installer) above.",
            "Save the file to a known location.",
            "Also download the MT4 or MT5 Expert Advisor for your platform.",
          ]} />}
          {activeStep === "install" && <Steps title="Install the Windows Agent" steps={[
            "Double-click PredictATrade-Agent-Setup.exe to start the installer.",
            "If Windows SmartScreen appears, click 'More info' then 'Run anyway'.",
            "Follow the installation wizard — accept the default path.",
            "The agent installs as a Windows Service and starts automatically.",
          ]} />}
          {activeStep === "connect" && <Steps title="Enter Your License Key" steps={[
            "Right-click the Predict-A-Trade agent icon in the system tray.",
            "Select 'Settings' or 'Configure'.",
            "Paste your license key (from the box above) into the License Key field.",
            "Click 'Save' or 'Connect'. The agent will show 'Connected' status.",
            "Your device and hardware fingerprint will be registered to the platform.",
          ]} />}
          {activeStep === "mt4" && <Steps title="Install MT4 Expert Advisor" steps={[
            "Open MetaTrader 4 → File → Open Data Folder → MQL4 → Experts.",
            "Copy PredictATrade_MT4.mq4 into the Experts folder.",
            "In MT4, refresh the Navigator (Ctrl+N) → Right-click 'Expert Advisors' → 'Refresh'.",
            "Drag PredictATrade_MT4 onto a XAUUSD chart.",
            "Check 'Allow live trading' → OK. Enable the 'AutoTrading' button (green).",
          ]} />}
          {activeStep === "mt5" && <Steps title="Install MT5 Expert Advisor" steps={[
            "Open MetaTrader 5 → File → Open Data Folder → MQL5 → Experts.",
            "Copy PredictATrade_MT5.mq5 into the Experts folder.",
            "In MT5, refresh the Navigator → Right-click 'Expert Advisors' → 'Refresh'.",
            "Drag PredictATrade_MT5 onto a XAUUSD chart.",
            "Check 'Allow Algo Trading' → OK. Enable 'Algo Trading' button (green).",
            "Press F7 in MetaEditor to compile if the file shows as .mq5 (not .ex5).",
          ]} />}
          {activeStep === "verify" && <Steps title="Verify Your Connection" steps={[
            "Check the Windows Agent system tray icon — should show 'Connected'.",
            "In MT4/MT5, check the Experts tab — should show 'PAT: Connected to agent'.",
            "Your device and terminal details should appear in the 'Registered Devices' section above.",
            "The hardware fingerprint binds your license to this machine — no other machine can use your license.",
            "If you see 'Agent: Disconnected', restart the Windows Agent service from the system tray.",
          ]} />}
        </div>
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

function Steps({ title, steps }: { title: string; steps: string[] }) {
  return (
    <div>
      <h3 className="text-sm font-medium text-pat-text-primary mb-2">{title}</h3>
      <ol className="space-y-1.5">
        {steps.map((step, i) => (
          <li key={i} className="flex items-start gap-2 text-xs text-pat-text-secondary">
            <span className="flex items-center justify-center w-5 h-5 rounded-full bg-pat-bg-surface-secondary text-[10px] text-pat-text-muted shrink-0 mt-0.5">{i + 1}</span>
            <span className="leading-relaxed">{step}</span>
          </li>
        ))}
      </ol>
    </div>
  );
}
