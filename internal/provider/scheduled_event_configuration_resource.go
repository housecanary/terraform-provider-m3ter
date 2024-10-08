// Copyright (c) HouseCanary, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework-validators/int32validator"
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
var _ resource.Resource = &ScheduledEventConfigurationResource{}
var _ resource.ResourceWithImportState = &ScheduledEventConfigurationResource{}

func NewScheduledEventConfigurationResource() resource.Resource {
	return &ScheduledEventConfigurationResource{}
}

// ScheduledEventConfigurationResource defines the resource implementation.
type ScheduledEventConfigurationResource struct {
	client *m3terClient
}

// ScheduledEventConfigurationResourceModel describes the resource data model.
type ScheduledEventConfigurationResourceModel struct {
	Name    types.String `tfsdk:"name"`
	Entity  types.String `tfsdk:"entity"`
	Field   types.String `tfsdk:"field"`
	Offset  types.Int32  `tfsdk:"offset"`
	Id      types.String `tfsdk:"id"`
	Version types.Int64  `tfsdk:"version"`
}

func (r *ScheduledEventConfigurationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_scheduled_event_configuration"
}

func (r *ScheduledEventConfigurationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Scheduled event configuration resource",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the scheduled event",
				Required:            true,
			},
			"entity": schema.StringAttribute{
				MarkdownDescription: "Entity to schedule the event for",
				Required:            true,
			},
			"field": schema.StringAttribute{
				MarkdownDescription: "Field to schedule the event for",
				Required:            true,
			},
			"offset": schema.Int32Attribute{
				MarkdownDescription: "Offset in days to schedule the event",
				Required:            true,
				Validators: []validator.Int32{
					int32validator.AtLeast(1),
				},
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

func (r *ScheduledEventConfigurationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ScheduledEventConfigurationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ScheduledEventConfigurationResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	restData := make(map[string]any)
	r.write(ctx, &data, restData, &resp.Diagnostics)

	var updatedRestData map[string]any
	err := r.client.execute(ctx, "POST", "/scheduledevents/configurations", nil, restData, &updatedRestData)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create scheduled event configuration, got error: %s", err))
		return
	}

	r.read(ctx, &data, updatedRestData, &resp.Diagnostics)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ScheduledEventConfigurationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ScheduledEventConfigurationResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var restData map[string]any
	err := r.client.execute(ctx, "GET", "/scheduledevents/configurations/"+url.PathEscape(data.Id.ValueString()), nil, nil, &restData)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read example, got error: %s", err))
		return
	}

	r.read(ctx, &data, restData, &resp.Diagnostics)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ScheduledEventConfigurationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ScheduledEventConfigurationResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	restData := make(map[string]any)
	r.write(ctx, &data, restData, &resp.Diagnostics)

	var updatedRestData map[string]any
	err := r.client.execute(ctx, "PUT", "/scheduledevents/configurations/"+url.PathEscape(data.Id.ValueString()), nil, restData, &updatedRestData)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create scheduled event configuration, got error: %s", err))
	}

	r.read(ctx, &data, updatedRestData, &resp.Diagnostics)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ScheduledEventConfigurationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ScheduledEventConfigurationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.execute(ctx, "DELETE", "/scheduledevents/configurations/"+url.PathEscape(data.Id.ValueString()), nil, nil, nil)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete scheduled event configuration, got error: %s", err))
	}
}

func (r *ScheduledEventConfigurationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *ScheduledEventConfigurationResource) read(ctx context.Context, data *ScheduledEventConfigurationResourceModel, restModel map[string]any, diagnostics *diag.Diagnostics) {
	m := &mapper{
		ctx:         ctx,
		diagnostics: diagnostics,
		v:           restModel,
	}

	m.to("id", &data.Id)
	m.to("version", &data.Version)
	m.to("name", &data.Name)
	m.to("entity", &data.Entity)
	m.to("field", &data.Field)
	m.to("offset", &data.Offset)
}

func (r *ScheduledEventConfigurationResource) write(ctx context.Context, data *ScheduledEventConfigurationResourceModel, restModel map[string]any, diagnostics *diag.Diagnostics) {
	m := &mapper{
		ctx:         ctx,
		diagnostics: diagnostics,
		v:           restModel,
	}

	m.from(data.Id, "id")
	m.from(data.Version, "version")
	m.from(data.Name, "name")
	m.from(data.Entity, "entity")
	m.from(data.Field, "field")
	m.from(data.Offset, "offset")
}
