package main

import (
	"reflect"
	"testing"

	cloud "github.com/calyptia/api/types"
)

func Test_aggregatorsKeys(t *testing.T) {
	tt := []struct {
		name  string
		given []cloud.Aggregator
		want  []string
	}{
		{
			given: []cloud.Aggregator{{ID: "id-1", Name: "name-1"}, {ID: "id-2", Name: "name-2"}},
			want:  []string{"name-1", "name-2"},
		},
		{
			given: []cloud.Aggregator{{ID: "id-1", Name: "name"}, {ID: "id-2", Name: "name"}},
			want:  []string{"id-1", "id-2"},
		},
		{
			given: []cloud.Aggregator{{ID: "id-1", Name: "name"}, {ID: "id-2", Name: "name"}, {ID: "id-3", Name: "other-name"}},
			want:  []string{"id-1", "id-2", "other-name"},
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			if got := aggregatorsKeys(tc.given); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Aggregators.Keys(%+v) = %v, want %v", tc.given, got, tc.want)
			}
		})
	}
}
