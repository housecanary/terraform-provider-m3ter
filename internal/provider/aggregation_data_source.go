// Copyright (c) HouseCanary, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &AggregationDataSource{}

func NewAggregationDataSource() datasource.DataSource {
	return &AggregationDataSource{}
}

// AggregationDataSource defines the data source implementation.
type AggregationDataSource struct {
	client *m3terClient
}

type AggregationDataSourceModel struct {
	Name         types.String  `tfsdk:"name"`
	Code         types.String  `tfsdk:"code"`
	CustomFields types.Dynamic `tfsdk:"custom_fields"`
	Segments     types.List    `tfsdk:"segments"`
	Id           types.String  `tfsdk:"id"`
	Version      types.Int64   `tfsdk:"version"`
}

func (r *AggregationDataSourceModel) GetId() types.String {
	return r.Id
}

func (r *AggregationDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_aggregation"
}

func (r *AggregationDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Aggregation data source",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Descriptive name for the Aggregation.",
				Optional:            true,
				Computed:            true,
			},
			"code": schema.StringAttribute{
				MarkdownDescription: "Code of the Aggregation. A unique short code to identify the Aggregation.",
				Optional:            true,
				Computed:            true,
			},
			"custom_fields": schema.DynamicAttribute{
				MarkdownDescription: "User defined fields enabling you to attach custom data. The value for a custom field can be either a string or a number.",
				Optional:            true,
				Computed:            true,
			},
			"segments": schema.ListAttribute{
				MarkdownDescription: "Used when creating a segmented Aggregation, which segments the usage data collected by a single Meter. Works together with `segmentedFields`.\n\nContains the values that are to be used as the segments, read from the fields in the meter pointed at by `segmentedFields`.",
				Computed:            true,
				ElementType: types.MapType{
					ElemType: types.StringType,
				},
			},
			"id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The UUID of the entity.",
			},
			"version": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "The version number.",
			},
		},
	}
}

func (r *AggregationDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (r *AggregationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data AggregationDataSourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if !data.Id.IsUnknown() && !data.Id.IsNull() {
		var restData map[string]any
		err := r.client.execute(ctx, "GET", "/aggregations/"+url.PathEscape(data.Id.ValueString()), nil, nil, &restData)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read aggregation, got error: %s", err))
			return
		}

		r.read(ctx, &data, restData, &resp.Diagnostics)

		// Save updated data into Terraform state
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	var matches []map[string]any
	queryParams := make(url.Values)
	queryParams.Set("pageSize", "200")
	for {
		var response struct {
			Data      []map[string]any `json:"data"`
			NextToken string           `json:"nextToken"`
		}
		err := r.client.execute(ctx, "GET", "/aggregations", queryParams, nil, &response)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list products, got error: %s", err))
			return
		}

		for _, restData := range response.Data {
			if !data.Name.IsUnknown() && !data.Name.IsNull() {
				name := data.Name.ValueString()
				productName, ok := restData["name"].(string)
				if !ok {
					continue
				}
				if productName != name {
					continue
				}
			}

			if !data.Code.IsUnknown() && !data.Code.IsNull() {
				code := data.Code.ValueString()
				productCode, ok := restData["code"].(string)
				if !ok {
					continue
				}

				if productCode != code {
					continue
				}
			}

			matches = append(matches, restData)
		}

		if response.NextToken == "" {
			break
		}

		queryParams.Set("nextToken", response.NextToken)
	}

	if len(matches) == 0 {
		resp.Diagnostics.AddError("No matching aggregation found", "No aggregation found matching the specified criteria")
		return
	}

	if len(matches) > 1 {
		resp.Diagnostics.AddError("Multiple matching aggregation found", "Multiple aggregation found matching the specified criteria")
		return
	}

	r.read(ctx, &data, matches[0], &resp.Diagnostics)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AggregationDataSource) read(ctx context.Context, data *AggregationDataSourceModel, restData map[string]any, diagnostics *diag.Diagnostics) {
	m := &mapper{
		ctx:         ctx,
		diagnostics: diagnostics,
		v:           restData,
	}
	m.to("id", &data.Id)
	m.to("version", &data.Version)
	m.to("name", &data.Name)
	m.to("code", &data.Code)
	m.customFieldsTo(&data.CustomFields)

	if segments, ok := restData["segments"].([]any); ok {
		items := make([]attr.Value, 0, len(segments))
		for _, segment := range segments {
			if segment, ok := segment.(map[string]any); ok {
				mapEntries := make(map[string]attr.Value, len(segment))
				for k, v := range segment {
					if v, ok := v.(string); ok {
						mapEntries[k] = types.StringValue(v)
					}
				}
				m, diag := types.MapValue(types.StringType, mapEntries)
				diagnostics.Append(diag...)
				items = append(items, m)
			}
		}

		lv, diag := types.ListValue(types.MapType{
			ElemType: types.StringType,
		}, items)
		diagnostics.Append(diag...)
		data.Segments = lv
	} else {
		data.Segments = types.ListNull(types.MapType{
			ElemType: types.StringType,
		})
	}
}
