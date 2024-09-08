// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"reflect"

	"github.com/davidjspooner/dsflow/pkg/job"
	"github.com/davidjspooner/terraform-provider-kubernetes/internal/kresource"

	"github.com/davidjspooner/dsvalue/pkg/path"
	"github.com/davidjspooner/dsvalue/pkg/reflected"
	"github.com/davidjspooner/dsvalue/pkg/value"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ResourceResource{}
var _ resource.ResourceWithImportState = &ResourceResource{}

func NewManifestResource() resource.Resource {
	return &ResourceResource{}
}

// ResourceResource defines the resource implementation.
type ResourceResource struct {
	provider *KubernetesProvider
}

// ManifestResourceModel describes the resource data model.
type ManifestResourceModel struct {
	Manifest types.String          `tfsdk:"manifest"`
	Retry    *kresource.RetryModel `tfsdk:"retry"`
}

func (r *ResourceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource"
}

func (r *ResourceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Example resource",

		Attributes: map[string]schema.Attribute{
			"manifest": schema.StringAttribute{
				MarkdownDescription: "Manifest to apply",
				Optional:            true,
			},
			"retry": kresource.RetryModelSchema(),
		},
	}
}

func (r *ResourceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ResourceResource) newCrudHelper(retryModel *kresource.RetryModel) (*kresource.CrudHelper, error) {
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

func (r *ResourceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ManifestResourceModel

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
	err = helper.Plan.ParseAndAttach(manifestStr)
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

func (r *ResourceResource) diff(left, right interface{}) ([]string, error) {
	var diffs []string
	leftRoot, err := reflected.NewReflectedObject(reflect.ValueOf(left), nil)
	if err != nil {
		return nil, err
	}
	rightRoot, err := reflected.NewReflectedObject(reflect.ValueOf(right), nil)
	if err != nil {
		return nil, err
	}
	err = path.Diff(leftRoot, rightRoot, func(p path.Path, left, right value.Value) error {
		pathString := p.String()
		diffs = append(diffs, pathString)
		return nil
	})
	return diffs, err
}

func (r *ResourceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ManifestResourceModel

	// Read Terraform prior state data into the model
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	manifestStr := state.Manifest.ValueString()

	stateResource, err := kresource.ParseResourceYaml(manifestStr)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read resource tfstate", err.Error())
		return
	}

	actualUnstructured, err := r.provider.Shared.Get(ctx, &stateResource.Key)

	if err != nil {
		if kresource.ErrorIsNotFound(err) {
			//ok so we have to update the state to say it is stale
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError("Failed to fetch resource", err.Error())
		}
	} else {
		//compare state with current
		diffs, err := r.diff(stateResource.Unstructured.Object, actualUnstructured.Object)
		for _, diff := range diffs {
			resp.Diagnostics.AddWarning(fmt.Sprintf("Read.Diff: %s", diff), "")
		}
		if err != nil {
			resp.Diagnostics.AddError("Failed to diff state and actual", err.Error())
			return
		} else {
			s, err := kresource.FormatYaml(*actualUnstructured)
			if err != nil {
				resp.Diagnostics.AddError("Failed to format actual", err.Error())
				return
			}
			state.Manifest = types.StringValue(s)
		}
	}
}

func (r *ResourceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ManifestResourceModel
	var state ManifestResourceModel

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

	err = helper.State.ParseAndAttach(state.Manifest.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse resource tfstate", err.Error())
		return
	}
	err = helper.Plan.ParseAndAttach(plan.Manifest.ValueString())
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

func (r *ResourceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ManifestResourceModel

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
	err = helper.State.ParseAndAttach(manifestStr)
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

func (r *ResourceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.AddError("Import not supported", "This resource does not support import")
}
