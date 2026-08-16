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

type projectDataSource struct{ client *client.ClearMLClient }

type projectDataSourceModel struct {
	ID                       types.String `tfsdk:"id"`
	Name                     types.String `tfsdk:"name"`
	Description              types.String `tfsdk:"description"`
	Tags                     types.Set    `tfsdk:"tags"`
	DefaultOutputDestination types.String `tfsdk:"default_output_destination"`
}

func newProjectDataSource() datasource.DataSource { return &projectDataSource{} }

func (d *projectDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (d *projectDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "Looks up one ClearML project by its exact full name.", Attributes: map[string]schema.Attribute{
		"id": schema.StringAttribute{Computed: true},
		"name": schema.StringAttribute{
			Required: true, Validators: []validator.String{stringvalidator.LengthAtLeast(1)},
		},
		"description":                schema.StringAttribute{Computed: true},
		"tags":                       schema.SetAttribute{Computed: true, ElementType: types.StringType},
		"default_output_destination": schema.StringAttribute{Computed: true},
	}}
}

func (d *projectDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configuredClient(req.ProviderData, &resp.Diagnostics, "data source")
}

func (d *projectDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state projectDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	project, err := d.client.FindProject(ctx, state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Read ClearML project", err.Error())
		return
	}
	tags, diagnostics := stringSet(ctx, project.Tags)
	resp.Diagnostics.Append(diagnostics...)
	state.ID = types.StringValue(project.ID)
	state.Name = types.StringValue(project.Name)
	state.Description = types.StringValue(project.Description)
	state.Tags = tags
	state.DefaultOutputDestination = types.StringValue(project.DefaultOutputDestination)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

var _ datasource.DataSource = &projectDataSource{}
var _ datasource.DataSourceWithConfigure = &projectDataSource{}
