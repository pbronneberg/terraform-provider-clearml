package provider

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestListStrings(t *testing.T) {
	t.Parallel()
	list, diagnostics := types.ListValueFrom(context.Background(), types.StringType, []string{"one", "two"})
	if diagnostics.HasError() {
		t.Fatalf("ListValueFrom() diagnostics = %v", diagnostics)
	}
	got, diagnostics := listStrings(context.Background(), list)
	if diagnostics.HasError() {
		t.Fatalf("listStrings() diagnostics = %v", diagnostics)
	}
	if !reflect.DeepEqual(got, []string{"one", "two"}) {
		t.Fatalf("listStrings() = %v, want [one two]", got)
	}
}
