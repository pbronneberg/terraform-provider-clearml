package client

import (
	"context"
	"net/http"
)

type ServiceAccount struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	Role                string `json:"role"`
	AllowRunningAsOwner bool   `json:"allow_running_as_owner"`
	Credentials         int64  `json:"credentials"`
}

type UserGroup struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Assignable  bool   `json:"assignable"`
	System      bool   `json:"system"`
	Users       []struct {
		ID string `json:"id"`
	} `json:"users"`
}

func (c *ClearMLClient) FindServiceAccount(ctx context.Context, name string) (ServiceAccount, error) {
	var response struct {
		Data struct {
			Users []ServiceAccount `json:"users"`
		} `json:"data"`
	}
	if err := c.request(ctx, http.MethodPost, "/users.get_service_users", struct{}{}, &response); err != nil {
		return ServiceAccount{}, err
	}
	matches := make([]ServiceAccount, 0, 1)
	for _, account := range response.Data.Users {
		if account.Name == name {
			matches = append(matches, account)
		}
	}
	return exactlyOne("service account", name, matches, func(account ServiceAccount) string { return account.ID })
}

func (c *ClearMLClient) FindUserGroup(ctx context.Context, name string) (UserGroup, error) {
	request := struct {
		IncludePermissions bool `json:"include_permissions"`
	}{IncludePermissions: false}
	var response struct {
		Data struct {
			Groups []UserGroup `json:"groups"`
		} `json:"data"`
	}
	if err := c.request(ctx, http.MethodPost, "/permissions.get_user_groups", request, &response); err != nil {
		return UserGroup{}, err
	}
	matches := make([]UserGroup, 0, 1)
	for _, group := range response.Data.Groups {
		if group.Name == name {
			matches = append(matches, group)
		}
	}
	return exactlyOne("user group", name, matches, func(group UserGroup) string { return group.ID })
}
