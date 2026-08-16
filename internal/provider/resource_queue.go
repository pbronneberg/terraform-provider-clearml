package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/pbronneberg/terraform-provider-clearml/internal/client"
)

type queueResource struct{ client *client.ClearMLClient }

type queueResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	DisplayName types.String `tfsdk:"display_name"`
	Tags        types.Set    `tfsdk:"tags"`
	Metadata    types.Map    `tfsdk:"metadata"`
}

func newQueueResource() resource.Resource { return &queueResource{} }

func (r *queueResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_queue"
}

func (r *queueResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A ClearML task execution queue.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true},
			"name": schema.StringAttribute{
				Required:    true,
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
				Description: "The unique queue name.",
			},
			"display_name": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Description:   "An optional human-readable queue name. Omit it to preserve the remote value.",
			},
			"tags": schema.SetAttribute{
				Optional: true, Computed: true, ElementType: types.StringType,
				PlanModifiers: []planmodifier.Set{setplanmodifier.UseStateForUnknown()},
				Description:   "User-defined tags. Omit them to preserve the remote value.",
			},
			"metadata": schema.MapAttribute{
				Optional: true, Computed: true,
				ElementType:   types.ObjectType{AttrTypes: metadataItemTypes},
				PlanModifiers: []planmodifier.Map{mapplanmodifier.UseStateForUnknown()},
				Description:   "Typed metadata keyed by metadata key. Omit it to preserve the remote value.",
			},
		},
	}
}

func (r *queueResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configuredClient(req.ProviderData, &resp.Diagnostics, "resource")
}

func queueInput(ctx context.Context, model queueResourceModel) (client.QueueInput, diag.Diagnostics) {
	tags, diagnostics := setStrings(ctx, model.Tags)
	metadata, metadataDiagnostics := metadataItems(ctx, model.Metadata)
	diagnostics.Append(metadataDiagnostics...)
	return client.QueueInput{
		Name: model.Name.ValueString(), DisplayName: optionalString(model.DisplayName),
		Tags: tags, Metadata: metadata,
	}, diagnostics
}

func (r *queueResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan queueResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	input, diagnostics := queueInput(ctx, plan)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, err := r.client.CreateQueue(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Create ClearML queue", err.Error())
		return
	}
	plan.ID = types.StringValue(id)
	r.read(ctx, &plan, &resp.Diagnostics)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	}
}

func (r *queueResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state queueResourceModel
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

func (r *queueResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state queueResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	plan.ID = state.ID
	input, diagnostics := queueInput(ctx, plan)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.UpdateQueue(ctx, state.ID.ValueString(), input); err != nil {
		resp.Diagnostics.AddError("Update ClearML queue", err.Error())
		return
	}
	r.read(ctx, &plan, &resp.Diagnostics)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	}
}

func (r *queueResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state queueResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteQueue(ctx, state.ID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Delete ClearML queue", "ClearML refused the safe non-force deletion. Ensure the queue is empty. "+err.Error())
	}
}

func (r *queueResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *queueResource) read(ctx context.Context, state *queueResourceModel, diagnostics *diag.Diagnostics) bool {
	queue, err := r.client.GetQueue(ctx, state.ID.ValueString())
	if client.IsNotFound(err) {
		return false
	}
	if err != nil {
		diagnostics.AddError("Read ClearML queue", err.Error())
		return true
	}
	tags, tagDiagnostics := stringSet(ctx, queue.Tags)
	metadata, metadataDiagnostics := metadataMap(ctx, queue.Metadata)
	diagnostics.Append(tagDiagnostics...)
	diagnostics.Append(metadataDiagnostics...)
	state.ID = types.StringValue(queue.ID)
	state.Name = types.StringValue(queue.Name)
	state.DisplayName = types.StringValue(queue.DisplayName)
	state.Tags = tags
	state.Metadata = metadata
	return true
}

var _ resource.Resource = &queueResource{}
var _ resource.ResourceWithConfigure = &queueResource{}
var _ resource.ResourceWithImportState = &queueResource{}
