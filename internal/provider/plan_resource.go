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
var _ resource.Resource = &PlanResource{}
var _ resource.ResourceWithImportState = &PlanResource{}

func NewPlanResource() resource.Resource {
	return &PlanResource{}
}

// PlanResource defines the resource implementation.
type PlanResource struct {
	client *m3terClient
}

// PlanResourceModel describes the resource data model.
type PlanResourceModel struct {
	Name                        types.String  `tfsdk:"name"`
	Code                        types.String  `tfsdk:"code"`
	CustomFields                types.Dynamic `tfsdk:"custom_fields"`
	PlanTemplateId              types.String  `tfsdk:"plan_template_id"`
	StandingCharge              types.Float64 `tfsdk:"standing_charge"`
	StandingChargeDescription   types.String  `tfsdk:"standing_charge_description"`
	MinimumSpend                types.Float64 `tfsdk:"minimum_spend"`
	MinimumSpendDescription     types.String  `tfsdk:"minimum_spend_description"`
	StandingChargeBillInAdvance types.Bool    `tfsdk:"standing_charge_bill_in_advance"`
	MinimumSpendBillInAdvance   types.Bool    `tfsdk:"minimum_spend_bill_in_advance"`
	Id                          types.String  `tfsdk:"id"`
	Version                     types.Int64   `tfsdk:"version"`
}

func (r *PlanResourceModel) GetId() types.String {
	return r.Id
}

func (r *PlanResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_plan"
}

func (r *PlanResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Plan resource",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the plan",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 200),
				},
			},
			"code": schema.StringAttribute{
				MarkdownDescription: "The short code for the Plan.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 80),
				},
			},
			"custom_fields": schema.DynamicAttribute{
				MarkdownDescription: "The name of the Event that triggers the Plan.",
				Optional:            true,
			},
			"plan_template_id": schema.StringAttribute{
				MarkdownDescription: "The plan ID that this plan belongs to",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"standing_charge": schema.Float64Attribute{
				MarkdownDescription: "The standing charge of the plan",
				Optional:            true,
				Validators: []validator.Float64{
					float64validator.AtLeast(0),
				},
			},
			"standing_charge_description": schema.StringAttribute{
				MarkdownDescription: "The standing charge description of the plan",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(200),
				},
			},
			"minimum_spend": schema.Float64Attribute{
				MarkdownDescription: "The minimum spend of the plan",
				Optional:            true,
				Validators: []validator.Float64{
					float64validator.AtLeast(0),
				},
			},
			"minimum_spend_description": schema.StringAttribute{
				MarkdownDescription: "The minimum spend description of the plan",
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
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Plan identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"version": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Plan version",
			},
		},
	}
}

func (r *PlanResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *PlanResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	genericCreate[PlanResourceModel](ctx, req, resp, r.client, "/plans", "plan", r.read, r.write)
}

func (r *PlanResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	genericRead[PlanResourceModel](ctx, req, resp, r.client, "/plans", "plan", r.read)
}

func (r *PlanResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	genericUpdate[PlanResourceModel](ctx, req, resp, r.client, "/plans", "plan", r.read, r.write)
}

func (r *PlanResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	genericDelete[PlanResourceModel](ctx, req, resp, r.client, "/plans", "plan")
}

func (r *PlanResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *PlanResource) read(ctx context.Context, data *PlanResourceModel, restData map[string]any, diagnostics *diag.Diagnostics) {
	m := &mapper{
		ctx:         ctx,
		diagnostics: diagnostics,
		v:           restData,
	}
	m.to("id", &data.Id)
	m.to("version", &data.Version)
	m.to("name", &data.Name)
	m.to("code", &data.Code)
	m.to("planTemplateId", &data.PlanTemplateId)
	m.to("standingCharge", &data.StandingCharge)
	m.to("standingChargeDescription", &data.StandingChargeDescription)
	m.to("minimumSpend", &data.MinimumSpend)
	m.to("minimumSpendDescription", &data.MinimumSpendDescription)
	m.to("standingChargeBillInAdvance", &data.StandingChargeBillInAdvance)
	m.to("minimumSpendBillInAdvance", &data.MinimumSpendBillInAdvance)
	m.customFieldsTo(&data.CustomFields)
}

func (r *PlanResource) write(ctx context.Context, data *PlanResourceModel, restData map[string]any, diagnostics *diag.Diagnostics) {
	m := &mapper{
		ctx:         ctx,
		diagnostics: diagnostics,
		v:           restData,
	}
	m.from(data.Id, "id")
	m.from(data.Version, "version")
	m.from(data.Name, "name")
	m.from(data.Code, "code")
	m.from(data.PlanTemplateId, "planTemplateId")
	m.from(data.StandingCharge, "standingCharge")
	m.from(data.StandingChargeDescription, "standingChargeDescription")
	m.from(data.MinimumSpend, "minimumSpend")
	m.from(data.MinimumSpendDescription, "minimumSpendDescription")
	m.from(data.StandingChargeBillInAdvance, "standingChargeBillInAdvance")
	m.from(data.MinimumSpendBillInAdvance, "minimumSpendBillInAdvance")
	m.customFieldsFrom(data.CustomFields)
}
