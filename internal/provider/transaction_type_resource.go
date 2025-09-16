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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &TransactionTypeResource{}
var _ resource.ResourceWithImportState = &TransactionTypeResource{}

func NewTransactionTypeResource() resource.Resource {
	return &TransactionTypeResource{}
}

// TransactionTypeResource defines the resource implementation.
type TransactionTypeResource struct {
	client *m3terClient
}

// TransactionTypeResourceModel describes the resource data model.
type TransactionTypeResourceModel struct {
	Name     types.String `tfsdk:"name"`
	Archived types.Bool   `tfsdk:"archived"`
	Code     types.String `tfsdk:"code"`
	Id       types.String `tfsdk:"id"`
	Version  types.Int64  `tfsdk:"version"`
}

func (r *TransactionTypeResourceModel) GetId() types.String {
	return r.Id
}

func (r *TransactionTypeResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_transaction_type"
}

func (r *TransactionTypeResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Transaction type resource",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Descriptive name for the Transaction Type.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 200),
				},
			},
			"archived": schema.BoolAttribute{
				MarkdownDescription: "Whether the Transaction Type is archived.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"code": schema.StringAttribute{
				MarkdownDescription: "Code of the Transaction Type - unique short code used to identify the Transaction Type.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 80),
					stringvalidator.RegexMatches(regexp.MustCompile(`^[\p{L}_$][\p{L}_$0-9]*$`), "The code must start with a letter or underscore and contain only letters, numbers, and underscores."),
				},
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Transaction Type identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"version": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Transaction Type version",
			},
		},
	}
}

func (r *TransactionTypeResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *TransactionTypeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	genericCreate[TransactionTypeResourceModel](ctx, req, resp, r.client, "/picklists/transactiontypes", "transaction_type", r.read, r.write)
}

func (r *TransactionTypeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	genericRead[TransactionTypeResourceModel](ctx, req, resp, r.client, "/picklists/transactiontypes", "transaction_type", r.read)
}

func (r *TransactionTypeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	genericUpdate[TransactionTypeResourceModel](ctx, req, resp, r.client, "/picklists/transactiontypes", "transaction_type", r.read, r.write)
}

func (r *TransactionTypeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	genericDelete[TransactionTypeResourceModel](ctx, req, resp, r.client, "/picklists/transactiontypes", "transaction_type")
}

func (r *TransactionTypeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var restData map[string]any
	err := r.client.execute(ctx, "GET", "/picklists/transactiontypes/"+url.PathEscape(req.ID), nil, nil, &restData)
	if sc, ok := err.(*statusCodeError); ok && sc.StatusCode == 404 {
		urlValues := url.Values{}
		urlValues.Set("pageSize", "1")
		urlValues.Set("codes", req.ID)

		var transactionTypeListResponse struct {
			Data []struct {
				Id      string `json:"id"`
				Code    string `json:"code"`
				Version int64  `json:"version"`
			} `json:"data"`
			NextToken string `json:"next_token"`
		}
		err := r.client.execute(ctx, "GET", "/picklists/transactiontypes", nil, nil, &transactionTypeListResponse)
		if err != nil {
			resp.Diagnostics.AddError("Failed to list transaction types", err.Error())
			return
		}
		for _, transactionType := range transactionTypeListResponse.Data {
			if transactionType.Code == req.ID {
				resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), transactionType.Id)...)
				return
			}
		}
		resp.Diagnostics.AddError("Transaction Type not found", "The transaction type with code "+req.ID+" does not exist.")
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *TransactionTypeResource) read(ctx context.Context, data *TransactionTypeResourceModel, restData map[string]any, diagnostics *diag.Diagnostics) {
	m := &mapper{
		ctx:         ctx,
		diagnostics: diagnostics,
		v:           restData,
	}
	m.to("id", &data.Id)
	m.to("version", &data.Version)
	m.to("name", &data.Name)
	m.to("archived", &data.Archived)
	m.to("code", &data.Code)
}

func (r *TransactionTypeResource) write(ctx context.Context, data *TransactionTypeResourceModel, restData map[string]any, diagnostics *diag.Diagnostics) {
	m := &mapper{
		ctx:         ctx,
		diagnostics: diagnostics,
		v:           restData,
	}

	m.from(data.Id, "id")
	m.from(data.Version, "version")
	m.from(data.Name, "name")
	m.from(data.Archived, "archived")
	m.from(data.Code, "code")
}
