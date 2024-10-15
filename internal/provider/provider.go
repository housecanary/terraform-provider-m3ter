// Copyright (c) HouseCanary, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	"golang.org/x/time/rate"
)

// Ensure M3terProvider satisfies various provider interfaces.
var _ provider.Provider = &M3terProvider{}
var _ provider.ProviderWithFunctions = &M3terProvider{}

// M3terProvider defines the provider implementation.
type M3terProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// M3terProviderModel describes the provider data model.
type M3terProviderModel struct {
	OrganizationID types.String `tfsdk:"organization_id"`
	AccessKey      types.String `tfsdk:"access_key"`
	SecretKey      types.String `tfsdk:"secret_key"`
}

func (p *M3terProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "m3ter"
	resp.Version = p.version
}

func (p *M3terProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"organization_id": schema.StringAttribute{
				MarkdownDescription: "M3ter organization ID.",
				Optional:            true,
			},
			"access_key": schema.StringAttribute{
				MarkdownDescription: "M3ter access key.",
				Optional:            true,
			},
			"secret_key": schema.StringAttribute{
				MarkdownDescription: "M3ter secret key.",
				Optional:            true,
				Sensitive:           true,
			},
		},
	}
}

func (p *M3terProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data M3terProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.OrganizationID.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("organization_id"),
			"Unknown M3ter Organization ID",
			"The provider cannot create the M3ter API client as there is an unknown configuration value for the M3ter Organization ID. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the M3TER_ORGANIZATION_ID environment variable.",
		)
	}

	if data.AccessKey.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("access_key"),
			"Unknown M3ter Access Key",
			"The provider cannot create the M3ter API client as there is an unknown configuration value for the M3ter Access Key. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the M3TER_ACCESS_KEY environment variable.",
		)
	}

	if data.SecretKey.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("organization_id"),
			"Unknown M3ter Secret Key",
			"The provider cannot create the M3ter API client as there is an unknown configuration value for the M3ter Secret Key. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the M3TER_SECRET_KEY environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	organizationID := os.Getenv("M3TER_ORGANIZATION_ID")
	accessKey := os.Getenv("M3TER_ACCESS_KEY")
	secretKey := os.Getenv("M3TER_SECRET_KEY")

	if !data.OrganizationID.IsNull() {
		organizationID = data.OrganizationID.ValueString()
	}

	if !data.AccessKey.IsNull() {
		accessKey = data.AccessKey.ValueString()
	}

	if !data.SecretKey.IsNull() {
		secretKey = data.SecretKey.ValueString()
	}

	if organizationID == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("organization_id"),
			"Missing M3ter Organization ID",
			"The provider cannot create the M3ter API client as there is no configuration value for the M3ter Organization ID. "+
				"Set the value statically in the configuration or use the M3TER_ORGANIZATION_ID environment variable.",
		)
	}

	if accessKey == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("access_key"),
			"Missing M3ter Access Key",
			"The provider cannot create the M3ter API client as there is no configuration value for the M3ter Access Key. "+
				"Set the value statically in the configuration or use the M3TER_ACCESS_KEY environment variable.",
		)
	}

	if secretKey == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("secret_key"),
			"Missing M3ter Secret Key",
			"The provider cannot create the M3ter API client as there is no configuration value for the M3ter Secret Key. "+
				"Set the value statically in the configuration or use the M3TER_SECRET_KEY environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	cnf := clientcredentials.Config{
		ClientID:     accessKey,
		ClientSecret: secretKey,
		TokenURL:     "https://api.m3ter.com/oauth/token",
		AuthStyle:    oauth2.AuthStyleInHeader,
	}

	client := &m3terClient{
		organizationID: organizationID,
		client:         cnf.Client(context.Background()),
		limit:          rate.NewLimiter(rate.Limit(10), 1),
	}
	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *M3terProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewIntegrationConfigurationResource,
		NewNotificationResource,
		NewScheduledEventConfigurationResource,
		NewWebhookDestinationResource,
		NewOrganizationConfigResource,
		NewProductResource,
		NewPricingResource,
		NewPlanTemplateResource,
		NewPlanResource,
		NewPlanGroupResource,
		NewPlanGroupLinkResource,
		NewAggregationResource,
		NewMeterResource,
	}
}

func (p *M3terProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewProductDataSource,
		NewAggregationDataSource,
	}
}

func (p *M3terProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &M3terProvider{
			version: version,
		}
	}
}
