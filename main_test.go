package main

import (
	"encoding/csv"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPostWritesDailyCSV(t *testing.T) {
	dir := t.TempDir()
	handler := &receiver{dir: dir}
	body := `{"timestamp":"2026-09-25T12:00:00Z","content":"9780306406157","format":"EAN-13","deviceId":"scanner-1"}`
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body)))

	if response.Code != http.StatusOK || response.Body.String() != "OK" {
		t.Fatalf("unexpected response: %d %q", response.Code, response.Body.String())
	}
	path := filepath.Join(dir, "isbn-"+time.Now().Format("2006-01-02")+".csv")
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	rows, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[1][1] != "9780306406157" {
		t.Fatalf("unexpected CSV rows: %#v", rows)
	}
}

func TestRejectsInvalidJSON(t *testing.T) {
	handler := &receiver{dir: t.TempDir()}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{")))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("got status %d, want %d", response.Code, http.StatusBadRequest)
	}
}