// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &GraphDataSource{}

func NewGraphDataSource() datasource.DataSource {
	return &GraphDataSource{}
}

// GraphDataSource defines the data source implementation.
type GraphDataSource struct {
	popts ProviderOpts
}

// GraphDataSourceModel describes the data source data model.
type GraphDataSourceModel struct {
	Configs []types.Object `tfsdk:"configs"`
	Id      types.String   `tfsdk:"id"`
}

func (d *GraphDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_graph"
}

func (d *GraphDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Graph data source",

		Attributes: map[string]schema.Attribute{
			"configs": schema.ListAttribute{
				MarkdownDescription: "List of configs",
				Required:            true,
				ElementType: basetypes.ObjectType{
					AttrTypes: configSchema.AttrTypes,
				},
			},
			"deps": schema.MapAttribute{
				MarkdownDescription: "Map of dependencies: this -> [needs]",
				Computed:            true,
				ElementType: basetypes.ListType{
					ElemType: basetypes.StringType{},
				},
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Graph identifier",
				Computed:            true,
			},
		},
	}
}

func (d *GraphDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *GraphDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data GraphDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// ID is the sha256 of the JSON-serialized input configs,
	// to ensure the data source changes if any config changes.
	b, err := json.Marshal(data.Configs)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", "failed to marshal configs")
		return
	}
	data.Id = types.StringValue(fmt.Sprintf("%x", sha256.Sum256(b)))

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
