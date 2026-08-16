package provider

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/pbronneberg/terraform-provider-clearml/internal/client"
)

func TestAccAccessRuleLifecycle(t *testing.T) {
	requireEnterpriseAcceptance(t, "CLEARML_ENTERPRISE_TEST_GROUP_ID")
	name := acceptanceQueueName(t)
	groupID := os.Getenv("CLEARML_ENTERPRISE_TEST_GROUP_ID")
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acceptanceProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccAccessRuleConfig(name, "read", groupID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("clearml_access_rule.test", "id"),
					resource.TestCheckResourceAttr("clearml_access_rule.test", "permission", "read"),
					resource.TestCheckResourceAttr("clearml_access_rule.test", "group_ids.#", "1"),
				),
			},
			{
				Config:             testAccAccessRuleConfig(name, "read_write", groupID),
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("clearml_access_rule.test", "permission", "read_write"),
					testAccRemoveRemote("clearml_access_rule.test", "", func(ctx context.Context, apiClient *client.ClearMLClient, id string) error {
						return apiClient.DeleteAccessRule(ctx, id)
					}),
				),
			},
			{
				Config: testAccAccessRuleConfig(name, "read_write", groupID),
				Check:  resource.TestCheckResourceAttrSet("clearml_access_rule.test", "id"),
			},
			{ResourceName: "clearml_access_rule.test", ImportState: true, ImportStateVerify: true},
		},
	})
}

func testAccAccessRuleConfig(name, permission, groupID string) string {
	return fmt.Sprintf(`
resource "clearml_access_rule" "test" {
  name        = %q
  description = "Terraform acceptance test"
  entity_type = "project"
  permission  = %q
  group_ids   = [%q]
}
`, name, permission, groupID)
}
