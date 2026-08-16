package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/pbronneberg/terraform-provider-clearml/internal/client"
)

type queueDataSource struct{ client *client.ClearMLClient }

type queueDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	DisplayName types.String `tfsdk:"display_name"`
	Tags        types.Set    `tfsdk:"tags"`
	Metadata    types.Map    `tfsdk:"metadata"`
}

func newQueueDataSource() datasource.DataSource { return &queueDataSource{} }

func (d *queueDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_queue"
}

func (d *queueDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "Looks up one ClearML queue by its exact name.", Attributes: map[string]schema.Attribute{
		"id": schema.StringAttribute{Computed: true},
		"name": schema.StringAttribute{
			Required: true, Validators: []validator.String{stringvalidator.LengthAtLeast(1)},
		},
		"display_name": schema.StringAttribute{Computed: true},
		"tags":         schema.SetAttribute{Computed: true, ElementType: types.StringType},
		"metadata":     schema.MapAttribute{Computed: true, ElementType: types.ObjectType{AttrTypes: metadataItemTypes}},
	}}
}

func (d *queueDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configuredClient(req.ProviderData, &resp.Diagnostics, "data source")
}

func (d *queueDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state queueDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	queue, err := d.client.FindQueue(ctx, state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Read ClearML queue", err.Error())
		return
	}
	tags, tagDiagnostics := stringSet(ctx, queue.Tags)
	metadata, metadataDiagnostics := metadataMap(ctx, queue.Metadata)
	resp.Diagnostics.Append(tagDiagnostics...)
	resp.Diagnostics.Append(metadataDiagnostics...)
	state.ID = types.StringValue(queue.ID)
	state.Name = types.StringValue(queue.Name)
	state.DisplayName = types.StringValue(queue.DisplayName)
	state.Tags = tags
	state.Metadata = metadata
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

var _ datasource.DataSource = &queueDataSource{}
var _ datasource.DataSourceWithConfigure = &queueDataSource{}
