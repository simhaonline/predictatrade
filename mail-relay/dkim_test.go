package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"strings"
	"testing"
)

// TestDKIMSignRoundTrip — sign a message, then verify the RSA signature
// against the EXACT hash input the signer built (dkimBuildHeader). This proves
// signer/verifier agreement and guards the production signing path
// (2026-09-10 fix: DKIM fields were declared but never used).
// Body tamper detection is covered separately (TestDKIMSignTamperDetection),
// and the relaxed canonicalization rules by TestDKIMBodyCanonicalization.
func TestDKIMSignRoundTrip(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("genkey: %v", err)
	}

	msg := "From: no-reply@predictatrade.com\r\n" +
		"To: someone@example.com\r\n" +
		"Subject: Test\r\n" +
		"Date: Thu, 10 Sep 2026 09:00:00 +0000\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain\r\n" +
		"\r\n" +
		"Hello DKIM world.\r\n"

	signed := string(dkimSign(key, "pat1", "predictatrade.com", msg))
	if !strings.HasPrefix(signed, "DKIM-Signature:") {
		t.Fatalf("signed message must start with DKIM-Signature, got %.40q", signed)
	}
	for _, tag := range []string{"v=1;", "a=rsa-sha256;", "c=relaxed/relaxed;", "d=predictatrade.com;", "s=pat1;", "bh="} {
		if !strings.Contains(signed, tag) {
			t.Errorf("signature missing tag %q", tag)
		}
	}
	// Original message must be preserved verbatim after the sig header.
	if !strings.Contains(signed, msg) {
		t.Errorf("original message not preserved after DKIM header")
	}

	// Signer-path verification: b= must verify against the hashInput from
	// dkimBuildHeader (the exact bytes the signer hashed).
	_, hashInput, ok := dkimBuildHeader("pat1", "predictatrade.com", msg)
	if !ok {
		t.Fatal("dkimBuildHeader returned !ok for well-formed message")
	}
	dhEnd := strings.Index(signed, "\r\n\r\n")
	dh := signed[:dhEnd]
	b64sig := extractBValue(dh)
	sig, err := base64.StdEncoding.DecodeString(b64sig)
	if err != nil {
		t.Fatalf("b= not base64 (%.20q): %v", b64sig, err)
	}
	digest := sha256.Sum256(hashInput)
	if err := rsa.VerifyPKCS1v15(&key.PublicKey, crypto.SHA256, digest[:], sig); err != nil {
		t.Errorf("signature does NOT verify against signer hash input: %v", err)
	}

	// bh= must equal relaxed-canonicalized body of the original message.
	origBody := strings.SplitN(msg, "\r\n\r\n", 2)[1]
	b64bh := extractTag(dh, "bh")
	bh, err := base64.StdEncoding.DecodeString(b64bh)
	if err != nil {
		t.Fatalf("bh not base64: %v", err)
	}
	canonHash := sha256.Sum256([]byte(dkimRelaxedBody(origBody)))
	if string(canonHash[:]) != string(bh) { // raw digest equality (bh IS a digest)
		t.Errorf("bh= mismatch: body hash does not match original body")
	}
}

// extractBValue finds the b= tag value in a DKIM-Signature header. The signer
// emits b= as the last tag with no folding inside the value, so the value ends
// at the first CRLF after the tag (the header block may continue with other
// headers — e.g. when dh spans sig header + original headers).
func extractBValue(dh string) string {
	i := strings.LastIndex(dh, "\r\n	b=")
	if i < 0 {
		i = strings.LastIndex(dh, " b=")
		if i < 0 {
			return ""
		}
		v := dh[i+3:]
		if j := strings.Index(v, "\r\n"); j >= 0 {
			v = v[:j]
		}
		return strings.TrimSpace(v)
	}
	v := dh[i+5:] // "\r\n	b=" is 5 chars
	if j := strings.Index(v, "\r\n"); j >= 0 {
		v = v[:j]
	}
	return strings.TrimSpace(v)
}

// extractTag pulls a DKIM tag value (from tag= to the next ";" or end).
func extractTag(dh, tag string) string {
	i := strings.Index(dh, tag+"=")
	if i < 0 {
		return ""
	}
	v := dh[i+len(tag)+1:]
	if j := strings.Index(v, ";"); j >= 0 {
		v = v[:j]
	}
	return strings.TrimSpace(v)
}

// TestDKIMSignTamperDetection — flipping one body byte must break bh=.
func TestDKIMSignTamperDetection(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	msg := "From: a@predictatrade.com\r\nSubject: x\r\nDate: d\r\nMIME-Version: 1.0\r\n\r\nbody\r\n"
	signed := string(dkimSign(key, "pat1", "predictatrade.com", msg))
	tampered := strings.Replace(signed, "body", "bady", 1)
	if tampered == signed {
		t.Fatal("tamper setup failed — replacement changed nothing")
	}
	// Receiver: strip sig header, canonicalize tampered body, compare to bh=.
	parts := strings.SplitN(tampered, "\r\n\r\n", 2)
	dh := parts[0]
	var tamBody string
	if rp := strings.SplitN(parts[1], "\r\n\r\n", 2); len(rp) == 2 {
		tamBody = rp[1]
	} else {
		tamBody = parts[1]
	}
	b64bh := extractTag(dh, "bh")
	bh, err := base64.StdEncoding.DecodeString(b64bh)
	if err != nil {
		t.Fatalf("bh not base64: %v", err)
	}
	canonHash := sha256.Sum256([]byte(dkimRelaxedBody(tamBody)))
	if string(canonHash[:]) == string(bh) {
		t.Errorf("tampered body hash matched — tamper detection broken")
	}
}

// TestDKIMBodyCanonicalization — relaxed body rules: collapse WS per line,
// strip TRAILING empty lines (internal empties are kept per RFC 6376 §3.4.4),
// no trailing CRLF.
func TestDKIMBodyCanonicalization(t *testing.T) {
	in := "line one  with\tspaces  \r\n\r\n\r\ntrailing empties\r\n\r\n"
	want := "line one with spaces\r\n\r\n\r\ntrailing empties"
	if got := dkimRelaxedBody(in); got != want {
		t.Errorf("canonical body = %q, want %q", got, want)
	}
	// Empty body canonicalizes to empty string (RFC: empty → CRLF → "").
	if got := dkimRelaxedBody("\r\n"); got != "" {
		t.Errorf("empty body = %q, want \"\"", got)
	}
}

// TestDKIMKeyLoadPKCS1 — the generated key format parses.
func TestDKIMKeyLoadPKCS1(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	if k, ok := loadDKIMKeyFromBytes(pemBytes).(*rsa.PrivateKey); !ok {
		t.Fatalf("expected *rsa.PrivateKey, got %T", k)
	}
}

func loadDKIMKeyFromBytes(pemBytes []byte) interface{} {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil
	}
	if k, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return k
	}
	if k, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		return k
	}
	return nil
}