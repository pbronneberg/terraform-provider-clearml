package provider

import (
	"context"
	"fmt"
	"strings"

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

type resourcePolicyProfileConnectionResource struct{ client *client.ClearMLClient }

type resourcePolicyProfileConnectionModel struct {
	PolicyID    types.String `tfsdk:"policy_id"`
	ProfileID   types.String `tfsdk:"profile_id"`
	QueueName   types.String `tfsdk:"queue_name"`
	DisplayName types.String `tfsdk:"display_name"`
	QueueID     types.String `tfsdk:"queue_id"`
}

func newResourcePolicyProfileConnectionResource() resource.Resource {
	return &resourcePolicyProfileConnectionResource{}
}

func (r *resourcePolicyProfileConnectionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource_policy_profile_connection"
}

func requiredReplacement(description string) schema.StringAttribute {
	return schema.StringAttribute{
		Required: true, Description: description,
		Validators:    []validator.String{stringvalidator.LengthAtLeast(1)},
		PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
	}
}

func (r *resourcePolicyProfileConnectionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Connects a ClearML Enterprise resource policy to a vendor-provisioned profile and creates its execution queue.",
		Attributes: map[string]schema.Attribute{
			"policy_id":  requiredReplacement("Resource policy ID."),
			"profile_id": requiredReplacement("Resource profile ID."),
			"queue_name": requiredReplacement("Name of the queue created by this connection."),
			"display_name": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown(), stringplanmodifier.RequiresReplace()},
				Description:   "Optional display name for the generated queue. Omit it to preserve the remote value.",
			},
			"queue_id": schema.StringAttribute{
				Computed: true, Description: "ID of the queue created by the connection.",
			},
		},
	}
}

func (r *resourcePolicyProfileConnectionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configuredClient(req.ProviderData, &resp.Diagnostics, "resource")
}

func (r *resourcePolicyProfileConnectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan resourcePolicyProfileConnectionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	queueID, err := r.client.ConnectResourcePolicyProfile(
		ctx, plan.PolicyID.ValueString(), plan.ProfileID.ValueString(),
		plan.QueueName.ValueString(), optionalString(plan.DisplayName),
	)
	if err != nil {
		resp.Diagnostics.AddError("Connect ClearML resource policy profile", err.Error())
		return
	}
	plan.QueueID = types.StringValue(queueID)
	r.read(ctx, &plan, &resp.Diagnostics)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	}
}

func (r *resourcePolicyProfileConnectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state resourcePolicyProfileConnectionModel
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

func (r *resourcePolicyProfileConnectionResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Unexpected ClearML resource policy profile update", "All configurable connection attributes require replacement.")
}

func (r *resourcePolicyProfileConnectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state resourcePolicyProfileConnectionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DisconnectResourcePolicyProfile(ctx, state.QueueID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Disconnect ClearML resource policy profile", "Ensure the generated queue is empty first. "+err.Error())
	}
}

func (r *resourcePolicyProfileConnectionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError("Invalid ClearML resource policy profile import", fmt.Sprintf("Expected policy-id/queue-id, got %q.", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("policy_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("queue_id"), parts[1])...)
}

func (r *resourcePolicyProfileConnectionResource) read(ctx context.Context, state *resourcePolicyProfileConnectionModel, diagnostics *diag.Diagnostics) bool {
	profile, queue, err := r.client.GetResourcePolicyProfileConnection(ctx, state.PolicyID.ValueString(), state.QueueID.ValueString())
	if client.IsNotFound(err) {
		return false
	}
	if err != nil {
		diagnostics.AddError("Read ClearML resource policy profile connection", err.Error())
		return true
	}
	state.ProfileID = types.StringValue(profile.ID)
	state.QueueID = types.StringValue(queue.ID)
	state.QueueName = types.StringValue(queue.Name)
	state.DisplayName = types.StringValue(queue.DisplayName)
	return true
}

var _ resource.Resource = &resourcePolicyProfileConnectionResource{}
var _ resource.ResourceWithConfigure = &resourcePolicyProfileConnectionResource{}
var _ resource.ResourceWithImportState = &resourcePolicyProfileConnectionResource{}
