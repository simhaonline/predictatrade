"use client";
import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import StatusBadge from "@/components/ui/status-badge";
import {
  IconDownload, IconBrandWindows, IconDeviceDesktop,
  IconClipboard, IconCheck, IconTerminal2, IconRefresh,
  IconFingerprint, IconShieldCheck, IconLicense, IconCloud,
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

type InstallStep = "license" | "download" | "allowlist" | "mt4" | "mt5" | "verify";

export default function UserMtClientPage() {
  const [copiedKey, setCopiedKey] = useState(false);

  const { data: devices, isLoading } = useQuery<UserDevice[]>({
    queryKey: ["user-mt-devices"],
    queryFn: async () => (await customInstance.get("/licensing/devices")).data,
    refetchInterval: 5000,
  });

  // Per-user terminal status — derived from the user's OWN registered devices.
  // Option B (v1.19.0): EAs talk to the cloud directly over HTTPS. A device is
  // "live" when the control plane has seen it recently (edge-poll/heartbeat
  // refresh devices.last_seen_at); terminal link state lives on the device's
  // `activations` array, keyed by client_type (MT4/MT5).
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

  // Ready-to-run compiled EAs only — no source distribution. The .ex4/.ex5
  // binaries are built and published per release (mql/compiled_executable).
  const downloadFiles = [
    { name: "MT5 Expert Advisor — Ready to Run (.ex5)", file: "https://downloads.predictatrade.com/downloads/Predict-A-Trade.ex5", desc: "Compiled client EA for MetaTrader 5 — drop it in, no compiler needed", size: "~260 KB", icon: IconTerminal2, type: "EX5", primary: true },
    { name: "MT4 Expert Advisor — Ready to Run (.ex4)", file: "https://downloads.predictatrade.com/downloads/Predict-A-Trade.ex4", desc: "Compiled client EA for MetaTrader 4 — drop it in, no compiler needed", size: "~290 KB", icon: IconTerminal2, type: "EX4" },
  ];

  const installSteps: { id: InstallStep; label: string }[] = [
    { id: "license", label: "1. Copy License" },
    { id: "download", label: "2. Download EA" },
    { id: "allowlist", label: "3. Allow Cloud URL" },
    { id: "mt4", label: "4. Install MT4 EA" },
    { id: "mt5", label: "5. Install MT5 EA" },
    { id: "verify", label: "6. Verify" },
  ];

  // Full installation steps shown inline on the dashboard (no toggling required).
  const guideSteps: Record<InstallStep, { title: string; steps: string[] }> = {
    download: {
      title: "Download the Expert Advisors (ready-to-run binaries)",
      steps: [
        "No Windows Agent needed — the EA talks to the Predict-A-Trade cloud directly over HTTPS.",
        "Download the compiled EA above: Predict-A-Trade.ex5 (MT5) or Predict-A-Trade.ex4 (MT4). No MetaEditor, no compiling — the file runs as-is.",
        "Check the version chip on the download card before installing — always use the latest published build.",
        "Only one of MT4/MT5 is required; install on every terminal you want signals delivered to (each terminal = 1 device slot).",
      ],
    },
    allowlist: {
      title: "Allow the Cloud API in your terminal (one-time, per terminal)",
      steps: [
        "In MetaTrader: Tools → Options → Expert Advisors tab.",
        "Tick 'Allow WebRequest for listed URL'.",
        "Add: https://api.predictatrade.com",
        "Click OK. Without this the EA cannot reach the cloud (WebRequest is blocked by default).",
        "Restart the terminal if the EA still shows 'WebRequest not allowed' in the Experts log.",
      ],
    },
    license: {
      title: "Copy Your License Key",
      steps: [
        "Copy your license key from the box above (click the copy icon).",
        "You will paste this key into the EA inputs — it activates your cloud device automatically.",
        "Your license determines which strategies you can receive:",
        "  FREE: Standard Scalping only",
        "  STANDARD: Standard Scalping + Standard Swing",
        "  PRO: All scalping + swing strategies",
        "  ELITE: All strategies including Ultra Scalping and EQFE",
        "Signals are filtered server-side by your subscription plan — a FREE device never receives PRO/ELITE strategy signals.",
      ],
    },
    mt4: {
      title: "Install MT4 Expert Advisor",
      steps: [
        "Open MetaTrader 4 → File → Open Data Folder → MQL4 → Experts.",
        "Copy Predict-A-Trade.ex4 into the Experts folder.",
        "Delete any older PredictATrade .ex4/.mq4 files from Experts first — leaving both causes two panels on the chart.",
        "In MT4, open Navigator (Ctrl+N) → Right-click 'Expert Advisors' → 'Refresh'.",
        "Drag 'Predict-A-Trade' onto an XAUUSD chart (M1–H1 as instructed for your plan).",
        "In the EA inputs, paste your License Key into the LicenseKey field.",
        "Check 'Allow live trading' → OK.",
        "Enable the 'AutoTrading' button at the top (should turn green).",
        "The EA activates its device against the cloud and starts polling for signals every few seconds.",
      ],
    },
    mt5: {
      title: "Install MT5 Expert Advisor",
      steps: [
        "Open MetaTrader 5 → File → Open Data Folder → MQL5 → Experts.",
        "Copy Predict-A-Trade.ex5 into the Experts folder.",
        "Delete any older PredictATrade .ex5/.mq5 files from Experts first — avoids loading a stale build.",
        "In MT5, open Navigator → Right-click 'Expert Advisors' → 'Refresh'.",
        "Drag 'Predict-A-Trade' onto an XAUUSD chart (M1–H1 as instructed for your plan).",
        "In the EA inputs, paste your License Key into the LicenseKey field.",
        "Check 'Allow Algo Trading' → OK.",
        "Enable the 'Algo Trading' button at the top (should turn green).",
        "The EA connects directly to api.predictatrade.com — no local agent, no ports to open.",
      ],
    },
    verify: {
      title: "Verify Your Connection",
      steps: [
        "In MT4/MT5, check the Experts tab — should show 'Device activated: …' followed by 'License validated — ACTIVE'.",
        "On the chart you should see the Predict-A-Trade dashboard panel (two-column status board with a session map).",
        "This dashboard lists your device under 'Your Registered Devices' with status Online within ~15 seconds.",
        "Signals appear on the chart panel and in the Experts log when the engine publishes them.",
        "The EA will only execute strategies allowed by your license plan.",
        "Stop-loss is enforced by the server — trades without SL are automatically closed.",
        "To update: download the new .ex4/.ex5, replace the old file in Experts, then remove and re-attach the EA on the chart.",
      ],
    },
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">MetaTrader Client</h1>
        <p className="text-sm text-pat-text-secondary mt-1">
          Install the ready-to-run EA directly in MetaTrader 4/5 — no Windows Agent, no compiler. The EA connects to the Predict-A-Trade cloud over HTTPS, validates your license, and receives only the signals your subscription plan allows.
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

      {/* EA cloud link — connection & signal delivery */}
      <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-4">
        <div className="flex items-center justify-between flex-wrap gap-3 mb-3">
          <div className="flex items-center gap-2">
            <IconCloud size={18} className="text-pat-info" />
            <h2 className="text-sm font-semibold text-pat-text-primary">EA Cloud Link — Signal Delivery</h2>
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
            EA Link: {(agentsStatus?.agents_connected ?? 0) > 0 ? "LIVE" : "OFFLINE"}
          </span>
          <span>
            {agentsStatus?.agents_connected ?? 0} device(s) polling the cloud
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
          Your EA polls the Predict-A-Trade cloud every few seconds (encrypted, license-gated).
          Signals are delivered <strong>only</strong> for strategies included in your subscription plan,
          and only when the engine marks them executable.
        </div>
      </div>

      {/* Registered devices with terminal details */}
      <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-5">
        <h2 className="text-sm font-semibold text-pat-text-primary mb-4">Your Registered Devices &amp; Terminals</h2>
        {isLoading && <div className="text-sm text-pat-text-secondary">Loading...</div>}
        {!isLoading && (!devices || devices.length === 0) && (
          <div className="flex items-center gap-2 py-6">
            <IconDeviceDesktop size={20} className="text-pat-text-muted" />
            <div className="text-sm text-pat-text-muted">No devices registered yet. Install the EA on an XAUUSD chart — it registers automatically with your license key.</div>
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
                      {d.os} | Host: {d.hostname} | EA: {d.agent_version || "—"}
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

      {/* Download section — compiled binaries only */}
      <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-5">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-sm font-semibold text-pat-text-primary">Download Files</h2>
          <span className="text-[10px] px-2 py-1 rounded-lg bg-pat-success/10 text-pat-success border border-pat-success/20 font-medium">NO COMPILING NEEDED</span>
        </div>
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
{/*
        <div className="mt-3 text-[11px] text-pat-text-muted">
          Master Data Node: if your account was provisioned with a data-feed role, its compiled binaries are available on request — contact support.
        </div>
*/}
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

      {/* EA configuration reference */}
      <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-5">
        <h2 className="text-sm font-semibold text-pat-text-primary mb-4">EA Configuration Reference (v1.29)</h2>
        <div className="space-y-4 text-xs">
          <div>
            <div className="text-pat-text-primary font-semibold mb-1.5">Required input</div>
            <ul className="space-y-1">
              <li className="flex items-start gap-2 text-pat-text-secondary"><span className="mt-1.5 w-1 h-1 rounded-full bg-pat-text-muted shrink-0" /><span><span className="font-mono text-pat-text-primary">LicenseKey</span> — paste your license key exactly as shown above. Everything else works on defaults.</span></li>
            </ul>
          </div>
          <div>
            <div className="text-pat-text-primary font-semibold mb-1.5">Execution</div>
            <ul className="space-y-1">
              <li className="flex items-start gap-2 text-pat-text-secondary"><span className="mt-1.5 w-1 h-1 rounded-full bg-pat-text-muted shrink-0" /><span><span className="font-mono text-pat-text-primary">AutoExecute</span> (default false = display-only) — set true to auto-trade EXECUTABLE signals. This is the main on/off switch.</span></li>
              <li className="flex items-start gap-2 text-pat-text-secondary"><span className="mt-1.5 w-1 h-1 rounded-full bg-pat-text-muted shrink-0" /><span><span className="font-mono text-pat-text-primary">ExecuteCandidates</span> (default false) — also execute BUY_CANDIDATE/SELL_CANDIDATE signals. Keep off unless advised.</span></li>
              <li className="flex items-start gap-2 text-pat-text-secondary"><span className="mt-1.5 w-1 h-1 rounded-full bg-pat-text-muted shrink-0" /><span><span className="font-mono text-pat-text-primary">ChartTimeframe</span> (default "M1") — the timeframe this EA instance trades; used for chart identity and session reporting.</span></li>
              <li className="flex items-start gap-2 text-pat-text-secondary"><span className="mt-1.5 w-1 h-1 rounded-full bg-pat-text-muted shrink-0" /><span><span className="font-mono text-pat-text-primary">PATPollMs</span> (3000) — signal poll interval; keep 3000 (ULTRA scalping TTL is 3 minutes).</span></li>
              <li className="flex items-start gap-2 text-pat-text-secondary"><span className="mt-1.5 w-1 h-1 rounded-full bg-pat-text-muted shrink-0" /><span>Signal TTL: signals older than the server expiry are never executed (fail-closed, ~300s default).</span></li>
            </ul>
          </div>
          <div>
            <div className="text-pat-text-primary font-semibold mb-1.5">Risk &amp; session protection</div>
            <ul className="space-y-1">
              <li className="flex items-start gap-2 text-pat-text-secondary"><span className="mt-1.5 w-1 h-1 rounded-full bg-pat-text-muted shrink-0" /><span>Daily loss limit 6% (warning at 3%), floating-drawdown breaker 5%, equity floor — all terminal-local, server gates apply on top.</span></li>
              <li className="flex items-start gap-2 text-pat-text-secondary"><span className="mt-1.5 w-1 h-1 rounded-full bg-pat-text-muted shrink-0" /><span><span className="font-mono text-pat-text-primary">AvoidSwapCharges</span> / <span className="font-mono text-pat-text-primary">AvoidRolloverWindow</span> (default on) — skip entries in the swap-cutoff / broker-rollover windows; <span className="font-mono text-pat-text-primary">AvoidTripleSwapDay</span> skips the triple-swap weekday.</span></li>
              <li className="flex items-start gap-2 text-pat-text-secondary"><span className="mt-1.5 w-1 h-1 rounded-full bg-pat-text-muted shrink-0" /><span><span className="font-mono text-pat-text-primary">MaxTradesPerDay</span> (50) and per-strategy position caps guard against runaway execution.</span></li>
            </ul>
          </div>
          <div>
            <div className="text-pat-text-primary font-semibold mb-1.5">Dashboard panel</div>
            <ul className="space-y-1">
              <li className="flex items-start gap-2 text-pat-text-secondary"><span className="mt-1.5 w-1 h-1 rounded-full bg-pat-text-muted shrink-0" /><span><span className="font-mono text-pat-text-primary">PAT_ShowDashboard</span> (default true) — show the on-chart status panel.</span></li>
              <li className="flex items-start gap-2 text-pat-text-secondary"><span className="mt-1.5 w-1 h-1 rounded-full bg-pat-text-muted shrink-0" /><span><span className="font-mono text-pat-text-primary">PAT_DashFontSize</span> (8, range 6–12) and <span className="font-mono text-pat-text-primary">PAT_DashRefreshMs</span> (400) — panel readability / CPU trade-off.</span></li>
              <li className="flex items-start gap-2 text-pat-text-secondary"><span className="mt-1.5 w-1 h-1 rounded-full bg-pat-text-muted shrink-0" /><span>Panel controls: drag the header to move (position is remembered), click the header to collapse, press <span className="font-mono text-pat-text-primary">F</span> or the PAUSE button to halt/resume execution. The server keeps enforcing your plan either way.</span></li>
            </ul>
          </div>
        </div>
      </div>

      {/* Risk protection info */}
      <div className="rounded-xl border border-pat-warning/20 bg-pat-warning/5 p-5">
        <h2 className="text-sm font-semibold text-pat-text-primary mb-2 flex items-center gap-2">
          <IconShieldCheck size={16} className="text-pat-warning" /> License &amp; Capital Protection
        </h2>
        <div className="text-xs text-pat-text-muted space-y-1">
          <div>• Your EA authenticates with a per-device credential set derived from your license key — one device = one binding.</div>
          <div>• Signals are delivered ONLY for strategies your subscription plan allows (enforced server-side at enqueue and again at poll time).</div>
          <div>• Non-executable signals (advisory, gate-blocked) are never delivered to your EA — fail-closed delivery.</div>
          <div>• Capital protection: daily loss limits, per-trade risk cap, partial TP ladder, server-enforced stop-loss.</div>
          <div>• Swap, slippage and rollover protection per strategy.</div>
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