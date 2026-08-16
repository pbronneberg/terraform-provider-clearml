package provider

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/pbronneberg/terraform-provider-clearml/internal/client"
)

func acceptanceProviderFactories() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"clearml": providerserver.NewProtocol6WithError(New("acceptance")()),
	}
}

type acceptanceDeleteFunc func(context.Context, *client.ClearMLClient, string) error

func testAccRemoveRemote(address, idAttribute string, deleteObject acceptanceDeleteFunc) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		instance, ok := state.RootModule().Resources[address]
		if !ok {
			return fmt.Errorf("resource %s not found in state", address)
		}
		id := instance.Primary.ID
		if idAttribute != "" {
			id = instance.Primary.Attributes[idAttribute]
		}
		apiHost := os.Getenv("CLEARML_API_HOST")
		if apiHost == "" {
			apiHost = "https://api.clear.ml"
		}
		apiClient, err := client.NewClearMLClient(
			context.Background(), "terraform-provider-clearml/acceptance-disappearance",
			os.Getenv("CLEARML_API_ACCESS_KEY"), os.Getenv("CLEARML_API_SECRET_KEY"), apiHost,
		)
		if err != nil {
			return fmt.Errorf("configure disappearance test client: %w", err)
		}
		return deleteObject(context.Background(), apiClient, id)
	}
}

func requireCoreAcceptance(t *testing.T) {
	t.Helper()
	if os.Getenv("TF_ACC") != "1" {
		t.Skip("set TF_ACC=1 to run acceptance tests")
	}
	if os.Getenv("CLEARML_API_ACCESS_KEY") == "" || os.Getenv("CLEARML_API_SECRET_KEY") == "" {
		t.Fatal("CLEARML_API_ACCESS_KEY and CLEARML_API_SECRET_KEY must be set for acceptance tests")
	}
}

func requireEnterpriseAcceptance(t *testing.T, variables ...string) {
	t.Helper()
	requireCoreAcceptance(t)
	if os.Getenv("CLEARML_ENTERPRISE_ACC") != "1" {
		t.Skip("set CLEARML_ENTERPRISE_ACC=1 to run Enterprise acceptance tests")
	}
	for _, variable := range variables {
		if os.Getenv(variable) == "" {
			t.Fatalf("%s must be set for this Enterprise acceptance test", variable)
		}
	}
}
