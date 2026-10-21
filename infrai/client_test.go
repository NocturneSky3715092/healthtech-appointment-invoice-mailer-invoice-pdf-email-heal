package infrai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPDFAndEmailUseSameBoundary(t *testing.T) {
	var paths []string
	var bodies []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("authorization = %q", got)
		}
		if r.Header.Get("Idempotency-Key") == "" {
			t.Error("missing idempotency key")
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		bodies = append(bodies, body)
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/v1/pdf/generate" {
			json.NewEncoder(w).Encode(map[string]any{"ok": true, "data": map[string]string{"url": "https://files.example/invoice.pdf"}})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"ok": true, "data": map[string]string{"message_id": "msg-42"}})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", server.Client())
	pdfURL, err := client.GeneratePDF(context.Background(), "<h1>Invoice</h1>", "apt-1042:pdf")
	if err != nil {
		t.Fatal(err)
	}
	messageID, err := client.SendEmail(context.Background(), "patient@example.com", "Invoice", pdfURL, "apt-1042:email")
	if err != nil {
		t.Fatal(err)
	}
	if messageID != "msg-42" {
		t.Fatalf("message ID = %q", messageID)
	}
	if len(paths) != 2 || paths[0] != "/v1/pdf/generate" || paths[1] != "/v1/email/send" {
		t.Fatalf("paths = %v", paths)
	}
	if got := bodies[0]["idempotency_key"]; got != "apt-1042:pdf" {
		t.Errorf("PDF idempotency_key = %v", got)
	}
	if got := bodies[1]["idempotency_key"]; got != "apt-1042:email" {
		t.Errorf("email idempotency_key = %v", got)
	}
}
