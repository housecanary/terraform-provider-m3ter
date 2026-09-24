// Copyright (c) HouseCanary, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestPlanResourceReadPreservesNullDescriptions(t *testing.T) {
	t.Parallel()

	data := PlanResourceModel{
		StandingChargeDescription: types.StringNull(),
		MinimumSpendDescription:   types.StringNull(),
	}
	diagnostics := diag.Diagnostics{}

	(&PlanResource{}).read(context.Background(), &data, map[string]any{
		"standingChargeDescription": "",
		"minimumSpendDescription":   "",
	}, &diagnostics)

	if diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diagnostics)
	}

	if !data.StandingChargeDescription.IsNull() {
		t.Errorf("standing_charge_description = %q, want null", data.StandingChargeDescription.ValueString())
	}

	if !data.MinimumSpendDescription.IsNull() {
		t.Errorf("minimum_spend_description = %q, want null", data.MinimumSpendDescription.ValueString())
	}
}

func TestPlanResourceReadPreservesEmptyDescriptions(t *testing.T) {
	t.Parallel()

	data := PlanResourceModel{
		StandingChargeDescription: types.StringValue(""),
		MinimumSpendDescription:   types.StringValue(""),
	}
	diagnostics := diag.Diagnostics{}

	(&PlanResource{}).read(context.Background(), &data, map[string]any{
		"standingChargeDescription": "",
		"minimumSpendDescription":   "",
	}, &diagnostics)

	if diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diagnostics)
	}

	if data.StandingChargeDescription.IsNull() || data.StandingChargeDescription.ValueString() != "" {
		t.Errorf("standing_charge_description = %q, want empty string", data.StandingChargeDescription.ValueString())
	}

	if data.MinimumSpendDescription.IsNull() || data.MinimumSpendDescription.ValueString() != "" {
		t.Errorf("minimum_spend_description = %q, want empty string", data.MinimumSpendDescription.ValueString())
	}
}
