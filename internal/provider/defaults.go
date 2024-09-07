package provider

import "golang.org/x/exp/constraints"

func FirstNonEmpty[E constraints.Ordered](values ...E) E {
	var null E
	for _, v := range values {
		if v != null {
			return v
		}

	}
	return values[len(values)-1]
}
