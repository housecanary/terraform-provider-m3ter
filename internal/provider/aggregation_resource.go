// Copyright (c) HouseCanary, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/url"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
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
var _ resource.Resource = &AggregationResource{}
var _ resource.ResourceWithImportState = &AggregationResource{}

func NewAggregationResource() resource.Resource {
	return &AggregationResource{}
}

// AggregationResource defines the resource implementation.
type AggregationResource struct {
	client *m3terClient
}

// AggregationResourceModel describes the resource data model.
type AggregationResourceModel struct {
	Name            types.String  `tfsdk:"name"`
	CustomFields    types.Dynamic `tfsdk:"custom_fields"`
	Rounding        types.String  `tfsdk:"rounding"`
	QuantityPerUnit types.Float64 `tfsdk:"quantity_per_unit"`
	Unit            types.String  `tfsdk:"unit"`
	Code            types.String  `tfsdk:"code"`
	MeterId         types.String  `tfsdk:"meter_id"`
	TargetField     types.String  `tfsdk:"target_field"`
	Aggregation     types.String  `tfsdk:"aggregation"`
	SegmentedFields types.List    `tfsdk:"segmented_fields"`
	Segments        types.List    `tfsdk:"segments"`
	DefaultValue    types.Float64 `tfsdk:"default_value"`
	Id              types.String  `tfsdk:"id"`
	Version         types.Int64   `tfsdk:"version"`
}

func (r *AggregationResourceModel) GetId() types.String {
	return r.Id
}

func (r *AggregationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_aggregation"
}

func (r *AggregationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Scheduled event configuration resource",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Descriptive name for the Aggregation.",
				Required:            true,
			},
			"custom_fields": schema.DynamicAttribute{
				MarkdownDescription: "User defined fields enabling you to attach custom data. The value for a custom field can be either a string or a number.",
				Required:            true,
			},
			"rounding": schema.StringAttribute{
				MarkdownDescription: "Specifies how you want to deal with non-integer, fractional number Aggregation values.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("UP", "DOWN", "NEAREST", "NONE"),
				},
			},
			"quantity_per_unit": schema.Float64Attribute{
				MarkdownDescription: "Defines how much of a quantity equates to 1 unit. Used when setting the price per unit for billing purposes - if charging for kilobytes per second (KiBy/s) at rate of $0.25 per 500 KiBy/s, then set quantityPerUnit to 500 and price Plan at $0.25 per unit.",
				Required:            true,
				Validators: []validator.Float64{
					float64validator.AtLeast(0),
				},
			},
			"unit": schema.StringAttribute{
				MarkdownDescription: "User defined label for units shown for Bill line items, indicating to your customers what they are being charged for.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 50),
				},
			},
			"code": schema.StringAttribute{
				MarkdownDescription: "Code of the new Aggregation. A unique short code to identify the Aggregation.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(80),
					stringvalidator.RegexMatches(regexp.MustCompile(`^[\p{L}_$][\p{L}_$0-9]*$`), "must be a code"),
				},
			},
			"meter_id": schema.StringAttribute{
				MarkdownDescription: "The UUID of the Meter used as the source of usage data for the Aggregation.",
				Required:            true,
			},
			"target_field": schema.StringAttribute{
				MarkdownDescription: "Code of the target dataField or derivedField on the Meter used as the basis for the Aggregation.",
				Required:            true,
			},
			"aggregation": schema.StringAttribute{
				MarkdownDescription: "Specifies the computation method applied to usage data collected in targetField.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("SUM", "MIN", "MAX", "COUNT", "LATEST", "MEAN", "UNIQUE"),
				},
			},
			"segmented_fields": schema.ListAttribute{
				MarkdownDescription: "Used when creating a segmented Aggregation, which segments the usage data collected by a single Meter. Works together with segments.",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"segments": schema.ListAttribute{
				MarkdownDescription: "Used when creating a segmented Aggregation, which segments the usage data collected by a single Meter. Works together with segmentedFields.",
				Optional:            true,
				ElementType: types.MapType{
					ElemType: types.StringType,
				},
			},
			"default_value": schema.Float64Attribute{
				MarkdownDescription: "Aggregation value used when no usage data is available to be aggregated.",
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

func (r *AggregationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AggregationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	genericCreate(ctx, req, resp, r.client, "/aggregations", "aggregation", r.read, r.write)
}

func (r *AggregationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	genericRead(ctx, req, resp, r.client, "/aggregations", "aggregation", r.read)
}

func (r *AggregationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	genericUpdate(ctx, req, resp, r.client, "/aggregations", "aggregation", r.read, r.write)
}

func (r *AggregationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	genericDelete[AggregationResourceModel](ctx, req, resp, r.client, "/aggregations", "aggregation")
}

func (r *AggregationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var restData map[string]any
	err := r.client.execute(ctx, "GET", "/aggregations/"+url.PathEscape(req.ID), nil, nil, &restData)
	if sc, ok := err.(*statusCodeError); ok && sc.StatusCode == 404 {
		urlValues := url.Values{}
		urlValues.Set("pageSize", "1")
		urlValues.Set("codes", req.ID)

		var aggregationListResponse struct {
			Data []struct {
				Id      string `json:"id"`
				Code    string `json:"code"`
				Version int64  `json:"version"`
			} `json:"data"`
			NextToken string `json:"next_token"`
		}
		err := r.client.execute(ctx, "GET", "/aggregations", nil, nil, &aggregationListResponse)
		if err != nil {
			resp.Diagnostics.AddError("Failed to list aggregations", err.Error())
			return
		}
		for _, aggregation := range aggregationListResponse.Data {
			if aggregation.Code == req.ID {
				resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), aggregation.Id)...)
				return
			}
		}
		resp.Diagnostics.AddError("Aggregation not found", "The aggregation with code "+req.ID+" does not exist.")
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *AggregationResource) read(ctx context.Context, data *AggregationResourceModel, restModel map[string]any, diagnostics *diag.Diagnostics) {
	m := &mapper{
		ctx:         ctx,
		diagnostics: diagnostics,
		v:           restModel,
	}

	m.to("id", &data.Id)
	m.to("version", &data.Version)
	m.to("name", &data.Name)
	m.customFieldsTo(&data.CustomFields)
	m.to("rounding", &data.Rounding)
	m.to("quantityPerUnit", &data.QuantityPerUnit)
	m.to("unit", &data.Unit)
	m.to("code", &data.Code)
	m.to("meterId", &data.MeterId)
	m.to("targetField", &data.TargetField)
	m.to("aggregation", &data.Aggregation)
	m.listTo("segmentedFields", &data.SegmentedFields, types.StringType, func(v any) (attr.Value, diag.Diagnostics) {
		if s, ok := v.(string); ok {
			return types.StringValue(s), nil
		}

		return nil, diag.Diagnostics{diag.NewErrorDiagnostic("expected a string in segmented fields", "expected a string in segmented fields")}
	})

	m.listTo("segments", &data.Segments, types.MapType{ElemType: types.StringType}, func(v any) (attr.Value, diag.Diagnostics) {
		var diags diag.Diagnostics

		m, ok := v.(map[string]any)
		if !ok {
			diags = append(diags, diag.NewErrorDiagnostic("expected a map in segments", "expected a map in segments"))
			return nil, diags
		}

		segment := make(map[string]attr.Value)
		for k, v := range m {
			if s, ok := v.(string); ok {
				segment[k] = types.StringValue(s)
			} else {
				diags = append(diags, diag.NewErrorDiagnostic("expected a string in segment", "expected a string in segment"))
			}
		}

		return types.MapValue(types.StringType, segment)
	})

	m.to("defaultValue", &data.DefaultValue)
}

func (r *AggregationResource) write(ctx context.Context, data *AggregationResourceModel, restModel map[string]any, diagnostics *diag.Diagnostics) {
	m := &mapper{
		ctx:         ctx,
		diagnostics: diagnostics,
		v:           restModel,
	}

	m.from(data.Id, "id")
	m.from(data.Version, "version")
	m.from(data.Name, "name")
	m.customFieldsFrom(data.CustomFields)
	m.from(data.Rounding, "rounding")
	m.from(data.QuantityPerUnit, "quantityPerUnit")
	m.from(data.Unit, "unit")
	m.from(data.Code, "code")
	m.from(data.MeterId, "meterId")
	m.from(data.TargetField, "targetField")
	m.from(data.Aggregation, "aggregation")
	m.listFrom(data.SegmentedFields, "segmentedFields", func(v attr.Value) (any, diag.Diagnostics) {
		s, ok := v.(types.String)
		if !ok {
			return nil, diag.Diagnostics{diag.NewErrorDiagnostic("expected a string in segmented fields", "expected a string in segmented fields")}
		}
		return s.ValueString(), nil
	})

	m.listFrom(data.Segments, "segments", func(v attr.Value) (any, diag.Diagnostics) {
		m, ok := v.(types.Map)
		if !ok {
			return nil, diag.Diagnostics{diag.NewErrorDiagnostic("expected a map in segments", "expected a map in segments")}
		}

		segment := make(map[string]any)
		for k, v := range m.Elements() {
			if s, ok := v.(types.String); ok {
				segment[k] = s.ValueString()
			}
		}

		return segment, nil
	})

	m.from(data.DefaultValue, "defaultValue")
}
