package provider

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/pbronneberg/terraform-provider-clearml/internal/client"
)

func TestAccQueueLifecycle(t *testing.T) {
	requireCoreAcceptance(t)

	initialName := acceptanceQueueName(t)
	updatedName := acceptanceQueueName(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acceptanceProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccQueueConfig(initialName, "Initial queue", []string{"acceptance", "initial"}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("clearml_queue.test", "id"),
					resource.TestCheckResourceAttr("clearml_queue.test", "name", initialName),
					resource.TestCheckResourceAttr("clearml_queue.test", "tags.#", "2"),
					resource.TestCheckResourceAttr("clearml_queue.test", "metadata.owner.type", "string"),
					resource.TestCheckResourceAttr("clearml_queue.test", "metadata.owner.value", "platform"),
				),
			},
			{
				Config:             testAccQueueConfig(updatedName, "", []string{}),
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("clearml_queue.test", "id"),
					resource.TestCheckResourceAttr("clearml_queue.test", "name", updatedName),
					resource.TestCheckResourceAttr("clearml_queue.test", "display_name", ""),
					resource.TestCheckResourceAttr("clearml_queue.test", "tags.#", "0"),
					testAccRemoveRemote("clearml_queue.test", "", func(ctx context.Context, apiClient *client.ClearMLClient, id string) error {
						return apiClient.DeleteQueue(ctx, id)
					}),
				),
			},
			{
				Config: testAccQueueConfig(updatedName, "", []string{}),
				Check:  resource.TestCheckResourceAttrSet("clearml_queue.test", "id"),
			},
			{
				ResourceName:      "clearml_queue.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func acceptanceQueueName(t *testing.T) string {
	t.Helper()
	var randomBytes [16]byte
	if _, err := rand.Read(randomBytes[:]); err != nil {
		t.Fatalf("generate acceptance queue identifier: %v", err)
	}
	version := strings.NewReplacer(".", "-", " ", "-").Replace(os.Getenv("TF_ACC_TERRAFORM_VERSION"))
	if version == "" {
		version = "0-0-0"
	}
	runID := os.Getenv("GITHUB_RUN_ID")
	if runID == "" {
		runID = "0"
	}
	attempt := os.Getenv("GITHUB_RUN_ATTEMPT")
	if attempt == "" {
		attempt = "0"
	}
	return fmt.Sprintf("tfacc-clearml-%d-%s-%s-%s-%s", time.Now().UTC().Unix(), version, runID, attempt, hex.EncodeToString(randomBytes[:]))
}

func testAccQueueConfig(name, displayName string, tags []string) string {
	tagValues := make([]string, len(tags))
	for index, tag := range tags {
		tagValues[index] = fmt.Sprintf("%q", tag)
	}
	return fmt.Sprintf(`
resource "clearml_queue" "test" {
  name         = %q
  display_name = %q
  tags         = [%s]

  metadata = {
    owner = {
      type  = "string"
      value = "platform"
    }
  }
}
`, name, displayName, strings.Join(tagValues, ", "))
}
