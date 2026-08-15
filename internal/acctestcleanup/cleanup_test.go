package acctestcleanup

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pbronneberg/terraform-provider-clearml/internal/client"
)

func TestIsStaleOwnedQueue(t *testing.T) {
	t.Parallel()
	cutoff := time.Unix(1_700_000_000, 0).UTC()
	for _, test := range []struct {
		name string
		want bool
	}{
		{"tfacc-clearml-1699999999-1-15-8-123-1-0123456789abcdef0123456789abcdef", true},
		{"tfacc-clearml-1700000000-1-15-8-123-1-0123456789abcdef0123456789abcdef", false},
		{"tfacc-clearml-9999999999-1-15-8-123-1-0123456789abcdef0123456789abcdef", false},
		{"tfacc-clearml-1699999999-local-123-1-0123456789abcdef0123456789abcdef", false},
		{"personal-queue", false},
	} {
		if got := IsStaleOwnedQueue(test.name, cutoff); got != test.want {
			t.Errorf("IsStaleOwnedQueue(%q) = %t, want %t", test.name, got, test.want)
		}
	}
}

func TestPrune(t *testing.T) {
	t.Parallel()
	fake := &fakeClient{pages: [][]client.Queue{
		makeQueues(PageSize),
		{{ID: "old", Name: "tfacc-clearml-1699999999-1-15-8-123-1-0123456789abcdef0123456789abcdef"}, {ID: "new", Name: "tfacc-clearml-1700000000-1-15-8-123-1-0123456789abcdef0123456789abcdef"}, {ID: "other", Name: "personal-queue"}},
	}}
	deleted, err := Prune(t.Context(), fake, time.Unix(1_700_086_400, 0), 24*time.Hour)
	if err != nil {
		t.Fatalf("Prune() error = %v", err)
	}
	if deleted != 1 || len(fake.deleted) != 1 || fake.deleted[0] != "old" {
		t.Fatalf("Prune() = %d, deleted %v; want 1, [old]", deleted, fake.deleted)
	}
}

func TestPruneDeleteFailure(t *testing.T) {
	t.Parallel()
	fake := &fakeClient{pages: [][]client.Queue{{{ID: "old", Name: "tfacc-clearml-1699999999-1-15-8-123-1-0123456789abcdef0123456789abcdef"}}}, deleteErr: errors.New("denied")}
	_, err := Prune(t.Context(), fake, time.Unix(1_700_086_400, 0), 24*time.Hour)
	if err == nil {
		t.Fatal("Prune() error = nil, want delete error")
	}
}

func TestPruneAlreadyDeletedQueue(t *testing.T) {
	t.Parallel()
	fake := &fakeClient{pages: [][]client.Queue{{{ID: "old", Name: "tfacc-clearml-1699999999-1-15-8-123-1-0123456789abcdef0123456789abcdef"}}}, deleteErr: &client.APIError{StatusCode: 404}}
	deleted, err := Prune(t.Context(), fake, time.Unix(1_700_086_400, 0), 24*time.Hour)
	if err != nil || deleted != 1 {
		t.Fatalf("Prune() = %d, %v; want 1, nil", deleted, err)
	}
}

type fakeClient struct {
	pages     [][]client.Queue
	deleted   []string
	deleteErr error
}

func (f *fakeClient) ListQueues(_ context.Context, _ string, page, _ int) (client.QueuePage, error) {
	if page >= len(f.pages) {
		return client.QueuePage{}, nil
	}
	return client.QueuePage{Queues: f.pages[page]}, nil
}

func (f *fakeClient) DeleteQueue(_ context.Context, id string) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	f.deleted = append(f.deleted, id)
	return nil
}

func makeQueues(count int) []client.Queue {
	queues := make([]client.Queue, count)
	for index := range queues {
		queues[index] = client.Queue{ID: "ignored", Name: "unrelated"}
	}
	return queues
}
