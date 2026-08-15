package client

import (
	"context"
	"net/http"
)

// Queue is the ClearML queue representation required by this provider.
type Queue struct {
	ID   string   `json:"id"`
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

// QueuePage is one page returned by the ClearML queues.get_all endpoint.
type QueuePage struct {
	Queues []Queue
}

func (c *ClearMLClient) CreateQueue(ctx context.Context, name string, tags []string) (string, error) {
	var response struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := c.request(ctx, http.MethodPost, "/queues.create", map[string]any{"name": name, "tags": tags}, &response); err != nil {
		return "", err
	}
	return response.Data.ID, nil
}

func (c *ClearMLClient) GetQueue(ctx context.Context, id string) (Queue, error) {
	var response struct {
		Data struct {
			Queue Queue `json:"queue"`
		} `json:"data"`
	}
	if err := c.request(ctx, http.MethodPost, "/queues.get_by_id", map[string]any{"queue": id}, &response); err != nil {
		return Queue{}, err
	}
	return response.Data.Queue, nil
}

// ListQueues returns one page of queues matching namePattern. It is used by
// acceptance-test cleanup and deliberately projects only the fields it needs.
func (c *ClearMLClient) ListQueues(ctx context.Context, namePattern string, page, pageSize int) (QueuePage, error) {
	var response struct {
		Data struct {
			Queues []Queue `json:"queues"`
		} `json:"data"`
	}
	payload := map[string]any{
		"name":        namePattern,
		"only_fields": []string{"id", "name"},
		"page":        page,
		"page_size":   pageSize,
	}
	if err := c.request(ctx, http.MethodPost, "/queues.get_all", payload, &response); err != nil {
		return QueuePage{}, err
	}
	return QueuePage{Queues: response.Data.Queues}, nil
}

func (c *ClearMLClient) UpdateQueue(ctx context.Context, id, name string, tags []string) error {
	return c.request(ctx, http.MethodPost, "/queues.update", map[string]any{"queue": id, "name": name, "tags": tags}, nil)
}

func (c *ClearMLClient) DeleteQueue(ctx context.Context, id string) error {
	return c.request(ctx, http.MethodPost, "/queues.delete", map[string]any{"queue": id}, nil)
}
