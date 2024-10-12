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
				MarkdownDescription: "Name of the scheduled event",
				Required:            true,
			},
			"custom_fields": schema.DynamicAttribute{
				MarkdownDescription: "Custom fields",
				Required:            true,
			},
			"rounding": schema.StringAttribute{
				MarkdownDescription: "Rounding",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("UP", "DOWN", "NEAREST", "NONE"),
				},
			},
			"quantity_per_unit": schema.Float64Attribute{
				MarkdownDescription: "Quantity per unit",
				Required:            true,
				Validators: []validator.Float64{
					float64validator.AtLeast(0),
				},
			},
			"unit": schema.StringAttribute{
				MarkdownDescription: "Unit",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 50),
				},
			},
			"code": schema.StringAttribute{
				MarkdownDescription: "Code",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(80),
					stringvalidator.RegexMatches(regexp.MustCompile(`^[\p{L}_$][\p{L}_$0-9]*$`), "must be a code"),
				},
			},
			"meter_id": schema.StringAttribute{
				MarkdownDescription: "Meter ID",
				Required:            true,
			},
			"target_field": schema.StringAttribute{
				MarkdownDescription: "Target field",
				Required:            true,
			},
			"aggregation": schema.StringAttribute{
				MarkdownDescription: "Aggregation",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("SUM", "MIN", "MAX", "COUNT", "LATEST", "MEAN", "UNIQUE"),
				},
			},
			"segmented_fields": schema.ListAttribute{
				MarkdownDescription: "Segmented fields",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"segments": schema.ListAttribute{
				MarkdownDescription: "Segments",
				Optional:            true,
				ElementType: types.MapType{
					ElemType: types.StringType,
				},
			},
			"default_value": schema.Float64Attribute{
				MarkdownDescription: "Default value",
				Optional:            true,
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Scheduled Event Configuration identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"version": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Scheduled Event Configuration version",
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
		return s, nil
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
