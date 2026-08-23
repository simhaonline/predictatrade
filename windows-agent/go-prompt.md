# Predict-A-Trade Windows Agent — Windows Termination Fix + Installer

You are working inside the existing **Predict-A-Trade XAUUSD** codebase.

The Windows agent is currently being **terminated, quarantined, blocked, or failing to remain running on Windows**.

Your task is to diagnose and fix the **Windows agent, build process, Windows Service installation, signing/package metadata, installer, configuration collection, startup, logging, and health monitoring**.

## PRIMARY OBJECTIVE

Make the existing Predict-A-Trade Windows agent install and run reliably on:

* Windows 10 x64
* Windows 11 x64
* Windows Server 2019+
* Windows Server 2022+
* Windows Server 2025 where compatible

Do **not** redesign the application.

Do **not** over-engineer.

Do **not** weaken Windows Defender, SmartScreen, UAC, firewall, or other Windows security controls.

Do **not** create antivirus exclusions or attempt to evade antivirus detection.

Instead, determine **why Windows is terminating the agent and fix the legitimate underlying cause**.

---

# 1. AUDIT EXISTING WINDOWS IMPLEMENTATION FIRST

Before changing anything, inspect the existing codebase and identify:

* Windows agent source
* Go entry point
* current `build.ps1`
* installer scripts
* service-installation scripts
* `settings.json`
* logging
* health endpoint
* Scheduled Tasks
* startup logic
* Windows Service implementation
* notification implementation
* version metadata
* resource files
* signing implementation
* update/uninstall implementation

Do not replace working functionality unnecessarily.

Produce a short audit showing:

```text
Existing Windows Agent:
Existing Build Process:
Existing Installer:
Existing Service Mechanism:
Existing Signing:
Existing Configuration:
Existing Logging:
Likely Termination Cause:
Changes Required:
```

Then implement only the required fixes.

---

# 2. INVESTIGATE WHY WINDOWS TERMINATES THE AGENT

Determine whether termination is caused by:

* Windows Defender detection/quarantine
* Microsoft Defender for Endpoint
* SmartScreen
* missing/invalid executable signature
* unsigned binary reputation
* malformed PE/resources
* missing DLL/runtime dependency
* Windows Service crash
* service timeout
* panic/unhandled exception
* incorrect working directory
* permissions problem
* UAC requirement
* file-access permissions
* network-access failure
* port conflict
* duplicate process
* application self-termination
* watchdog
* Scheduled Task conflict
* installer problem
* Windows Firewall
* architecture mismatch
* corrupted configuration
* missing environment variables

Add diagnostic collection where appropriate for:

* Windows Event Viewer
* Windows Service Control Manager
* Application log
* Defender operational log
* agent logs
* process exit code
* service exit code

The goal is to identify the actual reason rather than masking it.

---

# 3. WINDOWS BUILD

Create or repair:

```text
build.ps1
```

The build should produce a normal production Go executable.

Example output:

```text
dist/
  PredictATradeAgent.exe
```

Requirements:

* `GOOS=windows`
* `GOARCH=amd64`
* production build
* deterministic version injection where practical
* proper Windows executable metadata
* no unnecessary packers
* no UPX
* no obfuscation
* no antivirus-evasion techniques
* no strange runtime extraction
* no hidden temporary executable generation

Use normal Go compiler/linker options.

Do not aggressively strip information purely to reduce antivirus detection.

---

# 4. WINDOWS RESOURCE METADATA

Create or repair:

```text
winres.json
```

Embed legitimate application metadata such as:

```text
CompanyName: Predict-A-Trade
ProductName: Predict-A-Trade Windows Agent
FileDescription: Predict-A-Trade Windows Agent
InternalName: PredictATradeAgent
OriginalFilename: PredictATradeAgent.exe
LegalCopyright: Predict-A-Trade
ProductVersion: <project version>
FileVersion: <project version>
```

Use the existing project version automatically where possible.

Include a legitimate application icon if one already exists in the repository.

Do not fabricate Microsoft or third-party publisher information.

---

# 5. CODE SIGNING

Support proper Authenticode signing.

Preferred production mechanism:

```text
signtool.exe
```

Support certificate configuration through environment variables or installer/build parameters.

Example:

```text
PAT_SIGN_CERT
PAT_SIGN_CERT_PASSWORD
PAT_TIMESTAMP_URL
```

Use SHA-256.

Use a trusted RFC3161 timestamp service configured by the operator.

Example conceptual signing flow:

```powershell
signtool sign `
  /fd SHA256 `
  /tr $TimestampURL `
  /td SHA256 `
  /f $Certificate `
  /p $CertificatePassword `
  .\dist\PredictATradeAgent.exe
```

IMPORTANT:

Do **not** automatically generate a self-signed certificate and present it as production signing.

A self-signed certificate may be supported only for clearly labelled local development/testing.

Production builds must support a legitimate organization code-signing certificate.

After signing, verify with:

```powershell
Get-AuthenticodeSignature
```

and/or:

```text
signtool verify
```

Build must report:

```text
Unsigned
Development Signed
Production Signed
```

clearly.

---

# 6. INSTALLER

Create or repair a simple Windows installer script:

```text
install.ps1
```

The installer must be easy for a normal operator to execute.

It should:

1. verify Administrator privileges if required
2. locate the agent executable
3. create the application directory
4. install/copy required files
5. collect configuration interactively
6. validate entered configuration
7. save configuration
8. restrict configuration-file permissions
9. install the Windows Service
10. start the service
11. verify the health endpoint
12. report success/failure clearly

Recommended application directory:

```text
C:\Program Files\Predict-A-Trade\Agent\
```

Recommended configuration/data directory:

```text
C:\ProgramData\Predict-A-Trade\Agent\
```

Example:

```text
C:\ProgramData\Predict-A-Trade\Agent\settings.json
C:\ProgramData\Predict-A-Trade\Agent\logs\
```

Do not store mutable configuration inside `Program Files` unless the current architecture specifically requires it.

---

# 7. INTERACTIVE SETTINGS CONFIGURATION

The installer must ask the user for notification and health-monitoring configuration instead of requiring manual editing of `settings.json`.

Prompt:

```text
Predict-A-Trade Windows Agent Configuration
```

Ask:

```text
Notification Type:
1. Telegram
2. Discord
3. Email
4. None
```

Save one of:

```json
"notification_type": "telegram"
```

```json
"notification_type": "discord"
```

```json
"notification_type": "email"
```

or:

```json
"notification_type": "none"
```

---

## Telegram

If Telegram is selected, ask:

```text
Telegram Bot Token:
Telegram Chat ID:
```

Save:

```json
{
  "telegram_bot_token": "...",
  "telegram_chat_id": "..."
}
```

Do not ask Discord or Email questions.

---

## Discord

If Discord is selected, ask:

```text
Discord Webhook URL:
```

Save:

```json
{
  "discord_webhook": "..."
}
```

Do not ask Telegram or Email questions.

---

## Email

If Email is selected, ask:

```text
SMTP Server:
SMTP Port:
SMTP Username:
SMTP Password:
From Email:
To Email:
```

Save equivalent existing schema fields:

```json
{
  "email_smtp": "...",
  "email_to": "...",
  "email_from": "...",
  "email_port": 587,
  "email_user": "...",
  "email_password": "..."
}
```

Use the project's existing field names if they already exist.

Do not create duplicate configuration schemas.

Passwords/tokens must not be printed back to the terminal after entry.

Where practical use:

```powershell
Read-Host -AsSecureString
```

for secrets.

---

# 8. HEALTH CHECK SETTINGS

Ask:

```text
Health Check URL
```

Default:

```text
http://localhost:9000/health
```

Ask:

```text
Health Check Timeout Seconds
```

Default:

```text
5
```

Ask:

```text
Health Check Interval Minutes
```

Default:

```text
1
```

Save:

```json
{
  "health_check_url": "http://localhost:9000/health",
  "health_check_timeout_seconds": 5,
  "health_check_interval_minutes": 1
}
```

Pressing ENTER must accept the default.

Validate:

* URL format
* timeout > 0
* interval >= 1

---

# 9. SETTINGS.JSON

Generate the final file automatically.

Example:

```json
{
  "notification_type": "telegram",

  "telegram_bot_token": "",
  "telegram_chat_id": "",

  "discord_webhook": "",

  "email_smtp": "",
  "email_to": "",
  "email_from": "",
  "email_port": 587,
  "email_user": "",
  "email_password": "",

  "health_check_url": "http://localhost:9000/health",
  "health_check_timeout_seconds": 5,
  "health_check_interval_minutes": 1
}
```

However:

Preserve any additional existing Predict-A-Trade configuration fields.

Do not overwrite unrelated configuration.

If `settings.json` already exists:

```text
Existing Predict-A-Trade configuration detected.
```

Load it and use existing values as defaults.

Ask before replacing changed values.

Create a backup:

```text
settings.json.bak
```

before modifying an existing configuration.

---

# 10. CONFIGURATION SECURITY

Because the configuration can contain:

* Telegram token
* Discord webhook
* SMTP credentials

restrict access to the configuration file.

At minimum ensure normal unrelated Windows users cannot freely read it.

Prefer permissions appropriate for:

* Administrators
* SYSTEM
* the actual Predict-A-Trade service account

Do not unnecessarily expose secrets in:

* logs
* console output
* Event Viewer
* command-line arguments
* process listings

Never log:

```text
telegram_bot_token
discord_webhook
email_password
```

---

# 11. WINDOWS SERVICE

The agent should operate as a proper Windows Service if its architecture is intended to run continuously.

Service name:

```text
PredictATradeAgent
```

Display name:

```text
Predict-A-Trade Windows Agent
```

Use the existing Windows service library if one is already present.

Do not introduce NSSM or another third-party wrapper unless absolutely necessary.

Configure:

```text
Startup Type: Automatic
```

and sensible Windows Service recovery:

```text
First failure  -> Restart service
Second failure -> Restart service
Subsequent     -> Restart service
```

Avoid restart loops.

The service must:

* start
* stop
* restart
* shutdown cleanly

Example:

```powershell
Get-Service PredictATradeAgent
```

should return a normal Windows service.

---

# 12. SERVICE WORKING DIRECTORY

Ensure that when Windows starts the service it does **not** assume:

```text
C:\Windows\System32
```

as its application directory.

Resolve paths relative to:

* executable directory
* ProgramData configuration directory

as appropriate.

This is a common Windows service failure and must be checked.

---

# 13. AGENT LOGGING

Create useful logs such as:

```text
C:\ProgramData\Predict-A-Trade\Agent\logs\agent.log
```

At startup log:

```text
Predict-A-Trade Windows Agent starting
Version:
Build:
Windows Version:
Architecture:
Executable Path:
Configuration Path:
Service Mode:
Health Endpoint:
```

Do not log secrets.

On shutdown log:

```text
Agent stopping
Shutdown reason:
Exit code:
```

Add reasonable log rotation if the existing logging system already supports it.

Do not add a complicated logging infrastructure solely for this task.

---

# 14. CRASH PROTECTION

Audit the Go agent for:

* unhandled panic
* nil pointer
* fatal logging
* goroutine panic
* network timeout
* file access failure
* temporary API outage

The agent must not unnecessarily terminate because a remote service is temporarily unavailable.

Transient failures should normally:

```text
log -> retry/backoff -> continue
```

rather than:

```text
fatal -> exit
```

where safe.

Do not hide genuine unrecoverable failures.

---

# 15. HEALTH ENDPOINT

Verify:

```text
http://localhost:9000/health
```

or the configured endpoint.

Expected successful response should represent:

```text
HTTP 200
```

The health system should distinguish between:

```text
Agent process running
Agent healthy
Backend reachable
MT4/MT5 connection healthy
```

if those states already exist in the current architecture.

Do not invent fake healthy states.

---

# 16. HEALTH MONITOR

If the existing implementation uses a Windows Scheduled Task for health monitoring, repair it rather than inventing another monitoring system.

Use:

```text
health_check_interval_minutes
```

Default:

```text
1 minute
```

The health monitor should:

1. call configured health URL
2. wait configured timeout
3. record failure
4. optionally trigger configured notification
5. restart the service only when appropriate

Prevent continuous restart loops.

---

# 17. NOTIFICATION TEST DURING INSTALLATION

After configuration, ask:

```text
Send a test notification now? [Y/n]
```

If Telegram:

```text
Predict-A-Trade Windows Agent
Telegram notification test successful.
```

If Discord:

```text
Predict-A-Trade Windows Agent
Discord notification test successful.
```

If Email:

```text
Predict-A-Trade Windows Agent
Email notification test successful.
```

Clearly report API/authentication errors without exposing credentials.

Notification failure should not corrupt or roll back an otherwise valid installation unless notification is mandatory in the existing specification.

---

# 18. DEFENDER / SMARTSCREEN DIAGNOSTICS

If the executable is being removed or terminated by Windows Security, gather legitimate diagnostic evidence.

Inspect where available:

```text
Microsoft-Windows-Windows Defender/Operational
Windows Event Viewer
Protection History
Get-MpThreatDetection
Get-MpComputerStatus
```

Do not modify Defender configuration automatically.

Do not add exclusion paths.

Do not disable:

```text
Real-time protection
Cloud-delivered protection
Tamper protection
SmartScreen
Defender
```

If Defender identifies the program, report:

```text
Detection Name:
Affected File:
Timestamp:
Action:
Executable SHA256:
Signature Status:
```

Then determine whether the cause appears to be:

```text
actual unsafe behavior
unsigned/untrusted binary
known false positive
malformed packaging
installer behavior
runtime behavior
```

Fix the software where possible.

For a suspected false positive, provide the binary hash and evidence needed for the operator to submit it through Microsoft's legitimate false-positive/submission process.

---

# 19. REMOVE SUSPICIOUS BUILD BEHAVIOUR

Audit the Windows executable and installer for behavior likely to cause legitimate endpoint-security concern, including unnecessary:

* PowerShell spawning
* cmd.exe spawning
* temp executable creation
* executable self-modification
* registry persistence
* scheduled-task persistence beyond required monitoring
* hidden child processes
* credential scraping
* arbitrary process termination
* downloading and executing binaries
* shellcode
* memory injection
* DLL injection
* process hollowing
* obfuscation
* packing
* unsigned helper binaries

If such functionality exists without being required by Predict-A-Trade, remove it.

Do not attempt to disguise such behavior.

---

# 20. FIREWALL

If the agent only exposes:

```text
localhost:9000
```

do not create unnecessary inbound Windows Firewall rules.

If external access is genuinely required, create the narrowest appropriate rule.

Do not globally disable Windows Firewall.

---

# 21. INSTALLER VALIDATION

At the end of installation run:

```powershell
Get-Service PredictATradeAgent
```

Verify:

```text
STATUS = Running
```

Then check:

```text
configured health URL
```

Then verify:

```text
settings.json exists
logs directory exists
service exists
service is running
health endpoint returns HTTP 200
binary signature status
```

Final output should look similar to:

```text
=================================================
 Predict-A-Trade Windows Agent Installation
=================================================

Executable:       OK
Configuration:    OK
Configuration ACL: OK
Windows Service:  Installed
Service Status:   Running
Startup Type:     Automatic
Health Check:     HTTP 200
Notification:     Telegram / Discord / Email / None
Code Signature:   Valid / Dev / Unsigned
Logs:             C:\ProgramData\Predict-A-Trade\Agent\logs

Installation completed successfully.
=================================================
```

---

# 22. UNINSTALLER

Provide a simple:

```text
uninstall.ps1
```

It should:

* stop service
* remove service
* remove Predict-A-Trade Scheduled Tasks created by this installer
* remove executable/application files

Ask before deleting:

```text
settings.json
logs
```

Never delete unrelated Predict-A-Trade files.

---

# 23. BUILD SCRIPT OUTPUT

`build.ps1` should perform only the necessary operations:

```text
1. validate Go
2. determine version
3. generate Windows resources
4. compile Windows executable
5. sign executable when certificate is configured
6. verify executable signature
7. calculate SHA256
8. output build summary
```

Example:

```text
Predict-A-Trade Windows Build

Version:          2.x.x
Architecture:     windows/amd64
Executable:       dist\PredictATradeAgent.exe
Resource Metadata: OK
Signature:        VALID / UNSIGNED
SHA256:           ...
Build:            SUCCESS
```

Do not make the build script install certificates or weaken local Windows security.

---

# 24. FILES EXPECTED

Where applicable, leave the project with:

```text
winres.json
build.ps1
install.ps1
uninstall.ps1
settings.example.json
```

Reuse existing equivalents instead of creating duplicates.

Modify Go source only where required to fix:

* Windows service lifecycle
* configuration loading
* clean shutdown
* crash handling
* health endpoint
* notification support
* Windows-specific path behavior

---

# 25. DO NOT DAMAGE EXISTING PREDICT-A-TRADE FUNCTIONALITY

This is critical.

Do not modify unrelated:

* XAUUSD signal engine
* indicators
* mathematical scoring
* trade management
* MT4 logic
* MT5 logic
* licensing
* device activation
* subscriptions
* billing
* referrals
* admin dashboard
* user dashboard
* database schema

unless the Windows agent directly depends on something that is broken.

Preserve existing:

```text
API contracts
ports
configuration keys
service communication
MT4/MT5 bridge protocol
licensing protocol
```

---

# 26. TEST

Run all applicable existing tests.

Also test:

### Build

```text
Windows binary builds successfully.
```

### Installation

```text
Fresh installation works.
```

### Upgrade

```text
Existing settings are preserved.
```

### Service

```text
start
stop
restart
automatic startup
```

### Configuration

Test:

```text
Telegram
Discord
Email
None
```

### Health

Test:

```text
healthy endpoint
timeout
connection refused
HTTP 500
```

### Failure

Temporarily simulate backend unavailability and confirm the agent does not unnecessarily crash.

### Security

Confirm secrets are not visible in:

```text
logs
command-line arguments
console
```

---

# 27. FINAL REPORT

When complete, return:

```text
PREDICT-A-TRADE WINDOWS AGENT — FINAL STATUS

Root Cause:
Files Modified:
Files Added:

Windows Build: PASS / FAIL
Windows Service: PASS / FAIL
Installer: PASS / FAIL
Configuration Wizard: PASS / FAIL
Telegram: PASS / FAIL / NOT TESTED
Discord: PASS / FAIL / NOT TESTED
Email: PASS / FAIL / NOT TESTED
Health Monitor: PASS / FAIL
Automatic Startup: PASS / FAIL
Logging: PASS / FAIL
Crash Recovery: PASS / FAIL
Authenticode Signing Support: PASS / FAIL
Signature Verification: PASS / FAIL
Defender Diagnostics: PASS / FAIL
Secrets Protection: PASS / FAIL
Uninstaller: PASS / FAIL

Windows Defender Evidence:
SmartScreen Evidence:
Service Exit/Error Evidence:

Remaining External Requirements:
1.
2.

FINAL DECISION:
GO / CONDITIONAL GO / NO-GO
```

## MOST IMPORTANT RULE

The objective is **not to bypass Windows Defender**.

The objective is to make Predict-A-Trade behave like a legitimate, correctly packaged, correctly signed, stable Windows application and to identify the exact reason Windows currently terminates it.

Keep the solution focused, maintainable, and production-ready.

Do not redesign the platform.
Do not over-engineer.
Fix the Windows agent and installer only.
