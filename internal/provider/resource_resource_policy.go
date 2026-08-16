package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/pbronneberg/terraform-provider-clearml/internal/client"
)

type resourcePolicyResource struct{ client *client.ClearMLClient }

type resourcePolicyResourceModel struct {
	ID          types.String  `tfsdk:"id"`
	Name        types.String  `tfsdk:"name"`
	Description types.String  `tfsdk:"description"`
	Reservation types.Float64 `tfsdk:"reservation"`
	Limit       types.Float64 `tfsdk:"limit"`
	UserGroupID types.String  `tfsdk:"user_group_id"`
}

func newResourcePolicyResource() resource.Resource { return &resourcePolicyResource{} }

func (r *resourcePolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource_policy"
}

func (r *resourcePolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A ClearML Enterprise resource policy for one user group. The vendor-managed resource policy manager must be enabled.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true},
			"name": schema.StringAttribute{
				Required: true, Validators: []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"description": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Description:   "Policy description. Omit it to preserve the remote value.",
			},
			"reservation": schema.Float64Attribute{
				Required: true, Validators: []validator.Float64{float64validator.AtLeast(0)},
			},
			"limit": schema.Float64Attribute{
				Required: true, Validators: []validator.Float64{float64validator.AtLeast(0)},
			},
			"user_group_id": schema.StringAttribute{
				Required: true, Validators: []validator.String{stringvalidator.LengthAtLeast(1)},
			},
		},
	}
}

func (r *resourcePolicyResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config resourcePolicyResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() || config.Reservation.IsNull() || config.Reservation.IsUnknown() || config.Limit.IsNull() || config.Limit.IsUnknown() {
		return
	}
	if config.Reservation.ValueFloat64() > config.Limit.ValueFloat64() {
		resp.Diagnostics.AddAttributeError(path.Root("reservation"), "Invalid ClearML resource policy", "reservation cannot exceed limit.")
	}
}

func (r *resourcePolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configuredClient(req.ProviderData, &resp.Diagnostics, "resource")
}

func resourcePolicyInput(model resourcePolicyResourceModel) client.ResourcePolicyInput {
	return client.ResourcePolicyInput{
		Name: model.Name.ValueString(), Description: optionalString(model.Description),
		Reservation: model.Reservation.ValueFloat64(), Limit: model.Limit.ValueFloat64(),
		UserGroupID: model.UserGroupID.ValueString(),
	}
}

func (r *resourcePolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan resourcePolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, err := r.client.CreateResourcePolicy(ctx, resourcePolicyInput(plan))
	if err != nil {
		resp.Diagnostics.AddError("Create ClearML resource policy", err.Error())
		return
	}
	plan.ID = types.StringValue(id)
	r.read(ctx, &plan, &resp.Diagnostics)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	}
}

func (r *resourcePolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state resourcePolicyResourceModel
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

func (r *resourcePolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state resourcePolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	plan.ID = state.ID
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.UpdateResourcePolicy(ctx, state.ID.ValueString(), resourcePolicyInput(plan)); err != nil {
		resp.Diagnostics.AddError("Update ClearML resource policy", err.Error())
		return
	}
	r.read(ctx, &plan, &resp.Diagnostics)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	}
}

func (r *resourcePolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state resourcePolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteResourcePolicy(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Delete ClearML resource policy", "Remove profile connections and ensure their queues are empty first. "+err.Error())
	}
}

func (r *resourcePolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *resourcePolicyResource) read(ctx context.Context, state *resourcePolicyResourceModel, diagnostics *diag.Diagnostics) bool {
	policy, err := r.client.GetResourcePolicy(ctx, state.ID.ValueString())
	if client.IsNotFound(err) {
		return false
	}
	if err != nil {
		diagnostics.AddError("Read ClearML resource policy", err.Error())
		return true
	}
	state.ID = types.StringValue(policy.ID)
	state.Name = types.StringValue(policy.Name)
	state.Description = types.StringValue(policy.Description)
	state.Reservation = types.Float64Value(policy.Reservation)
	state.Limit = types.Float64Value(policy.Limit)
	state.UserGroupID = types.StringValue(policy.UserGroup)
	return true
}

var _ resource.Resource = &resourcePolicyResource{}
var _ resource.ResourceWithConfigure = &resourcePolicyResource{}
var _ resource.ResourceWithImportState = &resourcePolicyResource{}
var _ resource.ResourceWithValidateConfig = &resourcePolicyResource{}
