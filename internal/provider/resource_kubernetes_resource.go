// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/davidjspooner/terraform-provider-kubernetes/internal/job"
	"github.com/davidjspooner/terraform-provider-kubernetes/internal/kresource"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &GenericResource{}
var _ resource.ResourceWithImportState = &GenericResource{}

func NewGenericResource() resource.Resource {
	return &GenericResource{}
}

// GenericResource defines the resource implementation.
type GenericResource struct {
	provider *KubernetesProvider
}

// GenericResourceModel describes the resource data model.
type GenericResourceModel struct {
	Manifest types.String    `tfsdk:"manifest"`
	Retry    *job.RetryModel `tfsdk:"retry"`
}

func (r *GenericResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource"
}

func (r *GenericResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Example resource",

		Attributes: map[string]schema.Attribute{
			"manifest": schema.StringAttribute{
				MarkdownDescription: "Manifest to apply",
				Optional:            true,
			},
			"retry": job.RetryModelSchema(),
		},
	}
}

func (r *GenericResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *GenericResource) newCrudHelper(retryModel *job.RetryModel) (*kresource.CrudHelper, error) {
	helper := &kresource.CrudHelper{
		Shared: &r.provider.Shared,
	}
	var err error
	if retryModel != nil {
		helper.RetryHelper, err = retryModel.NewHelper(r.provider.DefaultRetry)
		if err != nil {
			return nil, err
		}
	} else if r.provider.DefaultRetry != nil {
		helper.RetryHelper = r.provider.DefaultRetry
	} else {
		helper.RetryHelper = &job.RetryHelper{}
	}
	return helper, nil
}

func (r *GenericResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan GenericResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}
	helper, err := r.newCrudHelper(plan.Retry)
	if err != nil {
		resp.Diagnostics.AddError("Initializing", err.Error())
		return
	}

	manifestStr := plan.Manifest.ValueString()
	helper.Plan, err = kresource.ParseSingleYamlManifest(manifestStr)
	if err != nil {
		resp.Diagnostics.AddError("Parsing manifest", err.Error())
		return
	}

	err = helper.CreateFromPlan(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create resource", err.Error())
		return
	}

	// Save plan into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *GenericResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state GenericResourceModel

	// Read Terraform prior state data into the model
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	manifestStr := state.Manifest.ValueString()

	stateResources, err := kresource.ParseYamlManifestList(manifestStr)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read resource tfstate", err.Error())
		return
	}
	if len(stateResources) != 1 {
		resp.Diagnostics.AddError("Expected one resource in state", fmt.Sprintf("Found %d", len(stateResources)))
		return
	}
	stateResource := stateResources[0]

	key := kresource.GetKey(stateResource)
	actualUnstructured, err := r.provider.Shared.Get(ctx, key)

	if err != nil {
		if kresource.ErrorIsNotFound(err) {
			//ok so we have to update the state to say it is stale
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError("Failed to fetch resource", err.Error())
		}
	} else {
		//compare state with current
		diffs, err := kresource.DiffResources(stateResource.Object, actualUnstructured.Object)
		for _, diff := range diffs {
			resp.Diagnostics.AddWarning(fmt.Sprintf("Read.Diff: %s", diff), "")
		}
		if err != nil {
			resp.Diagnostics.AddError("Failed to diff state and actual", err.Error())
			return
		} else {
			s, err := kresource.FormatYaml(actualUnstructured)
			if err != nil {
				resp.Diagnostics.AddError("Failed to format actual", err.Error())
				return
			}
			state.Manifest = types.StringValue(s)
		}
	}
}

func (r *GenericResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan GenericResourceModel
	var state GenericResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}
	helper, err := r.newCrudHelper(plan.Retry)
	if err != nil {
		resp.Diagnostics.AddError("Initializing", err.Error())
		return
	}

	helper.State, err = kresource.ParseSingleYamlManifest(state.Manifest.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse resource tfstate", err.Error())
		return
	}
	helper.Plan, err = kresource.ParseSingleYamlManifest(plan.Manifest.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse resource plan", err.Error())
		return
	}

	err = helper.Update(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update resource", err.Error())
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *GenericResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state GenericResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}
	helper, err := r.newCrudHelper(state.Retry)
	if err != nil {
		resp.Diagnostics.AddError("Initializing", err.Error())
		return
	}

	manifestStr := state.Manifest.ValueString()
	helper.State, err = kresource.ParseSingleYamlManifest(manifestStr)
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse reource", err.Error())
		return
	}

	err = helper.DeleteState(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Destroying resource", err.Error())
		return
	}
	req.State.RemoveResource(ctx)
}

func (r *GenericResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.AddError("Import not supported", "This resource does not support import")
}
