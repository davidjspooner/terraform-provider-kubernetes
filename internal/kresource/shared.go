package kresource

import (
	"context"
	"fmt"
	"sync"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

type Shared struct {
	lock sync.Mutex

	namespace string
	config    *rest.Config

	configPaths           []string
	configContext         string
	configContextAuthInfo string
	configContextCluster  string

	dynamic   *dynamic.DynamicClient
	discovery discovery.CachedDiscoveryInterface
}

func (shared *Shared) ResourceInterface(ctx context.Context, apiVersion, kind, namespace string) (dynamic.ResourceInterface, error) {

	shared.lock.Lock()
	defer shared.lock.Unlock()

	gv, err := schema.ParseGroupVersion(apiVersion)
	if err != nil {
		return nil, err
	}

	gvk := schema.GroupVersionKind{
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

func (shared *Shared) ReloadConfig(ctx context.Context) error {
	shared.lock.Lock()
	defer shared.lock.Unlock()
	return shared.reloadConfig(ctx)
}

func (shared *Shared) reloadConfig(ctx context.Context) error {
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

func (shared *Shared) SetConfigContext(context string) {
	shared.lock.Lock()
	defer shared.lock.Unlock()
	shared.configContext = context
}

func (shared *Shared) SetConfigContextAuthInfo(authInfo string) {
	shared.lock.Lock()
	defer shared.lock.Unlock()
	shared.configContextAuthInfo = authInfo
}

func (shared *Shared) SetConfigContextCluster(cluster string) {
	shared.lock.Lock()
	defer shared.lock.Unlock()
	shared.configContextCluster = cluster
}

func (shared *Shared) SetConfigPaths(paths []string) {
	shared.lock.Lock()
	defer shared.lock.Unlock()
	if len(paths) == 0 {
		paths = []string{"$KUBECONFIG", "~/.kube/config", "/etc/kubernetes/admin.conf", "/etc/kubernetes/kubelet.conf"}
	}
	shared.configPaths = paths
}

func (shared *Shared) SetNamespace(namespace string) {
	shared.lock.Lock()
	defer shared.lock.Unlock()
	shared.namespace = namespace
}

func (shared *Shared) GetNamespace(namespace *string) string {
	if namespace == nil || *namespace == "" {
		return shared.namespace
	}
	return *namespace
}

func (shared *Shared) Get(ctx context.Context, key *Key) (unstructured.Unstructured, error) {
	// fetch Object and update the model

	ri, err := shared.ResourceInterface(ctx, key.ApiVersion, key.Kind, shared.GetNamespace(key.MetaData.Namespace))

	if err != nil {
		return unstructured.Unstructured{}, err
	}
	u, err := ri.Get(ctx, key.MetaData.Name, metav1.GetOptions{})
	if err != nil {
		return unstructured.Unstructured{}, err
	}
	return *u, nil
}

func (shared *Shared) Apply(ctx context.Context, key *Key, u unstructured.Unstructured) error {
	ri, err := shared.ResourceInterface(ctx, key.ApiVersion, key.Kind, shared.GetNamespace(key.MetaData.Namespace))
	if err != nil {
		return err
	}

	reply, err := ri.Apply(ctx, key.MetaData.Name, &u, metav1.ApplyOptions{
		FieldManager: "terraform",
	})
	if err != nil {
		return err
	}

	_ = reply

	//todo apply the object
	return nil
}

func (shared *Shared) Delete(ctx context.Context, key *Key) error {
	ri, err := shared.ResourceInterface(ctx, key.ApiVersion, key.Kind, shared.GetNamespace(key.MetaData.Namespace))
	if err != nil {
		return err
	}

	err = ri.Delete(ctx, key.MetaData.Name, metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	return nil
}
