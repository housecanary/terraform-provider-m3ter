// Copyright (c) HouseCanary, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &PlanGroupResource{}
var _ resource.ResourceWithImportState = &PlanGroupResource{}

func NewPlanGroupResource() resource.Resource {
	return &PlanGroupResource{}
}

// PlanGroupResource defines the resource implementation.
type PlanGroupResource struct {
	client *m3terClient
}

// PlanGroupResourceModel describes the resource data model.
type PlanGroupResourceModel struct {
	Name                              types.String  `tfsdk:"name"`
	Code                              types.String  `tfsdk:"code"`
	CustomFields                      types.Dynamic `tfsdk:"custom_fields"`
	MinimumSpend                      types.Float64 `tfsdk:"minimum_spend"`
	MinimumSpendDescription           types.String  `tfsdk:"minimum_spend_description"`
	StandingCharge                    types.Float64 `tfsdk:"standing_charge"`
	StandingChargeDescription         types.String  `tfsdk:"standing_charge_description"`
	Currency                          types.String  `tfsdk:"currency"`
	StandingChargeBillInAdvance       types.Bool    `tfsdk:"standing_charge_bill_in_advance"`
	MinimumSpendBillInAdvance         types.Bool    `tfsdk:"minimum_spend_bill_in_advance"`
	MinimumSpendAccountingProductId   types.String  `tfsdk:"minimum_spend_accounting_product_id"`
	StandingChargeAccountingProductId types.String  `tfsdk:"standing_charge_accounting_product_id"`
	Id                                types.String  `tfsdk:"id"`
	Version                           types.Int64   `tfsdk:"version"`
}

func (r *PlanGroupResourceModel) GetId() types.String {
	return r.Id
}

func (r *PlanGroupResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_plan_group"
}

func (r *PlanGroupResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "PlanGroup resource",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the plan group",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 200),
				},
			},
			"code": schema.StringAttribute{
				MarkdownDescription: "The short code for the PlanGroup.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 80),
				},
			},
			"custom_fields": schema.DynamicAttribute{
				MarkdownDescription: "Custom fields for the plan group",
				Optional:            true,
			},

			"currency": schema.StringAttribute{
				MarkdownDescription: "The currency of the plan group",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(3, 3),
				},
			},
			"standing_charge": schema.Float64Attribute{
				MarkdownDescription: "The standing charge of the plan group",
				Required:            true,
				Validators: []validator.Float64{
					float64validator.AtLeast(0),
				},
			},
			"standing_charge_description": schema.StringAttribute{
				MarkdownDescription: "The standing charge description of the plan group",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(200),
				},
			},
			"minimum_spend": schema.Float64Attribute{
				MarkdownDescription: "The minimum spend of the plan group",
				Optional:            true,
				Validators: []validator.Float64{
					float64validator.AtLeast(0),
				},
			},
			"minimum_spend_description": schema.StringAttribute{
				MarkdownDescription: "The minimum spend description of the plan group",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(200),
				},
			},
			"standing_charge_bill_in_advance": schema.BoolAttribute{
				MarkdownDescription: "Boolean flag that sets the Standing Charge as a bill in advance.",
				Optional:            true,
			},
			"minimum_spend_bill_in_advance": schema.BoolAttribute{
				MarkdownDescription: "Boolean flag that sets the Minimum Spend as a bill in advance.",
				Optional:            true,
			},
			"minimum_spend_accounting_product_id": schema.StringAttribute{
				MarkdownDescription: "The accounting product ID for the minimum spend",
				Optional:            true,
			},
			"standing_charge_accounting_product_id": schema.StringAttribute{
				MarkdownDescription: "The accounting product ID for the standing charge",
				Optional:            true,
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "PlanGroup identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"version": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "PlanGroup version",
			},
		},
	}
}

func (r *PlanGroupResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*m3terClient)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *m3terClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *PlanGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	genericCreate[PlanGroupResourceModel](ctx, req, resp, r.client, "/plangroups", "plan group", r.read, r.write)
}

func (r *PlanGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	genericRead[PlanGroupResourceModel](ctx, req, resp, r.client, "/plangroups", "plan group", r.read)
}

func (r *PlanGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	genericUpdate[PlanGroupResourceModel](ctx, req, resp, r.client, "/plangroups", "plan group", r.read, r.write)
}

func (r *PlanGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	genericDelete[PlanGroupResourceModel](ctx, req, resp, r.client, "/plangroups", "plan group")
}

func (r *PlanGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *PlanGroupResource) read(ctx context.Context, data *PlanGroupResourceModel, restData map[string]any, diagnostics *diag.Diagnostics) {
	m := &mapper{
		ctx:         ctx,
		diagnostics: diagnostics,
		v:           restData,
	}
	m.to("id", &data.Id)
	m.to("version", &data.Version)
	m.to("name", &data.Name)
	m.to("code", &data.Code)
	m.to("currency", &data.Currency)
	m.to("standingCharge", &data.StandingCharge)
	m.to("standingChargeDescription", &data.StandingChargeDescription)
	m.to("minimumSpend", &data.MinimumSpend)
	m.to("minimumSpendDescription", &data.MinimumSpendDescription)
	m.to("standingChargeBillInAdvance", &data.StandingChargeBillInAdvance)
	m.to("minimumSpendBillInAdvance", &data.MinimumSpendBillInAdvance)
	m.to("minimumSpendAccountingProductId", &data.MinimumSpendAccountingProductId)
	m.to("standingChargeAccountingProductId", &data.StandingChargeAccountingProductId)
	m.customFieldsTo(&data.CustomFields)
}

func (r *PlanGroupResource) write(ctx context.Context, data *PlanGroupResourceModel, restData map[string]any, diagnostics *diag.Diagnostics) {
	m := &mapper{
		ctx:         ctx,
		diagnostics: diagnostics,
		v:           restData,
	}
	m.from(data.Id, "id")
	m.from(data.Version, "version")
	m.from(data.Name, "name")
	m.from(data.Code, "code")
	m.from(data.Currency, "currency")
	m.from(data.StandingCharge, "standingCharge")
	m.from(data.StandingChargeDescription, "standingChargeDescription")
	m.from(data.MinimumSpend, "minimumSpend")
	m.from(data.MinimumSpendDescription, "minimumSpendDescription")
	m.from(data.StandingChargeBillInAdvance, "standingChargeBillInAdvance")
	m.from(data.MinimumSpendBillInAdvance, "minimumSpendBillInAdvance")
	m.from(data.MinimumSpendAccountingProductId, "minimumSpendAccountingProductId")
	m.from(data.StandingChargeAccountingProductId, "standingChargeAccountingProductId")
	m.customFieldsFrom(data.CustomFields)
}
