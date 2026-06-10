package utils

import (
	"encoding/json"
	"testing"
)

func TestEmptyCollectionHelpersMarshalAsArrays(t *testing.T) {
	type payload struct {
		Data []int `json:"data"`
	}

	tests := map[string][]int{
		"each": Each[int, int](nil, func(value int) int {
			return value
		}),
		"each not nil": EachNotNil[int, int](nil, func(value int) *int {
			return &value
		}),
		"filter": Filter[int](nil, func(value int) bool {
			return value > 0
		}),
	}

	for name, data := range tests {
		body, err := json.Marshal(payload{Data: data})
		if err != nil {
			t.Fatalf("%s: marshal: %v", name, err)
		}

		if string(body) != `{"data":[]}` {
			t.Fatalf("%s: expected empty JSON array, got %s", name, body)
		}
	}
}
