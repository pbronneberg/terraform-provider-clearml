package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/pbronneberg/terraform-provider-clearml/internal/client"
)

var metadataItemTypes = map[string]attr.Type{
	"type":  types.StringType,
	"value": types.StringType,
}

type metadataItemModel struct {
	Type  types.String `tfsdk:"type"`
	Value types.String `tfsdk:"value"`
}

func configuredClient(providerData any, diagnostics *diag.Diagnostics, target string) *client.ClearMLClient {
	if providerData == nil {
		return nil
	}
	configured, ok := providerData.(*client.ClearMLClient)
	if !ok {
		diagnostics.AddError("Unexpected "+target+" configure type", fmt.Sprintf("Expected *client.ClearMLClient, got %T.", providerData))
		return nil
	}
	return configured
}

func optionalString(value types.String) *string {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	result := value.ValueString()
	return &result
}

func stringValue(value types.String) string {
	if result := optionalString(value); result != nil {
		return *result
	}
	return ""
}

func setStrings(ctx context.Context, value types.Set) (*[]string, diag.Diagnostics) {
	if value.IsNull() || value.IsUnknown() {
		return nil, nil
	}
	var result []string
	diagnostics := value.ElementsAs(ctx, &result, false)
	return &result, diagnostics
}

func stringSet(ctx context.Context, values []string) (types.Set, diag.Diagnostics) {
	if values == nil {
		values = []string{}
	}
	return types.SetValueFrom(ctx, types.StringType, values)
}

func optionalStringSet(ctx context.Context, values []string, current types.Set) (types.Set, diag.Diagnostics) {
	if len(values) == 0 && current.IsNull() {
		return types.SetNull(types.StringType), nil
	}
	return stringSet(ctx, values)
}

func metadataItems(ctx context.Context, value types.Map) (*map[string]client.MetadataItem, diag.Diagnostics) {
	if value.IsNull() || value.IsUnknown() {
		return nil, nil
	}
	var models map[string]metadataItemModel
	diagnostics := value.ElementsAs(ctx, &models, false)
	if diagnostics.HasError() {
		return nil, diagnostics
	}
	items := make(map[string]client.MetadataItem, len(models))
	for key, model := range models {
		items[key] = client.MetadataItem{Key: key, Type: model.Type.ValueString(), Value: model.Value.ValueString()}
	}
	return &items, diagnostics
}

func metadataMap(ctx context.Context, items map[string]client.MetadataItem) (types.Map, diag.Diagnostics) {
	models := make(map[string]metadataItemModel, len(items))
	for key, item := range items {
		models[key] = metadataItemModel{Type: types.StringValue(item.Type), Value: types.StringValue(item.Value)}
	}
	return types.MapValueFrom(ctx, types.ObjectType{AttrTypes: metadataItemTypes}, models)
}
