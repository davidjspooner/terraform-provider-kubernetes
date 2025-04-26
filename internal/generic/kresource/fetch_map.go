package kresource

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/davidjspooner/terraform-provider-kubernetes/internal/generic/vpath"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type compiledFetch struct {
	path   vpath.Path
	regexp *regexp.Regexp
}

type CompiledFetchMap struct {
	compiled map[string]compiledFetch
}

func (cfm *CompiledFetchMap) Add(key, field, pattern string) error {
	if cfm.compiled == nil {
		cfm.compiled = make(map[string]compiledFetch)
	}
	cf := compiledFetch{}
	if field == "" {
		return fmt.Errorf("field is empty for key %s", key)
	}
	var err error
	cf.path, err = vpath.Compile(field)
	if err != nil {
		return fmt.Errorf("invalid path: %s for key %s: %w", field, key, err)
	}
	if pattern != "" && pattern != "*" {
		cf.regexp, err = regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("invalid regex: %s for key %s: %w", pattern, key, err)
		}
	}
	cfm.compiled[key] = cf
	return nil
}

func (w *CompiledFetchMap) GetOutputFrom(data unstructured.Unstructured) (map[string]string, error) {
	if len(w.compiled) == 0 {
		return nil, nil
	} else {
		output := make(map[string]string)
		for key, cf := range w.compiled {
			v, err := cf.path.EvaluateFor(data.Object)
			if err != nil {
				return nil, fmt.Errorf("error evaluating output %s: %w", key, err)
			}
			s, ok := v.(string)
			if !ok {
				b, err := json.Marshal(v)
				if err != nil {
					return nil, fmt.Errorf("error marshalling output %s: %w", key, err)
				}
				s = string(b)
			}
			if cf.regexp != nil {
				if !cf.regexp.MatchString(s) {
					return nil, fmt.Errorf("output %s does not match regex %s", key, cf.regexp.String())
				}
			}
			output[key] = s
		}
		return output, nil
	}
}
