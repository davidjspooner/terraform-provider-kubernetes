package kresource

import (
	"context"
	"fmt"
	"reflect"
	"slices"

	"github.com/davidjspooner/terraform-provider-kubernetes/internal/job"
	"github.com/davidjspooner/terraform-provider-kubernetes/internal/pmodel"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
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

func (helper *CrudHelper) ReadActual(ctx context.Context, output *pmodel.OutputMetadata) (bool, error) {
	changed := false
	ctx, cancel := helper.RetryHelper.SetDeadline(ctx)
	defer cancel()
	key := GetKey(helper.State)
	var err error
	helper.Actual, err = helper.Shared.Get(ctx, key)
	if ErrorIsNotFound(err) {
		helper.State = unstructured.Unstructured{}
		return changed, nil
	}
	changed = helper.getResourceVersionInfo(output)
	return changed, nil
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

//type Difference struct {
//	Path  string
//	Left  interface{}
//	Right interface{}
//}

func DiffResources(left, right interface{}) ([]string, error) {
	var differences []string
	handleDifferences := DifferenceHandlerFunc(func(path string, left, right interface{}) error {
		differences = append(differences, path)
		return nil
	})
	err := compareReflect("", reflect.ValueOf(left), reflect.ValueOf(right), handleDifferences)
	return differences, err
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

func (helper *CrudHelper) getResourceVersionInfo(output *pmodel.OutputMetadata) bool {
	changed := false
	if helper.Actual.Object == nil {
		output.Generation = basetypes.NewInt64Value(0)
		output.ResourceVersion = basetypes.NewStringValue("")
		output.UID = basetypes.NewStringValue("")
		return changed
	}

	generation, found, err := unstructured.NestedInt64(helper.Actual.Object, "metadata", "generation")
	if err != nil || !found {
		generation = 0
	}
	prevGen := int64(0)
	if !output.Generation.IsNull() {
		prevGen = output.Generation.ValueInt64()
	}
	if prevGen != generation {
		output.Generation = basetypes.NewInt64Value(generation)
		changed = true
	}

	var s string

	s, found, err = unstructured.NestedString(helper.Actual.Object, "metadata", "resourceVersion")
	if err != nil || !found {
		s = ""
	}
	prvStr := ""

	if !output.ResourceVersion.IsNull() {
		prvStr = output.ResourceVersion.ValueString()
	}
	if s != prvStr {
		output.ResourceVersion = basetypes.NewStringValue(s)
		changed = true
	}

	s, found, err = unstructured.NestedString(helper.Actual.Object, "metadata", "uid")
	if err != nil || !found {
		s = ""
	}
	prvStr = ""

	if !output.ResourceVersion.IsNull() {
		prvStr = output.UID.ValueString()
	}
	if s != prvStr {
		output.UID = basetypes.NewStringValue(s)
		changed = true
	}

	return changed
}
