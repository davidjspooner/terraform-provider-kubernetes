// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/davidjspooner/terraform-provider-kubernetes/internal/job"
	"github.com/davidjspooner/terraform-provider-kubernetes/internal/kresource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &Namespace{}
var _ resource.ResourceWithImportState = &Namespace{}

func NewNamespace() resource.Resource {
	return &Namespace{
		prometheusTypeNameSuffix: "_namespace",
	}
}

// Namespace defines the resource implementation.
type Namespace struct {
	provider                 *KubernetesProvider
	prometheusTypeNameSuffix string
}

// NamespaceModel describes the resource data model.
type NamespaceModel struct {
	MetaData kresource.MetaData `tfsdk:"metadata"`

	Retry *job.RetryModel `tfsdk:"retry"`
	OutputMetadata
}

func (nsm *NamespaceModel) Manifest() (unstructured.Unstructured, error) {
	var manifest unstructured.Unstructured
	manifest.SetUnstructuredContent(map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Namespace",
		"metadata": map[string]interface{}{
			"name":        nsm.MetaData.Name,
			"namespace":   nsm.MetaData.Namespace,
			"labels":      nsm.MetaData.Labels,
			"annotations": nsm.MetaData.Annotations,
		},
	})
	return manifest, nil
}

func (r *Namespace) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.prometheusTypeNameSuffix
}

func (r *Namespace) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A Kubernetes Namespace resource. This resource manages the lifecycle of a Kubernetes namespace.",
		Attributes: map[string]schema.Attribute{
			"retry": job.RetryModelSchema(),
			"resource_version": schema.StringAttribute{
				MarkdownDescription: "The resource version.",
				Computed:            true,
			},
			"uid": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the resource.",
				Computed:            true,
			},
			"generation": schema.Int64Attribute{
				MarkdownDescription: "The generation of the resource.",
				Computed:            true,
			},
		},
		Blocks: map[string]schema.Block{
			"metadata": LongMetadataSchemaBlock(),
		},
	}
}

func (r *Namespace) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *Namespace) newCrudHelper(retryModel *job.RetryModel) (*CrudHelper, error) {
	helper := &CrudHelper{
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

func (r *Namespace) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan NamespaceModel

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

	helper.Plan, err = plan.Manifest()
	if err != nil {
		resp.Diagnostics.AddError("Parsing manifest", err.Error())
		return
	}

	err = helper.CreateFromPlan(ctx, &plan.OutputMetadata)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create resource", err.Error())
		return
	}

	// Save plan into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *Namespace) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state NamespaceModel

	// Read Terraform prior state data into the model
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	helper, err := r.newCrudHelper(nil)
	if err != nil {
		resp.Diagnostics.AddError("Initializing", err.Error())
		return
	}

	helper.State, err = state.Manifest()
	if err != nil {
		resp.Diagnostics.AddError("Failed to read resource tfstate", err.Error())
		return
	}

	changed, err := helper.ReadActual(ctx, &state.OutputMetadata)
	if err != nil {
		if kresource.ErrorIsNotFound(err) {
			//ok so we have to update the state to say it is stale
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError("Failed to fetch resource", err.Error())
		}
		return
	}

	// key := kresource.GetKey(stateResource)
	// actualUnstructured, err := r.provider.Shared.Get(ctx, key)
	//
	//	if err != nil {
	//		if kresource.ErrorIsNotFound(err) {
	//			//ok so we have to update the state to say it is stale
	//			resp.State.RemoveResource(ctx)
	//		} else {
	//			resp.Diagnostics.AddError("Failed to fetch resource", err.Error())
	//		}
	//	} else {
	//
	//		//compare state with current
	//		diffs, err := kresource.DiffResources(stateResource.Object, actualUnstructured.Object)
	//		for _, diff := range diffs {
	//			resp.Diagnostics.AddWarning(fmt.Sprintf("Read.Diff: %s", diff), "")
	//		}
	//		if err != nil {
	//			resp.Diagnostics.AddError("Failed to diff state and actual", err.Error())
	//			return
	//		} else {
	//			s, err := kresource.FormatYaml(actualUnstructured)
	//			if err != nil {
	//				resp.Diagnostics.AddError("Failed to format actual", err.Error())
	//				return
	//			}
	//			_ = s
	//		}
	//	}
	if changed {
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	}
}

func (r *Namespace) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan NamespaceModel
	var state NamespaceModel

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

	helper.State, err = state.Manifest()
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse resource tfstate", err.Error())
		return
	}
	helper.Plan, err = plan.Manifest()
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse resource plan", err.Error())
		return
	}

	err = helper.Update(ctx, &plan.OutputMetadata)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update resource", err.Error())
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *Namespace) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state NamespaceModel

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

	helper.State, err = state.Manifest()
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

func (r *Namespace) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.AddError("Import not supported", "This resource does not support import")
}
