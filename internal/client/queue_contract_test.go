package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

const fixtureVersion = "clearml-v3.28.8"

func TestQueueContract(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/auth.login":
			accessKey, secretKey, ok := r.BasicAuth()
			if !ok || accessKey != "access-key" || secretKey != "secret-key" {
				t.Fatalf("unexpected authentication: %q, %q, %t", accessKey, secretKey, ok)
			}
			writeFixture(t, w, "auth.login.json")
		case "/queues.create":
			assertBearerToken(t, r)
			assertPayload(t, r, map[string]any{"name": "example-queue", "tags": []any{"production", "gpu"}})
			writeFixture(t, w, "queues.create.json")
		case "/queues.get_by_id":
			assertBearerToken(t, r)
			assertPayload(t, r, map[string]any{"queue": "queue-123"})
			writeFixture(t, w, "queues.get_by_id.json")
		case "/queues.update":
			assertBearerToken(t, r)
			assertPayload(t, r, map[string]any{"queue": "queue-123", "name": "renamed-queue", "tags": []any{"production"}})
			writeFixture(t, w, "empty.json")
		case "/queues.delete":
			assertBearerToken(t, r)
			assertPayload(t, r, map[string]any{"queue": "queue-123"})
			writeFixture(t, w, "empty.json")
		default:
			t.Fatalf("unexpected endpoint: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := NewClearMLClient(t.Context(), "terraform-provider-clearml/test", "access-key", "secret-key", server.URL)
	if err != nil {
		t.Fatalf("NewClearMLClient() error = %v", err)
	}

	id, err := client.CreateQueue(t.Context(), "example-queue", []string{"production", "gpu"})
	if err != nil {
		t.Fatalf("CreateQueue() error = %v", err)
	}
	if id != "queue-123" {
		t.Fatalf("CreateQueue() id = %q, want queue-123", id)
	}

	queue, err := client.GetQueue(t.Context(), id)
	if err != nil {
		t.Fatalf("GetQueue() error = %v", err)
	}
	if queue.Name != "example-queue" {
		t.Fatalf("GetQueue() name = %q, want example-queue", queue.Name)
	}

	if err := client.UpdateQueue(t.Context(), id, "renamed-queue", []string{"production"}); err != nil {
		t.Fatalf("UpdateQueue() error = %v", err)
	}
	if err := client.DeleteQueue(t.Context(), id); err != nil {
		t.Fatalf("DeleteQueue() error = %v", err)
	}
}

func assertBearerToken(t *testing.T, r *http.Request) {
	t.Helper()
	if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
		t.Fatalf("Authorization = %q, want Bearer test-token", got)
	}
}

func assertPayload(t *testing.T, r *http.Request, want map[string]any) {
	t.Helper()
	var got map[string]any
	if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if !equalJSON(got, want) {
		t.Fatalf("payload = %#v, want %#v", got, want)
	}
}

func equalJSON(got, want map[string]any) bool {
	gotJSON, _ := json.Marshal(got)
	wantJSON, _ := json.Marshal(want)
	return string(gotJSON) == string(wantJSON)
}

func writeFixture(t *testing.T, w http.ResponseWriter, name string) {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("testdata", fixtureVersion, name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	if _, err := w.Write(body); err != nil {
		t.Fatalf("write fixture %s: %v", name, err)
	}
}
