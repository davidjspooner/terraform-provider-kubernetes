package kresource

import "strings"

type ErrorList []error

func (el ErrorList) Error() string {
	var s strings.Builder
	for i, e := range el {
		if i > 0 {
			s.WriteString(", ")
		}
		s.WriteString(e.Error())
	}
	return s.String()
}
