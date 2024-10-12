// Copyright (c) HouseCanary, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/int32validator"
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
var _ resource.Resource = &PlanTemplateResource{}
var _ resource.ResourceWithImportState = &PlanTemplateResource{}

func NewPlanTemplateResource() resource.Resource {
	return &PlanTemplateResource{}
}

// PlanTemplateResource defines the resource implementation.
type PlanTemplateResource struct {
	client *m3terClient
}

// PlanTemplateResourceModel describes the resource data model.
type PlanTemplateResourceModel struct {
	Name                        types.String  `tfsdk:"name"`
	Code                        types.String  `tfsdk:"code"`
	CustomFields                types.Dynamic `tfsdk:"custom_fields"`
	ProductId                   types.String  `tfsdk:"product_id"`
	Currency                    types.String  `tfsdk:"currency"`
	StandingCharge              types.Float64 `tfsdk:"standing_charge"`
	StandingChargeDescription   types.String  `tfsdk:"standing_charge_description"`
	StandingChargeInterval      types.Int32   `tfsdk:"standing_charge_interval"`
	StandingChargeOffset        types.Int32   `tfsdk:"standing_charge_offset"`
	BillFrequencyInterval       types.Int32   `tfsdk:"bill_frequency_interval"`
	BillFrequency               types.String  `tfsdk:"bill_frequency"`
	MinimumSpend                types.Float64 `tfsdk:"minimum_spend"`
	MinimumSpendDescription     types.String  `tfsdk:"minimum_spend_description"`
	StandingChargeBillInAdvance types.Bool    `tfsdk:"standing_charge_bill_in_advance"`
	MinimumSpendBillInAdvance   types.Bool    `tfsdk:"minimum_spend_bill_in_advance"`
	Id                          types.String  `tfsdk:"id"`
	Version                     types.Int64   `tfsdk:"version"`
}

func (r *PlanTemplateResourceModel) GetId() types.String {
	return r.Id
}

func (r *PlanTemplateResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_plan_template"
}

func (r *PlanTemplateResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "PlanTemplate resource",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Descriptive name for the PlanTemplate.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 200),
				},
			},
			"code": schema.StringAttribute{
				MarkdownDescription: "A unique, short code reference for the PlanTemplate. This code should not contain control characters or spaces.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 80),
					stringvalidator.RegexMatches(regexp.MustCompile(`^([^\p{Cc}\s])|([^\p{Cc}\s][[^\p{Cc}\s] ]*[^\p{Cc}\s])$`), "The code must not contain control characters or start/end with whitespace."),
				},
			},
			"custom_fields": schema.DynamicAttribute{
				MarkdownDescription: "User defined fields enabling you to attach custom data. The value for a custom field can be either a string or a number.",
				Required:            true,
			},
			"product_id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier (UUID) of the Product associated with this PlanTemplate.",
				Required:            true,
			},
			"currency": schema.StringAttribute{
				MarkdownDescription: "The ISO currency code for the currency used to charge end users - for example USD, GBP, EUR. This defines the pricing currency and is inherited by any Plans based on the Plan Template.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(3, 3),
				},
			},
			"standing_charge": schema.Float64Attribute{
				MarkdownDescription: "The fixed charge (standing charge) applied to customer bills. This charge is prorated and must be a non-negative number.",
				Required:            true,
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
			"standing_charge_interval": schema.Int32Attribute{
				MarkdownDescription: "How often the standing charge is applied. For example, if the bill is issued every three months and standingChargeInterval is 2, then the standing charge is applied every six months.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.Int32{
					int32validator.Between(1, 365),
				},
			},
			"standing_charge_offset": schema.Int32Attribute{
				MarkdownDescription: "Defines an offset for when the standing charge is first applied. For example, if the bill is issued every three months and the standingChargeOfset is 0, then the charge is applied to the first bill (at three months); if 1, it would be applied to the second bill (at six months), and so on.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.Int32{
					int32validator.Between(0, 364),
				},
			},
			"bill_frequency_interval": schema.Int32Attribute{
				MarkdownDescription: "How often bills are issued. For example, if billFrequency is Monthly and billFrequencyInterval is 3, bills are issued every three months.",
				Optional:            true,
				Validators: []validator.Int32{
					int32validator.Between(1, 365),
				},
			},
			"bill_frequency": schema.StringAttribute{
				MarkdownDescription: "Defines how often Bills are generated.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("DAILY", "WEEKLY", "MONTHLY", "ANNUALLY", "AD_HOC", "MIXED"),
				},
			},
			"minimum_spend": schema.Float64Attribute{
				MarkdownDescription: "The Product minimum spend amount per billing cycle for end customer Accounts on a pricing Plan based on the PlanTemplate. This must be a non-negative number.",
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
				MarkdownDescription: "A boolean that determines when the standing charge is billed.",
				Optional:            true,
			},
			"minimum_spend_bill_in_advance": schema.BoolAttribute{
				MarkdownDescription: "A boolean that determines when the minimum spend is billed.",
				Optional:            true,
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

func (r *PlanTemplateResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *PlanTemplateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	genericCreate[PlanTemplateResourceModel](ctx, req, resp, r.client, "/plantemplates", "plan template", r.read, r.write)
}

func (r *PlanTemplateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	genericRead[PlanTemplateResourceModel](ctx, req, resp, r.client, "/plantemplates", "plan template", r.read)
}

func (r *PlanTemplateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	genericUpdate[PlanTemplateResourceModel](ctx, req, resp, r.client, "/plantemplates", "plan template", r.read, r.write)
}

func (r *PlanTemplateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	genericDelete[PlanTemplateResourceModel](ctx, req, resp, r.client, "/plantemplates", "plan template")
}

func (r *PlanTemplateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *PlanTemplateResource) read(ctx context.Context, data *PlanTemplateResourceModel, restData map[string]any, diagnostics *diag.Diagnostics) {
	m := &mapper{
		ctx:         ctx,
		diagnostics: diagnostics,
		v:           restData,
	}
	m.to("id", &data.Id)
	m.to("version", &data.Version)
	m.to("name", &data.Name)
	m.to("code", &data.Code)
	m.to("productId", &data.ProductId)
	m.to("currency", &data.Currency)
	m.to("standingCharge", &data.StandingCharge)
	m.to("standingChargeDescription", &data.StandingChargeDescription)
	m.to("standingChargeInterval", &data.StandingChargeInterval)
	m.to("standingChargeOffset", &data.StandingChargeOffset)
	m.to("billFrequencyInterval", &data.BillFrequencyInterval)
	m.to("billFrequency", &data.BillFrequency)
	m.to("minimumSpend", &data.MinimumSpend)
	m.to("minimumSpendDescription", &data.MinimumSpendDescription)
	m.to("standingChargeBillInAdvance", &data.StandingChargeBillInAdvance)
	m.to("minimumSpendBillInAdvance", &data.MinimumSpendBillInAdvance)
	m.customFieldsTo(&data.CustomFields)
}

func (r *PlanTemplateResource) write(ctx context.Context, data *PlanTemplateResourceModel, restData map[string]any, diagnostics *diag.Diagnostics) {
	m := &mapper{
		ctx:         ctx,
		diagnostics: diagnostics,
		v:           restData,
	}
	m.from(data.Id, "id")
	m.from(data.Version, "version")
	m.from(data.Name, "name")
	m.from(data.Code, "code")
	m.from(data.ProductId, "productId")
	m.from(data.Currency, "currency")
	m.from(data.StandingCharge, "standingCharge")
	m.from(data.StandingChargeDescription, "standingChargeDescription")
	m.from(data.StandingChargeInterval, "standingChargeInterval")
	m.from(data.StandingChargeOffset, "standingChargeOffset")
	m.from(data.BillFrequencyInterval, "billFrequencyInterval")
	m.from(data.BillFrequency, "billFrequency")
	m.from(data.MinimumSpend, "minimumSpend")
	m.from(data.MinimumSpendDescription, "minimumSpendDescription")
	m.from(data.StandingChargeBillInAdvance, "standingChargeBillInAdvance")
	m.from(data.MinimumSpendBillInAdvance, "minimumSpendBillInAdvance")
	m.customFieldsFrom(data.CustomFields)
}
