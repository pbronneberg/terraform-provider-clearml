package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestAccessRulePlanValidationRequiresSubject(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	resourceUnderTest := newAccessRuleResource().(*accessRuleResource)
	schemaResponse := &resource.SchemaResponse{}
	resourceUnderTest.Schema(ctx, resource.SchemaRequest{}, schemaResponse)
	rawType := schemaResponse.Schema.Type().TerraformType(ctx)
	raw := tftypes.NewValue(rawType, map[string]tftypes.Value{
		"id":                  tftypes.NewValue(tftypes.String, nil),
		"name":                tftypes.NewValue(tftypes.String, "rule"),
		"description":         tftypes.NewValue(tftypes.String, "description"),
		"entity_type":         tftypes.NewValue(tftypes.String, "project"),
		"entity_id":           tftypes.NewValue(tftypes.String, nil),
		"permission":          tftypes.NewValue(tftypes.String, "read"),
		"group_ids":           tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, nil),
		"service_account_ids": tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, nil),
	})
	request := resource.ValidateConfigRequest{Config: tfsdk.Config{Schema: schemaResponse.Schema, Raw: raw}}
	response := &resource.ValidateConfigResponse{}
	resourceUnderTest.ConfigValidators(ctx)[0].ValidateResource(ctx, request, response)
	if !response.Diagnostics.HasError() || !strings.Contains(response.Diagnostics.Errors()[0].Detail(), "group_ids") {
		t.Fatalf("diagnostics = %v, want missing-subject error", response.Diagnostics)
	}
}

func TestResourcePolicyPlanValidationRejectsReservationAboveLimit(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	resourceUnderTest := newResourcePolicyResource().(*resourcePolicyResource)
	schemaResponse := &resource.SchemaResponse{}
	resourceUnderTest.Schema(ctx, resource.SchemaRequest{}, schemaResponse)
	rawType := schemaResponse.Schema.Type().TerraformType(ctx)
	raw := tftypes.NewValue(rawType, map[string]tftypes.Value{
		"id":            tftypes.NewValue(tftypes.String, nil),
		"name":          tftypes.NewValue(tftypes.String, "policy"),
		"description":   tftypes.NewValue(tftypes.String, nil),
		"reservation":   tftypes.NewValue(tftypes.Number, 2.0),
		"limit":         tftypes.NewValue(tftypes.Number, 1.0),
		"user_group_id": tftypes.NewValue(tftypes.String, "group-1"),
	})
	response := &resource.ValidateConfigResponse{}
	resourceUnderTest.ValidateConfig(ctx, resource.ValidateConfigRequest{Config: tfsdk.Config{Schema: schemaResponse.Schema, Raw: raw}}, response)
	if !response.Diagnostics.HasError() || !strings.Contains(response.Diagnostics.Errors()[0].Detail(), "reservation cannot exceed limit") {
		t.Fatalf("diagnostics = %v, want invalid quota error", response.Diagnostics)
	}
}
