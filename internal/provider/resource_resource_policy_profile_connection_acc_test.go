package provider

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/pbronneberg/terraform-provider-clearml/internal/client"
)

func TestAccResourcePolicyProfileConnectionLifecycle(t *testing.T) {
	requireEnterpriseAcceptance(t, "CLEARML_ENTERPRISE_TEST_GROUP_ID", "CLEARML_ENTERPRISE_TEST_PROFILE_ID")
	name := acceptanceQueueName(t)
	config := testAccResourcePolicyProfileConnectionConfig(
		name, os.Getenv("CLEARML_ENTERPRISE_TEST_GROUP_ID"), os.Getenv("CLEARML_ENTERPRISE_TEST_PROFILE_ID"),
	)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acceptanceProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:             config,
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("clearml_resource_policy_profile_connection.test", "queue_id"),
					resource.TestCheckResourceAttr("clearml_resource_policy_profile_connection.test", "queue_name", name),
					testAccRemoveRemote("clearml_resource_policy_profile_connection.test", "queue_id", func(ctx context.Context, apiClient *client.ClearMLClient, id string) error {
						return apiClient.DisconnectResourcePolicyProfile(ctx, id)
					}),
				),
			},
			{
				Config: config,
				Check:  resource.TestCheckResourceAttrSet("clearml_resource_policy_profile_connection.test", "queue_id"),
			},
			{
				ResourceName:        "clearml_resource_policy_profile_connection.test",
				ImportState:         true,
				ImportStateIdPrefix: "",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					connection := state.RootModule().Resources["clearml_resource_policy_profile_connection.test"]
					return connection.Primary.Attributes["policy_id"] + "/" + connection.Primary.Attributes["queue_id"], nil
				},
				ImportStateVerify: true,
			},
		},
	})
}

func testAccResourcePolicyProfileConnectionConfig(name, groupID, profileID string) string {
	return fmt.Sprintf(`
resource "clearml_resource_policy" "test" {
  name          = %q
  reservation   = 1
  limit         = 2
  user_group_id = %q
}

resource "clearml_resource_policy_profile_connection" "test" {
  policy_id   = clearml_resource_policy.test.id
  profile_id  = %q
  queue_name  = %q
  display_name = "Terraform acceptance queue"
}
`, name+"-policy", groupID, profileID, name)
}
