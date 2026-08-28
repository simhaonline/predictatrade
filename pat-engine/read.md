# Predict-A-Trade XAUUSD — Windows Agent (pat-engine / new project)

The agent is built from `pat-engine/cmd/agent` and installed **locally** from
`pat-engine/scripts/`. It is the client/execution side only (the new
architecture has **no master-node role**); it feeds bars to the gateway and is
the local side of SL-enforcement / `EMERGENCY_STOP` / `KILL_SWITCH`.

> **Reference — old project (`windows-agent`):** that project shipped one-line
> hosted commands, e.g.
> `irm https://downloads.predictatrade.com/windows-agent/install.ps1 | iex`
> (install), the **same command** to update, and
> `irm https://downloads.predictatrade.com/windows-agent/uninstall.ps1 | iex`
> to remove. The new project keeps the **same three operations** (install /
> update / uninstall) but installs from the local `scripts/` folder after
> building the exe — so the commands below are the local equivalents. Do **not**
> point the new project at `downloads.predictatrade.com` (that is the legacy host).

## Build (once)
```bash
./scripts/build-windows-agent.sh   # -> dist/pat-windows-agent.exe
```
Copy `dist/pat-windows-agent.exe` **and** `scripts/*.ps1` to the Windows trading
machine (same folder), then run the commands below as **Administrator**.

## Install (single command)
```powershell
.\install-client.ps1 -EngineHost api.predictatrade.com -LicenseKey "PAT1-XXXX-XXXX-XXXX-XXXX-XXXX-XX"
```
`-EngineHost` is where the gateway runs: use `localhost` for same-box dev, the
public domain for production, or a LAN IP for a separate server.

## Update (same command — re-run detects the existing install)
```powershell
.\install-client.ps1 -EngineHost api.predictatrade.com -LicenseKey "PAT1-XXXX-XXXX-XXXX-XXXX-XXXX-XX"
```
Re-running stops the service, swaps the exe, and restarts — preserving config.

## Uninstall
```powershell
.\uninstall-windows-agent.ps1
```
Removes the `pat-agent-client` service, the binary, EA + `PAT_license.txt`, and
the Defender exclusion.

## Verify
```powershell
Get-Service pat-agent-client          # Status should be "Running"
[Environment]::GetEnvironmentVariable("GATEWAY","Machine")   # https://<EngineHost>/bar (or http://<EngineHost>:80/bar for localhost)
```
