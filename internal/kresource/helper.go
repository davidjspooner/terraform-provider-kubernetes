package kresource

import (
	"context"
	"fmt"
	"slices"

	"github.com/davidjspooner/terraform-provider-kubernetes/internal/job"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type CrudHelper struct {
	Plan, Actual, State unstructured.Unstructured
	RetryHelper         *job.RetryHelper
	Shared              *Shared
}

func GetKey(r unstructured.Unstructured) *Key {
	if r.Object == nil {
		return nil
	}
	k := Key{}
	k.ApiVersion = r.GetAPIVersion()
	k.Kind = r.GetKind()
	k.MetaData.Name = r.GetName()
	namespace := r.GetNamespace()
	k.MetaData.Namespace = &namespace
	return &k
}

func (helper *CrudHelper) Read(ctx context.Context) error {
	ctx, cancel := helper.RetryHelper.SetDeadline(ctx)
	defer cancel()
	key := GetKey(helper.Plan)
	var err3 error
	helper.State, err3 = helper.Shared.Get(ctx, key)
	if ErrorIsNotFound(err3) {
		helper.State = unstructured.Unstructured{}
		return nil
	}
	//TODO check if it is what we want
	return nil
}
func (helper *CrudHelper) ApplyPlan(ctx context.Context) error {
	ctx, cancel := helper.RetryHelper.SetDeadline(ctx)
	defer cancel()

	key := GetKey(helper.Plan)
	err := helper.RetryHelper.Retry(ctx, func(ctx context.Context, attempt int) error {
		err2 := helper.Shared.Apply(ctx, key, helper.Plan)
		return err2
	})
	return err
}

func (helper *CrudHelper) CreateFromPlan(ctx context.Context) error {
	return helper.ApplyPlan(ctx)
}

var invariants = []string{".metadata.name", ".metadata.namespace", ".kind", ".apiVersion"}

func DiffContainsInvariant(diffs []string) bool {
	for _, invariant := range invariants {
		if slices.Contains(diffs, invariant) {
			return true
		}
	}
	return false
}

func (helper *CrudHelper) Update(ctx context.Context) error {

	//check for invariant violations
	needRecreate := false

	key := GetKey(helper.State)
	actual, err := helper.Shared.Get(ctx, key)
	if err == nil {
		diffs, _ := DiffResources(actual, helper.Plan)
		if DiffContainsInvariant(diffs) {
			needRecreate = true
		}
		return nil
	}

	if err != nil {
		return err
	}
	if needRecreate {
		//delete things that need to be recreated
		err := helper.delete(ctx, key)
		if err != nil {
			return err
		}
	}
	return helper.ApplyPlan(ctx)
}

func (helper *CrudHelper) delete(ctx context.Context, key *Key) error {
	err := helper.RetryHelper.Retry(ctx, func(ctx context.Context, attempt int) error {
		err := helper.Shared.Delete(ctx, key)
		if err != nil && !ErrorIsNotFound(err) {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	err = helper.RetryHelper.Retry(ctx, func(ctx context.Context, attempt int) error {
		//wait for them to be gone
		_, err := helper.Shared.Get(ctx, key)
		if err != nil {
			return nil
		}
		return fmt.Errorf("resource still exists")
	})
	return err
}

func (helper *CrudHelper) DeleteState(ctx context.Context) error {
	ctx, cancel := helper.RetryHelper.SetDeadline(ctx)
	defer cancel()

	key := GetKey(helper.State)
	err := helper.delete(ctx, key)
	if err != nil {
		return err
	}
	helper.State = unstructured.Unstructured{}

	return err
}

func (helper *CrudHelper) Retry(ctx context.Context, task func(context.Context, int) error) error {
	return helper.RetryHelper.Retry(ctx, task)
}
