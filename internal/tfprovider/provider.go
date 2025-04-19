// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfprovider

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/davidjspooner/terraform-provider-kubernetes/internal/job"
	"github.com/davidjspooner/terraform-provider-kubernetes/internal/kresource"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	runtimeschema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	memory "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure ScaffoldingProvider satisfies various provider interfaces.
var _ provider.Provider = &KubernetesResourceProvider{}
var _ provider.ProviderWithFunctions = &KubernetesResourceProvider{}

// KubernetesResourceProvider defines the provider implementation.
type KubernetesResourceProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.

	version string

	Shared            sharedApi
	DefaultApiOptions *APIOptions
}

// KubernetesProviderModel describes the provider data model.
type KubernetesProviderModel struct {
	ConfigPaths           []types.String `tfsdk:"config_paths"`
	ConfigContext         types.String   `tfsdk:"config_context"`
	ConfigContextAuthInfo types.String   `tfsdk:"config_context_auth_info"`
	ConfigContextCluster  types.String   `tfsdk:"config_context_cluster"`
	Namespace             types.String   `tfsdk:"namespace"`
	DefaultApiOptions     *APIOptions    `tfsdk:"api_options"`
}

func (p *KubernetesResourceProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "kubernetes"
	resp.Version = p.version
}

func (p *KubernetesResourceProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
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
			"api_options": ApiOptionsModelSchema(),
		},
	}
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

	if data.DefaultApiOptions != nil {
		data.DefaultApiOptions = &APIOptions{}
	}

	var err error
	defaultDefaults := &APIOptions{
		Retry: &job.RetryModel{},
	}
	//TODO set more setDefaults

	p.DefaultApiOptions, err = MergeKubenetesAPIOptions(defaultDefaults, data.DefaultApiOptions)

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
			version: version,
		}
	}
}

type ResourceType struct {
	Kind       string
	APIVersion string
	Namespaced bool
}

type sharedApi struct {
	lock sync.Mutex

	fieldManager string

	namespace string
	config    *rest.Config

	configPaths           []string
	configContext         string
	configContextAuthInfo string
	configContextCluster  string

	resourceTypes map[string]*ResourceType

	dynamic   *dynamic.DynamicClient
	discovery discovery.CachedDiscoveryInterface
}

func (shared *sharedApi) ResourceInterface(ctx context.Context, apiVersion, kind, namespace string) (dynamic.ResourceInterface, error) {

	if shared.fieldManager == "" {
		shared.fieldManager = "terraform-provider-kubernetes"
	}

	shared.lock.Lock()
	defer shared.lock.Unlock()

	gv, err := runtimeschema.ParseGroupVersion(apiVersion)
	if err != nil {
		return nil, err
	}

	gvk := runtimeschema.GroupVersionKind{
		Group:   gv.Group,
		Version: gv.Version,
		Kind:    kind,
	}

	if shared.discovery == nil || shared.dynamic == nil {
		err = shared.reloadConfig(context.Background())
		if err != nil {
			return nil, err
		}
	}

	mapper := restmapper.NewDeferredDiscoveryRESTMapper(shared.discovery)
	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		shared.discovery.Invalidate()
		return nil, err
	}

	var dr dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		namespace = kresource.FirstNonNullString(namespace, shared.namespace, "default")
		dr = shared.dynamic.Resource(mapping.Resource).Namespace(namespace)
	} else {
		// for cluster-wide resources
		dr = shared.dynamic.Resource(mapping.Resource)
	}
	return dr, nil
}

func (shared *sharedApi) ReloadConfig(ctx context.Context) error {
	shared.lock.Lock()
	defer shared.lock.Unlock()
	return shared.reloadConfig(ctx)
}

func (shared *sharedApi) reloadConfig(ctx context.Context) error {
	overrides := &clientcmd.ConfigOverrides{}
	loader := &clientcmd.ClientConfigLoadingRules{}

	var kubeConfigPaths []string

	if len(shared.configPaths) == 0 {
		path, _ := kresource.ExpandEnv("$KUBECONFIG")
		if path != "" {
			kubeConfigPaths = append(kubeConfigPaths, path)
		}
		path, _ = kresource.ExpandEnv("$HOME/.kube/config")
		if path != "" {
			kubeConfigPaths = append(kubeConfigPaths, path)
		}
	} else {
		for _, pathValue := range shared.configPaths {
			path, _ := kresource.ExpandEnv(pathValue)
			if path != "" {
				kubeConfigPaths = append(kubeConfigPaths, path)
			}
		}
	}
	switch len(kubeConfigPaths) {
	case 0:
		return fmt.Errorf("kubeconfig file not found")
	case 1:
		loader.ExplicitPath = kubeConfigPaths[0]
	default:
		loader.Precedence = kubeConfigPaths
	}

	//copied from https://github.com/hashicorp/terraform-provider-kubernetes/blob/main/internal/framework/provider/provider_configure.go

	ctxSuffix := "; default context"

	if shared.configContext != "" || shared.configContextAuthInfo != "" || shared.configContextCluster != "" || shared.namespace != "" {
		ctxSuffix = "; overridden context"
		if shared.configContext != "" {
			overrides.CurrentContext = shared.configContext
			ctxSuffix += fmt.Sprintf("; config ctx: %s", overrides.CurrentContext)
			tflog.Debug(ctx, "Using custom current context", map[string]any{"context": overrides.CurrentContext})
		}

		overrides.Context = api.Context{}
		if shared.configContextAuthInfo != "" {
			overrides.Context.AuthInfo = shared.configContextAuthInfo
			ctxSuffix += fmt.Sprintf("; auth_info: %s", overrides.Context.AuthInfo)
		}
		if shared.configContextCluster != "" {
			overrides.Context.Cluster = shared.configContextCluster
			ctxSuffix += fmt.Sprintf("; cluster: %s", overrides.Context.Cluster)
		}
		if shared.namespace != "" {
			overrides.Context.Namespace = shared.namespace
			ctxSuffix += fmt.Sprintf("; namespace: %s", overrides.Context.Namespace)
		}
	}
	tflog.Debug(ctx, "Using kubeconfig", map[string]any{"path": loader.ExplicitPath, "precedence": loader.Precedence, "context": ctxSuffix})

	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loader, overrides)

	var err error
	shared.config, err = clientConfig.ClientConfig()
	if err != nil {
		return err
	}

	innerDiscovery, err := discovery.NewDiscoveryClientForConfig(shared.config)
	if err != nil {
		return err
	}

	shared.discovery = memory.NewMemCacheClient(innerDiscovery)

	shared.dynamic, err = dynamic.NewForConfig(shared.config)
	if err != nil {
		return err
	}
	return nil
}

func (shared *sharedApi) SetConfigContext(context string) {
	shared.lock.Lock()
	defer shared.lock.Unlock()
	shared.configContext = context
}

func (shared *sharedApi) SetConfigContextAuthInfo(authInfo string) {
	shared.lock.Lock()
	defer shared.lock.Unlock()
	shared.configContextAuthInfo = authInfo
}

func (shared *sharedApi) SetConfigContextCluster(cluster string) {
	shared.lock.Lock()
	defer shared.lock.Unlock()
	shared.configContextCluster = cluster
}

func (shared *sharedApi) SetConfigPaths(paths []string) {
	shared.lock.Lock()
	defer shared.lock.Unlock()
	if len(paths) == 0 {
		paths = []string{"$KUBECONFIG", "~/.kube/config", "/etc/kubernetes/admin.conf", "/etc/kubernetes/kubelet.conf"}
	}
	shared.configPaths = paths
}

func (shared *sharedApi) SetNamespace(namespace string) {
	shared.lock.Lock()
	defer shared.lock.Unlock()
	shared.namespace = namespace
}

func (shared *sharedApi) fetchResourceTypes() error {
	if shared.discovery == nil {
		return fmt.Errorf("discovery client is not initialized")
	}

	resourceTypes := make(map[string]*ResourceType)

	apiResources, err := shared.discovery.ServerPreferredNamespacedResources()
	if err != nil {
		return err
	}

	for _, apiResourceList := range apiResources {
		for _, apiResource := range apiResourceList.APIResources {
			resourceTypes[apiResource.Kind] = &ResourceType{
				Kind:       apiResource.Kind,
				APIVersion: apiResourceList.GroupVersion,
				Namespaced: apiResource.Namespaced,
			}
		}
	}

	apiResources, err = shared.discovery.ServerPreferredResources()
	if err != nil {
		for _, apiResourceList := range apiResources {
			for _, apiResource := range apiResourceList.APIResources {
				resourceTypes[apiResource.Kind] = &ResourceType{
					Kind:       apiResource.Kind,
					APIVersion: apiResourceList.GroupVersion,
					Namespaced: apiResource.Namespaced,
				}
			}
		}
	}

	shared.resourceTypes = resourceTypes
	return nil
}

func (shared *sharedApi) GetNamespace(namespace *string) string {
	if namespace == nil || *namespace == "" {
		return shared.namespace
	}
	return *namespace
}

func (shared *sharedApi) getNamespaceForKind(kind string, namespace *string) string {
	shared.lock.Lock()
	defer shared.lock.Unlock()

	kind = strings.ToLower(kind)
	rType := shared.resourceTypes[kind]
	if rType == nil {
		shared.fetchResourceTypes()
		rType = shared.resourceTypes[kind]
	}
	if rType != nil && !rType.Namespaced {
		return ""
	}
	return shared.GetNamespace(namespace)
}

func (shared *sharedApi) Get(ctx context.Context, key *kresource.Key, apiOptions *APIOptions) (unstructured.Unstructured, error) {
	// fetch Object and update the model

	ri, err := shared.ResourceInterface(ctx, key.ApiVersion, key.Kind, shared.getNamespaceForKind(key.Kind, key.MetaData.Namespace))

	if err != nil {
		return unstructured.Unstructured{}, err
	}
	u, err := ri.Get(ctx, key.MetaData.Name, metav1.GetOptions{})
	if err != nil {
		return unstructured.Unstructured{}, err
	}
	return *u, nil
}

func (shared *sharedApi) Apply(ctx context.Context, key *kresource.Key, u unstructured.Unstructured, apiOptions *APIOptions) error {
	ri, err := shared.ResourceInterface(ctx, key.ApiVersion, key.Kind, shared.getNamespaceForKind(key.Kind, key.MetaData.Namespace))
	if err != nil {
		return err
	}

	reply, err := ri.Apply(ctx, key.MetaData.Name, &u, metav1.ApplyOptions{
		FieldManager: shared.fieldManager,
	})
	if err != nil {
		return err
	}

	_ = reply

	//todo apply the object
	return nil
}

func (shared *sharedApi) Delete(ctx context.Context, key *kresource.Key, apiOptions *APIOptions) error {
	ri, err := shared.ResourceInterface(ctx, key.ApiVersion, key.Kind, shared.getNamespaceForKind(key.Kind, key.MetaData.Namespace))
	if err != nil {
		return err
	}

	err = ri.Delete(ctx, key.MetaData.Name, metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	return nil
}
