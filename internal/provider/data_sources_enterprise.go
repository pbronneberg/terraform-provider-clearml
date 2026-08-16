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

type serviceAccountDataSource struct{ client *client.ClearMLClient }
type userGroupDataSource struct{ client *client.ClearMLClient }
type resourceProfileDataSource struct{ client *client.ClearMLClient }

func requiredName() schema.StringAttribute {
	return schema.StringAttribute{Required: true, Validators: []validator.String{stringvalidator.LengthAtLeast(1)}}
}

func newServiceAccountDataSource() datasource.DataSource { return &serviceAccountDataSource{} }

func (d *serviceAccountDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_account"
}

func (d *serviceAccountDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "Looks up one ClearML Enterprise service account by exact name.", Attributes: map[string]schema.Attribute{
		"id":                     schema.StringAttribute{Computed: true},
		"name":                   requiredName(),
		"role":                   schema.StringAttribute{Computed: true},
		"allow_running_as_owner": schema.BoolAttribute{Computed: true},
		"credentials":            schema.Int64Attribute{Computed: true},
	}}
}

func (d *serviceAccountDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configuredClient(req.ProviderData, &resp.Diagnostics, "data source")
}

func (d *serviceAccountDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state struct {
		ID                  types.String `tfsdk:"id"`
		Name                types.String `tfsdk:"name"`
		Role                types.String `tfsdk:"role"`
		AllowRunningAsOwner types.Bool   `tfsdk:"allow_running_as_owner"`
		Credentials         types.Int64  `tfsdk:"credentials"`
	}
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	account, err := d.client.FindServiceAccount(ctx, state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Read ClearML service account", err.Error())
		return
	}
	state.ID = types.StringValue(account.ID)
	state.Name = types.StringValue(account.Name)
	state.Role = types.StringValue(account.Role)
	state.AllowRunningAsOwner = types.BoolValue(account.AllowRunningAsOwner)
	state.Credentials = types.Int64Value(account.Credentials)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func newUserGroupDataSource() datasource.DataSource { return &userGroupDataSource{} }

func (d *userGroupDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_group"
}

func (d *userGroupDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "Looks up one ClearML Enterprise user group by exact name.", Attributes: map[string]schema.Attribute{
		"id":          schema.StringAttribute{Computed: true},
		"name":        requiredName(),
		"description": schema.StringAttribute{Computed: true},
		"assignable":  schema.BoolAttribute{Computed: true},
		"system":      schema.BoolAttribute{Computed: true},
		"user_ids":    schema.SetAttribute{Computed: true, ElementType: types.StringType},
	}}
}

func (d *userGroupDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configuredClient(req.ProviderData, &resp.Diagnostics, "data source")
}

func (d *userGroupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state struct {
		ID          types.String `tfsdk:"id"`
		Name        types.String `tfsdk:"name"`
		Description types.String `tfsdk:"description"`
		Assignable  types.Bool   `tfsdk:"assignable"`
		System      types.Bool   `tfsdk:"system"`
		UserIDs     types.Set    `tfsdk:"user_ids"`
	}
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	group, err := d.client.FindUserGroup(ctx, state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Read ClearML user group", err.Error())
		return
	}
	userIDs := make([]string, 0, len(group.Users))
	for _, user := range group.Users {
		userIDs = append(userIDs, user.ID)
	}
	users, diagnostics := stringSet(ctx, userIDs)
	resp.Diagnostics.Append(diagnostics...)
	state.ID = types.StringValue(group.ID)
	state.Name = types.StringValue(group.Name)
	state.Description = types.StringValue(group.Description)
	state.Assignable = types.BoolValue(group.Assignable)
	state.System = types.BoolValue(group.System)
	state.UserIDs = users
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func newResourceProfileDataSource() datasource.DataSource { return &resourceProfileDataSource{} }

func (d *resourceProfileDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource_profile"
}

func (d *resourceProfileDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "Looks up one vendor-provisioned ClearML resource profile by exact name.", Attributes: map[string]schema.Attribute{
		"id":           schema.StringAttribute{Computed: true},
		"name":         requiredName(),
		"description":  schema.StringAttribute{Computed: true},
		"profile_cost": schema.Float64Attribute{Computed: true},
		"cost_field":   schema.StringAttribute{Computed: true},
	}}
}

func (d *resourceProfileDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configuredClient(req.ProviderData, &resp.Diagnostics, "data source")
}

func (d *resourceProfileDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state struct {
		ID          types.String  `tfsdk:"id"`
		Name        types.String  `tfsdk:"name"`
		Description types.String  `tfsdk:"description"`
		ProfileCost types.Float64 `tfsdk:"profile_cost"`
		CostField   types.String  `tfsdk:"cost_field"`
	}
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	profile, err := d.client.FindResourceProfile(ctx, state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Read ClearML resource profile", err.Error())
		return
	}
	state.ID = types.StringValue(profile.ID)
	state.Name = types.StringValue(profile.Name)
	state.Description = types.StringValue(profile.Description)
	state.ProfileCost = types.Float64Value(profile.ProfileCost)
	state.CostField = types.StringValue(profile.CostField)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

var _ datasource.DataSource = &serviceAccountDataSource{}
var _ datasource.DataSource = &userGroupDataSource{}
var _ datasource.DataSource = &resourceProfileDataSource{}
var _ datasource.DataSourceWithConfigure = &serviceAccountDataSource{}
var _ datasource.DataSourceWithConfigure = &userGroupDataSource{}
var _ datasource.DataSourceWithConfigure = &resourceProfileDataSource{}
