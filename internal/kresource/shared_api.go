package kresource

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/davidjspooner/terraform-provider-kubernetes/internal/job"
	"github.com/hashicorp/terraform-plugin-log/tflog"
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
)

type APIOptions struct {
	Retry          *job.RetryModel
	FieldManager   *string
	ForceConflicts *bool
}

func MergeAPIOptions(
	models ...*APIOptions,
) (*APIOptions, error) {
	merged := &APIOptions{}
	var err error
	for _, model := range models {
		if model.Retry != nil {
			merged.Retry, err = job.MergeRetryModels(merged.Retry, model.Retry)
			if err != nil {
				return nil, err
			}
		}
		if model.FieldManager != nil {
			merged.FieldManager = model.FieldManager
		}
		if model.ForceConflicts != nil {
			merged.ForceConflicts = model.ForceConflicts
		}
	}
	if merged.FieldManager == nil || *merged.FieldManager == "" {
		s := "terraform-provider-kubernetes"
		merged.FieldManager = &s
	}
	if merged.Retry == nil {
		merged.Retry = &job.RetryModel{}
	}
	if merged.Retry.MaxAttempts == nil {
		i := int64(3)
		merged.Retry.MaxAttempts = &i
	}
	if merged.Retry.InitialPause == nil {
		s := "0s"
		merged.Retry.InitialPause = &s
	}
	if merged.Retry.Interval == nil {
		s := "2s"
		merged.Retry.Interval = &s
	}
	if merged.Retry.Timeout == nil {
		s := "30s"
		merged.Retry.Timeout = &s
	}
	if merged.Retry.FastFail == nil {
		fastFail := []string{
			"AlreadyExists",
			"Conflict",
			"Invalid",
			"Forbidden",
			"Unauthorized",
		}
		merged.Retry.FastFail = &fastFail
	}

	return merged, nil
}

type SharedApi struct {
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

type ResourceType struct {
	Kind       string
	APIVersion string
	Namespaced bool
}

func (shared *SharedApi) ResourceInterface(ctx context.Context, apiVersion, kind, namespace string) (dynamic.ResourceInterface, error) {

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
		namespace = FirstNonNullString(namespace, shared.namespace, "default")
		dr = shared.dynamic.Resource(mapping.Resource).Namespace(namespace)
	} else {
		// for cluster-wide resources
		dr = shared.dynamic.Resource(mapping.Resource)
	}
	return dr, nil
}

func (shared *SharedApi) ReloadConfig(ctx context.Context) error {
	shared.lock.Lock()
	defer shared.lock.Unlock()
	return shared.reloadConfig(ctx)
}

func (shared *SharedApi) reloadConfig(ctx context.Context) error {
	overrides := &clientcmd.ConfigOverrides{}
	loader := &clientcmd.ClientConfigLoadingRules{}

	var kubeConfigPaths []string

	if len(shared.configPaths) == 0 {
		path, _ := ExpandEnv("$KUBECONFIG")
		if path != "" {
			kubeConfigPaths = append(kubeConfigPaths, path)
		}
		path, _ = ExpandEnv("$HOME/.kube/config")
		if path != "" {
			kubeConfigPaths = append(kubeConfigPaths, path)
		}
	} else {
		for _, pathValue := range shared.configPaths {
			path, _ := ExpandEnv(pathValue)
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

func (shared *SharedApi) SetConfigContext(context string) {
	shared.lock.Lock()
	defer shared.lock.Unlock()
	shared.configContext = context
}

func (shared *SharedApi) SetConfigContextAuthInfo(authInfo string) {
	shared.lock.Lock()
	defer shared.lock.Unlock()
	shared.configContextAuthInfo = authInfo
}

func (shared *SharedApi) SetConfigContextCluster(cluster string) {
	shared.lock.Lock()
	defer shared.lock.Unlock()
	shared.configContextCluster = cluster
}

func (shared *SharedApi) SetConfigPaths(paths []string) {
	shared.lock.Lock()
	defer shared.lock.Unlock()
	if len(paths) == 0 {
		paths = []string{"$KUBECONFIG", "~/.kube/config", "/etc/kubernetes/admin.conf", "/etc/kubernetes/kubelet.conf"}
	}
	shared.configPaths = paths
}

func (shared *SharedApi) SetNamespace(namespace string) {
	shared.lock.Lock()
	defer shared.lock.Unlock()
	shared.namespace = namespace
}

func (shared *SharedApi) fetchResourceTypes() error {
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

func (shared *SharedApi) GetNamespace(namespace *string) string {
	if namespace == nil || *namespace == "" {
		return shared.namespace
	}
	return *namespace
}

func (shared *SharedApi) getNamespaceForKind(kind string, namespace *string) string {
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

func (shared *SharedApi) Get(ctx context.Context, key *Key, apiOptions *APIOptions) (unstructured.Unstructured, error) {
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

func (shared *SharedApi) Apply(ctx context.Context, key *Key, u unstructured.Unstructured, apiOptions *APIOptions) error {
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

func (shared *SharedApi) Delete(ctx context.Context, key *Key, apiOptions *APIOptions) error {
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
