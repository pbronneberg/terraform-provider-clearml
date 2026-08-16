package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/pbronneberg/terraform-provider-clearml/internal/client"
)

type accessRuleResource struct{ client *client.ClearMLClient }

type accessRuleResourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	Description       types.String `tfsdk:"description"`
	EntityType        types.String `tfsdk:"entity_type"`
	EntityID          types.String `tfsdk:"entity_id"`
	Permission        types.String `tfsdk:"permission"`
	GroupIDs          types.Set    `tfsdk:"group_ids"`
	ServiceAccountIDs types.Set    `tfsdk:"service_account_ids"`
}

var accessEntityTypes = []string{
	"project", "task", "model", "dataview", "dataset", "dataset_version",
	"queue", "route", "app", "app_category",
}

func newAccessRuleResource() resource.Resource { return &accessRuleResource{} }

func (r *accessRuleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_access_rule"
}

func (r *accessRuleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A ClearML Enterprise entity access rule.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true},
			"name": schema.StringAttribute{
				Required: true, Validators: []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"description": schema.StringAttribute{Required: true},
			"entity_type": schema.StringAttribute{
				Required: true, Validators: []validator.String{stringvalidator.OneOf(accessEntityTypes...)},
				Description: "Protected ClearML entity type.",
			},
			"entity_id": schema.StringAttribute{
				Optional:    true,
				Description: "Entity ID. Omit it to grant access to all entities of the selected type.",
			},
			"permission": schema.StringAttribute{
				Required: true, Validators: []validator.String{stringvalidator.OneOf("read", "read_write")},
				Description: "Access level: read or read_write.",
			},
			"group_ids": schema.SetAttribute{
				Optional: true, ElementType: types.StringType,
				Validators:  []validator.Set{setvalidator.SizeAtLeast(1)},
				Description: "User group IDs receiving access.",
			},
			"service_account_ids": schema.SetAttribute{
				Optional: true, ElementType: types.StringType,
				Validators:  []validator.Set{setvalidator.SizeAtLeast(1)},
				Description: "Service account IDs receiving access.",
			},
		},
	}
}

func (r *accessRuleResource) ConfigValidators(context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{resourcevalidator.AtLeastOneOf(
		path.MatchRoot("group_ids"), path.MatchRoot("service_account_ids"),
	)}
}

func (r *accessRuleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configuredClient(req.ProviderData, &resp.Diagnostics, "resource")
}

func accessRuleInput(ctx context.Context, model accessRuleResourceModel) (client.AccessRuleInput, diag.Diagnostics) {
	groups, diagnostics := setStrings(ctx, model.GroupIDs)
	accounts, accountDiagnostics := setStrings(ctx, model.ServiceAccountIDs)
	diagnostics.Append(accountDiagnostics...)
	input := client.AccessRuleInput{
		Name: model.Name.ValueString(), Description: model.Description.ValueString(),
		EntityType: model.EntityType.ValueString(), EntityID: stringValue(model.EntityID),
		Permission: model.Permission.ValueString(),
	}
	if groups != nil {
		input.GroupIDs = *groups
	} else {
		input.GroupIDs = []string{}
	}
	if accounts != nil {
		input.ServiceAccountIDs = *accounts
	} else {
		input.ServiceAccountIDs = []string{}
	}
	return input, diagnostics
}

func (r *accessRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan accessRuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	input, diagnostics := accessRuleInput(ctx, plan)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, err := r.client.CreateAccessRule(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Create ClearML access rule", err.Error())
		return
	}
	plan.ID = types.StringValue(id)
	r.read(ctx, &plan, &resp.Diagnostics)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	}
}

func (r *accessRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state accessRuleResourceModel
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

func (r *accessRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state accessRuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	plan.ID = state.ID
	input, diagnostics := accessRuleInput(ctx, plan)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.UpdateAccessRule(ctx, state.ID.ValueString(), input); err != nil {
		resp.Diagnostics.AddError("Update ClearML access rule", err.Error())
		return
	}
	r.read(ctx, &plan, &resp.Diagnostics)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	}
}

func (r *accessRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state accessRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteAccessRule(ctx, state.ID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Delete ClearML access rule", err.Error())
	}
}

func (r *accessRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *accessRuleResource) read(ctx context.Context, state *accessRuleResourceModel, diagnostics *diag.Diagnostics) bool {
	rule, err := r.client.GetAccessRule(ctx, state.ID.ValueString())
	if client.IsNotFound(err) {
		return false
	}
	if err != nil {
		diagnostics.AddError("Read ClearML access rule", err.Error())
		return true
	}
	groups := make([]string, 0, len(rule.Groups))
	for _, group := range rule.Groups {
		groups = append(groups, group.ID)
	}
	accounts := make([]string, 0, len(rule.Users))
	for _, account := range rule.Users {
		accounts = append(accounts, account.ID)
	}
	groupSet, groupDiagnostics := optionalStringSet(ctx, groups, state.GroupIDs)
	accountSet, accountDiagnostics := optionalStringSet(ctx, accounts, state.ServiceAccountIDs)
	diagnostics.Append(groupDiagnostics...)
	diagnostics.Append(accountDiagnostics...)
	state.ID = types.StringValue(rule.ID)
	state.Name = types.StringValue(rule.Name)
	state.Description = types.StringValue(rule.Description)
	state.EntityType = types.StringValue(rule.EntityType)
	entityIDWasConfigured := !state.EntityID.IsNull()
	state.EntityID = types.StringNull()
	if rule.EntityID != "" || entityIDWasConfigured {
		state.EntityID = types.StringValue(rule.EntityID)
	}
	state.Permission = types.StringValue(rule.AccessPermission)
	state.GroupIDs = groupSet
	state.ServiceAccountIDs = accountSet
	return true
}

var _ resource.Resource = &accessRuleResource{}
var _ resource.ResourceWithConfigure = &accessRuleResource{}
var _ resource.ResourceWithConfigValidators = &accessRuleResource{}
var _ resource.ResourceWithImportState = &accessRuleResource{}
