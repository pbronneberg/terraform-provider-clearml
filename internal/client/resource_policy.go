package client

import (
	"context"
	"errors"
	"net/http"
)

type ResourceProfile struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	ProfileCost float64 `json:"profile_cost"`
	CostField   string  `json:"cost_field"`
}

type ResourcePolicy struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Description  string  `json:"description"`
	Reservation  float64 `json:"reservation"`
	Limit        float64 `json:"limit"`
	UserGroup    string  `json:"user_group"`
	ProfileLinks []struct {
		Profile ResourceProfile `json:"profile"`
		Queue   Queue           `json:"queue"`
	} `json:"profile_links"`
	Tasks []struct {
		ID string `json:"id"`
	} `json:"tasks"`
}

type ResourcePolicyInput struct {
	Name        string
	Description *string
	Reservation float64
	Limit       float64
	UserGroupID string
}

type resourcePolicyMutationRequest struct {
	ID          string  `json:"id,omitempty"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Reservation float64 `json:"reservation"`
	Limit       float64 `json:"limit"`
	UserGroupID string  `json:"user_group,omitempty"`
}

func (c *ClearMLClient) resourceProfiles(ctx context.Context) ([]ResourceProfile, error) {
	request := struct {
		Edit bool `json:"edit"`
	}{Edit: false}
	var response struct {
		Data struct {
			Configuration struct {
				Profiles []ResourceProfile `json:"profiles"`
			} `json:"configuration"`
		} `json:"data"`
	}
	if err := c.request(ctx, http.MethodPost, "/resources.get_resource_configuration", request, &response); err != nil {
		return nil, err
	}
	return response.Data.Configuration.Profiles, nil
}

func (c *ClearMLClient) FindResourceProfile(ctx context.Context, name string) (ResourceProfile, error) {
	profiles, err := c.resourceProfiles(ctx)
	if err != nil {
		return ResourceProfile{}, err
	}
	matches := make([]ResourceProfile, 0, 1)
	for _, profile := range profiles {
		if profile.Name == name {
			matches = append(matches, profile)
		}
	}
	return exactlyOne("resource profile", name, matches, func(profile ResourceProfile) string { return profile.ID })
}

func resourcePolicyPayload(id string, input ResourcePolicyInput) resourcePolicyMutationRequest {
	return resourcePolicyMutationRequest{
		ID: id, Name: input.Name, Description: input.Description, Reservation: input.Reservation,
		Limit: input.Limit, UserGroupID: input.UserGroupID,
	}
}

func (c *ClearMLClient) CreateResourcePolicy(ctx context.Context, input ResourcePolicyInput) (string, error) {
	var response struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := c.request(ctx, http.MethodPost, "/resources.create_policy", resourcePolicyPayload("", input), &response); err != nil {
		return "", err
	}
	if response.Data.ID == "" {
		return "", invalidResponse("resources.create_policy", "id")
	}
	return response.Data.ID, nil
}

func (c *ClearMLClient) GetResourcePolicy(ctx context.Context, id string) (ResourcePolicy, error) {
	request := struct {
		ID string `json:"id"`
	}{ID: id}
	var response struct {
		Data ResourcePolicy `json:"data"`
	}
	if err := c.request(ctx, http.MethodPost, "/resources.get_policy_data", request, &response); err != nil {
		return ResourcePolicy{}, err
	}
	if response.Data.ID == "" {
		return ResourcePolicy{}, &NotFoundError{Kind: "resource policy", Value: id}
	}
	return response.Data, nil
}

func (c *ClearMLClient) UpdateResourcePolicy(ctx context.Context, id string, input ResourcePolicyInput) error {
	current, err := c.GetResourcePolicy(ctx, id)
	if err != nil {
		return err
	}
	if current.UserGroup != input.UserGroupID {
		request := struct {
			ID          string `json:"id"`
			UserGroupID string `json:"user_group"`
		}{ID: id, UserGroupID: input.UserGroupID}
		var validation struct {
			Data struct {
				TasksToDequeue int `json:"tasks_to_dequeue"`
			} `json:"data"`
		}
		if err := c.request(ctx, http.MethodPost, "/resources.validate_move_policy", request, &validation); err != nil {
			return err
		}
		if validation.Data.TasksToDequeue != 0 {
			return errors.New("ClearML resource policy move would dequeue pending tasks")
		}
	}

	update := resourcePolicyPayload(id, input)
	update.UserGroupID = ""
	if err := c.request(ctx, http.MethodPost, "/resources.update_policy", update, nil); err != nil {
		return err
	}
	if current.UserGroup == input.UserGroupID {
		return nil
	}
	move := struct {
		ID          string `json:"id"`
		UserGroupID string `json:"user_group"`
	}{ID: id, UserGroupID: input.UserGroupID}
	return c.request(ctx, http.MethodPost, "/resources.move_policy", move, nil)
}

func (c *ClearMLClient) DeleteResourcePolicy(ctx context.Context, id string) error {
	policy, err := c.GetResourcePolicy(ctx, id)
	if IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(policy.ProfileLinks) != 0 {
		return errors.New("ClearML resource policy still has profile connections")
	}
	if len(policy.Tasks) != 0 {
		return errors.New("ClearML resource policy still has dependent tasks")
	}
	request := struct {
		ID string `json:"id"`
	}{ID: id}
	return c.request(ctx, http.MethodPost, "/resources.delete_policy", request, nil)
}

func (c *ClearMLClient) ConnectResourcePolicyProfile(ctx context.Context, policyID, profileID, queueName string, displayName *string) (string, error) {
	request := struct {
		PolicyID    string  `json:"policy"`
		ProfileID   string  `json:"profile"`
		QueueName   string  `json:"queue_name"`
		DisplayName *string `json:"display_name,omitempty"`
	}{PolicyID: policyID, ProfileID: profileID, QueueName: queueName, DisplayName: displayName}
	var response struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := c.request(ctx, http.MethodPost, "/resources.connect_policy_profile", request, &response); err != nil {
		return "", err
	}
	if response.Data.ID == "" {
		return "", invalidResponse("resources.connect_policy_profile", "id")
	}
	return response.Data.ID, nil
}

func (c *ClearMLClient) GetResourcePolicyProfileConnection(ctx context.Context, policyID, queueID string) (ResourceProfile, Queue, error) {
	policy, err := c.GetResourcePolicy(ctx, policyID)
	if err != nil {
		return ResourceProfile{}, Queue{}, err
	}
	for _, link := range policy.ProfileLinks {
		if link.Queue.ID == queueID {
			return link.Profile, link.Queue, nil
		}
	}
	return ResourceProfile{}, Queue{}, &NotFoundError{Kind: "resource policy profile connection", Value: queueID}
}

func (c *ClearMLClient) DisconnectResourcePolicyProfile(ctx context.Context, queueID string) error {
	one := 1
	queue, err := c.getQueue(ctx, queueID, &one)
	if IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(queue.Entries) != 0 {
		return errors.New("ClearML resource policy queue still contains tasks")
	}
	request := struct {
		QueueID string `json:"queue"`
	}{QueueID: queueID}
	return c.request(ctx, http.MethodPost, "/resources.disconnect_policy_profile", request, nil)
}
