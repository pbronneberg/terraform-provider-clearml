package client

import (
	"context"
	"errors"
	"net/http"
	"regexp"
)

type Project struct {
	ID                       string   `json:"id"`
	Name                     string   `json:"name"`
	Description              string   `json:"description"`
	Tags                     []string `json:"tags"`
	DefaultOutputDestination string   `json:"default_output_destination"`
}

type ProjectInput struct {
	Name                     string
	Description              *string
	Tags                     *[]string
	DefaultOutputDestination *string
}

type projectMutationRequest struct {
	Project                  string    `json:"project,omitempty"`
	Name                     string    `json:"name"`
	Description              *string   `json:"description,omitempty"`
	Tags                     *[]string `json:"tags,omitempty"`
	DefaultOutputDestination *string   `json:"default_output_destination,omitempty"`
}

type projectDeletionImpact struct {
	Datasets           int `json:"datasets"`
	Dataviews          int `json:"dataviews"`
	HyperDatasets      int `json:"hyper_datasets"`
	Models             int `json:"models"`
	NonArchivedModels  int `json:"non_archived_models"`
	NonArchivedReports int `json:"non_archived_reports"`
	NonArchivedTasks   int `json:"non_archived_tasks"`
	Pipelines          int `json:"pipelines"`
	Reports            int `json:"reports"`
	Tasks              int `json:"tasks"`
}

func (impact projectDeletionImpact) empty() bool {
	return impact.Datasets+impact.Dataviews+impact.HyperDatasets+impact.Models+
		impact.NonArchivedModels+impact.NonArchivedReports+impact.NonArchivedTasks+
		impact.Pipelines+impact.Reports+impact.Tasks == 0
}

func (c *ClearMLClient) CreateProject(ctx context.Context, input ProjectInput) (string, error) {
	request := projectMutationRequest{Name: input.Name, Description: input.Description, Tags: input.Tags, DefaultOutputDestination: input.DefaultOutputDestination}
	var response struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := c.request(ctx, http.MethodPost, "/projects.create", request, &response); err != nil {
		return "", err
	}
	if response.Data.ID == "" {
		return "", invalidResponse("projects.create", "id")
	}
	return response.Data.ID, nil
}

func (c *ClearMLClient) GetProject(ctx context.Context, id string) (Project, error) {
	request := struct {
		Project string `json:"project"`
	}{Project: id}
	var response struct {
		Data struct {
			Project Project `json:"project"`
		} `json:"data"`
	}
	if err := c.request(ctx, http.MethodPost, "/projects.get_by_id", request, &response); err != nil {
		return Project{}, err
	}
	if response.Data.Project.ID == "" {
		return Project{}, &NotFoundError{Kind: "project", Value: id}
	}
	return response.Data.Project, nil
}

func (c *ClearMLClient) FindProject(ctx context.Context, name string) (Project, error) {
	request := struct {
		Name     string `json:"name"`
		Page     int    `json:"page"`
		PageSize int    `json:"page_size"`
	}{Name: "^" + regexp.QuoteMeta(name) + "$", Page: 0, PageSize: 2}
	var response struct {
		Data struct {
			Projects []Project `json:"projects"`
		} `json:"data"`
	}
	if err := c.request(ctx, http.MethodPost, "/projects.get_all", request, &response); err != nil {
		return Project{}, err
	}
	return exactlyOne("project", name, response.Data.Projects, func(project Project) string { return project.ID })
}

func (c *ClearMLClient) UpdateProject(ctx context.Context, id string, input ProjectInput) error {
	request := projectMutationRequest{Project: id, Name: input.Name, Description: input.Description, Tags: input.Tags, DefaultOutputDestination: input.DefaultOutputDestination}
	return c.request(ctx, http.MethodPost, "/projects.update", request, nil)
}

func (c *ClearMLClient) MoveProject(ctx context.Context, id, location string) error {
	request := struct {
		Project     string `json:"project"`
		NewLocation string `json:"new_location"`
	}{Project: id, NewLocation: location}
	return c.request(ctx, http.MethodPost, "/projects.move", request, nil)
}

func (c *ClearMLClient) DeleteProject(ctx context.Context, id string) error {
	request := struct {
		Project string `json:"project"`
	}{Project: id}
	var validation struct {
		Data projectDeletionImpact `json:"data"`
	}
	if err := c.request(ctx, http.MethodPost, "/projects.validate_delete", request, &validation); err != nil {
		return err
	}
	if !validation.Data.empty() {
		return errors.New("ClearML project is not empty")
	}
	deletion := struct {
		Project                 string `json:"project"`
		Force                   bool   `json:"force"`
		DeleteContents          bool   `json:"delete_contents"`
		DeleteExternalArtifacts bool   `json:"delete_external_artifacts"`
	}{Project: id}
	return c.request(ctx, http.MethodPost, "/projects.delete", deletion, nil)
}
