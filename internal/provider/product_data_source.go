// Copyright (c) HouseCanary, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &ProductDataSource{}

func NewProductDataSource() datasource.DataSource {
	return &ProductDataSource{}
}

// ProductDataSource defines the data source implementation.
type ProductDataSource struct {
	client *m3terClient
}

type ProductDataSourceModel struct {
	Name         types.String  `tfsdk:"name"`
	Code         types.String  `tfsdk:"code"`
	CustomFields types.Dynamic `tfsdk:"custom_fields"`
	Id           types.String  `tfsdk:"id"`
	Version      types.Int64   `tfsdk:"version"`
}

func (r *ProductDataSourceModel) GetId() types.String {
	return r.Id
}

func (r *ProductDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_product"
}

func (r *ProductDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Product data source",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Descriptive name for the Product providing context and information.",
				Optional:            true,
				Computed:            true,
			},
			"code": schema.StringAttribute{
				MarkdownDescription: "A unique short code to identify the Product. It should not contain control characters or spaces.",
				Optional:            true,
				Computed:            true,
			},
			"custom_fields": schema.DynamicAttribute{
				MarkdownDescription: "User defined fields enabling you to attach custom data. The value for a custom field can be either a string or a number.",
				Computed:            true,
			},
			"id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Product identifier",
			},
			"version": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Product version",
			},
		},
	}
}

func (r *ProductDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (r *ProductDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ProductDataSourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if !data.Id.IsUnknown() && !data.Id.IsNull() {
		var restData map[string]any
		err := r.client.execute(ctx, "GET", "/products/"+url.PathEscape(data.Id.ValueString()), nil, nil, &restData)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read product, got error: %s", err))
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
		err := r.client.execute(ctx, "GET", "/products", queryParams, nil, &response)
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
		resp.Diagnostics.AddError("No matching product found", "No product found matching the specified criteria")
		return
	}

	if len(matches) > 1 {
		resp.Diagnostics.AddError("Multiple matching products found", "Multiple products found matching the specified criteria")
		return
	}

	r.read(ctx, &data, matches[0], &resp.Diagnostics)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProductDataSource) read(ctx context.Context, data *ProductDataSourceModel, restData map[string]any, diagnostics *diag.Diagnostics) {
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
}
