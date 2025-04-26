package kube

import (
	"context"
	"errors"

	"github.com/davidjspooner/terraform-provider-kubernetes/internal/generic/job"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type StateInteraface interface {
	GetResouceKey() (ResourceKey, error)
	BuildManifest(manifest *unstructured.Unstructured) error
	UpdateFrom(manifest unstructured.Unstructured) error
}

type ResourceHelper struct {
	api         *APIClientWrapper
	options     *APIClientOptions
	retryHelper *job.RetryHelper
	key         ResourceKey
}

func (base *ResourceHelper) Create(ctx context.Context, plan StateInteraface) error {

	var manifest unstructured.Unstructured
	err := plan.BuildManifest(&manifest)
	if err != nil {
		return err
	}

	alreadyExists := false
	err = base.retryHelper.Retry(ctx, func(ctx context.Context, attempt int) error {
		err := base.api.Apply(ctx, &base.key, manifest, base.options)
		if apierrors.IsAlreadyExists(err) {
			alreadyExists = true
			return nil
		}
		return err
	})
	if alreadyExists {
		return err
	}
	if err != nil {
		return err
	}

	manifest, err = base.api.Get(ctx, &base.key, base.options)
	if err != nil {
		return err
	}
	err = plan.UpdateFrom(manifest)
	if err != nil {
		return err
	}
	return nil
}

func (base *ResourceHelper) Update(ctx context.Context, plan StateInteraface) error {
	var manifest unstructured.Unstructured
	err := plan.BuildManifest(&manifest)
	if err != nil {
		return err
	}
	err = base.retryHelper.Retry(ctx, func(ctx context.Context, attempt int) error {
		err := base.api.Apply(ctx, &base.key, manifest, base.options)
		return err
	})
	if err != nil {
		return err
	}
	manifest, err = base.api.Get(ctx, &base.key, base.options)
	if err != nil {
		return err
	}
	err = plan.UpdateFrom(manifest)
	if err != nil {
		return err
	}
	return nil
}
func (base *ResourceHelper) Delete(ctx context.Context, state StateInteraface) error {
	err := base.retryHelper.Retry(ctx, func(ctx context.Context, attempt int) error {
		base.api.Delete(ctx, &base.key, base.options)
		_, err := base.api.Get(ctx, &base.key, base.options)

		if err != nil && apierrors.IsNotFound(err) {
			//excellent
			return nil
		}
		return errors.New("resource still exists")
	})
	if err != nil {
		return err
	}
	return nil
}

type ExistRequirement int

const (
	MayOrMayNotExist = 0
	MustExit         = 1
	MustNotExit      = -1
)

func (base *ResourceHelper) Fetch(ctx context.Context, state StateInteraface, fetchMap *CompiledFetchMap, existRequirement ExistRequirement) (map[string]string, error) {
	var output map[string]string
	err := base.retryHelper.Retry(ctx, func(ctx context.Context, attempt int) error {
		manifest, err := base.api.Get(ctx, &base.key, base.options)
		if err != nil {
			if apierrors.IsNotFound(err) && existRequirement == MayOrMayNotExist {
				manifest = unstructured.Unstructured{}
			} else {
				return err
			}
		}
		err = state.UpdateFrom(manifest)
		if err != nil {
			return err
		}
		if fetchMap == nil {
			return nil
		}
		output, err = fetchMap.GetOutputFrom(manifest)
		if err != nil {
			return err
		}
		return nil
	})
	return output, err
}

func NewResourceHelper(ctx context.Context, sharedApi *APIClientWrapper, apiOptions *APIClientOptions, key ResourceKey) (*ResourceHelper, error) {
	retryHelper, err := apiOptions.Retry.NewHelper()
	if err != nil {
		return nil, err
	}
	h := &ResourceHelper{
		api:         sharedApi,
		options:     apiOptions,
		retryHelper: retryHelper,
		key:         key,
	}
	h.retryHelper.SetDeadline(ctx)
	return h, nil
}
