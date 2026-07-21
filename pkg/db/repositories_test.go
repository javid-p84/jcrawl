package db

import (
	"reflect"
	"testing"

	"github.com/lib/pq"
)

// Regression test for a bug where day_preference (INTEGER[]) failed to scan
// on every read (list/get) even though writes succeeded, because lib/pq's
// generic array Scan only supports []int64, not []int. See int64sToInts.
func TestInt64sToInts(t *testing.T) {
	tests := []struct {
		name string
		in   pq.Int64Array
		want []int
	}{
		{"nil", nil, nil},
		{"empty", pq.Int64Array{}, []int{}},
		{"values", pq.Int64Array{0, 5, 6}, []int{0, 5, 6}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := int64sToInts(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("int64sToInts(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
