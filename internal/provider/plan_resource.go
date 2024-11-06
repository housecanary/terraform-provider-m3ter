// Copyright (c) HouseCanary, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
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
	Bespoke                     types.Bool    `tfsdk:"bespoke"`
	MinimumSpend                types.Float64 `tfsdk:"minimum_spend"`
	MinimumSpendDescription     types.String  `tfsdk:"minimum_spend_description"`
	StandingChargeBillInAdvance types.Bool    `tfsdk:"standing_charge_bill_in_advance"`
	MinimumSpendBillInAdvance   types.Bool    `tfsdk:"minimum_spend_bill_in_advance"`
	AccountId                   types.String  `tfsdk:"account_id"`
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
				MarkdownDescription: "Descriptive name for the Plan.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 200),
				},
			},
			"code": schema.StringAttribute{
				MarkdownDescription: "Unique short code reference for the Plan.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 80),
					stringvalidator.RegexMatches(regexp.MustCompile(`^([^\p{Cc}\s])|([^\p{Cc}\s][[^\p{Cc}\s] ]*[^\p{Cc}\s])$`), "The code must not contain control characters or start/end with whitespace."),
				},
			},
			"custom_fields": schema.DynamicAttribute{
				MarkdownDescription: "User defined fields enabling you to attach custom data. The value for a custom field can be either a string or a number.",
				Required:            true,
			},
			"plan_template_id": schema.StringAttribute{
				MarkdownDescription: "UUID of the PlanTemplate the Plan belongs to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"standing_charge": schema.Float64Attribute{
				MarkdownDescription: "The standing charge applied to bills for end customers. This is prorated.",
				Optional:            true,
				Validators: []validator.Float64{
					float64validator.AtLeast(0),
				},
			},
			"standing_charge_description": schema.StringAttribute{
				MarkdownDescription: "Standing charge description (displayed on the bill line item).",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(200),
				},
			},
			"bespoke": schema.BoolAttribute{
				MarkdownDescription: "TRUE/FALSE flag indicating whether the plan is a custom/bespoke Plan for a particular Account.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"minimum_spend": schema.Float64Attribute{
				MarkdownDescription: "The product minimum spend amount per billing cycle for end customer Accounts on a priced Plan.",
				Optional:            true,
				Validators: []validator.Float64{
					float64validator.AtLeast(0),
				},
			},
			"minimum_spend_description": schema.StringAttribute{
				MarkdownDescription: "Minimum spend description (displayed on the bill line item).",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(200),
				},
			},
			"standing_charge_bill_in_advance": schema.BoolAttribute{
				MarkdownDescription: "When TRUE, standing charge is billed at the start of each billing period.\n\nWhen FALSE, standing charge is billed at the end of each billing period.",
				Optional:            true,
			},
			"minimum_spend_bill_in_advance": schema.BoolAttribute{
				MarkdownDescription: "When TRUE, minimum spend is billed at the start of each billing period.\n\nWhen FALSE, minimum spend is billed at the end of each billing period.",
				Optional:            true,
			},
			"account_id": schema.StringAttribute{
				MarkdownDescription: "Used to specify an Account for which the Plan will be a custom/bespoke Plan.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The UUID of the entity.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"version": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "The version number.",
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
	m.to("bespoke", &data.Bespoke)
	m.to("minimumSpend", &data.MinimumSpend)
	m.to("minimumSpendDescription", &data.MinimumSpendDescription)
	m.to("standingChargeBillInAdvance", &data.StandingChargeBillInAdvance)
	m.to("minimumSpendBillInAdvance", &data.MinimumSpendBillInAdvance)
	m.to("accountId", &data.AccountId)
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
	m.from(data.Bespoke, "bespoke")
	m.from(data.MinimumSpend, "minimumSpend")
	m.from(data.MinimumSpendDescription, "minimumSpendDescription")
	m.from(data.StandingChargeBillInAdvance, "standingChargeBillInAdvance")
	m.from(data.MinimumSpendBillInAdvance, "minimumSpendBillInAdvance")
	m.from(data.AccountId, "accountId")
	m.customFieldsFrom(data.CustomFields)
}
