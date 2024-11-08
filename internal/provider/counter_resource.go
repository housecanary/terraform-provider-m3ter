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
var _ resource.Resource = &CounterResource{}
var _ resource.ResourceWithImportState = &CounterResource{}

func NewCounterResource() resource.Resource {
	return &CounterResource{}
}

// CounterResource defines the resource implementation.
type CounterResource struct {
	client *m3terClient
}

// CounterResourceModel describes the resource data model.
type CounterResourceModel struct {
	Code      types.String `tfsdk:"code"`
	ProductId types.String `tfsdk:"product_id"`
	Name      types.String `tfsdk:"name"`
	Unit      types.String `tfsdk:"unit"`
	Id        types.String `tfsdk:"id"`
	Version   types.Int64  `tfsdk:"version"`
}

func (r *CounterResourceModel) GetId() types.String {
	return r.Id
}

func (r *CounterResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_counter"
}

func (r *CounterResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Counter resource",

		Attributes: map[string]schema.Attribute{
			"product_id": schema.StringAttribute{
				MarkdownDescription: "UUID of the product the Counter belongs to. (Optional) - if left blank, the Counter is global.",
				Optional:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Descriptive name for the Counter.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 200),
				},
			},
			"code": schema.StringAttribute{
				MarkdownDescription: "Code of the Counter - unique short code used to identify the Counter.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 80),
					stringvalidator.RegexMatches(regexp.MustCompile(`^([^\p{Cc}\s])|([^\p{Cc}\s][[^\p{Cc}\s] ]*[^\p{Cc}\s])$`), "The code must not contain control characters or start/end with whitespace."),
				},
			},
			"unit": schema.StringAttribute{
				MarkdownDescription: "User defined label for units shown on Bill line items, and indicating to your customers what they are being charged for.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Counter identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"version": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Counter version",
			},
		},
	}
}

func (r *CounterResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *CounterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	genericCreate(ctx, req, resp, r.client, "/counters", "counter", r.read, r.write)
}

func (r *CounterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	genericRead(ctx, req, resp, r.client, "/counters", "counter", r.read)
}

func (r *CounterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	genericUpdate(ctx, req, resp, r.client, "/counters", "counter", r.read, r.write)
}

func (r *CounterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	genericDelete[CounterResourceModel](ctx, req, resp, r.client, "/counters", "counter")
}

func (r *CounterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var restData map[string]any
	err := r.client.execute(ctx, "GET", "/counters/"+url.PathEscape(req.ID), nil, nil, &restData)
	if sc, ok := err.(*statusCodeError); ok && sc.StatusCode == 404 {
		urlValues := url.Values{}
		urlValues.Set("pageSize", "1")
		urlValues.Set("codes", req.ID)

		var counterListResponse struct {
			Data []struct {
				Id      string `json:"id"`
				Code    string `json:"code"`
				Version int64  `json:"version"`
			} `json:"data"`
			NextToken string `json:"next_token"`
		}
		err := r.client.execute(ctx, "GET", "/counters", nil, nil, &counterListResponse)
		if err != nil {
			resp.Diagnostics.AddError("Failed to list counters", err.Error())
			return
		}
		for _, counter := range counterListResponse.Data {
			if counter.Code == req.ID {
				resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), counter.Id)...)
				return
			}
		}
		resp.Diagnostics.AddError("Counter not found", "The counter with code "+req.ID+" does not exist.")
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *CounterResource) read(ctx context.Context, data *CounterResourceModel, restData map[string]any, diagnostics *diag.Diagnostics) {
	m := &mapper{
		ctx:         ctx,
		diagnostics: diagnostics,
		v:           restData,
	}
	m.to("id", &data.Id)
	m.to("version", &data.Version)
	m.to("productId", &data.ProductId)
	m.to("name", &data.Name)
	m.to("code", &data.Code)
	m.to("unit", &data.Unit)
}

func (r *CounterResource) write(ctx context.Context, data *CounterResourceModel, restData map[string]any, diagnostics *diag.Diagnostics) {
	m := &mapper{
		ctx:         ctx,
		diagnostics: diagnostics,
		v:           restData,
	}

	m.from(data.Id, "id")
	m.from(data.Version, "version")
	m.from(data.ProductId, "productId")
	m.from(data.Name, "name")
	m.from(data.Code, "code")
	m.from(data.Unit, "unit")
}
