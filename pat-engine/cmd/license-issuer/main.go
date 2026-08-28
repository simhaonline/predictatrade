package main

// cmd/license-issuer is a stand-in for the NestJS control plane's signing endpoint.
// It mints HMAC-signed license tokens (plan + allowed_strategies + optional device
// binding + expiry + broker-scalping override) using the SAME license.Sign the engine
// verifies with. In production this logic lives in the control plane; this binary lets
// us exercise the full issue -> validate loop end-to-end without the SaaS yet.
import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"pat-engine/internal/license"
)

func main() {
	secret := flag.String("secret", license.DefaultDevSecret, "HMAC signing secret (control plane holds the real one)")
	plan := flag.String("plan", "PRO", "plan name")
	strats := flag.String("strategies", "ULTRA_SCALPING,STANDARD_SWING,TREND_SWING", "comma-separated allowed strategies (* = all)")
	device := flag.String("device", "", "bind token to a device id (empty = no binding)")
	expiryDays := flag.Int("expiry", 0, "days until expiry (0 = never)")
	scalping := flag.String("scalping", "", "broker scalping override: allow|forbid|'' (unset)")
	validateURL := flag.String("validate", "", "if set, POST the token to this /licensing/validate URL and print the response")
	flag.Parse()

	var allowed []string
	if *strats == "*" {
		allowed = []string{"*"}
	} else {
		for _, s := range strings.Split(*strats, ",") {
			if s = strings.TrimSpace(s); s != "" {
				allowed = append(allowed, s)
			}
		}
	}

	var sc *bool
	switch *scalping {
	case "allow":
		b := true
		sc = &b
	case "forbid":
		b := false
		sc = &b
	}

	exp := int64(0)
	if *expiryDays > 0 {
		exp = time.Now().AddDate(0, 0, *expiryDays).Unix()
	}

	l := &license.License{
		Key:               fmt.Sprintf("CTRL-%s-%d", *plan, time.Now().UnixNano()),
		Plan:              *plan,
		AllowedStrategies: allowed,
		DeviceID:          *device,
		ExpiresAt:         exp,
		BrokerScalping:    sc,
	}
	tok, err := license.Sign(l, *secret)
	if err != nil {
		fmt.Fprintln(os.Stderr, "sign error:", err)
		os.Exit(1)
	}
	fmt.Println(tok)

	if *validateURL != "" {
		body, _ := json.Marshal(map[string]string{"license_key": tok, "device_id": *device})
		resp, err := http.Post(*validateURL, "application/json", bytes.NewReader(body))
		if err != nil {
			fmt.Fprintln(os.Stderr, "validate error:", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		var out bytes.Buffer
		_, _ = out.ReadFrom(resp.Body)
		fmt.Println("validate ->", out.String())
	}
}
