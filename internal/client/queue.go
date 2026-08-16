package client

import (
	"context"
	"net/http"
	"regexp"
)

type Queue struct {
	ID          string                  `json:"id"`
	Name        string                  `json:"name"`
	DisplayName string                  `json:"display_name"`
	Tags        []string                `json:"tags"`
	Metadata    map[string]MetadataItem `json:"metadata"`
	Entries     []QueueEntry            `json:"entries"`
}

type MetadataItem struct {
	Key   string `json:"key,omitempty"`
	Type  string `json:"type,omitempty"`
	Value string `json:"value,omitempty"`
}

type QueueEntry struct {
	Task string `json:"task"`
}

type QueueInput struct {
	Name        string
	DisplayName *string
	Tags        *[]string
	Metadata    *map[string]MetadataItem
}

type queueMutationRequest struct {
	Queue       string                   `json:"queue,omitempty"`
	Name        string                   `json:"name"`
	DisplayName *string                  `json:"display_name,omitempty"`
	Tags        *[]string                `json:"tags,omitempty"`
	Metadata    *map[string]MetadataItem `json:"metadata,omitempty"`
}

type QueuePage struct {
	Queues []Queue
}

func (c *ClearMLClient) CreateQueue(ctx context.Context, input QueueInput) (string, error) {
	request := queueMutationRequest{Name: input.Name, DisplayName: input.DisplayName, Tags: input.Tags, Metadata: input.Metadata}
	var response struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := c.request(ctx, http.MethodPost, "/queues.create", request, &response); err != nil {
		return "", err
	}
	if response.Data.ID == "" {
		return "", invalidResponse("queues.create", "id")
	}
	return response.Data.ID, nil
}

func (c *ClearMLClient) GetQueue(ctx context.Context, id string) (Queue, error) {
	return c.getQueue(ctx, id, nil)
}

func (c *ClearMLClient) getQueue(ctx context.Context, id string, maxTaskEntries *int) (Queue, error) {
	request := struct {
		Queue          string `json:"queue"`
		MaxTaskEntries *int   `json:"max_task_entries,omitempty"`
	}{Queue: id, MaxTaskEntries: maxTaskEntries}
	var response struct {
		Data struct {
			Queue Queue `json:"queue"`
		} `json:"data"`
	}
	if err := c.request(ctx, http.MethodPost, "/queues.get_by_id", request, &response); err != nil {
		return Queue{}, err
	}
	if response.Data.Queue.ID == "" {
		return Queue{}, &NotFoundError{Kind: "queue", Value: id}
	}
	return response.Data.Queue, nil
}

func (c *ClearMLClient) FindQueue(ctx context.Context, name string) (Queue, error) {
	request := struct {
		Name     string `json:"name"`
		Page     int    `json:"page"`
		PageSize int    `json:"page_size"`
	}{Name: "^" + regexp.QuoteMeta(name) + "$", Page: 0, PageSize: 2}
	var response struct {
		Data struct {
			Queues []Queue `json:"queues"`
		} `json:"data"`
	}
	if err := c.request(ctx, http.MethodPost, "/queues.get_all", request, &response); err != nil {
		return Queue{}, err
	}
	return exactlyOne("queue", name, response.Data.Queues, func(queue Queue) string { return queue.ID })
}

func (c *ClearMLClient) ListQueues(ctx context.Context, namePattern string, page, pageSize int) (QueuePage, error) {
	request := struct {
		Name       string   `json:"name"`
		OnlyFields []string `json:"only_fields"`
		Page       int      `json:"page"`
		PageSize   int      `json:"page_size"`
	}{Name: namePattern, OnlyFields: []string{"id", "name"}, Page: page, PageSize: pageSize}
	var response struct {
		Data struct {
			Queues []Queue `json:"queues"`
		} `json:"data"`
	}
	if err := c.request(ctx, http.MethodPost, "/queues.get_all", request, &response); err != nil {
		return QueuePage{}, err
	}
	return QueuePage{Queues: response.Data.Queues}, nil
}

func (c *ClearMLClient) UpdateQueue(ctx context.Context, id string, input QueueInput) error {
	request := queueMutationRequest{Queue: id, Name: input.Name, DisplayName: input.DisplayName, Tags: input.Tags, Metadata: input.Metadata}
	return c.request(ctx, http.MethodPost, "/queues.update", request, nil)
}

func (c *ClearMLClient) DeleteQueue(ctx context.Context, id string) error {
	request := struct {
		Queue string `json:"queue"`
		Force bool   `json:"force"`
	}{Queue: id, Force: false}
	return c.request(ctx, http.MethodPost, "/queues.delete", request, nil)
}
