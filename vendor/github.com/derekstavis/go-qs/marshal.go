package qs

import (
	"fmt"
	"net/url"
)

func Marshal(hash map[string]interface{}) (string, error) {
	return buildNestedQuery(hash, "")
}

func buildNestedQuery(value interface{}, prefix string) (string, error) {
	components := ""

	switch vv := value.(type) {
	case []interface{}:
		for i, v := range vv {
			component, err := buildNestedQuery(v, prefix+"[]")

			if err != nil {
				return "", err
			}

			components += component

			if i < len(vv)-1 {
				components += "&"
			}
		}

	case map[string]interface{}:
		length := len(vv)

		for k, v := range vv {
			childPrefix := ""

			if prefix != "" {
				childPrefix = prefix + "[" + url.QueryEscape(k) + "]"
			} else {
				childPrefix = url.QueryEscape(k)
			}

			component, err := buildNestedQuery(v, childPrefix)

			if err != nil {
				return "", err
			}

			components += component
			length -= 1

			if length > 0 {
				components += "&"
			}
		}

	case string:
		if prefix == "" {
			return "", fmt.Errorf("value must be a map[string]interface{}")
		}

		components += prefix + "=" + url.QueryEscape(vv)

	default:
		components += prefix
	}

	return components, nil
}
