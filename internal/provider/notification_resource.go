// Copyright (c) HouseCanary, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

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
var _ resource.Resource = &NotificationResource{}
var _ resource.ResourceWithImportState = &NotificationResource{}

func NewNotificationResource() resource.Resource {
	return &NotificationResource{}
}

// NotificationResource defines the resource implementation.
type NotificationResource struct {
	client *m3terClient
}

// NotificationResourceModel describes the resource data model.
type NotificationResourceModel struct {
	Name            types.String `tfsdk:"name"`
	Description     types.String `tfsdk:"description"`
	Active          types.Bool   `tfsdk:"active"`
	AlwaysFireEvent types.Bool   `tfsdk:"always_fire_event"`
	Calculation     types.String `tfsdk:"calculation"`
	Code            types.String `tfsdk:"code"`
	EventName       types.String `tfsdk:"event_name"`
	Id              types.String `tfsdk:"id"`
	Version         types.Int64  `tfsdk:"version"`
}

func (r *NotificationResourceModel) GetId() types.String {
	return r.Id
}

func (r *NotificationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_notification"
}

func (r *NotificationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Notification resource",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the notification",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the notification",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"active": schema.BoolAttribute{
				MarkdownDescription: "Boolean flag that sets the Notification as active or inactive. Only active Notifications are sent when triggered by the Event they are based on.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"always_fire_event": schema.BoolAttribute{
				MarkdownDescription: "A Boolean flag indicating whether the Notification is always triggered, regardless of other conditions and omitting reference to any calculation. This means the Notification will be triggered simply by the Event it is based on occurring and with no further conditions having to be met.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"calculation": schema.StringAttribute{
				MarkdownDescription: "A logical expression that that is evaluated to a Boolean. If it evaluates as True, a Notification for the Event is created and sent to the configured destination. Calculations can reference numeric, string, and boolean Event fields.",
				Optional:            true,
			},
			"code": schema.StringAttribute{
				MarkdownDescription: "The short code for the Notification.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"event_name": schema.StringAttribute{
				MarkdownDescription: "The name of the Event that triggers the Notification.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Notification identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"version": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Notification version",
			},
		},
	}
}

func (r *NotificationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *NotificationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	genericCreate(ctx, req, resp, r.client, "/notifications/configurations", "notification", r.read, r.write)
}

func (r *NotificationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	genericRead(ctx, req, resp, r.client, "/notifications/configurations", "notification", r.read)
}

func (r *NotificationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	genericUpdate(ctx, req, resp, r.client, "/notifications/configurations", "notification", r.read, r.write)
}

func (r *NotificationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	genericDelete[NotificationResourceModel](ctx, req, resp, r.client, "/notifications/configurations", "notification")
}

func (r *NotificationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *NotificationResource) read(ctx context.Context, data *NotificationResourceModel, restData map[string]any, diagnostics *diag.Diagnostics) {
	m := &mapper{
		ctx:         ctx,
		diagnostics: diagnostics,
		v:           restData,
	}
	m.to("id", &data.Id)
	m.to("version", &data.Version)
	m.to("name", &data.Name)
	m.to("description", &data.Description)
	m.to("active", &data.Active)
	m.to("always_fire_event", &data.AlwaysFireEvent)
	m.to("calculation", &data.Calculation)
	m.to("code", &data.Code)
	m.to("event_name", &data.EventName)
}

func (r *NotificationResource) write(ctx context.Context, data *NotificationResourceModel, restData map[string]any, diagnostics *diag.Diagnostics) {
	m := &mapper{
		ctx:         ctx,
		diagnostics: diagnostics,
		v:           restData,
	}
	m.from(data.Id, "id")
	m.from(data.Version, "version")
	m.from(data.Name, "name")
	m.from(data.Description, "description")
	m.from(data.Active, "active")
	m.from(data.AlwaysFireEvent, "alwaysFireEvent")
	m.from(data.Calculation, "calculation")
	m.from(data.Code, "code")
	m.from(data.EventName, "eventName")
}
