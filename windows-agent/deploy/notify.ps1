<#
.SYNOPSIS
    Predict-A-Trade XAUUSD — Notification Dispatcher with Windows Popup
.DESCRIPTION
    Sends crash/hang/restart notifications via:
    1. Windows toast notification (popup in bottom-right corner)
    2. Windows message box popup (blocking dialog — user must click OK)
    3. msg.exe broadcast (sends message to all logged-in users)
    4. Telegram / Discord / Email (external notifications)
.PARAMETER ExitCode
    The exit code of the agent process. Special values:
      0    = Clean stop (manual or normal shutdown)
      -999 = Health check detected hang and force-restarted
      1-5  = Various crash reasons
.NOTES
    Reads credentials from settings.json in the same directory.
    All actions are logged to the Windows Application Event Log.
#>

param(
    [Parameter(Mandatory=$true)]
    [int]$ExitCode
)

$ErrorActionPreference = "Stop"

# ─── Paths ───
$ScriptDir   = Split-Path -Parent $MyInvocation.MyCommand.Path
$SettingsFile = Join-Path $ScriptDir "settings.json"
$EventSource  = "pat-agent"

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
        $fallbackLog = Join-Path $ScriptDir "logs\notify_fallback.log"
        $dir = Split-Path -Parent $fallbackLog
        if (-not (Test-Path $dir)) { New-Item -ItemType Directory -Path $dir -Force | Out-Null }
        Add-Content -Path $fallbackLog -Value "[$(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')] [$Level] $Message"
    }
}

# ─── Helper: Show Windows Toast Notification ───
function Show-WindowsToast {
    param([string]$Title, [string]$Body)

    # Method 1: Try BurntToast module (if installed)
    if (Get-Module -ListAvailable -Name BurntToast -ErrorAction SilentlyContinue) {
        try {
            New-BurntToastNotification -Text $Title, $Body -AppLogo $null
            return
        } catch {}
    }

    # Method 2: Try raw Windows Runtime toast (Windows 10+)
    try {
        [Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
        [Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime] | Out-Null

        $template = @"
<toast>
    <visual>
        <binding template="ToastGeneric">
            <text>$Title</text>
            <text>$Body</text>
        </binding>
    </visual>
    <audio src="ms-winsoundevent:Notification.Default"/>
</toast>
"@

        $xml = New-Object Windows.Data.Xml.Dom.XmlDocument
        $xml.LoadXml($template)
        $toast = [Windows.UI.Notifications.ToastNotification]::new($xml)
        $notifier = [Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier("PredictATrade")
        $notifier.Show($toast)
        return
    } catch {}

    # Method 3: Fall back to msg.exe (sends popup to all logged-in users)
    try {
        $msgBody = "$Title`n`n$Body"
        & msg.exe * /TIME:60 $msgBody 2>&1 | Out-Null
    } catch {}
}

# ─── Helper: Show Windows Message Box Popup ───
function Show-WindowsPopup {
    param([string]$Title, [string]$Body, [string]$Icon = "Warning")

    try {
        Add-Type -AssemblyName System.Windows.Forms
        $iconEnum = switch ($Icon) {
            "Error"   { [System.Windows.Forms.MessageBoxIcon]::Error }
            "Warning" { [System.Windows.Forms.MessageBoxIcon]::Warning }
            default   { [System.Windows.Forms.MessageBoxIcon]::Warning }
        }
        # Show in a background thread so it doesn't block the service restart
        $params = @{
            Body  = $Body
            Title = $Title
            Icon  = $iconEnum
        }
        $job = Start-Job -ScriptBlock {
            param($Body, $Title, $Icon)
            Add-Type -AssemblyName System.Windows.Forms
            [System.Windows.Forms.MessageBox]::Show($Body, $Title, [System.Windows.Forms.MessageBoxButtons]::OK, $Icon) | Out-Null
        } -ArgumentList $Body, $Title, $iconEnum
        # Don't wait for the job — let it show the popup while we continue
    } catch {
        Write-PATEventLog -Message "notify.ps1: Could not show Windows popup: $_" -Level "Warning" -EventId 210
    }
}

# ─── Load settings ───
if (-not (Test-Path $SettingsFile)) {
    Write-PATEventLog -Message "notify.ps1: settings.json not found (exit code: $ExitCode)" -Level "Warning" -EventId 200
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
    -999   { "HEALTH CHECK: Agent was hung and has been force-restarted" }
    1      { "Crash (exit code 1 — general error)" }
    2      { "Crash (exit code 2 — configuration error)" }
    3      { "Crash (exit code 3 — connection error)" }
    4      { "Crash (exit code 4 — license/activation failure)" }
    5      { "Crash (exit code 5 — pipe/IPC error)" }
    default { "Crash (unexpected exit code: $ExitCode)" }
}

$popupTitle = "Predict-A-Trade XAUUSD Alert"
$popupBody = @"
$reasonText
Host: $hostname
Time: $timestamp

The agent will auto-restart in 5 seconds.
If it keeps crashing, run:
  irm https://downloads.predictatrade.com/windows-agent/install.ps1 | iex
"@

$fullMessage = @"
[Predict-A-Trade XAUUSD Alert]
Host:       $hostname
Timestamp:  $timestamp
Exit Code:  $ExitCode
Reason:     $reasonText
Service:    pat-agent
"@

Write-Host $fullMessage

# ─── 1. ALWAYS show Windows popup (unless clean exit code 0) ───
if ($ExitCode -ne 0) {
    $showPopup = $true
    if ($null -ne $settings.show_popup_notifications) {
        $showPopup = [bool]$settings.show_popup_notifications
    }
    if ($showPopup) {
        # Show toast notification (non-blocking, appears in notification center)
        Show-WindowsToast -Title $popupTitle -Body $reasonText
        # Show message box popup (blocking dialog — user must click OK)
        Show-WindowsPopup -Title $popupTitle -Body $popupBody -Icon "Warning"
    }
}

# ─── 2. Send external notification (Telegram/Discord/Email) ───
$notifType = $settings.notification_type
$notified  = $false

switch ($notifType) {
    "telegram" {
        $botToken = $settings.telegram_bot_token
        $chatId   = $settings.telegram_chat_id
        if ([string]::IsNullOrWhiteSpace($botToken) -or $botToken -eq "YOUR_BOT_TOKEN") {
            Write-PATEventLog -Message "notify.ps1: Telegram not configured — skipping" -Level "Warning" -EventId 202
            break
        }
        try {
            $uri = "https://api.telegram.org/bot$botToken/sendMessage"
            $body = @{ chat_id = $chatId; text = $fullMessage; parse_mode = "HTML" } | ConvertTo-Json
            $resp = Invoke-WebRequest -Uri $uri -Method Post -Body $body -ContentType "application/json" -UseBasicParsing -TimeoutSec 10
            if ($resp.StatusCode -eq 200) {
                $notified = $true
                Write-Host "[notify] Telegram notification sent."
            }
        } catch {
            Write-PATEventLog -Message "notify.ps1: Telegram send failed: $_" -Level "Error" -EventId 203
        }
    }
    "discord" {
        $webhook = $settings.discord_webhook
        if ([string]::IsNullOrWhiteSpace($webhook) -or $webhook -eq "YOUR_DISCORD_WEBHOOK_URL") {
            Write-PATEventLog -Message "notify.ps1: Discord not configured — skipping" -Level "Warning" -EventId 204
            break
        }
        try {
            $body = @{ content = $fullMessage } | ConvertTo-Json
            $resp = Invoke-WebRequest -Uri $webhook -Method Post -Body $body -ContentType "application/json" -UseBasicParsing -TimeoutSec 10
            if ($resp.StatusCode -eq 204 -or $resp.StatusCode -eq 200) {
                $notified = $true
                Write-Host "[notify] Discord notification sent."
            }
        } catch {
            Write-PATEventLog -Message "notify.ps1: Discord send failed: $_" -Level "Error" -EventId 205
        }
    }
    "email" {
        $smtp = $settings.email_smtp
        $to   = $settings.email_to
        $from = $settings.email_from
        $port = if ($settings.email_port) { $settings.email_port } else { 587 }
        $user = $settings.email_user
        $pass = $settings.email_password
        if ([string]::IsNullOrWhiteSpace($smtp) -or $smtp -eq "smtp.yourdomain.com") {
            Write-PATEventLog -Message "notify.ps1: Email not configured — skipping" -Level "Warning" -EventId 206
            break
        }
        try {
            $subject = "[Predict-A-Trade] Alert: $reasonText on $hostname"
            $smtpObj = New-Object Net.Mail.SmtpClient($smtp, $port)
            if ($user -and $pass) {
                $smtpObj.EnableSsl = $true
                $smtpObj.Credentials = New-Object System.Net.NetworkCredential($user, $pass)
            }
            $mailMsg = New-Object Net.Mail.MailMessage($from, $to, $subject, $fullMessage)
            $smtpObj.Send($mailMsg)
            $notified = $true
            Write-Host "[notify] Email notification sent."
        } catch {
            Write-PATEventLog -Message "notify.ps1: Email send failed: $_" -Level "Error" -EventId 207
        }
    }
    default {
        Write-PATEventLog -Message "notify.ps1: Unknown notification_type '$notifType'" -Level "Warning" -EventId 208
    }
}

# ─── Log result ───
if ($notified) {
    Write-PATEventLog -Message "notify.ps1: Notification sent via $notifType (exit code $ExitCode)" -EventId 100
} else {
    Write-PATEventLog -Message "notify.ps1: No external notification sent (type=$notifType, exitCode=$ExitCode) — Windows popup shown" -Level "Warning" -EventId 209
}

exit 0
