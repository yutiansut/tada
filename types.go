package tada

import "time"

type valueContainer struct {
	slice  interface{}
	name   string
	isNull []bool
}

// Series stub
type Series struct {
	values *valueContainer
	labels []*valueContainer
	err    error
}

// SeriesMutator stub
type SeriesMutator struct {
	series *Series
}

// DataFrame stub
type DataFrame struct {
	labels []*valueContainer
	values []*valueContainer
	name   string
	err    error
}

type floatValueContainer struct {
	slice  []float64
	isNull []bool
	index  []int
}

type stringValueContainer struct {
	slice  []string
	isNull []bool
	index  []int
}

type dateTimeValueContainer struct {
	slice  []time.Time
	isNull []bool
	index  []int
}

// Sorter stub
type Sorter struct {
	ColName    string
	Descending bool
	DType      DType
}

// Element stub
type Element struct {
	val    interface{}
	isNull bool
}

// GroupedSeries stub
type GroupedSeries struct {
	groups    map[string][]int
	reference *Series
	Err       error
}

// GroupedDataFrame stub
type GroupedDataFrame struct {
	groups    map[string][]int
	reference *DataFrame
	Err       error
}

// DType stub
type DType int

const (
	// Float stub
	Float DType = iota
	// String stub
	String
	// DateTime stub
	DateTime
)

// Dimension stub
type Dimension int

const (
	// Columns stub
	Columns Dimension = iota
	// Rows stub
	Rows
)

func (dim Dimension) String() string {
	return [...]string{"columns", "rows"}[dim]
}
