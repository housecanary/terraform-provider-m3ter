// Copyright (c) HouseCanary, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/url"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
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
var _ resource.Resource = &MeterResource{}
var _ resource.ResourceWithImportState = &MeterResource{}

func NewMeterResource() resource.Resource {
	return &MeterResource{}
}

// MeterResource defines the resource implementation.
type MeterResource struct {
	client *m3terClient
}

// MeterResourceModel describes the resource data model.
type MeterResourceModel struct {
	CustomFields  types.Dynamic `tfsdk:"custom_fields"`
	ProductId     types.String  `tfsdk:"product_id"`
	GroupId       types.String  `tfsdk:"group_id"`
	Name          types.String  `tfsdk:"name"`
	Code          types.String  `tfsdk:"code"`
	DataFields    types.List    `tfsdk:"data_fields"`
	DerivedFields types.List    `tfsdk:"derived_fields"`
	Id            types.String  `tfsdk:"id"`
	Version       types.Int64   `tfsdk:"version"`
}

var dataFieldsType = schema.NestedAttributeObject{
	Attributes: map[string]schema.Attribute{
		"category": schema.StringAttribute{
			MarkdownDescription: "The field type, which defines the type of data collected in the field.",
			Required:            true,
			Validators: []validator.String{
				stringvalidator.OneOf(
					"WHO",
					"WHAT",
					"WHERE",
					"OTHER",
					"METADATA",
					"MEASURE",
					"INCOME",
					"COST",
				),
			},
		},
		"code": schema.StringAttribute{
			MarkdownDescription: "Short code to identify the field",
			Required:            true,
			Validators: []validator.String{
				stringvalidator.LengthBetween(1, 80),
				stringvalidator.RegexMatches(regexp.MustCompile(`^[\p{L}_$][\p{L}_$0-9]*$`), "The code must start with a letter or underscore and contain only letters, numbers, and underscores."),
			},
		},
		"name": schema.StringAttribute{
			MarkdownDescription: "Descriptive name for the field",
			Required:            true,
			Validators: []validator.String{
				stringvalidator.LengthBetween(1, 200),
			},
		},
		"unit": schema.StringAttribute{
			MarkdownDescription: "The units to measure the data with. Should conform to Unified Code for Units of Measure (UCUM). Required only for numeric field categories.",
			Optional:            true,
			Validators: []validator.String{
				stringvalidator.LengthBetween(1, 50),
			},
		},
	},
}

var derivedFieldsType = schema.NestedAttributeObject{
	Attributes: map[string]schema.Attribute{
		"category": schema.StringAttribute{
			MarkdownDescription: "The field type, which defines the type of data collected in the field.",
			Required:            true,
			Validators: []validator.String{
				stringvalidator.OneOf(
					"WHO",
					"WHAT",
					"WHERE",
					"OTHER",
					"METADATA",
					"MEASURE",
					"INCOME",
					"COST",
				),
			},
		},
		"code": schema.StringAttribute{
			MarkdownDescription: "Short code to identify the field",
			Required:            true,
			Validators: []validator.String{
				stringvalidator.LengthBetween(1, 80),
				stringvalidator.RegexMatches(regexp.MustCompile(`^[\p{L}_$][\p{L}_$0-9]*$`), "The code must start with a letter or underscore and contain only letters, numbers, and underscores."),
			},
		},
		"name": schema.StringAttribute{
			MarkdownDescription: "Descriptive name for the field",
			Required:            true,
			Validators: []validator.String{
				stringvalidator.LengthBetween(1, 200),
			},
		},
		"unit": schema.StringAttribute{
			MarkdownDescription: "The units to measure the data with. Should conform to Unified Code for Units of Measure (UCUM). Required only for numeric field categories.",
			Optional:            true,
			Validators: []validator.String{
				stringvalidator.LengthBetween(1, 50),
			},
		},
		"calculation": schema.StringAttribute{
			MarkdownDescription: "The calculation used to transform the value of submitted dataFields in usage data. Calculation can reference dataFields, customFields, or system Timestamp fields.",
			Required:            true,
		},
	},
}

func (r *MeterResourceModel) GetId() types.String {
	return r.Id
}

func (r *MeterResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_meter"
}

func (r *MeterResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Meter resource",

		Attributes: map[string]schema.Attribute{
			"custom_fields": schema.DynamicAttribute{
				MarkdownDescription: "User defined fields enabling you to attach custom data. The value for a custom field can be either a string or a number.",
				Required:            true,
			},
			"product_id": schema.StringAttribute{
				MarkdownDescription: "UUID of the product the Meter belongs to. (Optional) - if left blank, the Meter is global.",
				Optional:            true,
			},
			"group_id": schema.StringAttribute{
				MarkdownDescription: "UUID of the group the Meter belongs to. (Optional).",
				Optional:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Descriptive name for the Meter.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 200),
				},
			},
			"code": schema.StringAttribute{
				MarkdownDescription: "Code of the Meter - unique short code used to identify the Meter.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 80),
					stringvalidator.RegexMatches(regexp.MustCompile(`^([^\p{Cc}\s])|([^\p{Cc}\s][[^\p{Cc}\s] ]*[^\p{Cc}\s])$`), "The code must not contain control characters or start/end with whitespace."),
				},
			},
			"data_fields": schema.ListNestedAttribute{
				MarkdownDescription: "Used to submit categorized raw usage data values for ingest into the platform - either numeric quantitative values or non-numeric data values. At least one required per Meter; maximum 15 per Meter.",
				Required:            true,
				NestedObject:        dataFieldsType,
				Validators: []validator.List{
					listvalidator.SizeBetween(1, 15),
				},
			},
			"derived_fields": schema.ListNestedAttribute{
				MarkdownDescription: "Used to submit usage data values for ingest into the platform that are the result of a calculation performed on dataFields, customFields, or system Timestamp fields. Raw usage data is not submitted using derivedFields. Maximum 15 per Meter.",
				Required:            true,
				NestedObject:        derivedFieldsType,
				Validators: []validator.List{
					listvalidator.SizeAtMost(15),
				},
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Meter identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"version": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Meter version",
			},
		},
	}
}

func (r *MeterResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *MeterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	genericCreate(ctx, req, resp, r.client, "/meters", "meter", r.read, r.write)
}

func (r *MeterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	genericRead(ctx, req, resp, r.client, "/meters", "meter", r.read)
}

func (r *MeterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	genericUpdate(ctx, req, resp, r.client, "/meters", "meter", r.read, r.write)
}

func (r *MeterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	genericDelete[MeterResourceModel](ctx, req, resp, r.client, "/meters", "meter")
}

func (r *MeterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var restData map[string]any
	err := r.client.execute(ctx, "GET", "/meters/"+url.PathEscape(req.ID), nil, nil, &restData)
	if sc, ok := err.(*statusCodeError); ok && sc.StatusCode == 404 {
		urlValues := url.Values{}
		urlValues.Set("pageSize", "1")
		urlValues.Set("codes", req.ID)

		var meterListResponse struct {
			Data []struct {
				Id      string `json:"id"`
				Code    string `json:"code"`
				Version int64  `json:"version"`
			} `json:"data"`
			NextToken string `json:"next_token"`
		}
		err := r.client.execute(ctx, "GET", "/meters", nil, nil, &meterListResponse)
		if err != nil {
			resp.Diagnostics.AddError("Failed to list meters", err.Error())
			return
		}
		for _, meter := range meterListResponse.Data {
			if meter.Code == req.ID {
				resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), meter.Id)...)
				return
			}
		}
		resp.Diagnostics.AddError("Meter not found", "The meter with code "+req.ID+" does not exist.")
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *MeterResource) read(ctx context.Context, data *MeterResourceModel, restData map[string]any, diagnostics *diag.Diagnostics) {
	m := &mapper{
		ctx:         ctx,
		diagnostics: diagnostics,
		v:           restData,
	}
	m.to("id", &data.Id)
	m.to("version", &data.Version)
	m.customFieldsTo(&data.CustomFields)
	m.to("productId", &data.ProductId)
	m.to("groupId", &data.GroupId)
	m.to("name", &data.Name)
	m.to("code", &data.Code)
	m.listTo("dataFields", &data.DataFields, dataFieldsType.Type(), func(v any) (attr.Value, diag.Diagnostics) {
		mv, ok := v.(map[string]any)
		if !ok {
			return nil, diag.Diagnostics{diag.NewErrorDiagnostic("data_fields must be a list of objects", "expected data_fields to be a list of objects")}
		}

		attrs := make(map[string]attr.Value)
		category, ok := mv["category"].(string)
		if !ok {
			return nil, diag.Diagnostics{diag.NewErrorDiagnostic("category must be a string", "expected category to be a string")}
		}
		attrs["category"] = types.StringValue(category)

		code, ok := mv["code"].(string)
		if !ok {
			return nil, diag.Diagnostics{diag.NewErrorDiagnostic("code must be a string", "expected code to be a string")}
		}
		attrs["code"] = types.StringValue(code)

		name, ok := mv["name"].(string)
		if !ok {
			return nil, diag.Diagnostics{diag.NewErrorDiagnostic("name must be a string", "expected name to be a string")}
		}
		attrs["name"] = types.StringValue(name)

		if _, ok := mv["unit"]; ok {
			unit, ok := mv["unit"].(string)
			if !ok {
				return nil, diag.Diagnostics{diag.NewErrorDiagnostic("unit must be a string", "expected unit to be a string")}
			}
			attrs["unit"] = types.StringValue(unit)
		} else {
			attrs["unit"] = types.StringNull()
		}

		ts := make(map[string]attr.Type)
		for k, v := range dataFieldsType.Attributes {
			ts[k] = v.GetType()
		}

		return types.ObjectValue(ts, attrs)
	})

	m.listTo("derivedFields", &data.DerivedFields, derivedFieldsType.Type(), func(v any) (attr.Value, diag.Diagnostics) {
		mv, ok := v.(map[string]any)
		if !ok {
			return nil, diag.Diagnostics{diag.NewErrorDiagnostic("derived_fields must be a list of objects", "expected derived_fields to be a list of objects")}
		}

		attrs := make(map[string]attr.Value)
		category, ok := mv["category"].(string)
		if !ok {
			return nil, diag.Diagnostics{diag.NewErrorDiagnostic("category must be a string", "expected category to be a string")}
		}
		attrs["category"] = types.StringValue(category)

		code, ok := mv["code"].(string)
		if !ok {
			return nil, diag.Diagnostics{diag.NewErrorDiagnostic("code must be a string", "expected code to be a string")}
		}
		attrs["code"] = types.StringValue(code)

		name, ok := mv["name"].(string)
		if !ok {
			return nil, diag.Diagnostics{diag.NewErrorDiagnostic("name must be a string", "expected name to be a string")}
		}
		attrs["name"] = types.StringValue(name)

		if _, ok := mv["unit"]; ok {
			unit, ok := mv["unit"].(string)
			if !ok {
				return nil, diag.Diagnostics{diag.NewErrorDiagnostic("unit must be a string", "expected unit to be a string")}
			}
			attrs["unit"] = types.StringValue(unit)
		} else {
			attrs["unit"] = types.StringNull()
		}

		calculation, ok := mv["calculation"].(string)
		if !ok {
			return nil, diag.Diagnostics{diag.NewErrorDiagnostic("calculation must be a string", "expected calculation to be a string")}
		}
		attrs["calculation"] = types.StringValue(calculation)

		ts := make(map[string]attr.Type)
		for k, v := range derivedFieldsType.Attributes {
			ts[k] = v.GetType()
		}

		return types.ObjectValue(ts, attrs)
	})

}

func (r *MeterResource) write(ctx context.Context, data *MeterResourceModel, restData map[string]any, diagnostics *diag.Diagnostics) {
	m := &mapper{
		ctx:         ctx,
		diagnostics: diagnostics,
		v:           restData,
	}

	m.from(data.Id, "id")
	m.from(data.Version, "version")
	m.customFieldsFrom(data.CustomFields)
	m.from(data.ProductId, "productId")
	m.from(data.GroupId, "groupId")
	m.from(data.Name, "name")
	m.from(data.Code, "code")
	m.listFrom(data.DataFields, "dataFields", func(v attr.Value) (any, diag.Diagnostics) {
		ov, ok := v.(types.Object)
		if !ok {
			return nil, diag.Diagnostics{diag.NewErrorDiagnostic("data_fields must be a list of objects", "expected data_fields to be a list of objects")}
		}

		m := make(map[string]any)
		attrs := ov.Attributes()

		category, ok := attrs["category"].(types.String)
		if !ok {
			return nil, diag.Diagnostics{diag.NewErrorDiagnostic("category must be a string", "expected category to be a string")}
		}
		m["category"] = category.ValueString()

		code, ok := attrs["code"].(types.String)
		if !ok {
			return nil, diag.Diagnostics{diag.NewErrorDiagnostic("code must be a string", "expected code to be a string")}
		}

		m["code"] = code.ValueString()

		name, ok := attrs["name"].(types.String)
		if !ok {
			return nil, diag.Diagnostics{diag.NewErrorDiagnostic("name must be a string", "expected name to be a string")}
		}

		m["name"] = name.ValueString()

		if _, ok := attrs["unit"]; ok {
			unit, ok := attrs["unit"].(types.String)
			if !ok {
				return nil, diag.Diagnostics{diag.NewErrorDiagnostic("unit must be a string", "expected unit to be a string")}
			}

			if !unit.IsUnknown() && !unit.IsNull() {
				m["unit"] = unit.ValueString()
			}
		}

		return m, nil
	})
	m.listFrom(data.DerivedFields, "derivedFields", func(v attr.Value) (any, diag.Diagnostics) {
		ov, ok := v.(types.Object)
		if !ok {
			return nil, diag.Diagnostics{diag.NewErrorDiagnostic("derived_fields must be a list of objects", "expected derived_fields to be a list of objects")}
		}

		m := make(map[string]any)
		attrs := ov.Attributes()

		category, ok := attrs["category"].(types.String)
		if !ok {
			return nil, diag.Diagnostics{diag.NewErrorDiagnostic("category must be a string", "expected category to be a string")}
		}
		m["category"] = category.ValueString()

		code, ok := attrs["code"].(types.String)
		if !ok {
			return nil, diag.Diagnostics{diag.NewErrorDiagnostic("code must be a string", "expected code to be a string")}
		}

		m["code"] = code.ValueString()

		name, ok := attrs["name"].(types.String)
		if !ok {
			return nil, diag.Diagnostics{diag.NewErrorDiagnostic("name must be a string", "expected name to be a string")}
		}

		m["name"] = name.ValueString()

		if _, ok := attrs["unit"]; ok {
			unit, ok := attrs["unit"].(types.String)
			if !ok {
				return nil, diag.Diagnostics{diag.NewErrorDiagnostic("unit must be a string", "expected unit to be a string")}
			}

			if !unit.IsUnknown() && !unit.IsNull() {
				m["unit"] = unit.ValueString()
			}
		}

		calculation, ok := attrs["calculation"].(types.String)
		if !ok {
			return nil, diag.Diagnostics{diag.NewErrorDiagnostic("calculation must be a string", "expected calculation to be a string")}
		}

		m["calculation"] = calculation.ValueString()

		return m, nil
	})
}
