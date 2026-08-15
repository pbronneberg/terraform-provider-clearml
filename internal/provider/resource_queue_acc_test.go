package provider

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccQueueLifecycle(t *testing.T) {
	if os.Getenv("TF_ACC") != "1" {
		t.Skip("set TF_ACC=1 to run acceptance tests")
	}
	if os.Getenv("CLEARML_ACCESS_KEY") == "" || os.Getenv("CLEARML_SECRET_KEY") == "" {
		t.Fatal("CLEARML_ACCESS_KEY and CLEARML_SECRET_KEY must be set for acceptance tests")
	}

	initialName := acceptanceQueueName(t)
	updatedName := acceptanceQueueName(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"clearml": providerserver.NewProtocol6WithError(New("acceptance")()),
		},
		Steps: []resource.TestStep{
			{
				Config: testAccQueueConfig(initialName, []string{"acceptance", "initial"}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("clearml_queue.test", "id"),
					resource.TestCheckResourceAttr("clearml_queue.test", "name", initialName),
					resource.TestCheckResourceAttr("clearml_queue.test", "tags.#", "2"),
				),
			},
			{
				Config: testAccQueueConfig(updatedName, []string{"acceptance", "updated"}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("clearml_queue.test", "id"),
					resource.TestCheckResourceAttr("clearml_queue.test", "name", updatedName),
					resource.TestCheckResourceAttr("clearml_queue.test", "tags.#", "2"),
				),
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

func testAccQueueConfig(name string, tags []string) string {
	return fmt.Sprintf(`
resource "clearml_queue" "test" {
  name = %q
  tags = [%q, %q]
}
`, name, tags[0], tags[1])
}
