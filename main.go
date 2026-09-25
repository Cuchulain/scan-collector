package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
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
	dir string
	mu  sync.Mutex
}

func (r *receiver) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

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

	log.Printf("Accepted ISBN: %s", item.Content)
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, "OK")
}

func (r *receiver) append(item entry) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := os.MkdirAll(r.dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(r.dir, "isbn-"+time.Now().Format("2006-01-02")+".csv")
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
		if err := w.Write([]string{"čas", "isbn", "formát", "zařízení"}); err != nil {
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
	addr := fmt.Sprintf("0.0.0.0:%s", port)
	log.Printf("Listening on %s; CSV files in %s", addr, dir)
	log.Fatal(http.ListenAndServe(addr, &receiver{dir: dir}))
}