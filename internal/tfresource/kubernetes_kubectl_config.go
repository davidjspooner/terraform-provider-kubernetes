// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfresource

import (
	"context"
	"fmt"

	"github.com/davidjspooner/terraform-provider-kubernetes/internal/kresource"
	"github.com/davidjspooner/terraform-provider-kubernetes/internal/tfprovider"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &Config{}
var _ resource.ResourceWithImportState = &Config{}

func init() {
	// Register the resource with the provider.
	tfprovider.RegisterResource(func() resource.Resource {
		return &Config{
			tfTypeNameSuffix: "_kubectl_config",
		}
	})
}

// Config defines the resource implementation.
type Config struct {
	provider         *tfprovider.KubernetesResourceProvider
	tfTypeNameSuffix string
}

// ConfigModel describes the resource data model.
type ConfigModel struct {
	Name           types.String `tfsdk:"name"`
	Server         types.String `tfsdk:"server"`
	SourceFilename types.String `tfsdk:"source_filename"`
	TargetFilename types.String `tfsdk:"target_filename"`
}

func (r *Config) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.tfTypeNameSuffix
}

func (r *Config) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Merges a Kubernetes cluster configuration file into another configuration file.",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Cluster name use in config",
				Optional:            true,
			},
			"server": schema.StringAttribute{
				MarkdownDescription: "Server to use in config ( eg https://localhost:16443 )",
				Optional:            true,
			},
			"source_filename": schema.StringAttribute{
				MarkdownDescription: "File to use as source of config to merge into target",
				Required:            true,
			},
			"target_filename": schema.StringAttribute{
				MarkdownDescription: "File to merge config into",
				Required:            true,
			},
		},
	}
}

func (r *Config) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	var ok bool
	r.provider, ok = req.ProviderData.(*tfprovider.KubernetesResourceProvider)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *KubernetesProvider, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}
}

func (r *Config) createOrUpdate(ctx context.Context, data *ConfigModel) error {
	pair := kresource.K8sConfigPair{}
	source_filename := data.SourceFilename.ValueString()
	target_filename := data.TargetFilename.ValueString()
	pair.LoadConfigs(source_filename, target_filename)
	name := data.Name.ValueString()
	server := data.Server.ValueString()
	err := pair.UpdateTemplate(name, server)
	if err != nil {
		return err
	}
	err = pair.MergeTemplateIntoTarget()
	if err != nil {
		return err
	}
	err = pair.Target.WriteToFile(target_filename)
	if err != nil {
		return err
	}

	err = r.provider.Shared.ReloadConfig(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (r *Config) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ConfigModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.createOrUpdate(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create or update cluster config", err.Error())
	} else {
		tflog.Info(ctx, "created a resource")
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *Config) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	resp.State.RemoveResource(ctx) //always recreate just in case
}

func (r *Config) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ConfigModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	err := r.createOrUpdate(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create or update cluster config", err.Error())
	} else {
		tflog.Info(ctx, "created a resource")
	}
	if err != nil {
		resp.Diagnostics.AddError("Failed to reload kubeconfig", err.Error())
	}

	tflog.Info(ctx, "updated a resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *Config) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ConfigModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	pair := kresource.K8sConfigPair{}

	source_filename := data.SourceFilename.ValueString()
	target_filename := data.TargetFilename.ValueString()
	pair.LoadConfigs(source_filename, target_filename)
	name := data.Name.ValueString()
	pair.RemoveClusterFromTarget(name)
	pair.Target.WriteToFile(target_filename)

	tflog.Info(ctx, "deleted a resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *Config) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
