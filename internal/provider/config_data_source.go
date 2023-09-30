// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	apkotypes "chainguard.dev/apko/pkg/build/types"
	"chainguard.dev/melange/pkg/config"
	"github.com/chainguard-dev/terraform-provider-apko/reflect"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/util/sets"
)

var configSchema basetypes.ObjectType

func init() {
	sch, err := reflect.GenerateType(config.Configuration{})
	if err != nil {
		panic(err)
	}
	configSchema = sch.(basetypes.ObjectType)
}

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &ConfigDataSource{}

func NewConfigDataSource() datasource.DataSource {
	return &ConfigDataSource{}
}

// ConfigDataSource defines the data source implementation.
type ConfigDataSource struct {
	popts ProviderOpts
}

// ConfigDataSourceModel describes the data source data model.
type ConfigDataSourceModel struct {
	ConfigContents types.String `tfsdk:"config_contents"`
	Config         types.Object `tfsdk:"config"`
	Id             types.String `tfsdk:"id"`
}

func (d *ConfigDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_config"
}

func (d *ConfigDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Config data source",

		Attributes: map[string]schema.Attribute{
			"config_contents": schema.StringAttribute{
				MarkdownDescription: "The raw contents of the melange configuration.",
				Optional:            true,
			},
			"config": schema.ObjectAttribute{
				MarkdownDescription: "The parsed structure of the melange configuration.",
				Computed:            true,
				AttributeTypes:      configSchema.AttrTypes,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Config identifier",
				Computed:            true,
			},
		},
	}
}

func (d *ConfigDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	popts, ok := req.ProviderData.(*ProviderOpts)
	if !ok || popts == nil {
		resp.Diagnostics.AddError("Client Error", "invalid provider data")
		return
	}
	d.popts = *popts
}

func (d *ConfigDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ConfigDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var cfg config.Configuration
	if err := yaml.UnmarshalStrict([]byte(data.ConfigContents.ValueString()), &cfg); err != nil {
		resp.Diagnostics.AddError("Unable to parse melange configuration", err.Error())
		return
	}

	// Append any provider-specified repositories and keys, if specified.
	cfg.Environment.Contents.Repositories = sets.List(sets.New(cfg.Environment.Contents.Repositories...).Insert(d.popts.repositories...))
	cfg.Environment.Contents.Keyring = sets.List(sets.New(cfg.Environment.Contents.Keyring...).Insert(d.popts.keyring...))
	cfg.Environment.Archs = apkotypes.ParseArchitectures(d.popts.archs)

	ov, diags := reflect.GenerateValue(cfg)
	resp.Diagnostics = append(resp.Diagnostics, diags...)
	if diags.HasError() {
		return
	}
	data.Config = ov.(basetypes.ObjectValue)

	hash := sha256.Sum256([]byte(data.ConfigContents.ValueString()))
	data.Id = types.StringValue(hex.EncodeToString(hash[:]))

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
