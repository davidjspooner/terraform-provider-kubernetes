// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ClusterConfig{}
var _ resource.ResourceWithImportState = &ClusterConfig{}

func NewClusterConfig() resource.Resource {
	return &ClusterConfig{}
}

// ClusterConfig defines the resource implementation.
type ClusterConfig struct {
	provider *KubernetesProvider
}

// ClusterConfigModel describes the resource data model.
type ClusterConfigModel struct {
	Name             types.String `tfsdk:"name"`
	Server           types.String `tfsdk:"server"`
	TemplateFilename types.String `tfsdk:"template_filename"`
	TargetFilename   types.String `tfsdk:"target_filename"`
}

func (r *ClusterConfig) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster_config"
}

func (r *ClusterConfig) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Imports/Updates a Kubernetes cluster configuration file into the config used by the provider.",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Cluster name use in config",
				Optional:            true,
			},
			"server": schema.StringAttribute{
				MarkdownDescription: "Server to use in config ( eg https://localhost:16443 )",
				Optional:            true,
			},
			"template_filename": schema.StringAttribute{
				MarkdownDescription: "File to use as template for config",
				Required:            true,
			},
			"target_filename": schema.StringAttribute{
				MarkdownDescription: "Where to write config",
				Required:            true,
			},
		},
	}
}

func (r *ClusterConfig) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	var ok bool
	r.provider, ok = req.ProviderData.(*KubernetesProvider)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *KubernetesProvider, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}
}

func (r *ClusterConfig) createOrUpdate(ctx context.Context, data *ClusterConfigModel) error {
	pair := K8sConfigPair{}
	template_filename := data.TemplateFilename.ValueString()
	target_filename := data.TargetFilename.ValueString()
	pair.LoadConfigs(template_filename, target_filename)
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

func (r *ClusterConfig) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ClusterConfigModel

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

func (r *ClusterConfig) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	resp.State.RemoveResource(ctx) //always recreate just in case
}

func (r *ClusterConfig) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ClusterConfigModel

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

func (r *ClusterConfig) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ClusterConfigModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	pair := K8sConfigPair{}

	template_filename := data.TemplateFilename.ValueString()
	target_filename := data.TargetFilename.ValueString()
	pair.LoadConfigs(template_filename, target_filename)
	name := data.Name.ValueString()
	pair.RemoveClusterFromTarget(name)
	pair.Target.WriteToFile(target_filename)

	tflog.Info(ctx, "deleted a resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ClusterConfig) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
