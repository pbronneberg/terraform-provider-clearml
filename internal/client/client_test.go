package client

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAPIErrorRedactsResponseBody(t *testing.T) {
	t.Parallel()
	const secret = "credential-that-must-not-appear"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth.login" {
			_, _ = w.Write([]byte(`{"data":{"token":"test-token"}}`))
			return
		}
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"data":{"secret":"` + secret + `"}}`))
	}))
	defer server.Close()

	client, err := NewClearMLClient(t.Context(), "test", "key", "secret", server.URL)
	if err != nil {
		t.Fatalf("NewClearMLClient() error = %v", err)
	}
	_, err = client.GetQueue(t.Context(), "queue")
	if err == nil {
		t.Fatal("GetQueue() returned no error")
	}
	if strings.Contains(err.Error(), secret) || !strings.Contains(err.Error(), "HTTP 403") {
		t.Fatalf("error = %q, want status-only diagnostic", err)
	}
}

func TestMissingObjectClassification(t *testing.T) {
	t.Parallel()
	for _, err := range []error{
		&NotFoundError{Kind: "queue", Value: "missing"},
		&APIError{StatusCode: http.StatusNotFound, Method: http.MethodPost, Path: "/queues.get_by_id"},
		&APIError{StatusCode: http.StatusBadRequest, ResultCode: 400, ResultSubcode: 401, Method: http.MethodPost, Path: "/projects.get_by_id"},
		&APIError{StatusCode: http.StatusBadRequest, ResultCode: 400, ResultSubcode: 701, Method: http.MethodPost, Path: "/queues.get_by_id"},
		errors.Join(errors.New("read queue"), &NotFoundError{Kind: "queue", Value: "missing"}),
	} {
		if !IsNotFound(err) {
			t.Errorf("IsNotFound(%v) = false, want true", err)
		}
	}
	if IsNotFound(errors.New("network failure")) {
		t.Fatal("IsNotFound(network failure) = true, want false")
	}
	if IsNotFound(&APIError{StatusCode: http.StatusBadRequest, ResultCode: 400, ResultSubcode: 702}) {
		t.Fatal("IsNotFound(queue not empty) = true, want false")
	}
}

func TestExactlyOneRejectsMissingDuplicateAndMalformedMatches(t *testing.T) {
	t.Parallel()
	type item struct{ id string }
	for _, test := range []struct {
		name   string
		values []item
		check  func(error) bool
	}{
		{name: "missing", check: IsNotFound},
		{name: "duplicate", values: []item{{id: "one"}, {id: "two"}}, check: func(err error) bool { var target *MultipleMatchesError; return errors.As(err, &target) }},
		{name: "malformed", values: []item{{}}, check: func(err error) bool { return strings.Contains(err.Error(), "did not contain id") }},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := exactlyOne("item", "name", test.values, func(value item) string { return value.id })
			if err == nil || !test.check(err) {
				t.Fatalf("exactlyOne() error = %v", err)
			}
		})
	}
}
