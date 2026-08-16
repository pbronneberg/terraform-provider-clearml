package provider

import (
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	frameworkprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/pbronneberg/terraform-provider-clearml/internal/client"
)

type clearmlProvider struct {
	version string
}

type providerModel struct {
	APIHost   types.String `tfsdk:"api_host"`
	AccessKey types.String `tfsdk:"access_key"`
	SecretKey types.String `tfsdk:"secret_key"`
}

func New(version string) func() frameworkprovider.Provider {
	return func() frameworkprovider.Provider { return &clearmlProvider{version: version} }
}

func (p *clearmlProvider) Metadata(_ context.Context, _ frameworkprovider.MetadataRequest, resp *frameworkprovider.MetadataResponse) {
	resp.TypeName = "clearml"
	resp.Version = p.version
}

func (p *clearmlProvider) Schema(_ context.Context, _ frameworkprovider.SchemaRequest, resp *frameworkprovider.SchemaResponse) {
	resp.Schema = schema.Schema{Attributes: map[string]schema.Attribute{
		"api_host": schema.StringAttribute{
			Optional:    true,
			Description: "ClearML API host. Defaults to CLEARML_API_HOST, then https://api.clear.ml.",
		},
		"access_key": schema.StringAttribute{
			Optional:    true,
			Sensitive:   true,
			Description: "ClearML access key. Defaults to CLEARML_API_ACCESS_KEY when unset.",
		},
		"secret_key": schema.StringAttribute{
			Optional:    true,
			Sensitive:   true,
			Description: "ClearML secret key. Defaults to CLEARML_API_SECRET_KEY when unset.",
		},
	}}
}

func (p *clearmlProvider) Configure(ctx context.Context, req frameworkprovider.ConfigureRequest, resp *frameworkprovider.ConfigureResponse) {
	var config providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiURL := configuredValue(config.APIHost, "CLEARML_API_HOST", "https://api.clear.ml")
	accessKey := configuredValue(config.AccessKey, "CLEARML_API_ACCESS_KEY", "")
	secretKey := configuredValue(config.SecretKey, "CLEARML_API_SECRET_KEY", "")
	if accessKey == "" || secretKey == "" {
		resp.Diagnostics.AddError("Missing ClearML credentials", "Set access_key and secret_key or the CLEARML_API_ACCESS_KEY and CLEARML_API_SECRET_KEY environment variables.")
		return
	}

	c, err := client.NewClearMLClient(ctx, fmt.Sprintf("terraform-provider-clearml/%s", p.version), accessKey, secretKey, apiURL)
	if err != nil {
		resp.Diagnostics.AddError("Configure ClearML client", err.Error())
		return
	}
	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *clearmlProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		newProjectResource,
		newQueueResource,
		newAccessRuleResource,
		newResourcePolicyResource,
		newResourcePolicyProfileConnectionResource,
	}
}

func (p *clearmlProvider) DataSources(context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		newProjectDataSource,
		newQueueDataSource,
		newServiceAccountDataSource,
		newUserGroupDataSource,
		newResourceProfileDataSource,
	}
}

func configuredValue(value types.String, environmentVariable, fallback string) string {
	if !value.IsNull() && !value.IsUnknown() && value.ValueString() != "" {
		return value.ValueString()
	}
	if environmentValue := os.Getenv(environmentVariable); environmentValue != "" {
		return environmentValue
	}
	return fallback
}

var _ frameworkprovider.Provider = &clearmlProvider{}
