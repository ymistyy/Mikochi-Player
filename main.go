package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
	"time"
)

const mediaRoot = "/"

type Config struct {
	Listen          string `json:"listen"`
	MikochiURL      string `json:"mikochi_url"`
	MikochiUsername string `json:"mikochi_username"`
	MikochiPassword string `json:"mikochi_password"`
	SessionTTLHours int    `json:"session_ttl_hours"`
}

type App struct {
	cfg          Config
	apiClient    *http.Client
	streamClient *http.Client
	sessionMu    sync.RWMutex
	sessions     map[string]time.Time
	mikochiMu    sync.Mutex
	mikochiJWT   string
}

type browseResponse struct {
	FileInfos []fileInfo `json:"fileInfos"`
	IsRoot    bool       `json:"isRoot"`
}

type fileInfo struct {
	Path  string `json:"path"`
	Size  int64  `json:"size"`
	IsDir bool   `json:"isDir"`
}

type loginResponse struct {
	Token string `json:"token"`
}

func main() {
	cfg, err := loadConfig("config.json")
	if err != nil {
		log.Fatal(err)
	}
	if cfg.Listen == "" {
		cfg.Listen = "127.0.0.1:8090"
	}
	if cfg.SessionTTLHours <= 0 {
		cfg.SessionTTLHours = 24
	}
	if cfg.MikochiURL == "" || cfg.MikochiUsername == "" || cfg.MikochiPassword == "" {
		log.Fatal("mikochi_url, mikochi_username and mikochi_password are required")
	}

	app := &App{
		cfg:          cfg,
		apiClient:    &http.Client{Timeout: 15 * time.Second},
		streamClient: &http.Client{Timeout: 0},
		sessions:     make(map[string]time.Time),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", app.static)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.HandleFunc("/api/login", app.login)
	mux.HandleFunc("/api/logout", app.logout)
	mux.HandleFunc("/api/me", app.me)
	mux.HandleFunc("/api/browse", app.browse)
	mux.HandleFunc("/api/stream", app.stream)
	mux.HandleFunc("/api/subtitle", app.subtitle)

	server := &http.Server{
		Addr:              cfg.Listen,
		Handler:           securityHeaders(logging(mux)),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	log.Printf("Mikochi Player listening on http://%s", cfg.Listen)
	log.Fatal(server.ListenAndServe())
}

func loadConfig(filename string) (Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return Config{}, fmt.Errorf("read %s: %w", filename, err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse %s: %w", filename, err)
	}
	return cfg, nil
}

func (a *App) static(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, "static/index.html")
}

func (a *App) login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var in struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 8<<10)).Decode(&in); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if !constantTimeEqual(in.Username, a.cfg.MikochiUsername) ||
		!constantTimeEqual(in.Password, a.cfg.MikochiPassword) {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	session, err := randomToken(32)
	if err != nil {
		http.Error(w, "failed to create session", http.StatusInternalServerError)
		return
	}

	a.sessionMu.Lock()
	a.sessions[session] = time.Now().Add(time.Duration(a.cfg.SessionTTLHours) * time.Hour)
	a.sessionMu.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:     "mikochi_player",
		Value:    session,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   r.TLS != nil,
		MaxAge:   a.cfg.SessionTTLHours * 3600,
	})

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *App) logout(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("mikochi_player")
	if err == nil {
		a.sessionMu.Lock()
		delete(a.sessions, c.Value)
		a.sessionMu.Unlock()
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "mikochi_player",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   r.TLS != nil,
		MaxAge:   -1,
	})
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *App) me(w http.ResponseWriter, r *http.Request) {
	if !a.authenticated(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *App) authenticated(r *http.Request) bool {
	c, err := r.Cookie("mikochi_player")
	if err != nil || c.Value == "" {
		return false
	}

	a.sessionMu.RLock()
	expires, ok := a.sessions[c.Value]
	a.sessionMu.RUnlock()

	if !ok {
		return false
	}
	if time.Now().After(expires) {
		a.sessionMu.Lock()
		delete(a.sessions, c.Value)
		a.sessionMu.Unlock()
		return false
	}
	return true
}

func (a *App) browse(w http.ResponseWriter, r *http.Request) {
	if !a.requireAuth(w, r) {
		return
	}

	p := cleanMikochiPath(r.URL.Query().Get("path"))

	target := strings.TrimRight(a.cfg.MikochiURL, "/") + "/api/browse" + p
	if q := r.URL.Query().Get("search"); q != "" {
		target += "?search=" + url.QueryEscape(q)
	}

	resp, err := a.mikochiRequest(r.Context(), http.MethodGet, target, nil)
	if err != nil {
		http.Error(w, "Mikochi is unavailable", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	copySafeJSON(w, resp)
}

func (a *App) stream(w http.ResponseWriter, r *http.Request) {
	if !a.requireAuth(w, r) {
		return
	}

	p := cleanMikochiPath(r.URL.Query().Get("path"))
	if p == "/" {
		http.Error(w, "invalid file path", http.StatusBadRequest)
		return
	}

	token, err := a.streamToken(r.Context(), p)
	if err != nil {
		http.Error(w, "failed to create stream token", http.StatusBadGateway)
		return
	}

	target := strings.TrimRight(a.cfg.MikochiURL, "/") + "/api/stream" + p + "?auth=" + url.QueryEscape(token)
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, target, nil)
	if err != nil {
		http.Error(w, "invalid stream request", http.StatusBadRequest)
		return
	}

	// Preserve byte ranges so the browser can seek without downloading the whole file.
	for _, name := range []string{"Range", "If-Range"} {
		if value := r.Header.Get(name); value != "" {
			req.Header.Set(name, value)
		}
	}

	resp, err := a.streamClient.Do(req)
	if err != nil {
		http.Error(w, "Mikochi is unavailable", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for _, name := range []string{
		"Content-Type",
		"Content-Length",
		"Content-Range",
		"Accept-Ranges",
		"Last-Modified",
		"ETag",
		"Content-Disposition",
	} {
		if value := resp.Header.Get(name); value != "" {
			w.Header().Set(name, value)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func (a *App) subtitle(w http.ResponseWriter, r *http.Request) {
	if !a.requireAuth(w, r) {
		return
	}

	p := cleanMikochiPath(r.URL.Query().Get("path"))
	lowerPath := strings.ToLower(p)
	if p == "/" || (!strings.HasSuffix(lowerPath, ".srt") && !strings.HasSuffix(lowerPath, ".vtt")) {
		http.Error(w, "invalid subtitle path", http.StatusBadRequest)
		return
	}

	token, err := a.streamToken(r.Context(), p)
	if err != nil {
		http.Error(w, "failed to create subtitle token", http.StatusBadGateway)
		return
	}

	target := strings.TrimRight(a.cfg.MikochiURL, "/") + "/api/stream" + p + "?auth=" + url.QueryEscape(token)
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, target, nil)
	if err != nil {
		http.Error(w, "invalid subtitle request", http.StatusBadRequest)
		return
	}

	resp, err := a.streamClient.Do(req)
	if err != nil {
		http.Error(w, "Mikochi is unavailable", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		http.Error(w, "subtitle file unavailable", resp.StatusCode)
		return
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		http.Error(w, "failed to read subtitle", http.StatusBadGateway)
		return
	}

	if strings.HasSuffix(lowerPath, ".srt") {
		data = srtToWebVTT(data)
	}

	w.Header().Set("Content-Type", "text/vtt; charset=utf-8")
	w.Header().Set("Cache-Control", "private, max-age=300")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func srtToWebVTT(data []byte) []byte {
	s := strings.TrimPrefix(string(data), "\ufeff")
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if strings.Contains(line, " --> ") {
			parts := strings.SplitN(line, " --> ", 2)
			parts[0] = strings.ReplaceAll(parts[0], ",", ".")
			parts[1] = strings.ReplaceAll(parts[1], ",", ".")
			lines[i] = parts[0] + " --> " + parts[1]
		}
	}
	return []byte("WEBVTT\n\n" + strings.Join(lines, "\n"))
}

func (a *App) streamToken(ctx context.Context, targetPath string) (string, error) {
	// A stream token is only valid for one path. Never reuse it for another file.
	// Do not hold mikochiMu here: mikochiRequest may need that same lock to refresh the JWT.
	endpoint := strings.TrimRight(a.cfg.MikochiURL, "/") + "/api/stream-token?target=" + url.QueryEscape(targetPath)

	resp, err := a.mikochiRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("stream-token returned %s", resp.Status)
	}

	var result loginResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 16<<10)).Decode(&result); err != nil {
		return "", err
	}
	if result.Token == "" {
		return "", errors.New("empty stream token")
	}
	return result.Token, nil
}

func (a *App) mikochiRequest(ctx context.Context, method, target string, body io.Reader) (*http.Response, error) {
	if err := a.ensureMikochiJWT(ctx); err != nil {
		return nil, err
	}

	resp, err := a.doMikochiRequest(ctx, method, target, body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusUnauthorized {
		return resp, nil
	}
	resp.Body.Close()

	if err := a.refreshMikochiJWT(ctx); err != nil {
		return nil, err
	}

	return a.doMikochiRequest(ctx, method, target, body)
}

func (a *App) doMikochiRequest(ctx context.Context, method, target string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, target, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	a.mikochiMu.Lock()
	jwt := a.mikochiJWT
	a.mikochiMu.Unlock()
	if jwt == "" {
		return nil, errors.New("Mikochi authentication token is empty")
	}
	req.Header.Set("Authorization", "Bearer "+jwt)

	return a.apiClient.Do(req)
}

func (a *App) ensureMikochiJWT(ctx context.Context) error {
	a.mikochiMu.Lock()
	defer a.mikochiMu.Unlock()

	if a.mikochiJWT != "" {
		return nil
	}
	return a.loginToMikochiLocked(ctx)
}

func (a *App) refreshMikochiJWT(ctx context.Context) error {
	a.mikochiMu.Lock()
	defer a.mikochiMu.Unlock()
	return a.loginToMikochiLocked(ctx)
}

func (a *App) loginToMikochiLocked(ctx context.Context) error {
	payload, err := json.Marshal(map[string]string{
		"username": a.cfg.MikochiUsername,
		"password": a.cfg.MikochiPassword,
	})
	if err != nil {
		return err
	}

	endpoint := strings.TrimRight(a.cfg.MikochiURL, "/") + "/api/login"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := a.apiClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Mikochi login returned %s", resp.Status)
	}

	var result loginResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 16<<10)).Decode(&result); err != nil {
		return err
	}
	if result.Token == "" {
		return errors.New("Mikochi returned an empty token")
	}

	a.mikochiJWT = result.Token
	return nil
}

func (a *App) requireAuth(w http.ResponseWriter, r *http.Request) bool {
	if !a.authenticated(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return false
	}
	return true
}

func cleanMikochiPath(p string) string {
	if p == "" {
		return "/"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	clean := path.Clean(p)
	if clean == "." || clean == "" {
		return "/"
	}
	return clean
}

func constantTimeEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func randomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func copySafeJSON(w http.ResponseWriter, resp *http.Response) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, io.LimitReader(resp.Body, 4<<20))
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; media-src 'self' blob:; style-src 'self' 'unsafe-inline'; script-src 'self'; img-src 'self' data:; frame-ancestors 'none'")
		next.ServeHTTP(w, r)
	})
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}
