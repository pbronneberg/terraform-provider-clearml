package provider

import (
	"context"
	"testing"

	frameworkprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
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
