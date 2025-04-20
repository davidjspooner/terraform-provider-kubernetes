package kresource

import (
	"context"
	"errors"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type StateInteraface interface {
	GetResouceKey() (ResourceKey, error)
	BuildManifest(manifest *unstructured.Unstructured) error
	FromManifest(manifest *unstructured.Unstructured) error
	GetApiOptions() *APIClientOptions
}

type ResourceHelper struct {
	Api     *APIClientWrapper
	Options *APIClientOptions
}

func (base *ResourceHelper) Create(ctx context.Context, plan StateInteraface) error {

	retryHelper, err := base.Options.Retry.NewHelper()
	if err != nil {
		return err
	}

	var manifest unstructured.Unstructured
	err = plan.BuildManifest(&manifest)
	if err != nil {
		return err
	}

	key, err := plan.GetResouceKey()
	if err != nil {
		return err
	}

	alreadyExists := false
	err = retryHelper.Retry(ctx, func(ctx context.Context, attempt int) error {
		err := base.Api.Apply(ctx, &key, manifest, base.Options)
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

	err = plan.FromManifest(&manifest)
	if err != nil {
		return err
	}
	return nil
}

func (base *ResourceHelper) trimMapElements(previousMap, currentMap map[string]string) {
	deleteKeys := make([]string, 0, len(previousMap))
	for k := range currentMap {
		_, existed := previousMap[k]
		if !existed {
			deleteKeys = append(deleteKeys, k)
		}
	}
	for _, deleteKey := range deleteKeys {
		delete(currentMap, deleteKey)
	}
}

func (base *ResourceHelper) Read(ctx context.Context, state StateInteraface) error {

	retryHelper, err := base.Options.Retry.NewHelper()
	if err != nil {
		return err
	}

	key, err := state.GetResouceKey()
	if err != nil {
		return err
	}

	var currentStateManifest unstructured.Unstructured
	found := false
	err = retryHelper.Retry(ctx, func(ctx context.Context, attempt int) error {
		var err error
		currentStateManifest, err = base.Api.Get(ctx, &key, base.Options)
		if err == nil {
			found = true
			return nil
		}
		if apierrors.IsNotFound(err) {
			// resource not found, return nil
			return nil
		}
		return err
	})
	if err != nil {
		return err
	}
	if !found {
		return nil
	}
	var previousStateManifest unstructured.Unstructured
	state.BuildManifest(&previousStateManifest)

	currentStateAnnotations := currentStateManifest.GetAnnotations()
	previousStateAnnotations := previousStateManifest.GetAnnotations()
	if previousStateAnnotations == nil {
		currentStateManifest.SetAnnotations(nil)
	} else if currentStateAnnotations != nil {
		base.trimMapElements(previousStateAnnotations, currentStateAnnotations)
		currentStateManifest.SetAnnotations(currentStateAnnotations)
	}
	currentStateLabels := currentStateManifest.GetLabels()
	previousStateLabels := previousStateManifest.GetLabels()
	if previousStateLabels == nil {
		currentStateManifest.SetLabels(nil)
	} else if currentStateLabels != nil {
		base.trimMapElements(previousStateLabels, currentStateLabels)
		currentStateManifest.SetLabels(currentStateLabels)
	}

	err = state.FromManifest(&currentStateManifest)

	if err != nil {
		return err
	}
	return nil
}
func (base *ResourceHelper) Update(ctx context.Context, plan StateInteraface) error {
	retryHelper, err := base.Options.Retry.NewHelper()
	if err != nil {
		return err
	}

	var manifest unstructured.Unstructured
	err = plan.BuildManifest(&manifest)
	if err != nil {
		return err
	}
	key, err := plan.GetResouceKey()
	if err != nil {
		return err
	}
	err = retryHelper.Retry(ctx, func(ctx context.Context, attempt int) error {
		err := base.Api.Apply(ctx, &key, manifest, base.Options)
		return err
	})
	if err != nil {
		return err
	}
	manifest, err = base.Api.Get(ctx, &key, base.Options)
	if err != nil {
		return err
	}
	err = plan.FromManifest(&manifest)
	if err != nil {
		return err
	}
	return nil
}
func (base *ResourceHelper) Delete(ctx context.Context, state StateInteraface) error {
	retryHelper, err := base.Options.Retry.NewHelper()
	if err != nil {
		return err
	}

	key, err := state.GetResouceKey()
	if err != nil {
		return err
	}
	err = retryHelper.Retry(ctx, func(ctx context.Context, attempt int) error {
		base.Api.Delete(ctx, &key, base.Options)
		_, err = base.Api.Get(ctx, &key, base.Options)

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

func NewResourceHelper(sharedApi *APIClientWrapper, apiOptions *APIClientOptions) *ResourceHelper {
	return &ResourceHelper{
		Api:     sharedApi,
		Options: apiOptions,
	}
}
