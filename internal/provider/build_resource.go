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

	"chainguard.dev/melange/pkg/build"
	"github.com/chainguard-dev/terraform-provider-apko/reflect"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"golang.org/x/sync/errgroup"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &BuildResource{}
var _ resource.ResourceWithImportState = &BuildResource{}

func NewBuildResource() resource.Resource {
	return &BuildResource{}
}

// BuildResource defines the resource implementation.
type BuildResource struct {
	popts ProviderOpts
}

// BuildResourceModel describes the resource data model.
type BuildResourceModel struct {
	Config         types.Object `tfsdk:"config"`
	ConfigContents types.String `tfsdk:"config_contents"`
	Id             types.String `tfsdk:"id"`
}

func (r *BuildResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_build"
}

func (r *BuildResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Example resource",

		Attributes: map[string]schema.Attribute{
			"config": schema.ObjectAttribute{
				MarkdownDescription: "Parsed melange config",
				Required:            true,
				AttributeTypes:      configSchema.AttrTypes,
			},
			"config_contents": schema.StringAttribute{
				MarkdownDescription: "The raw contents of the melange configuration.",
				Required:            true,
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Identifier of the resource",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *BuildResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	popts, ok := req.ProviderData.(*ProviderOpts)
	if !ok || popts == nil {
		resp.Diagnostics.AddError("Client Error", "invalid provider data")
		return
	}
	r.popts = *popts
}

func (r *BuildResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data BuildResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.doBuild(ctx, data); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	// ID is the sha256 of the JSON-serialized input config,
	// to ensure the resource is updated if the changes.
	b, err := json.Marshal(data.Config.String())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	data.Id = types.StringValue(fmt.Sprintf("%x", sha256.Sum256(b)))

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BuildResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data BuildResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// ID is the sha256 of the JSON-serialized input config,
	// to ensure the resource is updated if the changes.
	b, err := json.Marshal(data.Config.String())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	data.Id = types.StringValue(fmt.Sprintf("%x", sha256.Sum256(b)))

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BuildResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data BuildResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.doBuild(ctx, data); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	// ID is the sha256 of the JSON-serialized input config,
	// to ensure the resource is updated if the changes.
	b, err := json.Marshal(data.Config.String())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	data.Id = types.StringValue(fmt.Sprintf("%x", sha256.Sum256(b)))

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BuildResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data BuildResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Nothing to delete.
}

func (r *BuildResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *BuildResource) doBuild(ctx context.Context, data BuildResourceModel) error {
	var cfg Configuration
	if diags := reflect.AssignValue(data.Config, &cfg); diags.HasError() {
		return fmt.Errorf("assigning value: %v", diags.Errors())
	}

	// TODO: support force-overwrite, so you don't have to rm the package or bump the epoch while testing locally.

	var bcs []*build.Build
	for _, arch := range cfg.Environment.Archs {
		// See if we already have the package built.
		apk := fmt.Sprintf("%s-%s-r%d.apk", cfg.Package.Name, cfg.Package.Version, cfg.Package.Epoch)
		apkPath := filepath.Join("./packages", arch.ToAPK(), apk)
		if _, err := os.Stat(apkPath); err == nil {
			tflog.Trace(ctx, fmt.Sprintf("skipping %s, already built", apkPath))
			continue
		}

		// TODO: --dir
		sdir := filepath.Join(r.popts.dir, cfg.Package.Name)
		if _, err := os.Stat(sdir); os.IsNotExist(err) {
			if err := os.MkdirAll(sdir, os.ModePerm); err != nil {
				return fmt.Errorf("creating source directory %s: %v", sdir, err)
			}
		} else if err != nil {
			return fmt.Errorf("creating source directory: %v", err)
		}

		// Write the config to a temp file.
		// TODO(jason): This is kind of gross, but Melange's build API requires a file path.
		tmp, err := os.CreateTemp("", fmt.Sprintf("%s-*.yaml", cfg.Package.Name))
		if err != nil {
			return fmt.Errorf("creating temporary file: %v", err)
		}
		if err := os.WriteFile(tmp.Name(), []byte(data.ConfigContents.ValueString()), 0644); err != nil {
			return fmt.Errorf("writing config to temporary file: %v", err)
		}

		tflog.Trace(ctx, fmt.Sprintf("will build %s for %s", cfg.Package.Name, arch))
		bc, err := build.New(ctx,
			build.WithArch(arch),
			build.WithConfig(tmp.Name()),
			build.WithPipelineDir(filepath.Join(r.popts.dir, "pipelines")),
			//build.WithEnvFile(filepath.Join(p.opts.dir, fmt.Sprintf("build-%s.env", arch)), // TODO: ignore if it doesn't exist.
			build.WithOutDir(filepath.Join(r.popts.dir, "packages")),
			//build.WithSigningKey(filepath.Join(r.popts.dir, "local-melange.rsa")), // TODO: ignore if it doesn't exist.
			build.WithRunner("docker"), // TODO
			//build.WithNamespace("wolfi"), // TODO
			build.WithLogPolicy([]string{"builtin:stderr"}), // TODO: log dir instead, TF will swallow stderr
			build.WithSourceDir(sdir),
			build.WithGenerateIndex(true),
		)
		if err != nil {
			return fmt.Errorf("building %s for %s: %w", cfg.Package.Name, arch, err)
		}
		bcs = append(bcs, bc)
	}
	var errg errgroup.Group
	for _, bc := range bcs {
		bc := bc
		errg.Go(func() error {
			return bc.BuildPackage(ctx)
		})
	}
	if err := errg.Wait(); err != nil {
		return err
	}
	return nil
}
