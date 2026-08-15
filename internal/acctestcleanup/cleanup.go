// Package acctestcleanup safely removes stale queues created by CI acceptance tests.
package acctestcleanup

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/pbronneberg/terraform-provider-clearml/internal/client"
)

const (
	Prefix   = "tfacc-clearml-"
	PageSize = 100
)

var queueNamePattern = regexp.MustCompile(`^tfacc-clearml-([0-9]{10})-[0-9]+-[0-9]+-[0-9]+-[0-9]+-[0-9]+-[a-f0-9]{32}$`)

// Client is the minimum ClearML surface needed for stale acceptance-test cleanup.
type Client interface {
	ListQueues(context.Context, string, int, int) (client.QueuePage, error)
	DeleteQueue(context.Context, string) error
}

// IsStaleOwnedQueue reports whether name has the exact CI-owned format and is
// strictly older than cutoff. The creation timestamp is embedded in the name,
// so cleanup never relies on mutable remote metadata.
func IsStaleOwnedQueue(name string, cutoff time.Time) bool {
	matches := queueNamePattern.FindStringSubmatch(name)
	if matches == nil {
		return false
	}
	seconds, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return false
	}
	created := time.Unix(seconds, 0).UTC()
	return created.Before(cutoff)
}

// Prune deletes all CI-owned queues older than maxAge. It lists every page and
// verifies ownership locally before each deletion.
func Prune(ctx context.Context, c Client, now time.Time, maxAge time.Duration) (int, error) {
	cutoff := now.UTC().Add(-maxAge)
	var stale []client.Queue
	for page := 0; ; page++ {
		queues, err := c.ListQueues(ctx, `^tfacc-clearml-[0-9]{10}-.*$`, page, PageSize)
		if err != nil {
			return 0, fmt.Errorf("list stale acceptance queues page %d: %w", page, err)
		}
		for _, queue := range queues.Queues {
			if !IsStaleOwnedQueue(queue.Name, cutoff) {
				continue
			}
			stale = append(stale, queue)
		}
		if len(queues.Queues) < PageSize {
			break
		}
	}
	for index, queue := range stale {
		if err := c.DeleteQueue(ctx, queue.ID); err != nil && !client.IsNotFound(err) {
			return index, fmt.Errorf("delete stale acceptance queue %q (%s): %w", queue.Name, queue.ID, err)
		}
	}
	return len(stale), nil
}
