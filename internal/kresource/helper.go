package kresource

import (
	"context"
	"fmt"
	"reflect"
	"slices"

	"github.com/davidjspooner/dsflow/pkg/job"
	"github.com/davidjspooner/dsvalue/pkg/path"
	"github.com/davidjspooner/dsvalue/pkg/reflected"
	"github.com/davidjspooner/dsvalue/pkg/value"
)

type CrudHelper struct {
	Plan, Actual, State ResourceMap
	RetryHelper         *job.RetryHelper
	Shared              *Shared
}

func (helper *CrudHelper) Read(ctx context.Context) error {
	ctx, cancel := helper.RetryHelper.SetDeadline(ctx)
	defer cancel()
	err := helper.State.ForEach(func(key *Key, r *Resource) error {
		u, err3 := helper.Shared.Get(ctx, &r.Key)
		if ErrorIsNotFound(err3) {
			helper.State.Delete(key)
			return nil
		}
		//TODO check if it is what we want
		_ = u
		return nil
	})
	return err
}
func (helper *CrudHelper) ApplyPlan(ctx context.Context) error {
	ctx, cancel := helper.RetryHelper.SetDeadline(ctx)
	defer cancel()

	toCreateOrUpdate := helper.Plan.Clone()
	err := helper.RetryHelper.Retry(ctx, func(ctx context.Context, attempt int) error {
		var err2 error
		toCreateOrUpdate.ForEach(func(key *Key, r *Resource) error {
			err3 := helper.Shared.Apply(ctx, &r.Key, r.Unstructured)
			if err3 == nil {
				toCreateOrUpdate.Detach(key)
			}
			if err3 != nil && err2 == nil {
				err2 = err3
			}
			return nil
		})
		return err2
	})
	return err
}

func (helper *CrudHelper) CreateFromPlan(ctx context.Context) error {
	return helper.ApplyPlan(ctx)
}
func (helper *CrudHelper) Diff(ctx context.Context, a, b *Resource) ([]string, error) {
	var diffs []string
	leftRoot, err := reflected.NewReflectedObject(reflect.ValueOf(a.Unstructured.Object), nil)
	if err != nil {
		return nil, err
	}
	rightRoot, err := reflected.NewReflectedObject(reflect.ValueOf(b.Unstructured.Object), nil)
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
	needRecreate := ResourceMap{}

	err := helper.Plan.ForEach(func(key *Key, r *Resource) error {
		stateResource, ok := helper.State.Lookup(key)
		if !ok {
			return nil
		}
		diffs, _ := helper.Diff(ctx, stateResource, r)
		if DiffContainsInvariant(diffs) {
			needRecreate.Attach(r)
		}
		return nil
	})
	if err != nil {
		return err
	}
	if needRecreate.Len() > 0 {
		//delete things that need to be recreated
		err = needRecreate.ForEach(func(key *Key, r *Resource) error {
			err := helper.Shared.Delete(ctx, &r.Key)
			if err != nil && !ErrorIsNotFound(err) {
				return nil
			}
			return err
		})
		if err != nil {
			return err
		}
		//wait for them to be gone
		err = needRecreate.ForEach(func(key *Key, r *Resource) error {
			_, err := helper.Shared.Get(ctx, &r.Key)
			if err != nil {
				return nil
			}
			return fmt.Errorf("resource still exists")
		})
		if err != nil {
			return err
		}
	}
	return helper.ApplyPlan(ctx)
}
func (helper *CrudHelper) DeleteState(ctx context.Context) error {
	ctx, cancel := helper.RetryHelper.SetDeadline(ctx)
	defer cancel()

	//try and delete everything
	toDelete := helper.State.Clone()
	err := helper.RetryHelper.Retry(ctx, func(ctx context.Context, attempt int) error {
		var errList ErrorList
		toDelete.ForEach(func(key *Key, r *Resource) error {
			err3 := helper.Shared.Delete(ctx, &r.Key)
			if err3 == nil || ErrorIsNotFound(err3) {
				toDelete.Delete(key)
				return nil
			}
			errList = append(errList, err3)
			return nil
		})
		if len(errList) > 0 {
			return errList
		}
		return nil
	})
	if err != nil {
		return err
	}

	//and then wait to check if they are gone
	err = helper.RetryHelper.Retry(ctx, func(ctx context.Context, attempt int) error {
		var errList ErrorList
		helper.State.ForEach(func(key *Key, r *Resource) error {
			err3 := helper.Shared.Delete(ctx, &r.Key)
			if err3 == nil || ErrorIsNotFound(err3) {
				helper.State.Delete(key)
				return nil
			}
			errList = append(errList, err3)
			return nil
		})
		if len(errList) > 0 {
			return errList
		}
		return nil
	})

	return err
}

func (helper *CrudHelper) Retry(ctx context.Context, task func(context.Context, int) error) error {
	return helper.RetryHelper.Retry(ctx, task)
}
