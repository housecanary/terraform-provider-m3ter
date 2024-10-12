// Copyright (c) HouseCanary, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &PlanGroupLinkResource{}
var _ resource.ResourceWithImportState = &PlanGroupLinkResource{}

func NewPlanGroupLinkResource() resource.Resource {
	return &PlanGroupLinkResource{}
}

// PlanGroupLinkResource defines the resource implementation.
type PlanGroupLinkResource struct {
	client *m3terClient
}

// PlanGroupLinkResourceModel describes the resource data model.
type PlanGroupLinkResourceModel struct {
	PlanGroupId types.String `tfsdk:"plan_group_id"`
	PlanId      types.String `tfsdk:"plan_id"`
	Id          types.String `tfsdk:"id"`
	Version     types.Int64  `tfsdk:"version"`
}

func (r *PlanGroupLinkResourceModel) GetId() types.String {
	return r.Id
}

func (r *PlanGroupLinkResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_plan_group_link"
}

func (r *PlanGroupLinkResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "PlanGroupLink resource",

		Attributes: map[string]schema.Attribute{
			"plan_group_id": schema.StringAttribute{
				MarkdownDescription: "PlanGroupLink plan group identifier",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"plan_id": schema.StringAttribute{
				MarkdownDescription: "PlanGroupLink plan identifier",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "PlanGroupLink identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"version": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "PlanGroupLink version",
			},
		},
	}
}

func (r *PlanGroupLinkResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *PlanGroupLinkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	genericCreate[PlanGroupLinkResourceModel](ctx, req, resp, r.client, "/plangrouplinks", "plan group link", r.read, r.write)
}

func (r *PlanGroupLinkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	genericRead[PlanGroupLinkResourceModel](ctx, req, resp, r.client, "/plangrouplinks", "plan group link", r.read)
}

func (r *PlanGroupLinkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	genericUpdate[PlanGroupLinkResourceModel](ctx, req, resp, r.client, "/plangrouplinks", "plan group link", r.read, r.write)
}

func (r *PlanGroupLinkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	genericDelete[PlanGroupLinkResourceModel](ctx, req, resp, r.client, "/plangrouplinks", "plan group link")
}

func (r *PlanGroupLinkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *PlanGroupLinkResource) read(ctx context.Context, data *PlanGroupLinkResourceModel, restData map[string]any, diagnostics *diag.Diagnostics) {
	m := &mapper{
		ctx:         ctx,
		diagnostics: diagnostics,
		v:           restData,
	}
	m.to("id", &data.Id)
	m.to("version", &data.Version)
	m.to("planGroupId", &data.PlanGroupId)
	m.to("planId", &data.PlanId)
}

func (r *PlanGroupLinkResource) write(ctx context.Context, data *PlanGroupLinkResourceModel, restData map[string]any, diagnostics *diag.Diagnostics) {
	m := &mapper{
		ctx:         ctx,
		diagnostics: diagnostics,
		v:           restData,
	}
	m.from(data.Id, "id")
	m.from(data.Version, "version")
	m.from(data.PlanGroupId, "planGroupId")
	m.from(data.PlanId, "planId")
}
