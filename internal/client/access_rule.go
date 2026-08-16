package client

import (
	"context"
	"net/http"
)

type AccessRule struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Description      string `json:"description"`
	EntityType       string `json:"entity_type"`
	EntityID         string `json:"entity_id"`
	AccessPermission string `json:"access_permission"`
	Groups           []struct {
		ID string `json:"id"`
	} `json:"groups"`
	Users []struct {
		ID string `json:"id"`
	} `json:"users"`
}

type AccessRuleInput struct {
	Name              string
	Description       string
	EntityType        string
	EntityID          string
	Permission        string
	GroupIDs          []string
	ServiceAccountIDs []string
}

type accessRuleRequest struct {
	ID               string   `json:"id,omitempty"`
	Name             string   `json:"name"`
	Description      string   `json:"description"`
	EntityType       string   `json:"entity_type"`
	EntityID         string   `json:"entity_id"`
	AccessPermission string   `json:"access_permission"`
	Groups           []string `json:"groups"`
	Users            []string `json:"users"`
}

func accessRulePayload(id string, input AccessRuleInput) accessRuleRequest {
	return accessRuleRequest{
		ID: id, Name: input.Name, Description: input.Description, EntityType: input.EntityType,
		EntityID: input.EntityID, AccessPermission: input.Permission,
		Groups: input.GroupIDs, Users: input.ServiceAccountIDs,
	}
}

func (c *ClearMLClient) CreateAccessRule(ctx context.Context, input AccessRuleInput) (string, error) {
	var response struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := c.request(ctx, http.MethodPost, "/permissions.add_or_update_access_rule", accessRulePayload("", input), &response); err != nil {
		return "", err
	}
	if response.Data.ID == "" {
		return "", invalidResponse("permissions.add_or_update_access_rule", "id")
	}
	return response.Data.ID, nil
}

func (c *ClearMLClient) GetAccessRule(ctx context.Context, id string) (AccessRule, error) {
	request := struct {
		Rules []string `json:"rules"`
	}{Rules: []string{id}}
	var response struct {
		Data struct {
			Rules []AccessRule `json:"rules"`
		} `json:"data"`
	}
	if err := c.request(ctx, http.MethodPost, "/permissions.get_access_rules", request, &response); err != nil {
		return AccessRule{}, err
	}
	return exactlyOne("access rule", id, response.Data.Rules, func(rule AccessRule) string { return rule.ID })
}

func (c *ClearMLClient) UpdateAccessRule(ctx context.Context, id string, input AccessRuleInput) error {
	return c.request(ctx, http.MethodPost, "/permissions.add_or_update_access_rule", accessRulePayload(id, input), nil)
}

func (c *ClearMLClient) DeleteAccessRule(ctx context.Context, id string) error {
	request := struct {
		Rules []string `json:"rules"`
	}{Rules: []string{id}}
	return c.request(ctx, http.MethodPost, "/permissions.delete_access_rules", request, nil)
}
