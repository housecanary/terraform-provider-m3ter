// Copyright (c) HouseCanary, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/int32validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &OrganizationConfigResource{}
var _ resource.ResourceWithImportState = &OrganizationConfigResource{}

func NewOrganizationConfigResource() resource.Resource {
	return &OrganizationConfigResource{}
}

// OrganizationConfigResource defines the resource implementation.
type OrganizationConfigResource struct {
	client *m3terClient
}

// OrganizationConfigResourceModel describes the resource data model.
type OrganizationConfigResourceModel struct {
	Timezone                     types.String  `tfsdk:"timezone"`
	YearEpoch                    types.String  `tfsdk:"year_epoch"`
	MonthEpoch                   types.String  `tfsdk:"month_epoch"`
	WeekEpoch                    types.String  `tfsdk:"week_epoch"`
	DayEpoch                     types.String  `tfsdk:"day_epoch"`
	Currency                     types.String  `tfsdk:"currency"`
	CurrencyConversions          types.List    `tfsdk:"currency_conversions"`
	DaysBeforeBillDue            types.Int32   `tfsdk:"days_before_bill_due"`
	ScheduledBillInterval        types.Float64 `tfsdk:"scheduled_bill_interval"`
	StandingChargeBillInAdvance  types.Bool    `tfsdk:"standing_charge_bill_in_advance"`
	CommitmentFeeBillInAdvance   types.Bool    `tfsdk:"commitment_fee_bill_in_advance"`
	MinimumSpendBillInAdvance    types.Bool    `tfsdk:"minimum_spend_bill_in_advance"`
	ExternalInvoiceDate          types.String  `tfsdk:"external_invoice_date"`
	SuppressedEmptyBills         types.Bool    `tfsdk:"suppressed_empty_bills"`
	ConsolidateBills             types.Bool    `tfsdk:"consolidate_bills"`
	DefaultStatementDefinitionId types.String  `tfsdk:"default_statement_definition_id"`
	SequenceStartNumber          types.Int64   `tfsdk:"sequence_start_number"`
	AutoGenerateStatementMode    types.String  `tfsdk:"auto_generate_statement_mode"`
	CreditApplicationOrder       types.List    `tfsdk:"credit_application_order"`
	Id                           types.String  `tfsdk:"id"`
	Version                      types.Int64   `tfsdk:"version"`
}

var currencyConversionType = schema.NestedAttributeObject{
	Attributes: map[string]schema.Attribute{
		"from": schema.StringAttribute{
			MarkdownDescription: "Currency to convert from. For example: GBP.",
			Required:            true,
			Validators: []validator.String{
				stringvalidator.LengthAtLeast(1),
			},
		},
		"to": schema.StringAttribute{
			MarkdownDescription: "Currency to convert to. For example: USD.",
			Required:            true,
			Validators: []validator.String{
				stringvalidator.LengthAtLeast(1),
			},
		},
		"multiplier": schema.Float64Attribute{
			MarkdownDescription: "Conversion rate between currencies.",
			Required:            true,
			Validators: []validator.Float64{
				float64validator.AtLeast(0),
			},
		},
	},
}

func (r *OrganizationConfigResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_config"
}

func (r *OrganizationConfigResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Organization config resource",

		Attributes: map[string]schema.Attribute{
			"timezone": schema.StringAttribute{
				MarkdownDescription: "Specifies the time zone used for the generated Bills, ensuring alignment with the local time zone.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"year_epoch": schema.StringAttribute{
				MarkdownDescription: "Optional setting that defines the billing cycle date for Accounts that are billed yearly. Defines the date of the first Bill and then acts as reference for when subsequent Bills are created for the Account.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`\d{4}-\d{2}-\d{2}`), "must be in the format YYYY-MM-DD"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"month_epoch": schema.StringAttribute{
				MarkdownDescription: "Optional setting that defines the billing cycle date for Accounts that are billed monthly. Defines the date of the first Bill and then acts as reference for when subsequent Bills are created for the Account.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`\d{4}-\d{2}-\d{2}`), "must be in the format YYYY-MM-DD"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"week_epoch": schema.StringAttribute{
				MarkdownDescription: "Optional setting that defines the billing cycle date for Accounts that are billed weekly. Defines the date of the first Bill and then acts as reference for when subsequent Bills are created for the Account.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`\d{4}-\d{2}-\d{2}`), "must be in the format YYYY-MM-DD"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"day_epoch": schema.StringAttribute{
				MarkdownDescription: "Optional setting that defines the billing cycle date for Accounts that are billed daily. Defines the date of the first Bill and then acts as reference for when subsequent Bills are created for the Account.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`\d{4}-\d{2}-\d{2}`), "must be in the format YYYY-MM-DD"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"currency": schema.StringAttribute{
				MarkdownDescription: "The currency code for the Organization. For example: USD, GBP, or EUR.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"currency_conversions": schema.ListNestedAttribute{
				MarkdownDescription: "Define currency conversion rates from pricing currency to billing currency",
				Optional:            true,
				Computed:            true,
				NestedObject:        currencyConversionType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"days_before_bill_due": schema.Int32Attribute{
				MarkdownDescription: "The number of days after the Bill generation date that you want to show on Bills as the due date.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.Int32{
					int32validator.AtLeast(0),
				},
				PlanModifiers: []planmodifier.Int32{
					int32planmodifier.UseStateForUnknown(),
				},
			},
			"scheduled_bill_interval": schema.Float64Attribute{
				MarkdownDescription: "Sets the required interval for updating bills.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.Float64{
					float64validator.OneOf(
						0.25,
						0.5,
						1,
						2,
						3,
						4,
						6,
						8,
						0,
					),
				},
				PlanModifiers: []planmodifier.Float64{
					float64planmodifier.UseStateForUnknown(),
				},
			},
			"standing_charge_bill_in_advance": schema.BoolAttribute{
				MarkdownDescription: "Boolean flag that sets the Standing Charge as a bill in advance.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"commitment_fee_bill_in_advance": schema.BoolAttribute{
				MarkdownDescription: "Boolean flag that sets the Commitment Fee as a bill in advance.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"minimum_spend_bill_in_advance": schema.BoolAttribute{
				MarkdownDescription: "Boolean flag that sets the Minimum Spend as a bill in advance.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"external_invoice_date": schema.StringAttribute{
				MarkdownDescription: "The date on which the external invoice is generated.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("FIRST_DAY_OF_NEXT_PERIOD", "LAST_DAY_OF_CURRENT_PERIOD"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"suppressed_empty_bills": schema.BoolAttribute{
				MarkdownDescription: "Boolean flag that suppresses the generation of empty Bills.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"consolidate_bills": schema.BoolAttribute{
				MarkdownDescription: "Boolean flag that consolidates Bills.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"default_statement_definition_id": schema.StringAttribute{
				MarkdownDescription: "The default Statement Definition ID.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"sequence_start_number": schema.Int64Attribute{
				MarkdownDescription: "The sequence start number.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"auto_generate_statement_mode": schema.StringAttribute{
				MarkdownDescription: "The auto generate statement mode.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("JSON_AND_CSV", "JSON", "NONE"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"credit_application_order": schema.ListAttribute{
				MarkdownDescription: "The credit application order.",
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				Validators: []validator.List{
					listvalidator.ValueStringsAre(
						stringvalidator.OneOf("PREPAYMENT", "BALANCE"),
					),
				},
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Organization identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"version": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Organization version",
			},
		},
	}
}

func (r *OrganizationConfigResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *OrganizationConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data OrganizationConfigResourceModel

	var orgData map[string]any
	err := r.client.execute(ctx, "GET", "/organizationconfig", nil, nil, &orgData)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read organization, got error: %s", err))
		return
	}

	r.read(ctx, orgData, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	r.update(ctx, orgData, &data, &resp.Diagnostics)

	if resp.Diagnostics.HasError() {
		return
	}

	var updatedOrgData map[string]any
	err = r.client.execute(ctx, "PUT", "/organizationconfig", nil, orgData, &updatedOrgData)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update organization, got error: %s", err))
		return
	}

	r.read(ctx, updatedOrgData, &data, &resp.Diagnostics)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *OrganizationConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data OrganizationConfigResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var orgData map[string]any
	err := r.client.execute(ctx, "GET", "/organizationconfig", nil, nil, &orgData)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read organization, got error: %s", err))
		return
	}

	r.read(ctx, orgData, &data, &resp.Diagnostics)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *OrganizationConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data OrganizationConfigResourceModel

	var orgData map[string]any
	err := r.client.execute(ctx, "GET", "/organizationconfig", nil, nil, &orgData)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read organization, got error: %s", err))
		return
	}

	r.read(ctx, orgData, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	r.update(ctx, orgData, &data, &resp.Diagnostics)

	if resp.Diagnostics.HasError() {
		return
	}

	var updatedOrgData map[string]any
	err = r.client.execute(ctx, "PUT", "/organizationconfig", nil, orgData, &updatedOrgData)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update organization, got error: %s", err))
		return
	}

	r.read(ctx, updatedOrgData, &data, &resp.Diagnostics)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

}

func (r *OrganizationConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// No need to do anything here - this just removes the org settings from being managed by Terraform
}

func (r *OrganizationConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *OrganizationConfigResource) update(ctx context.Context, orgModel map[string]any, resourceModel *OrganizationConfigResourceModel, diagnostics *diag.Diagnostics) {
	m := &mapper{
		ctx:         ctx,
		diagnostics: diagnostics,
		v:           orgModel,
	}

	m.from(resourceModel.Version, "version")
	m.from(resourceModel.Timezone, "timezone")
	m.from(resourceModel.YearEpoch, "yearEpoch")
	m.from(resourceModel.MonthEpoch, "monthEpoch")
	m.from(resourceModel.WeekEpoch, "weekEpoch")
	m.from(resourceModel.DayEpoch, "dayEpoch")
	m.from(resourceModel.Currency, "currency")
	m.from(resourceModel.DaysBeforeBillDue, "daysBeforeBillDue")
	m.from(resourceModel.ScheduledBillInterval, "scheduledBillInterval")
	m.from(resourceModel.StandingChargeBillInAdvance, "standingChargeBillInAdvance")
	m.from(resourceModel.CommitmentFeeBillInAdvance, "commitmentFeeBillInAdvance")
	m.from(resourceModel.MinimumSpendBillInAdvance, "minimumSpendBillInAdvance")
	m.from(resourceModel.ExternalInvoiceDate, "externalInvoiceDate")
	m.from(resourceModel.SuppressedEmptyBills, "suppressedEmptyBills")
	m.from(resourceModel.ConsolidateBills, "consolidateBills")
	m.from(resourceModel.DefaultStatementDefinitionId, "defaultStatementDefinitionId")
	m.from(resourceModel.SequenceStartNumber, "sequenceStartNumber")
	m.from(resourceModel.AutoGenerateStatementMode, "autoGenerateStatementMode")

	m.listFrom(resourceModel.CreditApplicationOrder, "creditApplicationOrder", func(v attr.Value) (any, diag.Diagnostics) {
		if sv, ok := v.(types.String); ok {
			return sv.ValueString(), nil
		}

		return nil, diag.Diagnostics{diag.NewErrorDiagnostic("cannot map list element, expected string", "")}
	})

	m.listFrom(resourceModel.CurrencyConversions, "currencyConversions", func(v attr.Value) (any, diag.Diagnostics) {
		if ov, ok := v.(types.Object); ok {
			attrs := ov.Attributes()
			from, ok := attrs["from"].(types.String)
			if !ok {
				return nil, diag.Diagnostics{diag.NewErrorDiagnostic("cannot map list element, expected string", "")}
			}
			to, ok := attrs["to"].(types.String)
			if !ok {
				return nil, diag.Diagnostics{diag.NewErrorDiagnostic("cannot map list element, expected string", "")}
			}
			multiplier, ok := attrs["multiplier"].(types.Float64)
			if !ok {
				return nil, diag.Diagnostics{diag.NewErrorDiagnostic("cannot map list element, expected float", "")}
			}
			return map[string]any{
				"from":       from.ValueString(),
				"to":         to.ValueString(),
				"multiplier": multiplier.ValueFloat64(),
			}, nil
		}

		return nil, diag.Diagnostics{diag.NewErrorDiagnostic("cannot map list element, expected object", "")}
	})
}

func (r *OrganizationConfigResource) read(ctx context.Context, orgModel map[string]any, resourceModel *OrganizationConfigResourceModel, diagnostics *diag.Diagnostics) {
	m := &mapper{
		ctx:         ctx,
		diagnostics: diagnostics,
		v:           orgModel,
	}
	// convert the json data into the terraform model
	resourceModel.Id = types.StringValue(r.client.organizationID)
	m.to("version", &resourceModel.Version)
	m.to("timezone", &resourceModel.Timezone)
	m.to("yearEpoch", &resourceModel.YearEpoch)
	m.to("monthEpoch", &resourceModel.MonthEpoch)
	m.to("weekEpoch", &resourceModel.WeekEpoch)
	m.to("dayEpoch", &resourceModel.DayEpoch)
	m.to("currency", &resourceModel.Currency)
	m.to("daysBeforeBillDue", &resourceModel.DaysBeforeBillDue)
	m.to("scheduledBillInterval", &resourceModel.ScheduledBillInterval)
	m.to("standingChargeBillInAdvance", &resourceModel.StandingChargeBillInAdvance)
	m.to("commitmentFeeBillInAdvance", &resourceModel.CommitmentFeeBillInAdvance)
	m.to("minimumSpendBillInAdvance", &resourceModel.MinimumSpendBillInAdvance)
	m.to("externalInvoiceDate", &resourceModel.ExternalInvoiceDate)
	m.to("suppressedEmptyBills", &resourceModel.SuppressedEmptyBills)
	m.to("consolidateBills", &resourceModel.ConsolidateBills)
	m.to("defaultStatementDefinitionId", &resourceModel.DefaultStatementDefinitionId)
	m.to("sequenceStartNumber", &resourceModel.SequenceStartNumber)
	m.to("autoGenerateStatementMode", &resourceModel.AutoGenerateStatementMode)
	m.listTo("currencyConversions", &resourceModel.CurrencyConversions, currencyConversionType.Type(), func(v any) (attr.Value, diag.Diagnostics) {
		mv, ok := v.(map[string]any)

		if !ok {
			return nil, diag.Diagnostics{diag.NewErrorDiagnostic("cannot map list element, expected map", "")}
		}

		m := &mapper{
			ctx:         ctx,
			diagnostics: diagnostics,
			v:           mv,
		}
		var from types.String
		var to types.String
		var multiplier types.Float64

		m.to("from", &from)
		m.to("to", &to)
		m.to("multiplier", &multiplier)

		return types.ObjectValue(map[string]attr.Type{
			"from":       types.StringType,
			"to":         types.StringType,
			"multiplier": types.Float64Type,
		}, map[string]attr.Value{
			"from":       from,
			"to":         to,
			"multiplier": multiplier,
		})
	})
	m.listTo("creditApplicationOrder", &resourceModel.CreditApplicationOrder, types.StringType, func(v any) (attr.Value, diag.Diagnostics) {
		mv, ok := v.(string)

		if !ok {
			return nil, diag.Diagnostics{diag.NewErrorDiagnostic("cannot map list element, expected string", "")}
		}

		return types.StringValue(mv), nil
	})
}
