package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/pbronneberg/terraform-provider-clearml/internal/client"
)

func TestAccProjectLifecycle(t *testing.T) {
	requireCoreAcceptance(t)
	initialName := acceptanceQueueName(t) + "/initial"
	updatedName := acceptanceQueueName(t) + "/updated"
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acceptanceProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProjectConfig(initialName, "initial", []string{"acceptance", "initial"}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("clearml_project.test", "id"),
					resource.TestCheckResourceAttr("clearml_project.test", "name", initialName),
					resource.TestCheckResourceAttr("clearml_project.test", "tags.#", "2"),
				),
			},
			{
				Config:             testAccProjectConfig(updatedName, "", []string{}),
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("clearml_project.test", "name", updatedName),
					resource.TestCheckResourceAttr("clearml_project.test", "description", ""),
					resource.TestCheckResourceAttr("clearml_project.test", "tags.#", "0"),
					testAccRemoveRemote("clearml_project.test", "", func(ctx context.Context, apiClient *client.ClearMLClient, id string) error {
						return apiClient.DeleteProject(ctx, id)
					}),
				),
			},
			{
				Config: testAccProjectConfig(updatedName, "", []string{}),
				Check:  resource.TestCheckResourceAttrSet("clearml_project.test", "id"),
			},
			{ResourceName: "clearml_project.test", ImportState: true, ImportStateVerify: true},
		},
	})
}

func testAccProjectConfig(name, description string, tags []string) string {
	tagValues := ""
	for index, tag := range tags {
		if index > 0 {
			tagValues += ", "
		}
		tagValues += fmt.Sprintf("%q", tag)
	}
	return fmt.Sprintf(`
resource "clearml_project" "test" {
  name                       = %q
  description                = %q
  tags                       = [%s]
  default_output_destination = ""
}
`, name, description, tagValues)
}
