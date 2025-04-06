// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	"github.com/davidjspooner/terraform-provider-kubernetes/internal/job"
	"github.com/davidjspooner/terraform-provider-kubernetes/internal/kresource"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure ScaffoldingProvider satisfies various provider interfaces.
var _ provider.Provider = &KubernetesProvider{}
var _ provider.ProviderWithFunctions = &KubernetesProvider{}

// KubernetesProvider defines the provider implementation.
type KubernetesProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.

	version string

	Shared       kresource.Shared
	DefaultRetry *job.RetryHelper
}

// KubernetesProviderModel describes the provider data model.
type KubernetesProviderModel struct {
	ConfigPaths           []types.String  `tfsdk:"config_paths"`
	ConfigContext         types.String    `tfsdk:"config_context"`
	ConfigContextAuthInfo types.String    `tfsdk:"config_context_auth_info"`
	ConfigContextCluster  types.String    `tfsdk:"config_context_cluster"`
	Namespace             types.String    `tfsdk:"namespace"`
	Retry                 *job.RetryModel `tfsdk:"retry"`
}

func (p *KubernetesProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "kubernetes"
	resp.Version = p.version
}

func (p *KubernetesProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"config_paths": schema.ListAttribute{
				ElementType: types.StringType,
				Description: "A list of paths to kube config files. Can be set with KUBE_CONFIG_PATHS environment variable. Default is [\"$KUBECONFIG\", \"$HOME/.kube/config\"]",
				Optional:    true,
			},
			"config_context_cluster": schema.StringAttribute{
				Description: "Override the current context cluster in kubeconfig",
				Optional:    true,
			},
			"config_context": schema.StringAttribute{
				Description: "Override the current context in kubeconfig",
				Optional:    true,
			},
			"config_context_auth_info": schema.StringAttribute{
				Description: "Override the current context auth info in kubeconfig",
				Optional:    true,
			},
			"namespace": schema.StringAttribute{
				Description: "Default namespace to use",
				Optional:    true,
			},
			"retry": job.RetryModelSchema(),
		},
	}
}

func (p *KubernetesProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data KubernetesProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	configPaths := make([]string, len(data.ConfigPaths))
	for i, path := range data.ConfigPaths {
		configPaths[i] = path.ValueString()
	}
	p.Shared.SetConfigPaths(configPaths)

	p.Shared.SetConfigContextAuthInfo(data.ConfigContextAuthInfo.ValueString())
	p.Shared.SetConfigContextCluster(data.ConfigContextCluster.ValueString())
	p.Shared.SetConfigContext(data.ConfigContext.ValueString())
	p.Shared.SetNamespace(data.Namespace.ValueString())

	if data.Retry != nil {
		var err error
		p.DefaultRetry, err = data.Retry.NewHelper(nil)
		if err != nil {
			resp.Diagnostics.AddError("Failed to create default retry helper", err.Error())
		}
	} else {
		p.DefaultRetry = &job.RetryHelper{}
	}

	resp.DataSourceData = p
	resp.ResourceData = p
}

func (p *KubernetesProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewGenericResource,
		NewResourceGet,
		NewClusterConfig,
		NewConfigMap,
		NewSecret,
		NewNamespace,
	}
}

func (p *KubernetesProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewFileManifests,
	}
}

func (p *KubernetesProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{
		//NewExampleFunction,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &KubernetesProvider{
			version: version,
		}
	}
}
