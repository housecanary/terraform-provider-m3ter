// Copyright (c) HouseCanary, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &PricingResource{}
var _ resource.ResourceWithImportState = &PricingResource{}

func NewPricingResource() resource.Resource {
	return &PricingResource{}
}

// PricingResource defines the resource implementation.
type PricingResource struct {
	client *m3terClient
}

// PricingResourceModel describes the resource data model.
type PricingResourceModel struct {
	Description               types.String  `tfsdk:"description"`
	Code                      types.String  `tfsdk:"code"`
	AggregationId             types.String  `tfsdk:"aggregation_id"`
	CompoundAggregationId     types.String  `tfsdk:"compound_aggregation_id"`
	Type                      types.String  `tfsdk:"type"`
	Segment                   types.Map     `tfsdk:"segment"`
	TiersSpanPlan             types.Bool    `tfsdk:"tiers_span_plan"`
	MinimumSpend              types.Float64 `tfsdk:"minimum_spend"`
	MinimumSpendDescription   types.String  `tfsdk:"minimum_spend_description"`
	MinimumSpendBillInAdvance types.Bool    `tfsdk:"minimum_spend_bill_in_advance"`
	OveragePricingBands       types.List    `tfsdk:"overage_pricing_bands"`
	PlanId                    types.String  `tfsdk:"plan_id"`
	PlanTemplateId            types.String  `tfsdk:"plan_template_id"`
	Cumulative                types.Bool    `tfsdk:"cumulative"`
	StartDate                 types.String  `tfsdk:"start_date"`
	EndDate                   types.String  `tfsdk:"end_date"`
	PricingBands              types.List    `tfsdk:"pricing_bands"`
	Id                        types.String  `tfsdk:"id"`
	Version                   types.Int64   `tfsdk:"version"`
}

var pricingBandNestedObject = schema.NestedAttributeObject{
	Attributes: map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed: true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"lower_limit": schema.Float64Attribute{
			Required: true,
			Validators: []validator.Float64{
				float64validator.AtLeast(0),
			},
		},
		"fixed_price": schema.Float64Attribute{
			Required: true,
		},
		"unit_price": schema.Float64Attribute{
			Required: true,
		},
	},
}

func (r *PricingResourceModel) GetId() types.String {
	return r.Id
}

func (r *PricingResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pricing"
}

func (r *PricingResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Pricing resource",

		Attributes: map[string]schema.Attribute{
			"description": schema.StringAttribute{
				MarkdownDescription: "Displayed on Bill line items.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(200),
				},
			},
			"code": schema.StringAttribute{
				MarkdownDescription: "Unique short code for the Pricing.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 80),
					stringvalidator.RegexMatches(regexp.MustCompile(`^([^\p{Cc}\s])|([^\p{Cc}\s][[^\p{Cc}\s] ]*[^\p{Cc}\s])$`), "The code must not contain control characters or start/end with whitespace."),
				},
			},
			"aggregation_id": schema.StringAttribute{
				MarkdownDescription: "UUID of the Aggregation used to create the Pricing. Use this when creating a Pricing for a segmented aggregation.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"compound_aggregation_id": schema.StringAttribute{
				MarkdownDescription: "UUID of the Compound Aggregation used to create the Pricing.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "The type of the pricing.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("DEBIT", "PRODUCT_CREDIT", "GLOBAL_CREDIT"),
				},
			},
			"segment": schema.MapAttribute{
				MarkdownDescription: "Specifies the segment value which you are defining a Pricing for using this call.",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"tiers_span_plan": schema.BoolAttribute{
				MarkdownDescription: "If TRUE, usage accumulates over the entire period the priced Plan is active for the account, and is not reset for pricing band rates at the start of each billing period.\n\nIf FALSE, usage does not accumulate, and is reset for pricing bands at the start of each billing period.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"minimum_spend": schema.Float64Attribute{
				MarkdownDescription: "The minimum spend amount per billing cycle for end customer Accounts on a Plan to which the Pricing is applied.",
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
			"minimum_spend_bill_in_advance": schema.BoolAttribute{
				MarkdownDescription: "When TRUE, minimum spend is billed at the start of each billing period.\n\nWhen FALSE, minimum spend is billed at the end of each billing period.",
				Optional:            true,
			},
			"overage_pricing_bands": schema.ListNestedAttribute{
				MarkdownDescription: "Specify Prepayment/Balance overage pricing in pricing bands for the case of a Tiered pricing structure.",
				Optional:            true,
				NestedObject:        pricingBandNestedObject,
			},
			"plan_id": schema.StringAttribute{
				MarkdownDescription: "UUID of the Plan the Pricing is created for.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"plan_template_id": schema.StringAttribute{
				MarkdownDescription: "UUID of the Plan Template the Pricing is created for.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"cumulative": schema.BoolAttribute{
				MarkdownDescription: "Controls whether or not charge rates under a set of pricing bands configured for a Pricing are applied according to each separate band or at the highest band reached.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"start_date": schema.StringAttribute{
				MarkdownDescription: "The start date (in ISO-8601 format) for when the Pricing starts to be active for the Plan of Plan Template.",
				Required:            true,
			},
			"end_date": schema.StringAttribute{
				MarkdownDescription: "The end date (in ISO-8601 format) for when the Pricing ceases to be active for the Plan or Plan Template.",
				Optional:            true,
			},
			"pricing_bands": schema.ListNestedAttribute{
				MarkdownDescription: "The pricing bands of the pricing.",
				Required:            true,
				NestedObject:        pricingBandNestedObject,
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

func (r *PricingResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *PricingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	genericCreate[PricingResourceModel](ctx, req, resp, r.client, "/pricings", "pricing", r.read, r.write)
}

func (r *PricingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	genericRead[PricingResourceModel](ctx, req, resp, r.client, "/pricings", "pricing", r.read)
}

func (r *PricingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	genericUpdate[PricingResourceModel](ctx, req, resp, r.client, "/pricings", "pricing", r.read, r.write)
}

func (r *PricingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	genericDelete[PricingResourceModel](ctx, req, resp, r.client, "/pricings", "pricing")
}

func (r *PricingResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *PricingResource) read(ctx context.Context, data *PricingResourceModel, restData map[string]any, diagnostics *diag.Diagnostics) {
	m := &mapper{
		ctx:         ctx,
		diagnostics: diagnostics,
		v:           restData,
	}
	m.to("id", &data.Id)
	m.to("version", &data.Version)
	m.to("description", &data.Description)
	m.to("code", &data.Code)
	m.to("aggregationId", &data.AggregationId)
	m.to("compoundAggregationId", &data.CompoundAggregationId)
	m.to("type", &data.Type)
	if segments, ok := restData["segment"].(map[string]any); ok {
		elements := make(map[string]attr.Value)
		for k, v := range segments {
			if v, ok := v.(string); ok {
				elements[k] = types.StringValue(v)
			} else {
				diagnostics.AddError("Invalid segment", "Segment must be a map of strings")
			}
		}
		mv, diag := types.MapValue(types.StringType, elements)
		diagnostics.Append(diag...)
		data.Segment = mv
	}

	m.to("tiersSpanPlan", &data.TiersSpanPlan)
	m.to("minimumSpend", &data.MinimumSpend)
	m.to("minimumSpendDescription", &data.MinimumSpendDescription)
	m.to("minimumSpendBillInAdvance", &data.MinimumSpendBillInAdvance)

	if bands, ok := restData["overagePricingBands"].([]any); ok {
		if len(bands) > 0 {
			lv := readPricingBandList(bands, diagnostics)
			data.OveragePricingBands = lv
		}
	}
	m.to("planId", &data.PlanId)
	m.to("planTemplateId", &data.PlanTemplateId)
	m.to("cumulative", &data.Cumulative)
	m.to("startDate", &data.StartDate)
	m.to("endDate", &data.EndDate)
	if bands, ok := restData["pricingBands"].([]any); ok {
		lv := readPricingBandList(bands, diagnostics)
		data.PricingBands = lv
	}
}

func (r *PricingResource) write(ctx context.Context, data *PricingResourceModel, restData map[string]any, diagnostics *diag.Diagnostics) {
	m := &mapper{
		ctx:         ctx,
		diagnostics: diagnostics,
		v:           restData,
	}
	m.from(data.Id, "id")
	m.from(data.Version, "version")
	m.from(data.Description, "description")
	m.from(data.Code, "code")
	m.from(data.AggregationId, "aggregationId")
	m.from(data.CompoundAggregationId, "compoundAggregationId")
	m.from(data.Type, "type")
	if segment := data.Segment; !segment.IsUnknown() && !segment.IsNull() {
		elements := make(map[string]any)

		for k, v := range segment.Elements() {
			if v, ok := v.(types.String); ok {
				elements[k] = v.ValueString()
			}
		}
		m.v["segment"] = elements
	}
	m.from(data.TiersSpanPlan, "tiersSpanPlan")
	m.from(data.MinimumSpend, "minimumSpend")
	m.from(data.MinimumSpendDescription, "minimumSpendDescription")
	m.from(data.MinimumSpendBillInAdvance, "minimumSpendBillInAdvance")
	if bands := data.OveragePricingBands; !bands.IsUnknown() && !bands.IsNull() {
		bandList := writePricingBandList(bands, diagnostics)
		m.v["overagePricingBands"] = bandList
	}
	m.from(data.PlanId, "planId")
	m.from(data.PlanTemplateId, "planTemplateId")
	m.from(data.Cumulative, "cumulative")
	m.from(data.StartDate, "startDate")
	m.from(data.EndDate, "endDate")
	if bands := data.PricingBands; !bands.IsUnknown() {
		bandList := writePricingBandList(bands, diagnostics)
		m.v["pricingBands"] = bandList
	}
}

func writePricingBandList(bands types.List, diagnostics *diag.Diagnostics) []any {
	bandList := make([]any, 0, len(bands.Elements()))
	for _, band := range bands.Elements() {
		band, ok := band.(types.Object)
		if !ok {
			diagnostics.AddError("Invalid overage pricing band", "Pricing band must be an object")
			continue
		}

		attrs := band.Attributes()

		if !ok {
			diagnostics.AddError("Invalid overage pricing band", "Pricing band must have an id")
		}
		lowerLimit, ok := attrs["lower_limit"].(types.Float64)
		if !ok {
			diagnostics.AddError("Invalid overage pricing band", "Pricing band must have a lower limit")
		}

		fixedPrice, ok := attrs["fixed_price"].(types.Float64)
		if !ok {
			diagnostics.AddError("Invalid overage pricing band", "Pricing band must have a fixed price")
		}

		unitPrice, ok := attrs["unit_price"].(types.Float64)
		if !ok {
			diagnostics.AddError("Invalid overage pricing band", "Pricing band must have a unit price")
		}

		bandMap := map[string]any{
			"lowerLimit": lowerLimit.ValueFloat64(),
			"fixedPrice": fixedPrice.ValueFloat64(),
			"unitPrice":  unitPrice.ValueFloat64(),
		}
		id, ok := attrs["id"].(types.String)
		if ok && !id.IsUnknown() {
			bandMap["id"] = id.ValueString()
		}

		bandList = append(bandList, bandMap)
	}
	return bandList
}

func readPricingBandList(bands []any, diagnostics *diag.Diagnostics) types.List {
	elements := make([]attr.Value, 0, len(bands))
	for _, b := range bands {
		if b, ok := b.(map[string]any); ok {
			id, ok := b["id"].(string)
			if !ok {
				diagnostics.AddError("Invalid overage pricing band", "Pricing band must have an id")
			}

			lowerLimit, ok := b["lowerLimit"].(float64)
			if !ok {
				diagnostics.AddError("Invalid overage pricing band", "Pricing band must have a lower limit")
			}
			fixedPrice, ok := b["fixedPrice"].(float64)
			if !ok {
				diagnostics.AddError("Invalid overage pricing band", "Pricing band must have a fixed price")
			}
			unitPrice, ok := b["unitPrice"].(float64)
			if !ok {
				diagnostics.AddError("Invalid overage pricing band", "Pricing band must have a unit price")
			}

			band, diag := types.ObjectValue(map[string]attr.Type{
				"id":          types.StringType,
				"lower_limit": types.Float64Type,
				"fixed_price": types.Float64Type,
				"unit_price":  types.Float64Type,
			}, map[string]attr.Value{
				"id":          types.StringValue(id),
				"lower_limit": types.Float64Value(lowerLimit),
				"fixed_price": types.Float64Value(fixedPrice),
				"unit_price":  types.Float64Value(unitPrice),
			})
			diagnostics.Append(diag...)

			elements = append(elements, band)
		} else {
			diagnostics.AddError("Invalid overage pricing band", "Pricing band must be a map")
		}
	}
	lv, diag := types.ListValue(pricingBandNestedObject.Type(), elements)
	diagnostics.Append(diag...)
	return lv
}
