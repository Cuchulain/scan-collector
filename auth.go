package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const sessionCookieName = "scan_collector_session"

var loginPage = template.Must(template.New("login").Parse(`<!doctype html>
<html lang="cs"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<title data-i18n-page="login">Přihlášení · Scan Collector</title><style>
:root{color-scheme:light;font:16px/1.5 system-ui,sans-serif;color:#17212b;background:#f3f6f8}body{margin:0;min-height:100vh;display:grid;place-items:center}main{box-sizing:border-box;width:min(100% - 32px,420px);padding:30px;background:white;border-radius:12px;box-shadow:0 4px 24px #17212b14}h1{margin:0 0 8px}.muted{color:#5e6b75;margin:0 0 24px}.toolbar{display:flex;justify-content:flex-end;margin-bottom:12px}label.field{display:block;margin:16px 0 5px;font-weight:600}input[type=text],input[type=password]{box-sizing:border-box;width:100%;padding:11px;border:1px solid #aab5bf;border-radius:7px;font:inherit}.remember{display:flex;gap:9px;align-items:flex-start;margin:18px 0}button{width:100%;border:0;border-radius:7px;padding:11px 14px;background:#155eef;color:white;font:inherit;cursor:pointer}button.language-switch{width:auto;background:#e3eaf0;color:#17212b}.error{padding:10px 12px;border-radius:7px;background:#fff0f0;color:#a11;margin:16px 0 0}
</style></head><body><main><div class="toolbar"><button class="language-switch" type="button" data-language-switch>🇨🇿 Čeština</button></div><h1 data-i18n="appName">Scan Collector</h1><p class="muted" data-i18n="loginIntro">Přihlaste se pro zobrazení uložených skenů.</p>
<form method="post" action="/login"><label class="field" for="username" data-i18n="username">Uživatelské jméno</label><input id="username" name="username" type="text" autocomplete="username" required><label class="field" for="password" data-i18n="password">Heslo</label><input id="password" name="password" type="password" autocomplete="current-password" required><label class="remember"><input type="checkbox" name="extended" value="on"><span data-i18n="extendedLogin">Prodloužené přihlášení</span></label><button type="submit" data-i18n="login">Přihlásit se</button></form>
{{if .Error}}<p class="error" role="alert" data-i18n="loginError">Neplatné uživatelské jméno nebo heslo.</p>{{end}}<script src="/i18n.js" defer></script></main></body></html>`))

type loginPageData struct {
	Error bool
}

func openSessionStore(dataDir string) (*sql.DB, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", filepath.Join(dataDir, "sessions.sqlite3"))
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	for _, statement := range []string{
		"PRAGMA journal_mode=WAL",
		`CREATE TABLE IF NOT EXISTS sessions (
			token_hash TEXT PRIMARY KEY NOT NULL,
			expires_at INTEGER NOT NULL
		)`,
		"CREATE INDEX IF NOT EXISTS sessions_expiry ON sessions(expires_at)",
	} {
		if _, err := db.Exec(statement); err != nil {
			_ = db.Close()
			return nil, err
		}
	}
	return db, nil
}

func parseSessionDuration(name, fallback string) (time.Duration, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		value = fallback
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, err
	}
	if duration <= 0 {
		return 0, os.ErrInvalid
	}
	return duration, nil
}

func (r *receiver) hasValidSession(req *http.Request) bool {
	cookie, err := req.Cookie(sessionCookieName)
	if err != nil || r.sessions == nil {
		return false
	}
	var expiresAt int64
	err = r.sessions.QueryRow("SELECT expires_at FROM sessions WHERE token_hash = ?", hashSessionToken(cookie.Value)).Scan(&expiresAt)
	return err == nil && expiresAt > time.Now().Unix()
}

func hashSessionToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func (r *receiver) serveLogin(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		renderLogin(w, false, http.StatusOK)
	case http.MethodPost:
		r.handleLogin(w, req)
	default:
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func renderLogin(w http.ResponseWriter, hasError bool, status int) {
	var body bytes.Buffer
	if err := loginPage.Execute(&body, loginPageData{Error: hasError}); err != nil {
		http.Error(w, "could not render login page", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_, _ = w.Write(body.Bytes())
}

func (r *receiver) handleLogin(w http.ResponseWriter, req *http.Request) {
	req.Body = http.MaxBytesReader(w, req.Body, 64*1024)
	if err := req.ParseForm(); err != nil {
		renderLogin(w, true, http.StatusUnauthorized)
		return
	}
	providedUser := sha256.Sum256([]byte(req.FormValue("username")))
	expectedUser := sha256.Sum256([]byte(r.username))
	providedPassword := sha256.Sum256([]byte(req.FormValue("password")))
	expectedPassword := sha256.Sum256([]byte(r.password))
	userMatches := subtle.ConstantTimeCompare(providedUser[:], expectedUser[:])
	passwordMatches := subtle.ConstantTimeCompare(providedPassword[:], expectedPassword[:])
	if userMatches != 1 || passwordMatches != 1 {
		renderLogin(w, true, http.StatusUnauthorized)
		return
	}

	ttl := r.sessionTTL
	if req.FormValue("extended") == "on" {
		ttl = r.extendedTTL
	}
	var tokenBytes [32]byte
	if _, err := rand.Read(tokenBytes[:]); err != nil {
		http.Error(w, "could not create session", http.StatusInternalServerError)
		return
	}
	token := hex.EncodeToString(tokenBytes[:])
	expiresAt := time.Now().Add(ttl).Unix()
	if r.sessions == nil {
		http.Error(w, "session store unavailable", http.StatusInternalServerError)
		return
	}
	if _, err := r.sessions.Exec("DELETE FROM sessions WHERE expires_at <= ?", time.Now().Unix()); err != nil {
		http.Error(w, "could not save session", http.StatusInternalServerError)
		return
	}
	if _, err := r.sessions.Exec("INSERT INTO sessions(token_hash, expires_at) VALUES (?, ?)", hashSessionToken(token), expiresAt); err != nil {
		http.Error(w, "could not save session", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookieName, Value: token, Path: "/", HttpOnly: true,
		Secure: r.secureCookie, SameSite: http.SameSiteLaxMode,
		Expires: time.Unix(expiresAt, 0), MaxAge: int(ttl / time.Second),
	})
	http.Redirect(w, req, "/", http.StatusSeeOther)
}

func (r *receiver) serveLogout(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if cookie, err := req.Cookie(sessionCookieName); err == nil && r.sessions != nil {
		if _, err := r.sessions.Exec("DELETE FROM sessions WHERE token_hash = ?", hashSessionToken(cookie.Value)); err != nil {
			log.Printf("Session deletion failed: %v", err)
		}
	}
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookieName, Value: "", Path: "/", HttpOnly: true,
		Secure: r.secureCookie, SameSite: http.SameSiteLaxMode,
		Expires: time.Unix(1, 0), MaxAge: -1,
	})
	http.Redirect(w, req, "/login", http.StatusSeeOther)
}
