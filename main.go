package main

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const maxRequestBytes = 1 << 20

type entry struct {
	Timestamp string `json:"timestamp"`
	Content   string `json:"content"`
	Format    string `json:"format"`
	DeviceID  string `json:"deviceId"`
}

type receiver struct {
	dir          string
	mu           sync.Mutex
	username     string
	password     string
	sessions     *sql.DB
	sessionTTL   time.Duration
	extendedTTL  time.Duration
	secureCookie bool
}

type csvSet struct {
	Date  string
	Count int
}

type pageData struct {
	Sets    []csvSet
	Date    string
	Entries [][]string
	Total   int
}

var pages = template.Must(template.New("pages").Parse(`<!doctype html>
<html lang="cs">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{if .Date}}Záznamy {{.Date}}{{else}}Scan Collector{{end}}</title>
  <style>
    :root { color-scheme: light; font: 16px/1.5 system-ui, sans-serif; color: #17212b; background: #f3f6f8; }
    body { margin: 0; }
    main { max-width: 1000px; margin: 0 auto; padding: 32px 20px 64px; }
    h1 { margin: 0 0 8px; font-size: clamp(1.7rem, 4vw, 2.4rem); }
    .muted { color: #5e6b75; margin: 0 0 24px; }
    .toolbar { display: flex; flex-wrap: wrap; gap: 10px; margin: 20px 0; }
    a.button, button { display: inline-block; border: 0; border-radius: 7px; padding: 9px 14px; background: #155eef; color: white; font: inherit; text-decoration: none; cursor: pointer; }
    a.secondary, button.secondary { background: #e3eaf0; color: #17212b; }
    table { width: 100%; border-collapse: collapse; background: white; border-radius: 10px; overflow: hidden; box-shadow: 0 1px 4px #17212b12; }
    th, td { padding: 12px 14px; text-align: left; border-bottom: 1px solid #e5eaee; }
    th { background: #eaf0f5; font-size: .9rem; }
    tbody tr:last-child td { border-bottom: 0; }
    .count { font-variant-numeric: tabular-nums; }
    .empty { padding: 20px; background: white; border-radius: 10px; }
    .copy { white-space: nowrap; padding: 6px 10px; }
    #notice { align-self: center; color: #176b3a; }
  @media (max-width: 650px) { main { padding: 22px 12px 40px; } th, td { padding: 9px 8px; font-size: .9rem; } }
  </style>
</head>
<body><main>
  <div class="toolbar"><form action="/logout" method="post"><button class="secondary" type="submit">Odhlásit</button></form></div>
{{if .Date}}
  <h1>Záznamy z {{.Date}}</h1>
  <p class="muted">Celkem {{.Total}} {{if eq .Total 1}}záznam{{else}}záznamů{{end}}</p>
  <div class="toolbar">
    <a class="button secondary" href="/">← Přehled sad</a>
    <a class="button" href="/sets/{{.Date}}.csv">Stáhnout CSV</a>
    <button class="secondary" type="button" onclick="copyAll()">Kopírovat všechen obsah</button>
    <span id="notice" role="status" aria-live="polite"></span>
  </div>
  {{if .Entries}}
  <table id="records"><thead><tr><th>Čas</th><th>Obsah</th><th class="wide">Formát</th><th class="wide">Zařízení</th><th></th></tr></thead><tbody>
  {{range .Entries}}<tr><td>{{index . 0}}</td><td>{{index . 1}}</td><td class="wide">{{index . 2}}</td><td class="wide">{{index . 3}}</td><td><button class="copy" type="button" data-content="{{index . 1}}" onclick="copyText(this.dataset.content)">Kopírovat</button></td></tr>{{end}}
  </tbody></table>
  {{else}}<p class="empty">Tato sada zatím neobsahuje žádné záznamy.</p>{{end}}
  <script>
    async function copyText(value) { try { await navigator.clipboard.writeText(value); document.getElementById('notice').textContent = 'Zkopírováno'; setTimeout(() => document.getElementById('notice').textContent = '', 1800); } catch (_) { document.getElementById('notice').textContent = 'Kopírování není v tomto prohlížeči dostupné'; } }
    async function copyAll() { const values = [...document.querySelectorAll('#records tbody tr td:nth-child(2)')].map(cell => cell.textContent); try { await navigator.clipboard.writeText(values.join('\n')); document.getElementById('notice').textContent = 'Obsah zkopírován'; setTimeout(() => document.getElementById('notice').textContent = '', 1800); } catch (_) { document.getElementById('notice').textContent = 'Kopírování není v tomto prohlížeči dostupné'; } }
  </script>
{{else}}
  <h1>Scan Collector</h1>
  <p class="muted">Přehled uložených sad podle data</p>
  {{if .Sets}}
  <table><thead><tr><th>Datum</th><th>Počet záznamů</th><th></th></tr></thead><tbody>
  {{range .Sets}}<tr><td>{{.Date}}</td><td class="count">{{.Count}}</td><td><a href="/sets/{{.Date}}">Zobrazit záznamy →</a></td></tr>{{end}}
  </tbody></table>
  {{else}}<p class="empty">Zatím nejsou uložené žádné sady.</p>{{end}}
{{end}}
</main></body></html>`))

func (r *receiver) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if req.URL.Path == "/login" {
		r.serveLogin(w, req)
		return
	}
	if req.URL.Path == "/logout" {
		r.serveLogout(w, req)
		return
	}
	if req.Method == http.MethodPost && req.URL.Path == "/" {
		r.servePost(w, req)
		return
	}
	if !r.hasValidSession(req) {
		http.Redirect(w, req, "/login", http.StatusSeeOther)
		return
	}
	if req.Method == http.MethodGet {
		r.serveGet(w, req)
		return
	}
	w.Header().Set("Allow", "GET")
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func (r *receiver) servePost(w http.ResponseWriter, req *http.Request) {
	var item entry
	decoder := json.NewDecoder(http.MaxBytesReader(w, req.Body, maxRequestBytes))
	if err := decoder.Decode(&item); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		http.Error(w, "request must contain one JSON object", http.StatusBadRequest)
		return
	}
	if err := r.append(item); err != nil {
		log.Printf("CSV write failed: %v", err)
		http.Error(w, "could not save entry", http.StatusInternalServerError)
		return
	}

	log.Printf("Accepted scan content: %s", item.Content)
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, "OK")
}

func (r *receiver) serveGet(w http.ResponseWriter, req *http.Request) {
	if req.URL.Path == "/" {
		sets, err := r.listSets()
		if err != nil {
			log.Printf("CSV list failed: %v", err)
			http.Error(w, "could not read saved entries", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := pages.Execute(w, pageData{Sets: sets}); err != nil {
			log.Printf("Page render failed: %v", err)
		}
		return
	}

	path := strings.TrimPrefix(req.URL.Path, "/sets/")
	if !strings.HasPrefix(req.URL.Path, "/sets/") {
		http.NotFound(w, req)
		return
	}
	export := strings.HasSuffix(path, ".csv")
	date := strings.TrimSuffix(path, ".csv")
	if _, err := time.Parse("2006-01-02", date); err != nil {
		http.NotFound(w, req)
		return
	}
	filePath := filepath.Join(r.dir, "scan-"+date+".csv")
	r.mu.Lock()
	file, err := os.Open(filePath)
	if err != nil {
		r.mu.Unlock()
		if os.IsNotExist(err) {
			http.NotFound(w, req)
		} else {
			http.Error(w, "could not read saved entries", http.StatusInternalServerError)
		}
		return
	}
	if export {
		defer r.mu.Unlock()
		defer file.Close()
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=scan-%s.csv", date))
		if _, err := io.Copy(w, file); err != nil {
			log.Printf("CSV export failed: %v", err)
		}
		return
	}
	rows, err := csv.NewReader(file).ReadAll()
	_ = file.Close()
	r.mu.Unlock()
	if err != nil {
		log.Printf("CSV read failed: %v", err)
		http.Error(w, "could not read saved entries", http.StatusInternalServerError)
		return
	}
	if len(rows) > 0 {
		rows = rows[1:]
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pages.Execute(w, pageData{Date: date, Entries: rows, Total: len(rows)}); err != nil {
		log.Printf("Page render failed: %v", err)
	}
}

func (r *receiver) listSets() ([]csvSet, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	files, err := os.ReadDir(r.dir)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	var sets []csvSet
	for _, file := range files {
		name := file.Name()
		if file.IsDir() || !strings.HasPrefix(name, "scan-") || !strings.HasSuffix(name, ".csv") {
			continue
		}
		date := strings.TrimSuffix(strings.TrimPrefix(name, "scan-"), ".csv")
		if _, err := time.Parse("2006-01-02", date); err != nil {
			continue
		}
		f, err := os.Open(filepath.Join(r.dir, name))
		if err != nil {
			return nil, err
		}
		count := 0
		reader := csv.NewReader(f)
		for {
			_, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				_ = f.Close()
				return nil, err
			}
			count++
		}
		_ = f.Close()
		if count > 0 {
			count-- // header row
		}
		sets = append(sets, csvSet{Date: date, Count: count})
	}
	sort.Slice(sets, func(i, j int) bool { return sets[i].Date > sets[j].Date })
	return sets, nil
}

func (r *receiver) append(item entry) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := os.MkdirAll(r.dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(r.dir, "scan-"+time.Now().Format("2006-01-02")+".csv")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}
	w := csv.NewWriter(file)
	if info.Size() == 0 {
		if err := w.Write([]string{"čas", "obsah", "formát", "zařízení"}); err != nil {
			return err
		}
	}
	if err := w.Write([]string{item.Timestamp, item.Content, item.Format, item.DeviceID}); err != nil {
		return err
	}
	w.Flush()
	return w.Error()
}

func main() {
	port := strings.TrimPrefix(os.Getenv("PORT"), ":")
	if port == "" {
		port = "8765"
	}
	dir := os.Getenv("DATA_DIR")
	if dir == "" {
		dir = "/data"
	}
	authUser := os.Getenv("AUTH_USER")
	authPassword := os.Getenv("AUTH_PASSWORD")
	if authUser == "" || authPassword == "" {
		log.Fatal("AUTH_USER and AUTH_PASSWORD must be configured")
	}
	sessionTTL, err := parseSessionDuration("SESSION_TTL", "8h")
	if err != nil {
		log.Fatalf("invalid SESSION_TTL: %v", err)
	}
	extendedTTL, err := parseSessionDuration("SESSION_EXTENDED_TTL", "720h")
	if err != nil {
		log.Fatalf("invalid SESSION_EXTENDED_TTL: %v", err)
	}
	sessions, err := openSessionStore(dir)
	if err != nil {
		log.Fatalf("could not open session store: %v", err)
	}
	defer sessions.Close()
	receiver := &receiver{
		dir: dir, username: authUser, password: authPassword, sessions: sessions,
		sessionTTL: sessionTTL, extendedTTL: extendedTTL,
		secureCookie: strings.EqualFold(os.Getenv("COOKIE_SECURE"), "true"),
	}
	addr := fmt.Sprintf("0.0.0.0:%s", port)
	log.Printf("Listening on %s; CSV files in %s", addr, dir)
	log.Fatal(http.ListenAndServe(addr, receiver))
}
