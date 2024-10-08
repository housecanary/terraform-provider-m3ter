// Copyright (c) HouseCanary, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &WebhookDestinationResource{}
var _ resource.ResourceWithImportState = &WebhookDestinationResource{}

func NewWebhookDestinationResource() resource.Resource {
	return &WebhookDestinationResource{}
}

// WebhookDestinationResource defines the resource implementation.
type WebhookDestinationResource struct {
	client *m3terClient
}

// WebhookDestinationResourceModel describes the resource data model.
type WebhookDestinationResourceModel struct {
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Url         types.String `tfsdk:"url"`
	Code        types.String `tfsdk:"code"`
	Active      types.Bool   `tfsdk:"active"`
	Credentials types.Object `tfsdk:"credentials"`
	Id          types.String `tfsdk:"id"`
	Version     types.Int64  `tfsdk:"version"`
}

var credentialsAttributes = map[string]schema.Attribute{
	"api_key": schema.StringAttribute{
		MarkdownDescription: "The API key provided by m3ter. This key is part of the credential set required for signing requests and authenticating with m3ter services.",
		Required:            true,
		Validators: []validator.String{
			stringvalidator.LengthAtLeast(1),
		},
	},
	"secret": schema.StringAttribute{
		MarkdownDescription: "The secret associated with the API key. This secret is used in conjunction with the API key to generate a signature for secure authentication.",
		Required:            true,
		Sensitive:           true,
		Validators: []validator.String{
			stringvalidator.LengthAtLeast(1),
		},
	},
}

func (r *WebhookDestinationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook_destination"
}

func (r *WebhookDestinationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Webhook destination resource",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the Webhook Destination",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the Webhook Destination",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"url": schema.StringAttribute{
				MarkdownDescription: "The URL to which the Webhook Destination requests will be sent.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"code": schema.StringAttribute{
				Required: true,
			},
			"active": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"credentials": schema.SingleNestedAttribute{
				Attributes: credentialsAttributes,
				Required:   true,
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Webhook Destination identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"version": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Webhook Destination version",
			},
		},
	}
}

func (r *WebhookDestinationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *WebhookDestinationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data WebhookDestinationResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	webhookData := make(map[string]any)
	r.write(ctx, &data, webhookData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	var updatedWebhookData map[string]any
	err := r.client.execute(ctx, "POST", "/integrationdestinations/webhooks", nil, webhookData, &updatedWebhookData)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create webhook destination, got error: %s", err))
	}

	r.read(ctx, &data, updatedWebhookData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WebhookDestinationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data WebhookDestinationResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var webhookData map[string]any
	err := r.client.execute(ctx, "GET", "/integrationdestinations/webhooks/"+url.PathEscape(data.Id.ValueString()), nil, nil, &webhookData)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read webhook destination, got error: %s", err))
		return
	}

	r.read(ctx, &data, webhookData, &resp.Diagnostics)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WebhookDestinationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data WebhookDestinationResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var webhookData map[string]any
	err := r.client.execute(ctx, "GET", "/integrationdestinations/webhooks/"+url.PathEscape(data.Id.ValueString()), nil, nil, &webhookData)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read webhook destination, got error: %s", err))
		return
	}

	r.write(ctx, &data, webhookData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	var updatedWebhookData map[string]any
	err = r.client.execute(ctx, "PUT", "/integrationdestinations/webhooks/"+url.PathEscape(data.Id.ValueString()), nil, webhookData, &updatedWebhookData)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update webhook destination, got error: %s", err))
	}

	r.read(ctx, &data, updatedWebhookData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WebhookDestinationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data WebhookDestinationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.execute(ctx, "DELETE", "/integrationdestinations/webhooks/"+url.PathEscape(data.Id.ValueString()), nil, nil, nil)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete webhook destination, got error: %s", err))
	}
}

func (r *WebhookDestinationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *WebhookDestinationResource) read(ctx context.Context, data *WebhookDestinationResourceModel, webhookModel map[string]any, diagnostics *diag.Diagnostics) {
	m := &mapper{
		ctx:         ctx,
		diagnostics: diagnostics,
		v:           webhookModel,
	}

	m.to("id", &data.Id)
	m.to("version", &data.Version)
	m.to("name", &data.Name)
	m.to("description", &data.Description)
	m.to("url", &data.Url)
	m.to("code", &data.Code)
	m.to("active", &data.Active)

	// Never map the credentials back to the model since they are write-only
}

func (r *WebhookDestinationResource) write(ctx context.Context, data *WebhookDestinationResourceModel, webhookModel map[string]any, diagnostics *diag.Diagnostics) {
	m := &mapper{
		ctx:         ctx,
		diagnostics: diagnostics,
		v:           webhookModel,
	}

	m.from(data.Id, "id")
	m.from(data.Version, "version")
	m.from(data.Name, "name")
	m.from(data.Description, "description")
	m.from(data.Url, "url")
	m.from(data.Code, "code")
	m.from(data.Active, "active")

	creds, ok := webhookModel["credentials"].(map[string]any)
	if !ok {
		creds = make(map[string]any)
		webhookModel["credentials"] = creds
	}

	attrs := data.Credentials.Attributes()

	credsM := &mapper{
		ctx:         ctx,
		diagnostics: diagnostics,
		v:           creds,
	}

	credsM.from(attrs["api_key"], "apiKey")
	credsM.from(attrs["secret"], "secret")
	creds["type"] = "M3TER_SIGNED_REQUEST"
	creds["empty"] = false
}
