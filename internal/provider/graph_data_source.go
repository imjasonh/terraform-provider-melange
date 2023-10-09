// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/wolfi-dev/wolfictl/pkg/dag"
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
	Dir  types.String `tfsdk:"dir"`
	Deps types.Map    `tfsdk:"deps"`
	Id   types.String `tfsdk:"id"`
}

func (d *GraphDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_graph"
}

func (d *GraphDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Graph data source",

		Attributes: map[string]schema.Attribute{
			"dir": schema.StringAttribute{
				MarkdownDescription: "Dir to load configs from (overrides provider dir)",
				Optional:            true,
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

	dir := d.popts.dir
	if data.Dir.ValueString() != "" {
		dir = data.Dir.ValueString()
	}

	tflog.Trace(ctx, fmt.Sprintf("dir: %s", dir))

	pkgs, err := dag.NewPackages(os.DirFS(dir), dir, filepath.Join(dir, "pipelines"))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("failed to load packages: %v", err))
		return
	}
	if len(pkgs.Packages()) == 0 {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("no packages found in %s", dir))
		return
	}

	g, err := dag.NewGraph(pkgs,
		dag.WithRepos(d.popts.repositories...),
		dag.WithKeys(d.popts.keyring...),
	)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("failed to build graph: %v", err))
		return
	}

	tflog.Trace(ctx, fmt.Sprintf("configs: %v", pkgs.PackageNames()))

	// Only include local packages
	g, err = g.Filter(dag.FilterLocal())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("failed to filter graph to local packages: %v", err))
		return
	}
	// Only return main packages (configs)
	g, err = g.Filter(dag.OnlyMainPackages(pkgs))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("failed to filter graph to configs: %v", err))
		return
	}
	m, err := g.Graph.AdjacencyMap()
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("failed to get graph adjacency map: %v", err))
		return
	}
	out := map[string]attr.Value{}
	for k, v := range m {
		k, _, _ := strings.Cut(k, ":")
		var vels []string
		for vv := range v {
			vv, _, _ := strings.Cut(vv, ":")
			vels = append(vels, vv)
		}
		sort.Strings(vels)
		var els []attr.Value
		for _, vv := range vels {
			els = append(els, basetypes.NewStringValue(vv))
		}
		out[k] = basetypes.NewListValueMust(basetypes.StringType{}, els)
	}
	data.Deps = basetypes.NewMapValueMust(basetypes.ListType{
		ElemType: basetypes.StringType{},
	}, out)

	// ID is the sha256 of the JSON-serialized dep graph,
	// to ensure the data source changes if any config changes in a way that changes the graph.
	b, err := json.Marshal(data.Deps)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", "failed to marshal configs")
		return
	}
	data.Id = types.StringValue(fmt.Sprintf("%x", sha256.Sum256(b)))

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
