package provider

import (
	"context"
	"reflect"
	"sort"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/pbronneberg/terraform-provider-clearml/internal/client"
)

func TestQueueInputPreservesOmittedAndExplicitEmptyValues(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	omitted, diagnostics := queueInput(ctx, queueResourceModel{Name: types.StringValue("queue")})
	if diagnostics.HasError() {
		t.Fatalf("queueInput(omitted) diagnostics = %v", diagnostics)
	}
	if omitted.DisplayName != nil || omitted.Tags != nil || omitted.Metadata != nil {
		t.Fatalf("queueInput(omitted) = %#v, want nil optional values", omitted)
	}

	emptyTags, diagnostics := types.SetValueFrom(ctx, types.StringType, []string{})
	if diagnostics.HasError() {
		t.Fatalf("SetValueFrom() diagnostics = %v", diagnostics)
	}
	emptyMetadata, diagnostics := types.MapValueFrom(ctx, types.ObjectType{AttrTypes: metadataItemTypes}, map[string]metadataItemModel{})
	if diagnostics.HasError() {
		t.Fatalf("MapValueFrom() diagnostics = %v", diagnostics)
	}
	explicit, diagnostics := queueInput(ctx, queueResourceModel{
		Name: types.StringValue("queue"), DisplayName: types.StringValue(""), Tags: emptyTags, Metadata: emptyMetadata,
	})
	if diagnostics.HasError() {
		t.Fatalf("queueInput(explicit empty) diagnostics = %v", diagnostics)
	}
	if explicit.DisplayName == nil || *explicit.DisplayName != "" || explicit.Tags == nil || len(*explicit.Tags) != 0 || explicit.Metadata == nil || len(*explicit.Metadata) != 0 {
		t.Fatalf("queueInput(explicit empty) = %#v, want non-nil empty optional values", explicit)
	}
}

func TestSetAndMetadataRoundTripIsStable(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tagSet, diagnostics := stringSet(ctx, []string{"production", "gpu"})
	if diagnostics.HasError() {
		t.Fatalf("stringSet() diagnostics = %v", diagnostics)
	}
	tags, diagnostics := setStrings(ctx, tagSet)
	if diagnostics.HasError() {
		t.Fatalf("setStrings() diagnostics = %v", diagnostics)
	}
	sort.Strings(*tags)
	if !reflect.DeepEqual(*tags, []string{"gpu", "production"}) {
		t.Fatalf("set round trip = %v, want stable unique values", *tags)
	}

	want := map[string]client.MetadataItem{
		"owner": {Key: "owner", Type: "string", Value: "platform"},
		"cores": {Key: "cores", Type: "int", Value: "8"},
	}
	value, diagnostics := metadataMap(ctx, want)
	if diagnostics.HasError() {
		t.Fatalf("metadataMap() diagnostics = %v", diagnostics)
	}
	got, diagnostics := metadataItems(ctx, value)
	if diagnostics.HasError() {
		t.Fatalf("metadataItems() diagnostics = %v", diagnostics)
	}
	if !reflect.DeepEqual(*got, want) {
		t.Fatalf("metadata round trip = %#v, want %#v", *got, want)
	}
}

func TestOptionalSetPreservesNullAndExplicitEmpty(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	omitted, diagnostics := optionalStringSet(ctx, nil, types.SetNull(types.StringType))
	if diagnostics.HasError() || !omitted.IsNull() {
		t.Fatalf("optionalStringSet(omitted) = %v, %v; want null", omitted, diagnostics)
	}
	explicit, diagnostics := optionalStringSet(ctx, nil, types.SetValueMust(types.StringType, nil))
	if diagnostics.HasError() || explicit.IsNull() || len(explicit.Elements()) != 0 {
		t.Fatalf("optionalStringSet(explicit empty) = %v, %v; want empty set", explicit, diagnostics)
	}
}

func TestOptionalRemoteAttributesUseStateWhenOmitted(t *testing.T) {
	t.Parallel()
	response := &resource.SchemaResponse{}
	newQueueResource().Schema(context.Background(), resource.SchemaRequest{}, response)

	for _, name := range []string{"display_name", "tags", "metadata"} {
		attribute := response.Schema.Attributes[name]
		if !attribute.IsOptional() || !attribute.IsComputed() {
			t.Fatalf("%s must be Optional and Computed", name)
		}
		switch typed := attribute.(type) {
		case schema.StringAttribute:
			if len(typed.PlanModifiers) == 0 {
				t.Fatalf("%s has no state-preserving plan modifier", name)
			}
		case schema.SetAttribute:
			if len(typed.PlanModifiers) == 0 {
				t.Fatalf("%s has no state-preserving plan modifier", name)
			}
		case schema.MapAttribute:
			if len(typed.PlanModifiers) == 0 {
				t.Fatalf("%s has no state-preserving plan modifier", name)
			}
		default:
			t.Fatalf("unexpected schema type for %s: %T", name, attribute)
		}
	}
}

func TestProjectParent(t *testing.T) {
	t.Parallel()
	for name, want := range map[string]string{"root": "", "root/child": "root", "root/child/grandchild": "root/child"} {
		if got := projectParent(name); got != want {
			t.Errorf("projectParent(%q) = %q, want %q", name, got, want)
		}
	}
}
