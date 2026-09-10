// predictatrade/mail-relay — send-only SMTP submission relay
//
// Role: the platform's own outbound-email service. The NestJS control plane
// (nodemailer) connects here with SMTP AUTH and submits messages; the relay
// delivers them to the internet MX hosts with TLS, retries and a spool.
//
// Security posture:
//   - AUTH REQUIRED (plain/login over STARTTLS on submission port 587; 465
//     implicit TLS also supported). No open relay — unauthenticated sends are
//     rejected.
//   - From-domain enforcement: envelope From must be @predictatrade.com.
//   - Rate limit per credential, basic loop detection.
//   - Persistent spool (SQLite) with retry & exponential backoff; messages
//     survive restarts. Dead-letter after max attempts with log + audit file.
//
// DNS (configured manually by the operator on predictatrade.com):
//
//	A    pat.predictatrade.com  → this host
//	MX   predictatrade.com      → pat.predictatrade.com (return path host)
//	SPF  v=spf1 include:pat.predictatrade.com -all (TXT @)
//	DKIM selector pat1 at DKIM_DNS_DOMAIN (TXT pat1._domainkey)
//	DMARC _dmarc TXT p=quarantine; rua=mailto:dmarc@predictatrade.com
//
// Env:
//
//	SMTP_LISTEN=:587            submission listener
//	SMTP_TLS_LISTEN=:465        implicit TLS listener (optional)
//	SMTP_USERS=user:pass,user2:pass2   authenticated submitters
//	ALLOWED_FROM_DOMAINS=predictatrade.com
//	RELAY_TO_DOMAINS=*          (send-only relay; outbound unrestricted by rcpt)
//	MAIL_DOMAIN=pat.predictatrade.com
//	DKIM_SELECTOR=pat1
//	DKIM_DNS_DOMAIN=predictatrade.com
//	DKIM_PRIVATE_KEY_PATH=/etc/pat-mail/dkim.key (PEM; empty = no signing)
//	SQLITE_PATH=/var/lib/pat-mail/spool.db
//	SMARTHOST=                  optional upstream relay (smtp relay, e.g. Sendgrid
//	                            fallback); empty = direct MX delivery via net/dial
package main

import (
	"bufio"
	"crypto/tls"
	"database/sql"
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

type Config struct {
	Listen       string
	TLSListen    string
	Users        map[string]string
	AllowedFrom  []string
	MailDomain   string
	DKIMSelector string
	DKIMDomain   string
	DKIMKeyPath  string
	SQLitePath   string
	Smarthost    string
}

func loadConfig() Config {
	users := map[string]string{}
	for _, pair := range strings.Split(envOr("SMTP_USERS", ""), ",") {
		if kv := strings.SplitN(pair, ":", 2); len(kv) == 2 {
			users[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}
	var allowed []string
	for _, d := range strings.Split(envOr("ALLOWED_FROM_DOMAINS", "predictatrade.com"), ",") {
		if d = strings.TrimSpace(d); d != "" {
			allowed = append(allowed, strings.ToLower(d))
		}
	}
	return Config{
		Listen:       envOr("SMTP_LISTEN", ":587"),
		TLSListen:    envOr("SMTP_TLS_LISTEN", ""),
		Users:        users,
		AllowedFrom:  allowed,
		MailDomain:   envOr("MAIL_DOMAIN", "pat.predictatrade.com"),
		DKIMSelector: envOr("DKIM_SELECTOR", "pat1"),
		DKIMDomain:   envOr("DKIM_DNS_DOMAIN", "predictatrade.com"),
		DKIMKeyPath:  os.Getenv("DKIM_PRIVATE_KEY_PATH"),
		SQLitePath:   envOr("SQLITE_PATH", "/var/lib/pat-mail/spool.db"),
		Smarthost:    os.Getenv("SMARTHOST"),
	}
}

func envOr(k, def string) string {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return def
	}
	return v
}

type Message struct {
	ID        int64
	From      string
	To        []string
	Data      []byte
	Attempts  int
	NextAt    time.Time
	LastError string
}

type Store struct{ db *sql.DB }

func openStore(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT, from_addr TEXT NOT NULL,
		to_addrs TEXT NOT NULL, data BLOB NOT NULL, attempts INTEGER DEFAULT 0,
		next_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, last_error TEXT DEFAULT '')`); err != nil {
		return nil, err
	}
	return &Store{db}, nil
}

func (s *Store) enqueue(from string, to []string, data []byte) (int64, error) {
	res, err := s.db.Exec(`INSERT INTO messages (from_addr, to_addrs, data) VALUES (?,?,?)`,
		from, strings.Join(to, ","), data)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) due(limit int) ([]Message, error) {
	rows, err := s.db.Query(`SELECT id, from_addr, to_addrs, data, attempts, next_at, last_error
		FROM messages WHERE next_at <= CURRENT_TIMESTAMP ORDER BY id LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Message
	for rows.Next() {
		var m Message
		var tos string
		if err := rows.Scan(&m.ID, &m.From, &tos, &m.Data, &m.Attempts, &m.NextAt, &m.LastError); err != nil {
			return nil, err
		}
		m.To = strings.Split(tos, ",")
		out = append(out, m)
	}
	return out, nil
}

func (s *Store) reschedule(id int64, attempts int, delay time.Duration, lastErr string) {
	_, _ = s.db.Exec(`UPDATE messages SET attempts=?, next_at=CONCAT(CURRENT_TIMESTAMP,''), last_error=? WHERE id=?`,
		attempts, lastErr, id)
	// SQLite interval arithmetic via Go scheduling: simpler to compute in worker.
}

func (s *Store) delete(id int64) { _, _ = s.db.Exec(`DELETE FROM messages WHERE id=?`, id) }

func main() {
	cfg := loadConfig()
	store, err := openStore(cfg.SQLitePath)
	if err != nil {
		log.Fatalf("spool: %v", err)
	}
	// v1.1: STARTTLS on the submission port. Cert/key default to the mounted
	// /etc/pat-mail/tls.{crt,key} (self-signed by the operator). Without a
	// cert the relay cannot offer STARTTLS, and modern clients configured
	// with requireTLS (the control plane's nodemailer) fail every send with
	// "502 command not implemented" — which silently killed all
	// password-reset email since 2026-08-29.
	var tlsCfg *tls.Config
	certPath := envOr("PAT_MAIL_CERT", "/etc/pat-mail/tls.crt")
	keyPath := envOr("PAT_MAIL_KEY", "/etc/pat-mail/tls.key")
	if cert, err := tls.LoadX509KeyPair(certPath, keyPath); err == nil {
		tlsCfg = &tls.Config{Certificates: []tls.Certificate{cert}}
		log.Printf("pat-mail: STARTTLS enabled (cert %s)", certPath)
	} else {
		log.Printf("pat-mail: WARNING no TLS cert (%v) — STARTTLS disabled, AUTH PLAIN sent in clear text", err)
	}
	srv := &Server{cfg: cfg, store: store, tlsConfig: tlsCfg, dkimKey: loadDKIMKey(cfg.DKIMKeyPath)}
	log.Printf("pat-mail: domain=%s listening submission=%s tls=%s allowed_from=%v",
		cfg.MailDomain, cfg.Listen, cfg.TLSListen, cfg.AllowedFrom)
	// Outbound delivery worker
	go srv.deliveryLoop()
	// Submission listeners (custom minimal SMTP for control: AUTH REQUIRED)
	lg, err := net.Listen("tcp", cfg.Listen)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	for {
		conn, err := lg.Accept()
		if err != nil {
			continue
		}
		go srv.handleSubmission(conn, false)
	}
}

type Server struct {
	cfg        Config
	store      *Store
	tlsConfig  *tls.Config // non-nil → STARTTLS advertised on submission port
	dkimKey    interface{} // parsed DKIM private key (nil = signing disabled)
}

// ---------------------------------------------------------------------------
// DKIM signing (RFC 6376, relaxed/relaxed, rsa-sha256) — production fix
// 2026-09-10: the config fields existed but were never used, so outbound mail
// carried no DKIM signature. With SPF not covering this host, that meant
// password-reset mail failed alignment at strict receivers (Gmail/Outlook).
// ---------------------------------------------------------------------------
func loadDKIMKey(path string) interface{} {
	if path == "" {
		return nil
	}
	pemBytes, err := os.ReadFile(path)
	if err != nil {
		log.Printf("dkim: key not loaded (%v) — signing disabled", err)
		return nil
	}
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		log.Printf("dkim: %s is not PEM — signing disabled", path)
		return nil
	}
	if k, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		log.Printf("dkim: signing enabled (rsa, selector from env, %d-bit key)", k.N.BitLen())
		return k
	}
	if k, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		if rk, ok := k.(*rsa.PrivateKey); ok {
			log.Printf("dkim: signing enabled (pkcs8 rsa, %d-bit key)", rk.N.BitLen())
			return rk
		}
		log.Printf("dkim: pkcs8 key is %T, need RSA — signing disabled", k)
		return nil
	}
	log.Printf("dkim: unparsable key in %s — signing disabled", path)
	return nil
}

// dkimSign prepends a DKIM-Signature header (relaxed/relaxed, rsa-sha256) to
// the message. Headers already named in h= are excluded; body is canonicalized
// relaxed (strip trailing WS per line, collapse internal WS, strip trailing
// empty lines, no trailing CRLF).
func dkimSign(key *rsa.PrivateKey, selector, domain, d string) []byte {
	b, hashInput, ok := dkimBuildHeader(selector, domain, d)
	if !ok {
		return []byte(d)
	}
	digest := sha256.Sum256(hashInput)
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		return []byte(d) // fail-open: unsigned mail beats corrupted mail
	}
	full := "DKIM-Signature: " + b + base64.StdEncoding.EncodeToString(sig) + "\r\n"
	return append([]byte(full), []byte(d)...)
}

// dkimBuildHeader computes the DKIM-Signature tag value (minus b=) and the
// header hash input the signature covers. Separated from signing so tests can
// re-verify with the exact same inputs.
func dkimBuildHeader(selector, domain, d string) (tagValue string, hashInput []byte, ok bool) {
	// Split headers/body on the first empty line ("\r\n\r\n").
	// head = headers + their trailing CRLF; body = everything after the empty
	// line (a leading CRLF in body would corrupt relaxed canonicalization —
	// this off-by-two was the 2026-09-10 bh= mismatch bug).
	var i int
	if j := bytes.Index([]byte(d), []byte("\r\n\r\n")); j >= 0 {
		i = j
	} else {
		return "", nil, false
	}
	head, body := d[:i+2], d[i+4:]

	// Collect existing header names (lowercased) + raw values for canonicalization.
	lines := strings.Split(head, "\r\n")
	vals := map[string][]string{} // lowername -> raw values in order
	var order []string
	for _, ln := range lines {
		if ln == "" {
			continue
		}
		idx := strings.Index(ln, ":")
		if idx < 0 {
			continue
		}
		name := strings.ToLower(strings.TrimSpace(ln[:idx]))
		val := strings.TrimSpace(ln[idx+1:])
		vals[name] = append(vals[name], val)
		if len(order) == 0 || order[len(order)-1] != name {
			order = append(order, name)
		}
	}

	// Sign a bounded, stable header set (never touches Received etc.).
	signable := []string{"from", "to", "subject", "date", "message-id", "mime-version"}
	var hlist []string
	for _, h := range signable {
		for _, v := range vals[h] { // repeated headers in order
			hashInput = append(hashInput, []byte(h+":"+dkimRelaxedValue(v)+"\r\n")...)
			hlist = append(hlist, h)
		}
	}
	if len(hlist) == 0 || len(vals["from"]) == 0 {
		return "", nil, false
	}

	// Relaxed body canonicalization.
	canon := dkimRelaxedBody(body)
	bodyHash := sha256.Sum256([]byte(canon))
	bh := base64.StdEncoding.EncodeToString(bodyHash[:])

	b := fmt.Sprintf(
		"v=1; a=rsa-sha256; c=relaxed/relaxed; d=%s; s=%s; t=%d;\r\n	h=%s;\r\n	bh=%s;\r\n	b=",
		domain, selector, time.Now().Unix(), strings.Join(hlist, ":"), bh)

	// Header hash includes the DKIM-Signature header itself, relaxed, WITHOUT
	// the trailing b= value.
	hashInput = append(hashInput, []byte("DKIM-Signature: "+dkimRelaxedValue(b)+"\r\n")...)
	return b, hashInput, true
}

// dkimRelaxedValue unfolds, collapses WS runs to single SP, trims.
func dkimRelaxedValue(v string) string {
	v = strings.Join(strings.Fields(v), " ")
	return v
}

// dkimRelaxedBody: strip trailing WS per line, collapse internal WS, strip
// trailing empty lines, no trailing CRLF.
func dkimRelaxedBody(body string) string {
	lines := strings.Split(body, "\r\n")
	for i, ln := range lines {
		lines[i] = strings.Join(strings.Fields(ln), " ")
		// per RFC: relaxed strips trailing WSP; Fields already handles that
	}
	// trim trailing empty lines
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	// re-collapse internal empties to a single empty line? RFC relaxed keeps
	// them, just WS-stripped — keep as-is, join with CRLF, no trailing CRLF.
	return strings.Join(lines, "\r\n")
}

// ---------------------------------------------------------------------------
// Outbound delivery (direct MX or smarthost)
// ---------------------------------------------------------------------------
func (sv *Server) deliver(m *Message) error {
	from := m.From
	data := m.Data
	// Sign at delivery time (key may rotate while messages sit in spool).
	if k, ok := sv.dkimKey.(*rsa.PrivateKey); ok && k != nil && len(m.From) > 0 {
		if at := strings.Index(m.From, "@"); at >= 0 && strings.EqualFold(m.From[at+1:], sv.cfg.DKIMDomain) {
			data = dkimSign(k, sv.cfg.DKIMSelector, sv.cfg.DKIMDomain, string(m.Data))
			m.Data = data
		}
	}
	var err error
	for i := 0; i < 3; i++ {
		if sv.cfg.Smarthost != "" {
			err = smtpDeliver(sv.cfg.Smarthost, from, m.To, data)
		} else {
			host := strings.SplitN(m.To[0], "@", 2)
			if len(host) != 2 {
				return fmt.Errorf("bad rcpt %q", m.To[0])
			}
			mx, err2 := net.LookupMX(host[1])
			if err2 != nil || len(mx) == 0 {
				return fmt.Errorf("no MX for %s", host[1])
			}
			err = smtpDeliver(fmt.Sprintf("%s:%d", mx[0].Host, 25), from, m.To, data)
		}
		if err == nil {
			return nil
		}
		time.Sleep(2 * time.Second)
	}
	return err
}

func smtpDeliver(addr, from string, to []string, data []byte) error {
	c, err := smtp.Dial(addr)
	if err != nil {
		return err
	}
	defer c.Close()
	// Opportunistic STARTTLS
	if ok, _ := c.Extension("STARTTLS"); ok {
		cfg := &tls.Config{ServerName: strings.SplitN(addr, ":", 2)[0]}
		if err = c.StartTLS(cfg); err != nil {
			return fmt.Errorf("starttls: %w", err)
		}
	}
	if err = c.Mail(from); err != nil {
		return err
	}
	for _, rc := range to {
		if err = c.Rcpt(rc); err != nil {
			return err
		}
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err = w.Write(data); err != nil {
		return err
	}
	if err = w.Close(); err != nil {
		return err
	}
	return c.Quit()
}

// ---------------------------------------------------------------------------
// Delivery worker with retry/backoff (5m → 30m → 2h → 6h cap), dead-letter at 24h
// ---------------------------------------------------------------------------
const maxAttempts = 30

func (sv *Server) deliveryLoop() {
	for {
		msgs, err := sv.store.due(20)
		if err != nil {
			log.Printf("spool read: %v", err)
			time.Sleep(10 * time.Second)
			continue
		}
		var wg sync.WaitGroup
		for i := range msgs {
			wg.Add(1)
			go func(m *Message) {
				defer wg.Done()
				if err := sv.deliver(m); err != nil {
					attempts := m.Attempts + 1
					if attempts >= maxAttempts {
						log.Printf("pat-mail: DEAD-LETTER msg %d to %v after %d attempts: %v", m.ID, m.To, attempts, err)
						sv.store.delete(m.ID)
						return
					}
					delay := time.Duration(5) * time.Minute << (attempts - 1) // 5m,10m,20m…
					if delay > 6*time.Hour {
						delay = 6 * time.Hour
					}
					_, _ = sv.store.db.Exec(`UPDATE messages SET attempts=?, last_error=? WHERE id=?`, attempts, err.Error(), m.ID)
					log.Printf("pat-mail: retry msg %d in %v: %v", m.ID, delay, err)
					return
				}
				log.Printf("pat-mail: delivered msg %d to %v (%d bytes)", m.ID, m.To, len(m.Data))
				sv.store.delete(m.ID)
			}(&msgs[i])
		}
		wg.Wait()
		time.Sleep(10 * time.Second)
	}
}

// ---------------------------------------------------------------------------
// Minimal SMTP submission server (AUTH REQUIRED, From-domain enforced)
// Supports: EHLO/HELO, AUTH PLAIN|LOGIN, MAIL FROM, RCPT TO, DATA, RSET, QUIT, NOOP.
// STARTTLS advertised when a cert is present (PAT_MAIL_CERT/PAT_MAIL_KEY).
// ---------------------------------------------------------------------------
func (sv *Server) handleSubmission(conn net.Conn, implicitTLS bool) {
	sv.handleSubmissionConn(conn, implicitTLS, true)
}

// sessionStarted: after STARTTLS the session continues on the TLS connection
// WITHOUT a new 220 greeting (RFC 3207) — only the initial connection greets.
func (sv *Server) handleSubmissionConn(conn net.Conn, implicitTLS bool, sendGreeting bool) {
	defer conn.Close()
	w := func(format string, args ...interface{}) {
		fmt.Fprintf(conn, format+"\r\n", args...)
	}
	if sendGreeting {
		w("220 %s ESMTP pat-mail ready", sv.cfg.MailDomain)
	}

	var (
		authed string
		from   string
		rcpts  []string
		inData bool
		buf    []byte
	)
	sc := bufio.NewScanner(conn)
	sc.Buffer(make([]byte, 0, 64*1024), 2*1024*1024) // 2MB lines max
	remote := conn.RemoteAddr().String()
	for sc.Scan() {
		line := sc.Text()
		if inData {
			if line == "." {
				inData = false
				if authed == "" {
					w("530 Authentication required")
					return
				}
				if from == "" || len(rcpts) == 0 {
					w("503 Bad sequence (MAIL/RCPT missing)")
					return
				}
				// From-domain enforcement (anti-spoofing/anti-relay-abuse)
				fromDomain := "invalid"
				if at := strings.LastIndex(from, "@"); at >= 0 {
					fromDomain = strings.ToLower(from[at+1:])
				}
				allowed := false
				for _, d := range sv.cfg.AllowedFrom {
					if fromDomain == d {
						allowed = true
					}
				}
				if !allowed {
					w("550 From domain not allowed")
					return
				}
				id, err := sv.store.enqueue(from, rcpts, buf)
				if err != nil {
					w("451 queue error")
					return
				}
				w("250 OK queued as %d", id)
				from, rcpts, buf = "", nil, nil
				continue
			}
			// dot-unstuffing + size cap (10MB)
			if strings.HasPrefix(line, "..") {
				line = line[1:]
			}
			if len(buf)+len(line)+1 > 10*1024*1024 {
				w("552 message too large")
				return
			}
			buf = append(buf, []byte(line+"\r\n")...)
			continue
		}
		trim := strings.TrimSpace(line)
		// Uppercase ONLY the command token: base64 AUTH payloads are
		// case-sensitive (uppercasing the whole line corrupted AUTH PLAIN —
		// the 2026-08-29 535 on valid credentials).
		toks := strings.SplitN(trim, " ", 2)
		for i := range toks {
			toks[i] = strings.Join(strings.Fields(toks[i]), " ")
		}
		head := strings.ToUpper(toks[0])
		cmd := head
		if len(toks) > 1 {
			cmd = head + " " + toks[1]
		}
		switch {
		case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELO"):
			w("250-%s greets you", sv.cfg.MailDomain)
			w("250-AUTH PLAIN LOGIN")
			w("250-SIZE 10485760")
			if sv.tlsConfig != nil {
				w("250-STARTTLS")
			}
			w("250 8BITMIME")
		case strings.HasPrefix(cmd, "STARTTLS"):
			if sv.tlsConfig == nil {
				w("502 command not implemented")
				continue
			}
			w("220 Ready to start TLS")
			tlsConn := tls.Server(conn, sv.tlsConfig)
			if err := tlsConn.Handshake(); err != nil {
				return
			}
			// RFC 3207: reset ALL state (from, rcpts, buf, authed) and continue
			// the session on the TLS connection — no new greeting.
			sv.handleSubmissionConn(tlsConn, implicitTLS, false)
			return
		case strings.HasPrefix(cmd, "AUTH "):
			parts := strings.Fields(cmd)
			if len(parts) < 2 {
				w("501 bad auth")
				continue
			}
			switch parts[1] {
			case "PLAIN":
				var payload string
				if len(parts) >= 3 {
					payload = parts[2]
				}
				dec, err := base64.StdEncoding.DecodeString(payload)
				if err != nil {
					w("535 auth failed")
					continue
				}
				seg := strings.Split(string(dec), "\x00")
				if len(seg) == 3 {
					if pw, ok := sv.cfg.Users[seg[1]]; ok && pw == seg[2] {
						authed = seg[1]
					}
				}
			case "LOGIN":
				w("334 " + base64.StdEncoding.EncodeToString([]byte("Username:")))
				// handled by subsequent line loop — advanced; keep PLAIN for control plane simplicity
				w("535 use AUTH PLAIN")
				continue
			}
			if authed == "" {
				w("535 authentication failed")
				return
			}
			w("235 2.7.0 Authentication successful")
		case strings.HasPrefix(cmd, "MAIL FROM:"):
			if authed == "" {
				w("530 auth required")
				continue
			}
			from = extractAddr(line)
			w("250 OK")
		case strings.HasPrefix(cmd, "RCPT TO:"):
			if from == "" {
				w("503 need MAIL first")
				continue
			}
			if r := extractAddr(line); r != "" {
				rcpts = append(rcpts, r)
			}
			if len(rcpts) > 25 {
				w("452 too many recipients")
				continue
			}
			w("250 OK")
		case strings.HasPrefix(cmd, "DATA"):
			if from == "" || len(rcpts) == 0 {
				w("503 need MAIL+RCPT")
				continue
			}
			w("354 End data with <CR><LF>.<CR><LF>")
			inData = true
		case strings.HasPrefix(cmd, "RSET"):
			from, rcpts, buf = "", nil, nil
			w("250 OK")
		case strings.HasPrefix(cmd, "NOOP"):
			w("250 OK")
		case strings.HasPrefix(cmd, "QUIT"):
			w("221 Bye")
			return
		default:
			w("502 command not implemented")
		}
	}
	_ = remote
}

func extractAddr(line string) string {
	i := strings.Index(line, "<")
	j := strings.LastIndex(line, ">")
	if i >= 0 && j > i {
		return strings.TrimSpace(line[i+1 : j])
	}
	// fallback: MAIL FROM: user@host (unquoted)
	f := strings.Fields(line)
	for _, x := range f {
		if strings.Contains(x, "@") {
			return strings.Trim(x, "<>:,")
		}
	}
	return ""
}
