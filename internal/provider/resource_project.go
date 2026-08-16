package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/pbronneberg/terraform-provider-clearml/internal/client"
)

type projectResource struct{ client *client.ClearMLClient }

type projectResourceModel struct {
	ID                       types.String `tfsdk:"id"`
	Name                     types.String `tfsdk:"name"`
	Description              types.String `tfsdk:"description"`
	Tags                     types.Set    `tfsdk:"tags"`
	DefaultOutputDestination types.String `tfsdk:"default_output_destination"`
}

func newProjectResource() resource.Resource { return &projectResource{} }

func (r *projectResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (r *projectResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A hierarchical ClearML project.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true},
			"name": schema.StringAttribute{
				Required:    true,
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
				Description: "The unique full project name. Use slash-separated names for subprojects.",
			},
			"description": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Description:   "The project description. Omit it to preserve the remote value.",
			},
			"tags": schema.SetAttribute{
				Optional: true, Computed: true, ElementType: types.StringType,
				PlanModifiers: []planmodifier.Set{setplanmodifier.UseStateForUnknown()},
				Description:   "User-defined project tags. Omit them to preserve the remote value.",
			},
			"default_output_destination": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Description:   "Default output destination for new tasks. Omit it to preserve the remote value.",
			},
		},
	}
}

func (r *projectResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configuredClient(req.ProviderData, &resp.Diagnostics, "resource")
}

func projectInput(ctx context.Context, model projectResourceModel) (client.ProjectInput, diag.Diagnostics) {
	tags, diagnostics := setStrings(ctx, model.Tags)
	return client.ProjectInput{
		Name: model.Name.ValueString(), Description: optionalString(model.Description), Tags: tags,
		DefaultOutputDestination: optionalString(model.DefaultOutputDestination),
	}, diagnostics
}

func (r *projectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan projectResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	input, diagnostics := projectInput(ctx, plan)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, err := r.client.CreateProject(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Create ClearML project", err.Error())
		return
	}
	plan.ID = types.StringValue(id)
	r.read(ctx, &plan, &resp.Diagnostics)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	}
}

func (r *projectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state projectResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !r.read(ctx, &state, &resp.Diagnostics) {
		resp.State.RemoveResource(ctx)
		return
	}
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
	}
}

func (r *projectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state projectResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	plan.ID = state.ID
	input, diagnostics := projectInput(ctx, plan)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	if projectParent(state.Name.ValueString()) != projectParent(plan.Name.ValueString()) {
		if err := r.client.MoveProject(ctx, state.ID.ValueString(), projectParent(plan.Name.ValueString())); err != nil {
			resp.Diagnostics.AddError("Move ClearML project", err.Error())
			return
		}
	}
	if err := r.client.UpdateProject(ctx, state.ID.ValueString(), input); err != nil {
		resp.Diagnostics.AddError("Update ClearML project", err.Error())
		return
	}
	r.read(ctx, &plan, &resp.Diagnostics)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	}
}

func projectParent(name string) string {
	if index := strings.LastIndex(name, "/"); index >= 0 {
		return name[:index]
	}
	return ""
}

func (r *projectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state projectResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteProject(ctx, state.ID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Delete ClearML project", "ClearML refused the safe deletion. Ensure the project and its subprojects contain no tasks, models, datasets, or reports. "+err.Error())
	}
}

func (r *projectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *projectResource) read(ctx context.Context, state *projectResourceModel, diagnostics *diag.Diagnostics) bool {
	project, err := r.client.GetProject(ctx, state.ID.ValueString())
	if client.IsNotFound(err) {
		return false
	}
	if err != nil {
		diagnostics.AddError("Read ClearML project", err.Error())
		return true
	}
	tags, tagDiagnostics := stringSet(ctx, project.Tags)
	diagnostics.Append(tagDiagnostics...)
	state.ID = types.StringValue(project.ID)
	state.Name = types.StringValue(project.Name)
	state.Description = types.StringValue(project.Description)
	state.Tags = tags
	state.DefaultOutputDestination = types.StringValue(project.DefaultOutputDestination)
	return true
}

var _ resource.Resource = &projectResource{}
var _ resource.ResourceWithConfigure = &projectResource{}
var _ resource.ResourceWithImportState = &projectResource{}
