package agent

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Updater handles secure agent updates.
// SOW Section 23: Secure agent updates with version discovery, HTTPS download,
// checksum validation, atomic replacement, rollback.
//
// The updater fetches a static update-manifest.json from the download server
// (no NestJS API endpoint required). On Windows, the running binary is locked,
// so updates are applied via a helper batch script that stops the service,
// swaps the binary, and restarts.
type Updater struct {
	manifestURL   string
	fallbackURL   string // per-role (amd64) manifest, tried if the arch-specific one is missing
	currentVer    string
	dataDir       string
	updateChannel string
	serviceName   string // Windows service name to stop/start during swap (e.g. pat-agent-client)
}

// UpdateManifest represents the update metadata from the server.
type UpdateManifest struct {
	Version      string `json:"version"`
	DownloadURL  string `json:"download_url"`
	Checksum     string `json:"checksum"`     // SHA-256 hex
	Signature    string `json:"signature"`    // RSA signature (future)
	MinVersion   string `json:"min_version"`  // Minimum required version (downgrade protection)
	ReleaseNotes string `json:"release_notes"`
	Timestamp    string `json:"timestamp"`
}

// NewUpdater creates a new updater instance.
// manifestURL is the full URL to the arch-specific update-manifest.json on the
// download server. fallbackURL is the per-role (amd64) manifest used when the
// arch-specific one is unavailable, so updates never dead-end. serviceName is the
// exact Windows service name (must match what the installer registered).
func NewUpdater(manifestURL, fallbackURL, currentVer, dataDir, channel, serviceName string) *Updater {
	return &Updater{
		manifestURL:   manifestURL,
		fallbackURL:   fallbackURL,
		currentVer:    currentVer,
		dataDir:       dataDir,
		updateChannel: channel,
		serviceName:   serviceName,
	}
}

// updaterPublicKeyPEM is the RSA public key used to verify update-manifest
// signatures (W9). PRODUCTION MUST ROTATE THIS: replace with the public half
// of the code-signing keypair. While this is the placeholder (empty string),
// signature verification is SKIPPED and only the SHA-256 checksum is enforced,
// so updates still work in dev/test. Once a real key is embedded, a manifest
// without a valid signature is refused.
const updaterPublicKeyPEM = ""

// verifyManifestSignature verifies an RSA-PKCS1v15/SHA-256 signature over the
// canonical manifest payload (version|download_url|checksum|min_version|timestamp).
// Returns nil when the signature is valid. When no production key is embedded it
// logs a warning and skips verification (checksum-only mode).
func (u *Updater) verifyManifestSignature(manifest *UpdateManifest) error {
	if strings.TrimSpace(updaterPublicKeyPEM) == "" {
		logf("[updater] WARNING: no production code-signing public key embedded — SKIPPING signature verification (checksum-only). Set updaterPublicKeyPEM before production.")
		return nil
	}
	if strings.TrimSpace(manifest.Signature) == "" {
		return fmt.Errorf("update manifest has no signature — refusing to update")
	}
	block, _ := pem.Decode([]byte(updaterPublicKeyPEM))
	if block == nil {
		return fmt.Errorf("invalid embedded public key PEM")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse public key: %w", err)
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return fmt.Errorf("embedded key is not RSA")
	}
	signed := strings.Join([]string{
		manifest.Version,
		manifest.DownloadURL,
		manifest.Checksum,
		manifest.MinVersion,
		manifest.Timestamp,
	}, "|")
	digest := sha256.Sum256([]byte(signed))
	sig, err := base64.StdEncoding.DecodeString(manifest.Signature)
	if err != nil {
		return fmt.Errorf("manifest signature not base64: %w", err)
	}
	if err := rsa.VerifyPKCS1v15(rsaPub, crypto.SHA256, digest[:], sig); err != nil {
		return fmt.Errorf("manifest signature verification FAILED: %w", err)
	}
	return nil
}

// fetchManifest GETs a manifest URL and decodes it. It returns (manifest, status, err).
// A 204 means up-to-date, a non-200 non-204 returns manifest=nil with that status.
func (u *Updater) fetchManifest(url string) (*UpdateManifest, int, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, 0, fmt.Errorf("manifest fetch failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 204 {
		return nil, 204, nil
	}
	if resp.StatusCode != 200 {
		return nil, resp.StatusCode, nil
	}
	var m UpdateManifest
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to decode update manifest: %w", err)
	}
	return &m, 200, nil
}

// CheckForUpdate fetches the manifest from the server and compares versions.
// Returns the manifest if an update is available, nil if up-to-date.
func (u *Updater) CheckForUpdate() (*UpdateManifest, error) {
	manifest, status, err := u.fetchManifest(u.manifestURL)
	if err != nil {
		return nil, fmt.Errorf("update check failed: %w", err)
	}
	if status == 204 {
		return nil, nil // Up to date
	}
	// Arch-specific manifest missing (404) → fall back to the per-role (amd64) one
	// so the fleet still receives updates.
	if status == 404 && u.fallbackURL != "" && u.fallbackURL != u.manifestURL {
		if fb, fbStatus, fbErr := u.fetchManifest(u.fallbackURL); fbErr == nil && fbStatus == 200 {
			manifest, status = fb, 200
		}
	}
	if status != 200 {
		return nil, fmt.Errorf("update check returned HTTP %d", status)
	}

	// W9: verify the manifest signature before trusting any field (e.g. the
	// download URL could be redirected to a malicious binary).
	if err := u.verifyManifestSignature(manifest); err != nil {
		return nil, fmt.Errorf("update manifest rejected: %w", err)
	}

	// P0-WA1 fix: numeric semantic-version comparison.
	// The previous lexicographic string compare made "1.2.16" < "1.2.6",
	// leaving most of the fleet stuck on old versions (observed in prod).
	if belowMin, err := versionLessThan(u.currentVer, manifest.MinVersion); err == nil && manifest.MinVersion != "" && belowMin {
		// Below the floor is not an abort condition — it FORCES the update.
		return manifest, nil
	}

	if newer, err := versionLessThan(u.currentVer, manifest.Version); err == nil && !newer {
		return nil, nil // Already up to date or newer
	}

	return manifest, nil
}

// versionLessThan returns true if a < b under numeric semver comparison
// (major.minor.patch; missing components treated as 0).
func versionLessThan(a, b string) (bool, error) {
	parse := func(v string) ([3]int, error) {
		var out [3]int
		v = strings.TrimPrefix(strings.TrimSpace(v), "v")
		parts := strings.SplitN(v, ".", 3)
		if len(parts) == 0 {
			return out, fmt.Errorf("empty version")
		}
		for i := 0; i < len(parts) && i < 3; i++ {
			n, err := strconv.Atoi(parts[i])
			if err != nil {
				return out, fmt.Errorf("invalid version component %q: %w", parts[i], err)
			}
			out[i] = n
		}
		return out, nil
	}
	pa, err := parse(a)
	if err != nil {
		return false, err
	}
	pb, err := parse(b)
	if err != nil {
		return false, err
	}
	for i := 0; i < 3; i++ {
		if pa[i] != pb[i] {
			return pa[i] < pb[i], nil
		}
	}
	return false, nil
}

// DownloadAndVerify downloads the update, verifies checksum, and stages it.
// SOW Section 23: "Never execute an unverified downloaded binary."
func (u *Updater) DownloadAndVerify(manifest *UpdateManifest) (string, error) {
	stagingDir := filepath.Join(u.dataDir, "updates", "staging")
	if err := os.MkdirAll(stagingDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create staging dir: %w", err)
	}

	stagedPath := filepath.Join(stagingDir, fmt.Sprintf("agent_%s.exe", manifest.Version))

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(manifest.DownloadURL)
	if err != nil {
		return "", fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("download returned HTTP %d", resp.StatusCode)
	}

	tmpFile, err := os.Create(stagedPath + ".tmp")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tmpFile.Close()

	hasher := sha256.New()
	_, err = io.Copy(io.MultiWriter(tmpFile, hasher), resp.Body)
	if err != nil {
		os.Remove(stagedPath + ".tmp")
		return "", fmt.Errorf("download write failed: %w", err)
	}
	tmpFile.Close()

	// Verify checksum
	actualChecksum := hex.EncodeToString(hasher.Sum(nil))
	if actualChecksum != manifest.Checksum {
		os.Remove(stagedPath + ".tmp")
		return "", fmt.Errorf("checksum mismatch: expected %s, got %s", manifest.Checksum, actualChecksum)
	}

	if err := os.Rename(stagedPath+".tmp", stagedPath); err != nil {
		os.Remove(stagedPath + ".tmp")
		return "", fmt.Errorf("failed to stage update: %w", err)
	}

	return stagedPath, nil
}

// ApplyUpdateOnWindows performs a safe binary swap on Windows where the
// running executable is locked. It creates a helper batch script that:
//  1. Waits 2 seconds (for the agent to exit)
//  2. Stops the NSSM service
//  3. Backs up the current binary
//  4. Replaces it with the staged binary
//  5. Updates version.txt
//  6. Restarts the service
//  7. Deletes itself and the backup
//
// The batch script runs outside the agent process, so it can replace the
// locked binary after the service stops.
func (u *Updater) ApplyUpdateOnWindows(stagedPath, currentPath string, manifest *UpdateManifest) error {
	installDir := filepath.Dir(currentPath)
	backupPath := currentPath + ".bak"
	versionFile := filepath.Join(installDir, "version.txt")
	nssmPath := filepath.Join(installDir, "nssm.exe")
	// Use the exact service name registered by the installer (pat-agent-client /
	// pat-agent-master). A wrong name would stop/start the wrong service and the
	// update would never take effect, forcing a manual reinstall.
	serviceName := u.serviceName
	if serviceName == "" {
		serviceName = "pat-agent-client"
	}

	// Build the helper batch script
	batchContent := fmt.Sprintf(`@echo off
setlocal enabledelayedexpansion

rem === Predict-A-Trade Auto-Update Helper ===
rem This script is auto-generated by the agent updater.
rem It runs outside the agent process to swap the locked binary.

echo [update] Waiting for agent to exit...
timeout /t 3 /nobreak > nul

echo [update] Stopping service...
if exist "%s" (
  "%s" stop %s 2>nul
) else (
  sc stop %s 2>nul
)
timeout /t 2 /nobreak > nul

echo [update] Backing up current binary...
if exist "%s" move /Y "%s" "%s" 2>nul

echo [update] Installing new binary...
move /Y "%s" "%s" 2>nul

echo [update] Removing Mark-of-the-Web (Unblock-File) so Defender/SmartScreen won't block it...
powershell -NoProfile -Command "Unblock-File -Path '%s' -ErrorAction SilentlyContinue" 2>nul
echo [update] Ensuring Windows Defender exclusion for the install directory...
powershell -NoProfile -Command "Add-MpPreference -ExclusionPath '%s' -ErrorAction SilentlyContinue" 2>nul
echo [update] VERIFY: if Tamper Protection blocks the line above, the exclusion is NOT active.
echo [update] If the service fails to start now, add a Defender folder exclusion MANUALLY:
echo [update]   Windows Security ^> Virus ^& threat protection ^> Exclusions ^> Folder: %s
echo [update] then run: sc start %s

echo [update] Updating version file...
echo %s> "%s"

echo [update] Starting service...
if exist "%s" (
  "%s" start %s 2>nul
) else (
  sc start %s 2>nul
)

echo [update] Cleaning up...
if exist "%s" del /F /Q "%s" 2>nul
del /F /Q "%%~f0" 2>nul

echo [update] Done.
`, nssmPath, nssmPath, serviceName,
		serviceName,
		currentPath, currentPath, backupPath,
		stagedPath, currentPath,
		currentPath, installDir,
		manifest.Version, versionFile,
		nssmPath, nssmPath, serviceName,
		serviceName,
		backupPath, backupPath,
		installDir, serviceName)

	// Write the batch script
	batchPath := filepath.Join(installDir, "pat_update_helper.bat")
	if err := os.WriteFile(batchPath, []byte(batchContent), 0755); err != nil {
		return fmt.Errorf("failed to write update helper: %w", err)
	}

	// Log the update
	logf("[updater] Update staged at %s, helper script at %s", stagedPath, batchPath)
	logf("[updater] Helper script will stop service, swap binary, and restart")

	// Execute the helper batch script asynchronously (non-blocking)
	// It runs in a separate cmd.exe process, so the agent can exit safely
	cmd := exec.Command("cmd.exe", "/c", "start", "/min", "", batchPath)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to launch update helper: %w", err)
	}

	return nil
}

// RollbackUpdate restores the previous binary if the update fails.
func (u *Updater) RollbackUpdate(currentPath string) error {
	backupPath := currentPath + ".bak"
	if _, err := os.Stat(backupPath); err != nil {
		return fmt.Errorf("no backup found for rollback")
	}

	_ = os.Remove(currentPath)
	if err := os.Rename(backupPath, currentPath); err != nil {
		return fmt.Errorf("rollback failed: %w", err)
	}

	return nil
}
