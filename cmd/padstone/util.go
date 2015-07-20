package main

import (
	"fmt"
	"strings"
)

func decodeKVSpecs(specs []string) (map[string]string, error) {
	ret := map[string]string{}
	for _, spec := range specs {
		equalsIdx := strings.Index(spec, "=")
		if equalsIdx == -1 {
			return ret, fmt.Errorf("parameter %#v must be formatted as key=value", spec)
		}
		k := spec[:equalsIdx]
		v := spec[equalsIdx+1:]
		ret[k] = v
	}
	return ret, nil
}
