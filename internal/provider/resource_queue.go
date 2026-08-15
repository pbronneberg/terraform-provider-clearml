package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/pbronneberg/terraform-provider-clearml/internal/client"
)

type queueResource struct{ client *client.ClearMLClient }

type queueResourceModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	Tags types.List   `tfsdk:"tags"`
}

func newQueueResource() resource.Resource { return &queueResource{} }

func (r *queueResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_queue"
}

func (r *queueResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "A queue in ClearML.", Attributes: map[string]schema.Attribute{
		"id":   schema.StringAttribute{Computed: true},
		"name": schema.StringAttribute{Required: true, Description: "The name of the queue."},
		"tags": schema.ListAttribute{Optional: true, ElementType: types.StringType, Description: "Tags to set on the queue."},
	}}
}

func (r *queueResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.ClearMLClient)
	if !ok {
		resp.Diagnostics.AddError("Unexpected resource configure type", fmt.Sprintf("Expected *client.ClearMLClient, got %T.", req.ProviderData))
		return
	}
	r.client = c
}

func (r *queueResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan queueResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tags, diagnostics := listStrings(ctx, plan.Tags)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, err := r.client.CreateQueue(ctx, plan.Name.ValueString(), tags)
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
	r.read(ctx, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if state.ID.IsNull() {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *queueResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan queueResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state queueResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Computed attributes are unknown in an update plan. The remote queue ID is
	// therefore taken from the prior state, not plan.ID.
	plan.ID = state.ID
	tags, diagnostics := listStrings(ctx, plan.Tags)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.UpdateQueue(ctx, plan.ID.ValueString(), plan.Name.ValueString(), tags); err != nil {
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
		resp.Diagnostics.AddError("Delete ClearML queue", err.Error())
	}
}

func (r *queueResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *queueResource) read(ctx context.Context, state *queueResourceModel, diagnostics *diag.Diagnostics) {
	queue, err := r.client.GetQueue(ctx, state.ID.ValueString())
	if client.IsNotFound(err) {
		state.ID = types.StringNull()
		return
	}
	if err != nil {
		diagnostics.AddError("Read ClearML queue", err.Error())
		return
	}
	tags, tagDiagnostics := types.ListValueFrom(ctx, types.StringType, queue.Tags)
	diagnostics.Append(tagDiagnostics...)
	if diagnostics.HasError() {
		return
	}
	state.ID = types.StringValue(queue.ID)
	state.Name = types.StringValue(queue.Name)
	state.Tags = tags
}

func listStrings(ctx context.Context, list types.List) ([]string, diag.Diagnostics) {
	if list.IsNull() || list.IsUnknown() {
		return nil, nil
	}
	var values []string
	diagnostics := list.ElementsAs(ctx, &values, false)
	return values, diagnostics
}

var _ resource.Resource = &queueResource{}
var _ resource.ResourceWithConfigure = &queueResource{}
var _ resource.ResourceWithImportState = &queueResource{}
