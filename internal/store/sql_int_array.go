package store

import "github.com/lib/pq"

// intSliceParam converts nil to an empty slice so PostgreSQL cardinality() works as "no filter".
func intSliceParam(ids []int) interface{} {
	if ids == nil {
		ids = []int{}
	}
	return pq.Array(ids)
}
