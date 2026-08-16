package provider

import (
	"context"
	"reflect"
	"sort"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	frameworkprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

func TestProviderImplementation(t *testing.T) {
	t.Parallel()
	var _ frameworkprovider.Provider = New("test")()
	var _ resource.Resource = newQueueResource()
}

func TestProviderRegistersFocusedV1Interfaces(t *testing.T) {
	t.Parallel()
	p := New("test")()
	if got, want := resourceTypeNames(t, p.Resources(context.Background())), []string{
		"clearml_access_rule", "clearml_project", "clearml_queue", "clearml_resource_policy", "clearml_resource_policy_profile_connection",
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Resources() = %v, want %v", got, want)
	}
	if got, want := dataSourceTypeNames(t, p.DataSources(context.Background())), []string{
		"clearml_project", "clearml_queue", "clearml_resource_profile", "clearml_service_account", "clearml_user_group",
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("DataSources() = %v, want %v", got, want)
	}
}

func TestProviderMetadata(t *testing.T) {
	t.Parallel()
	p := New("1.0.0")()
	response := &frameworkprovider.MetadataResponse{}
	p.Metadata(context.Background(), frameworkprovider.MetadataRequest{}, response)
	if response.TypeName != "clearml" || response.Version != "1.0.0" {
		t.Fatalf("Metadata() = %#v, want clearml/1.0.0", response)
	}
}

func TestProviderUsesCanonicalConfiguration(t *testing.T) {
	t.Parallel()
	response := &frameworkprovider.SchemaResponse{}
	New("test")().Schema(context.Background(), frameworkprovider.SchemaRequest{}, response)

	for _, name := range []string{"api_host", "access_key", "secret_key"} {
		if response.Schema.Attributes[name] == nil {
			t.Fatalf("provider attribute %q is missing", name)
		}
	}
	for _, name := range []string{"api_url"} {
		if response.Schema.Attributes[name] != nil {
			t.Fatalf("legacy provider attribute %q is still registered", name)
		}
	}
	assertSensitive(t, response.Schema.Attributes["access_key"])
	assertSensitive(t, response.Schema.Attributes["secret_key"])
}

func TestConfiguredValuePrecedence(t *testing.T) {
	t.Setenv("CLEARML_TEST_VALUE", "environment")
	if got := configuredValue(types.StringValue("configuration"), "CLEARML_TEST_VALUE", "fallback"); got != "configuration" {
		t.Fatalf("explicit value = %q, want configuration", got)
	}
	if got := configuredValue(types.StringNull(), "CLEARML_TEST_VALUE", "fallback"); got != "environment" {
		t.Fatalf("environment value = %q, want environment", got)
	}
	t.Setenv("CLEARML_TEST_VALUE", "")
	if got := configuredValue(types.StringNull(), "CLEARML_TEST_VALUE", "fallback"); got != "fallback" {
		t.Fatalf("fallback value = %q, want fallback", got)
	}
}

func TestProviderProtocolSchema(t *testing.T) {
	t.Parallel()
	server := providerserver.NewProtocol6(New("test")())()
	response, err := server.GetProviderSchema(context.Background(), &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatalf("GetProviderSchema() error = %v", err)
	}
	for _, diagnostic := range response.Diagnostics {
		if diagnostic.Severity == tfprotov6.DiagnosticSeverityError {
			t.Errorf("provider schema diagnostic: %s: %s", diagnostic.Summary, diagnostic.Detail)
		}
	}
}

func resourceTypeNames(t *testing.T, factories []func() resource.Resource) []string {
	t.Helper()
	names := make([]string, 0, len(factories))
	for _, factory := range factories {
		response := &resource.MetadataResponse{}
		factory().Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "clearml"}, response)
		names = append(names, response.TypeName)
	}
	sort.Strings(names)
	return names
}

func dataSourceTypeNames(t *testing.T, factories []func() datasource.DataSource) []string {
	t.Helper()
	names := make([]string, 0, len(factories))
	for _, factory := range factories {
		response := &datasource.MetadataResponse{}
		factory().Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "clearml"}, response)
		names = append(names, response.TypeName)
	}
	sort.Strings(names)
	return names
}

func assertSensitive(t *testing.T, attribute schema.Attribute) {
	t.Helper()
	if !attribute.IsSensitive() {
		t.Fatal("credential attribute must be sensitive")
	}
}
