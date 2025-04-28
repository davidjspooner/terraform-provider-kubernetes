package tfprovider

import (
	"context"
	"sync"

	"github.com/davidjspooner/terraform-provider-kubernetes/internal/generic/job"
	"github.com/davidjspooner/terraform-provider-kubernetes/internal/generic/kube"
	"github.com/davidjspooner/terraform-provider-kubernetes/internal/terraform/tfparts"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure ScaffoldingProvider satisfies various provider interfaces.
var _ provider.Provider = &KubernetesResourceProvider{}
var _ provider.ProviderWithFunctions = &KubernetesResourceProvider{}

// KubernetesResourceProvider defines the provider implementation.
type KubernetesResourceProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.

	version  string
	typeName string

	Shared            kube.APIClientWrapper
	DefaultApiOptions *kube.APIClientOptions
}

// KubernetesProviderModel describes the provider data model.
type KubernetesProviderModel struct {
	ConfigPaths           []types.String `tfsdk:"config_paths"`
	ConfigContext         types.String   `tfsdk:"config_context"`
	ConfigContextAuthInfo types.String   `tfsdk:"config_context_auth_info"`
	ConfigContextCluster  types.String   `tfsdk:"config_context_cluster"`
	Namespace             types.String   `tfsdk:"namespace"`
	tfparts.APIOptionsModel
}

func (p *KubernetesResourceProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = p.typeName
	resp.Version = p.version
}

func (p *KubernetesResourceProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	attr := map[string]schema.Attribute{
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
	}
	resp.Schema = schema.Schema{
		Attributes: MergeProviderAttributes(
			attr,
			tfparts.ApiOptionProviderAttributes(),
		),
	}
}

func MergeProviderAttributes(attrs ...map[string]schema.Attribute) map[string]schema.Attribute {
	merged := make(map[string]schema.Attribute)
	for _, attr := range attrs {
		for k, v := range attr {
			merged[k] = v
		}
	}
	return merged
}

func (p *KubernetesResourceProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
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

	var err error
	defaultDefaults := &kube.APIClientOptions{
		Retry: job.RetryModel{
			MaxAttempts:  job.PointerTo(int64(3)),
			Interval:     job.PointerTo("10s,20s,30s"),
			Timeout:      job.PointerTo("5m"),
			InitialPause: job.PointerTo("0s"),
		},
		FieldManager: job.PointerTo("terraform-provider-kubernetes"),
	}

	p.DefaultApiOptions, err = kube.MergeAPIOptions(defaultDefaults, data.APIOptionsModel.Options())

	if err != nil {
		resp.Diagnostics.AddError("Failed to initialize provider api options", err.Error())
	}

	resp.DataSourceData = p
	resp.ResourceData = p
}

var lock = sync.Mutex{}

var supportedResources []func() resource.Resource
var supportedDataSources []func() datasource.DataSource
var supportedFunctions []func() function.Function

func RegisterResource(r func() resource.Resource) {
	lock.Lock()
	defer lock.Unlock()
	supportedResources = append(supportedResources, r)
}
func RegisterDataSource(d func() datasource.DataSource) {
	lock.Lock()
	defer lock.Unlock()
	supportedDataSources = append(supportedDataSources, d)
}
func RegisterFunction(f func() function.Function) {
	lock.Lock()
	defer lock.Unlock()
	supportedFunctions = append(supportedFunctions, f)
}

func (p *KubernetesResourceProvider) Resources(ctx context.Context) []func() resource.Resource {
	lock.Lock()
	defer lock.Unlock()
	return supportedResources
}

func (p *KubernetesResourceProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	lock.Lock()
	defer lock.Unlock()
	return supportedDataSources
}

func (p *KubernetesResourceProvider) Functions(ctx context.Context) []func() function.Function {
	lock.Lock()
	defer lock.Unlock()
	return supportedFunctions
}

func NewProvider(version string) func() provider.Provider {
	return func() provider.Provider {
		return &KubernetesResourceProvider{
			version:  version,
			typeName: "kube",
		}
	}
}
