package kresource

import "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

type Resource struct {
	Key          Key
	Unstructured unstructured.Unstructured
}

type ResourceMap struct {
	Resources map[string]Resource
}

func NewResourceMap() *ResourceMap {
	return &ResourceMap{
		Resources: make(map[string]Resource),
	}
}

func (rm *ResourceMap) Len() int {
	return len(rm.Resources)
}

func (rm *ResourceMap) ParseAndAttach(manifest string) error {
	r, err := ParseResourceYaml(manifest)
	if err != nil {
		return err
	}
	rm.Attach(r)
	return nil
}

func (rm *ResourceMap) Attach(r *Resource) {
	if rm.Resources == nil {
		rm.Resources = make(map[string]Resource)
	}
	rm.Resources[r.Key.String()] = *r
}
func (rm *ResourceMap) Detach(k *Key) {
	delete(rm.Resources, k.String())
}

func (rm *ResourceMap) Lookup(key *Key) (*Resource, bool) {
	r, ok := rm.Resources[key.String()]
	return &r, ok
}

func (rm *ResourceMap) Delete(key *Key) {
	delete(rm.Resources, key.String())
}

func (rm *ResourceMap) ForEach(fn func(key *Key, r *Resource) error) error {
	keys := make([]string, 0, len(rm.Resources))
	for _, r := range rm.Resources {
		keys = append(keys, r.Key.String())
	}
	for _, r := range keys {
		r, ok := rm.Resources[r]
		if !ok {
			continue
		}
		err := fn(&r.Key, &r)
		if err != nil {
			return err
		}
	}
	return nil
}

func WalkResourceMaps(a, b *ResourceMap, fn func(key *Key, a, b *unstructured.Unstructured) error) error {
	err := a.ForEach(func(key *Key, r *Resource) error {
		var innerErr error
		rB, ok := b.Lookup(key)
		if ok {
			innerErr = fn(key, &r.Unstructured, &rB.Unstructured)
		} else {
			innerErr = fn(key, &r.Unstructured, nil)
		}
		return innerErr
	})
	if err != nil {
		return err
	}
	err = b.ForEach(func(key *Key, r *Resource) error {
		var innerErr error
		_, ok := a.Lookup(key)
		if !ok {
			innerErr = fn(key, nil, &r.Unstructured)
		}
		return innerErr
	})
	if err != nil {
		return err
	}
	return nil
}

func (rm *ResourceMap) Clone() *ResourceMap {
	clone := NewResourceMap()
	rm.ForEach(func(key *Key, r *Resource) error {
		clone.Attach(r)
		return nil
	})
	return clone
}
