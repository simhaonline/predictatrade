<#
.SYNOPSIS
    Predict-A-Trade XAUUSD — Notification Dispatcher
.DESCRIPTION
    Sends crash/hang/restart notifications via Telegram, Discord, or Email.
    Triggered by NSSM on service exit or by health-check.ps1 on hang detection.
.PARAMETER ExitCode
    The exit code of the agent process. Special values:
      0    = Clean stop (manual or normal shutdown)
      -999 = Health check detected hang and force-restarted
      1-5  = Various crash reasons
.NOTES
    Reads credentials from settings.json in the same directory.
    All actions are logged to the Windows Application Event Log under
    source "PredictATradeXAUUSD".
#>

param(
    [Parameter(Mandatory=$true)]
    [int]$ExitCode
)

$ErrorActionPreference = "Stop"

# ─── Paths ───
$ScriptDir   = Split-Path -Parent $MyInvocation.MyCommand.Path
$SettingsFile = Join-Path $ScriptDir "settings.json"
$EventSource  = "PredictATradeXAUUSD"

# ─── Helper: Write to Event Log ───
function Write-PATEventLog {
    param([string]$Message, [string]$Level = "Information", [int]$EventId = 100)
    try {
        if (-not [System.Diagnostics.EventLog]::SourceExists($EventSource)) {
            [System.Diagnostics.EventLog]::CreateEventSource($EventSource, "Application")
        }
        $log = New-Object System.Diagnostics.EventLog("Application")
        $log.Source = $EventSource
        $entryType = switch ($Level) {
            "Error"       { [System.Diagnostics.EventLogEntryType]::Error }
            "Warning"     { [System.Diagnostics.EventLogEntryType]::Warning }
            default       { [System.Diagnostics.EventLogEntryType]::Information }
        }
        $log.WriteEntry($Message, $entryType, $EventId)
    } catch {
        # Fallback: write to a local log file if Event Log is unavailable
        $fallbackLog = Join-Path $ScriptDir "logs\notify_fallback.log"
        $dir = Split-Path -Parent $fallbackLog
        if (-not (Test-Path $dir)) { New-Item -ItemType Directory -Path $dir -Force | Out-Null }
        Add-Content -Path $fallbackLog -Value "[$(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')] [$Level] $Message"
    }
}

# ─── Load settings ───
if (-not (Test-Path $SettingsFile)) {
    Write-PATEventLog -Message "notify.ps1: settings.json not found at $SettingsFile — cannot send notification (exit code: $ExitCode)" -Level "Warning" -EventId 200
    exit 1
}

try {
    $settings = Get-Content $SettingsFile -Raw | ConvertFrom-Json
} catch {
    Write-PATEventLog -Message "notify.ps1: Failed to parse settings.json: $_" -Level "Error" -EventId 201
    exit 1
}

# ─── Build message ───
$hostname   = $env:COMPUTERNAME
$timestamp  = Get-Date -Format "yyyy-MM-dd HH:mm:ss zzz"
$reasonText = switch ($ExitCode) {
    0      { "Manual stop / clean shutdown" }
    -999   { "HEALTH CHECK: Agent is hung and has been force-restarted" }
    1      { "Crash (exit code 1 — general error)" }
    2      { "Crash (exit code 2 — configuration error)" }
    3      { "Crash (exit code 3 — connection error)" }
    4      { "Crash (exit code 4 — license/activation failure)" }
    5      { "Crash (exit code 5 — pipe/IPC error)" }
    default { "Crash (unexpected exit code: $ExitCode)" }
}

$message = @"
[Predict-A-Trade XAUUSD Alert]
Host:       $hostname
Timestamp:  $timestamp
Exit Code:  $ExitCode
Reason:     $reasonText
Service:    PredictATradeXAUUSD
"@

Write-Host $message

# ─── Send notification ───
$notifType = $settings.notification_type
$notified  = $false

switch ($notifType) {
    "telegram" {
        $botToken = $settings.telegram_bot_token
        $chatId   = $settings.telegram_chat_id
        if ([string]::IsNullOrWhiteSpace($botToken) -or $botToken -eq "YOUR_BOT_TOKEN") {
            Write-PATEventLog -Message "notify.ps1: Telegram bot token not configured — skipping notification" -Level "Warning" -EventId 202
            break
        }
        try {
            $uri = "https://api.telegram.org/bot$botToken/sendMessage"
            $body = @{ chat_id = $chatId; text = $message; parse_mode = "HTML" } | ConvertTo-Json
            $resp = Invoke-WebRequest -Uri $uri -Method Post -Body $body -ContentType "application/json" -UseBasicParsing -TimeoutSec 10
            if ($resp.StatusCode -eq 200) {
                $notified = $true
                Write-Host "[notify] Telegram notification sent successfully."
            }
        } catch {
            Write-PATEventLog -Message "notify.ps1: Telegram send failed: $_" -Level "Error" -EventId 203
        }
    }
    "discord" {
        $webhook = $settings.discord_webhook
        if ([string]::IsNullOrWhiteSpace($webhook) -or $webhook -eq "YOUR_DISCORD_WEBHOOK_URL") {
            Write-PATEventLog -Message "notify.ps1: Discord webhook not configured — skipping notification" -Level "Warning" -EventId 204
            break
        }
        try {
            $body = @{ content = $message } | ConvertTo-Json
            $resp = Invoke-WebRequest -Uri $webhook -Method Post -Body $body -ContentType "application/json" -UseBasicParsing -TimeoutSec 10
            if ($resp.StatusCode -eq 204 -or $resp.StatusCode -eq 200) {
                $notified = $true
                Write-Host "[notify] Discord notification sent successfully."
            }
        } catch {
            Write-PATEventLog -Message "notify.ps1: Discord send failed: $_" -Level "Error" -EventId 205
        }
    }
    "email" {
        $smtp   = $settings.email_smtp
        $to     = $settings.email_to
        $from   = $settings.email_from
        $port   = if ($settings.email_port) { $settings.email_port } else { 587 }
        $user   = $settings.email_user
        $pass   = $settings.email_password
        if ([string]::IsNullOrWhiteSpace($smtp) -or $smtp -eq "smtp.yourdomain.com") {
            Write-PATEventLog -Message "notify.ps1: Email SMTP not configured — skipping notification" -Level "Warning" -EventId 206
            break
        }
        try {
            $subject = "[Predict-A-Trade] Alert: $reasonText on $hostname"
            $smtpObj = New-Object Net.Mail.SmtpClient($smtp, $port)
            if ($user -and $pass) {
                $smtpObj.EnableSsl = $true
                $smtpObj.Credentials = New-Object System.Net.NetworkCredential($user, $pass)
            }
            $mailMsg = New-Object Net.Mail.MailMessage($from, $to, $subject, $message)
            $smtpObj.Send($mailMsg)
            $notified = $true
            Write-Host "[notify] Email notification sent successfully."
        } catch {
            Write-PATEventLog -Message "notify.ps1: Email send failed: $_" -Level "Error" -EventId 207
        }
    }
    default {
        Write-PATEventLog -Message "notify.ps1: Unknown notification_type '$notifType' in settings.json" -Level "Warning" -EventId 208
    }
}

# ─── Log result ───
if ($notified) {
    Write-PATEventLog -Message "notify.ps1: Notification sent via $notifType for exit code $ExitCode ($reasonText)" -EventId 100
} else {
    Write-PATEventLog -Message "notify.ps1: No notification sent (type=$notifType, exitCode=$ExitCode) — check settings.json configuration" -Level "Warning" -EventId 209
}

exit 0
