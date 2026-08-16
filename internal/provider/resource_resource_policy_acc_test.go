package provider

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/pbronneberg/terraform-provider-clearml/internal/client"
)

func TestAccResourcePolicyLifecycle(t *testing.T) {
	requireEnterpriseAcceptance(t, "CLEARML_ENTERPRISE_TEST_GROUP_ID")
	name := acceptanceQueueName(t)
	groupID := os.Getenv("CLEARML_ENTERPRISE_TEST_GROUP_ID")
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acceptanceProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccResourcePolicyConfig(name, "initial", groupID, 1, 2),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("clearml_resource_policy.test", "id"),
					resource.TestCheckResourceAttr("clearml_resource_policy.test", "reservation", "1"),
					resource.TestCheckResourceAttr("clearml_resource_policy.test", "limit", "2"),
				),
			},
			{
				Config:             testAccResourcePolicyConfig(name, "updated", groupID, 2, 4),
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("clearml_resource_policy.test", "description", "updated"),
					testAccRemoveRemote("clearml_resource_policy.test", "", func(ctx context.Context, apiClient *client.ClearMLClient, id string) error {
						return apiClient.DeleteResourcePolicy(ctx, id)
					}),
				),
			},
			{
				Config: testAccResourcePolicyConfig(name, "updated", groupID, 2, 4),
				Check:  resource.TestCheckResourceAttrSet("clearml_resource_policy.test", "id"),
			},
			{ResourceName: "clearml_resource_policy.test", ImportState: true, ImportStateVerify: true},
		},
	})
}

func testAccResourcePolicyConfig(name, description, groupID string, reservation, limit float64) string {
	return fmt.Sprintf(`
resource "clearml_resource_policy" "test" {
  name          = %q
  description   = %q
  reservation   = %g
  limit         = %g
  user_group_id = %q
}
`, name, description, reservation, limit, groupID)
}
