// Copyright (c) HouseCanary, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/url"
	"regexp"

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
var _ resource.Resource = &ProductResource{}
var _ resource.ResourceWithImportState = &ProductResource{}

func NewProductResource() resource.Resource {
	return &ProductResource{}
}

// ProductResource defines the resource implementation.
type ProductResource struct {
	client *m3terClient
}

// ProductResourceModel describes the resource data model.
type ProductResourceModel struct {
	Name         types.String  `tfsdk:"name"`
	Code         types.String  `tfsdk:"code"`
	CustomFields types.Dynamic `tfsdk:"custom_fields"`
	Id           types.String  `tfsdk:"id"`
	Version      types.Int64   `tfsdk:"version"`
}

func (r *ProductResourceModel) GetId() types.String {
	return r.Id
}

func (r *ProductResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_product"
}

func (r *ProductResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Product resource",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Descriptive name for the Product providing context and information.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 200),
				},
			},
			"code": schema.StringAttribute{
				MarkdownDescription: "A unique short code to identify the Product. It should not contain control chracters or spaces.",
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
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The UUID of the entity.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"version": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "The version number",
			},
		},
	}
}

func (r *ProductResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ProductResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	genericCreate[ProductResourceModel](ctx, req, resp, r.client, "/products", "product", r.read, r.write)
}

func (r *ProductResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	genericRead[ProductResourceModel](ctx, req, resp, r.client, "/products", "product", r.read)
}

func (r *ProductResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	genericUpdate[ProductResourceModel](ctx, req, resp, r.client, "/products", "product", r.read, r.write)
}

func (r *ProductResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	genericDelete[ProductResourceModel](ctx, req, resp, r.client, "/products", "product")
}

func (r *ProductResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var restData map[string]any
	err := r.client.execute(ctx, "GET", "/products/"+url.PathEscape(req.ID), nil, nil, &restData)
	if sc, ok := err.(*statusCodeError); ok && sc.StatusCode == 404 {
		urlValues := url.Values{}
		urlValues.Set("pageSize", "200")
		for {
			var productListResponse struct {
				Data []struct {
					Id      string `json:"id"`
					Code    string `json:"code"`
					Version int64  `json:"version"`
				} `json:"data"`
				NextToken string `json:"next_token"`
			}
			err := r.client.execute(ctx, "GET", "/products", nil, nil, &productListResponse)
			if err != nil {
				resp.Diagnostics.AddError("Failed to list products", err.Error())
				return
			}
			for _, product := range productListResponse.Data {
				if product.Code == req.ID {
					resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), product.Id)...)
					return
				}
			}
			if productListResponse.NextToken == "" {
				break
			}
			urlValues.Set("nextToken", productListResponse.NextToken)
		}

		resp.Diagnostics.AddError("Product not found", "The product with code "+req.ID+" does not exist.")
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *ProductResource) read(ctx context.Context, data *ProductResourceModel, restData map[string]any, diagnostics *diag.Diagnostics) {
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

func (r *ProductResource) write(ctx context.Context, data *ProductResourceModel, restData map[string]any, diagnostics *diag.Diagnostics) {
	m := &mapper{
		ctx:         ctx,
		diagnostics: diagnostics,
		v:           restData,
	}
	m.from(data.Id, "id")
	m.from(data.Version, "version")
	m.from(data.Name, "name")
	m.from(data.Code, "code")
	m.customFieldsFrom(data.CustomFields)
}
