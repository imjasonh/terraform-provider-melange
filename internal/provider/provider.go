// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// Ensure Provider satisfies various provider interfaces.
var _ provider.Provider = &Provider{}

// Provider defines the provider implementation.
type Provider struct {
	version string

	repositories, keyring, archs []string
}

// ProviderModel describes the provider data model.
type ProviderModel struct {
	ExtraRepositories []string              `tfsdk:"extra_repositories"`
	ExtraKeyring      []string              `tfsdk:"extra_keyring"`
	DefaultArchs      []string              `tfsdk:"default_archs"`
	Dir               basetypes.StringValue `tfsdk:"dir"`
}

type ProviderOpts struct {
	repositories, keyring, archs []string
	dir                          string
}

func (p *Provider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "melange"
	resp.Version = p.version
}

func (p *Provider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"extra_repositories": schema.ListAttribute{
				Description: "Additional repositories to search for packages",
				Optional:    true,
				ElementType: basetypes.StringType{},
			},
			"extra_keyring": schema.ListAttribute{
				Description: "Additional keys to use for package verification",
				Optional:    true,
				ElementType: basetypes.StringType{},
			},
			"default_archs": schema.ListAttribute{
				Description: "Default architectures to build for",
				Optional:    true,
				ElementType: basetypes.StringType{},
			},
			"dir": schema.StringAttribute{
				Description: "Directory to use for building packages",
				Optional:    true,
			},
		},
	}
}

func (p *Provider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data ProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	opts := &ProviderOpts{
		// This is only for testing, so we can inject provider config
		repositories: append(p.repositories, data.ExtraRepositories...),
		keyring:      append(p.keyring, data.ExtraKeyring...),
		archs:        append(p.archs, data.DefaultArchs...),
		dir:          data.Dir.ValueString(),
	}

	// Make provider opts available to resources and data sources.
	resp.ResourceData = opts
	resp.DataSourceData = opts
}

func (p *Provider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewBuildResource,
	}
}

func (p *Provider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewConfigDataSource,
		NewGraphDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &Provider{
			version: version,
		}
	}
}
