package provider

import (
	"context"
	"testing"

	frameworkprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestProviderImplementation(t *testing.T) {
	t.Parallel()
	var _ frameworkprovider.Provider = New("test")()
	var _ resource.Resource = newQueueResource()
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

func assertSensitive(t *testing.T, attribute schema.Attribute) {
	t.Helper()
	if !attribute.IsSensitive() {
		t.Fatal("credential attribute must be sensitive")
	}
}
