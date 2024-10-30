// Copyright (c) HouseCanary, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"
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
var _ resource.Resource = &IntegrationConfigurationResource{}
var _ resource.ResourceWithImportState = &IntegrationConfigurationResource{}

func NewIntegrationConfigurationResource() resource.Resource {
	return &IntegrationConfigurationResource{}
}

// IntegrationConfigurationResource defines the resource implementation.
type IntegrationConfigurationResource struct {
	client *m3terClient
}

// IntegrationConfigurationResourceModel describes the resource data model.
type IntegrationConfigurationResourceModel struct {
	EntityType               types.String `tfsdk:"entity_type"`
	EntityId                 types.String `tfsdk:"entity_id"`
	Destination              types.String `tfsdk:"destination"`
	DestinationId            types.String `tfsdk:"destination_id"`
	ConfigData               types.String `tfsdk:"config_data"`
	Name                     types.String `tfsdk:"name"`
	IntegrationCredentialsId types.String `tfsdk:"integration_credentials_id"`
	Id                       types.String `tfsdk:"id"`
	Version                  types.Int64  `tfsdk:"version"`
}

func (r *IntegrationConfigurationResourceModel) GetId() types.String {
	return r.Id
}

func (r *IntegrationConfigurationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_configuration"
}

func (r *IntegrationConfigurationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Integration Configuration resource",

		Attributes: map[string]schema.Attribute{
			"entity_type": schema.StringAttribute{
				MarkdownDescription: "Specifies the type of entity for which the integration configuration is being updated. Must be a valid alphanumeric string.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile("^[a-zA-Z0-9_-]*$"), "Must be a valid alphanumeric string"),
					stringvalidator.LengthAtLeast(1),
				},
			},
			"entity_id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier (UUID) of the entity. This field is used to specify which entity's integration configuration you're updating.",
				Optional:            true,
			},
			"destination": schema.StringAttribute{
				MarkdownDescription: "Denotes the integration destination. This field identifies the target platform or service for the integration.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile("^[a-zA-Z0-9_-]*$"), "Must be a valid alphanumeric string"),
					stringvalidator.LengthAtLeast(1),
				},
			},
			"destination_id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier (UUID) for the integration destination.",
				Optional:            true,
			},
			"config_data": schema.StringAttribute{
				MarkdownDescription: "A flexible object to include any additional configuration data specific to the integration.",
				Required:            true,
			},
			"integration_credentials_id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier (UUID) of the integration credentials. This field is used to specify the credentials used for the integration.",
				Optional:            true,
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the Integration Configuration",
				Required:            true,
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Integration Configuration identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"version": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Integration Configuration version",
			},
		},
	}
}

func (r *IntegrationConfigurationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *IntegrationConfigurationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	genericCreate(ctx, req, resp, r.client, "/integrationconfigs", "integration configuration", r.read, r.write)
}

func (r *IntegrationConfigurationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	genericRead(ctx, req, resp, r.client, "/integrationconfigs", "integration configuration", r.read)
}

func (r *IntegrationConfigurationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	genericUpdate(ctx, req, resp, r.client, "/integrationconfigs", "integration configuration", r.read, r.write)
}

func (r *IntegrationConfigurationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	genericDelete[IntegrationConfigurationResourceModel](ctx, req, resp, r.client, "/integrationconfigs", "integration configuration")
}

func (r *IntegrationConfigurationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *IntegrationConfigurationResource) read(ctx context.Context, data *IntegrationConfigurationResourceModel, restData map[string]any, diagnostics *diag.Diagnostics) {
	m := &mapper{
		ctx:         ctx,
		diagnostics: diagnostics,
		v:           restData,
	}

	m.to("id", &data.Id)
	m.to("version", &data.Version)
	m.to("name", &data.Name)
	m.to("entityType", &data.EntityType)
	m.to("entityId", &data.EntityId)
	m.to("destination", &data.Destination)
	m.to("destinationId", &data.DestinationId)
	if _, ok := restData["integrationCredentialsId"]; !ok {
		restData["integrationCredentialsId"] = ""
	}
	m.to("integrationCredentialsId", &data.IntegrationCredentialsId)
	configData, _ := json.Marshal(restData["configData"])
	data.ConfigData = types.StringValue(string(configData))
}

func (r *IntegrationConfigurationResource) write(ctx context.Context, data *IntegrationConfigurationResourceModel, restData map[string]any, diagnostics *diag.Diagnostics) {
	m := &mapper{
		ctx:         ctx,
		diagnostics: diagnostics,
		v:           restData,
	}
	m.from(data.Id, "id")
	m.from(data.Version, "version")
	m.from(data.Name, "name")
	m.from(data.EntityType, "entityType")
	m.from(data.EntityId, "entityId")
	m.from(data.Destination, "destination")
	m.from(data.DestinationId, "destinationId")
	if data.IntegrationCredentialsId.ValueString() != "" {
		m.from(data.IntegrationCredentialsId, "integrationCredentialsId")
	}
	restData["configData"] = json.RawMessage(data.ConfigData.ValueString())
}
