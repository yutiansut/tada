package tada

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"reflect"
	"regexp"

	"github.com/olekukonko/tablewriter"
	"github.com/ptiger10/tablediff"
)

// -- CONSTRUCTORS

// NewDataFrame stub
func NewDataFrame(slices []interface{}, labels ...interface{}) *DataFrame {
	// handle values
	var values []*valueContainer
	for i, slice := range slices {
		if !isSlice(slice) {
			return &DataFrame{err: fmt.Errorf(
				"NewDataFrame(): unsupported kind (%v) in `slices` (position %v); must be slice", reflect.TypeOf(slice).Kind(), i)}
		}
		if reflect.ValueOf(slice).Len() == 0 {
			return &DataFrame{err: fmt.Errorf("NewDataFrame(): empty slice in slices (position %v): cannot be empty", i)}
		}
		isNull := setNullsFromInterface(slice)
		if isNull == nil {
			return &DataFrame{err: fmt.Errorf(
				"NewDataFrame(): unable to calculate null values ([]%v not supported)", reflect.TypeOf(slice).Elem())}
		}
		// handle special case of []Element: convert to []interface{}
		elements := handleElementsSlice(slice)
		if elements != nil {
			slice = elements
		}
		values = append(values, &valueContainer{slice: slice, isNull: isNull, name: fmt.Sprintf("%d", i)})
	}

	// handle labels
	retLabels := make([]*valueContainer, len(labels))
	if len(retLabels) == 0 {
		// handle default labels
		defaultLabels := makeDefaultLabels(0, reflect.ValueOf(slices[0]).Len())
		retLabels = append(retLabels, defaultLabels)
	} else {
		// handle supplied labels
		for i := range retLabels {
			slice := labels[i]
			if !isSlice(slice) {
				return dataFrameWithError(fmt.Errorf("NewDataFrame(): unsupported label kind (%v) at level %d; must be slice", reflect.TypeOf(slice), i))
			}
			isNull := setNullsFromInterface(slice)
			if isNull == nil {
				return dataFrameWithError(fmt.Errorf(
					"NewDataFrame(): unable to calculate null values at level %d ([]%v not supported)", i, reflect.TypeOf(slice).Elem()))
			}
			// handle special case of []Element: convert to []interface{}
			elements := handleElementsSlice(slice)
			if elements != nil {
				slice = elements
			}
			retLabels[i] = &valueContainer{slice: slice, isNull: isNull, name: fmt.Sprintf("*%d", i)}
		}
	}

	return &DataFrame{values: values, labels: retLabels, colLevelNames: []string{"*0"}}
}

// Copy stub
func (df *DataFrame) Copy() *DataFrame {
	values := make([]*valueContainer, len(df.values))
	for j := range df.values {
		values[j] = df.values[j].copy()
	}

	labels := make([]*valueContainer, len(df.labels))
	for j := range df.labels {
		labels[j] = df.labels[j].copy()
	}
	colLevelNames := make([]string, len(df.colLevelNames))
	copy(colLevelNames, df.colLevelNames)

	return &DataFrame{
		values:        values,
		labels:        labels,
		err:           df.err,
		colLevelNames: colLevelNames,
		name:          df.name,
	}
}

// ReadCSV stub
func ReadCSV(csv [][]string, config *ReadConfig) (*DataFrame, error) {
	if len(csv) == 0 {
		return nil, fmt.Errorf("ReadCSV(): csv must have at least one row")
	}
	if len(csv[0]) == 0 {
		return nil, fmt.Errorf("ReadCSV(): csv must have at least one column")
	}
	if config.MajorDimIsCols {
		return readCSVByCols(csv, config), nil
	}
	return readCSVByRows(csv, config), nil
}

// ImportCSV stub
func ImportCSV(path string, config *ReadConfig) (*DataFrame, error) {
	if config == nil {
		config = &ReadConfig{
			NumHeaderRows: 1,
		}
	}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("ImportCSV(): %s", err)
	}
	reader := csv.NewReader(bytes.NewReader(data))

	csv, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("ImportCSV(): %s", err)
	}

	if len(csv) == 0 {
		return nil, fmt.Errorf("ImportCSV(): csv must have at least one row")
	}
	if len(csv[0]) == 0 {
		return nil, fmt.Errorf("ImportCSV(): csv must have at least one column")
	}
	return readCSVByRows(csv, config), nil

}

// ReadInterface stub
func ReadInterface(input [][]interface{}, config *ReadConfig) (*DataFrame, error) {
	if config == nil {
		config = &ReadConfig{
			NumHeaderRows: 1,
		}
	}

	if len(input) == 0 {
		return nil, fmt.Errorf("ReadInterface(): `input` must have at least one row")
	}
	if len(input[0]) == 0 {
		return nil, fmt.Errorf("ReadInterface(): `input` must at least one column")
	}
	// convert [][]interface to [][]string
	str := make([][]string, len(input))
	for j := range str {
		str[j] = make([]string, len(input[0]))
	}
	for i := range input {
		for j := range input[i] {
			str[i][j] = fmt.Sprint(input[i][j])
		}
	}
	if config.MajorDimIsCols {
		return readCSVByCols(str, config), nil
	}
	return readCSVByRows(str, config), nil
}

// ReadMatrix stub
func ReadMatrix(mat Matrix) *DataFrame {
	numRows, numCols := mat.Dims()
	csv := make([][]string, numCols)
	for k := range csv {
		csv[k] = make([]string, numRows)
		for i := 0; i < numRows; i++ {
			csv[k][i] = fmt.Sprint(mat.At(i, k))
		}
	}
	return readCSVByCols(csv, &ReadConfig{})
}

// ReadStruct stub
func ReadStruct(slice interface{}) (*DataFrame, error) {
	values, err := readStruct(slice)
	if err != nil {
		return nil, fmt.Errorf("ReadStruct(): %v", err)
	}
	defaultLabels := makeDefaultLabels(0, reflect.ValueOf(slice).Len())
	return &DataFrame{
		values:        values,
		labels:        []*valueContainer{defaultLabels},
		colLevelNames: []string{"*0"},
	}, nil
}

// ToSeries stub
func (df *DataFrame) ToSeries() *Series {
	if len(df.values) != 1 {
		return seriesWithError(fmt.Errorf("ToSeries(): DataFrame must have a single column"))
	}
	return &Series{
		values: df.values[0],
		labels: df.labels,
	}
}

// ToCSV converts a DataFrame to a [][]string with rows as the major dimension
func (df *DataFrame) ToCSV(ignoreLabels bool) [][]string {
	transposedStringValues, err := df.toCSVByRows(ignoreLabels)
	if err != nil {
		return nil
	}
	return transposedStringValues
}

// ExportCSV converts a DataFrame to a [][]string with rows as the major dimension, and writes the output to a csv file.
func (df *DataFrame) ExportCSV(file string, ignoreLabels bool) error {
	transposedStringValues, err := df.toCSVByRows(ignoreLabels)
	if err != nil {
		return fmt.Errorf("ToCSV(): %v", err)
	}
	var b bytes.Buffer
	w := csv.NewWriter(&b)
	err = w.WriteAll(transposedStringValues)
	if err != nil {
		return fmt.Errorf("ToCSV(): %v", err)
	}
	// ducks error because process is controlled
	err = ioutil.WriteFile(file, b.Bytes(), 0666)
	if err != nil {
		return fmt.Errorf("ToCSV(): %v", err)
	}
	return nil
}

// ToInterface exports a DataFrame to a [][]interface with rows as the major dimension.
func (df *DataFrame) ToInterface(ignoreLabels bool) ([][]interface{}, error) {
	transposedStringValues, err := df.toCSVByRows(ignoreLabels)
	if err != nil {
		return nil, fmt.Errorf("ToInterface(): %v", err)
	}
	ret := make([][]interface{}, len(transposedStringValues))
	for k := range ret {
		ret[k] = make([]interface{}, len(transposedStringValues[0]))
	}
	for i := range transposedStringValues {
		for k := range transposedStringValues[i] {
			ret[i][k] = transposedStringValues[i][k]
		}
	}
	return ret, nil
}

// EqualsCSV converts a dataframe to csv, compares it to another csv, and evaluates whether the two match and isolates their differences
func (df *DataFrame) EqualsCSV(csv [][]string, ignoreLabels bool) (bool, *tablediff.Differences) {
	compare := df.ToCSV(ignoreLabels)
	diffs, eq := tablediff.Diff(compare, csv)
	return eq, diffs
}

// WriteMockCSV writes a mock csv to `w` modeled after `src`.
func WriteMockCSV(src [][]string, w io.Writer, config *ReadConfig, outputRows int) error {
	if config == nil {
		config = &ReadConfig{NumHeaderRows: 1}
	}
	numPreviewRows := 10
	inferredTypes := make([]map[DType]int, 0)
	var headers [][]string
	// default: major dimension is rows
	if !config.MajorDimIsCols {
		if len(src) == 0 {
			return fmt.Errorf("WriteMockCSV(): csv must have at least one row")
		}
		if len(src[0]) == 0 {
			return fmt.Errorf("WriteMockCSV(): csv must have at least one column")
		}
		maxRows := len(src) - config.NumHeaderRows
		if maxRows < numPreviewRows {
			numPreviewRows = maxRows
		}
		// copy headers
		for i := 0; i < config.NumHeaderRows; i++ {
			headers = append(headers, src[i])
		}
		for range src[0] {
			inferredTypes = append(inferredTypes, map[DType]int{Float: 0, DateTime: 0, String: 0})
		}

		// offset preview by header rows
		preview := src[config.NumHeaderRows : numPreviewRows+config.NumHeaderRows]
		for i := range preview {
			for k := range preview[i] {
				dtype := inferType(src[i+config.NumHeaderRows][k])
				inferredTypes[k][dtype]++
			}
		}
	} else {
		if len(src) == 0 {
			return fmt.Errorf("WriteMockCSV(): csv must have at least one column")
		}
		if len(src[0]) == 0 {
			return fmt.Errorf("WriteMockCSV(): csv must have at least one row")
		}
		maxRows := len(src[0])
		if maxRows < numPreviewRows {
			numPreviewRows = maxRows
		}
		// copy headers

		for range src {
			inferredTypes = append(inferredTypes, map[DType]int{Float: 0, DateTime: 0, String: 0})
		}

		for k := range src {
			headers = append(headers, make([]string, config.NumHeaderRows))
			for l := 0; l < config.NumHeaderRows; l++ {
				headers[k][l] = src[k][l]
			}
			// offset by header rows
			for i := range src[config.NumHeaderRows : numPreviewRows+config.NumHeaderRows] {
				dtype := inferType(src[k][i])
				inferredTypes[k][dtype]++
			}
		}
	}
	mockCSV := mockCSVFromDTypes(inferredTypes, outputRows)
	mockCSV = append(headers, mockCSV...)
	writer := csv.NewWriter(w)
	return writer.WriteAll(mockCSV)
}

// -- GETTERS

func removeDefaultNameIndicator(name string) string {
	return regexp.MustCompile(`^\*`).ReplaceAllString(name, "")
}

func (df *DataFrame) String() string {
	// do not try to print all rows
	csv := df.Head(optionMaxRows).ToCSV(false)
	for k := range csv[0] {
		csv[0][k] = removeDefaultNameIndicator(csv[0][k])
	}
	var caption string
	if df.name != "" {
		caption = fmt.Sprintf("name: %v", df.name)
	}
	var buf bytes.Buffer
	table := tablewriter.NewWriter(&buf)
	table.SetHeader(csv[0])
	table.AppendBulk(csv[1:])
	table.SetAutoMergeCells(optionAutoMerge)
	if caption != "" {
		table.SetCaption(true, caption)
	}
	table.Render()
	return string(buf.Bytes())
}

// Len returns the number of rows in each column of the DataFrame.
func (df *DataFrame) Len() int {
	return reflect.ValueOf(df.values[0].slice).Len()
}

// Err returns the most recent error attached to the DataFrame, if any.
func (df *DataFrame) Err() error {
	return df.err
}

// numLevels returns the number of label columns in the DataFrame.
func (df *DataFrame) numLevels() int {
	return len(df.labels)
}

func listNames(columns []*valueContainer) []string {
	ret := make([]string, len(columns))
	for k := range columns {
		ret[k] = columns[k].name
	}
	return ret
}

// ListColumns returns the name of all the columns in the DataFrame
func (df *DataFrame) ListColumns() []string {
	return listNames(df.values)
}

// ListLevels returns the name and position of all the label levels in the DataFrame
func (df *DataFrame) ListLevels() []string {
	return listNames(df.labels)
}

// HasCols returns an error if the DataFrame does not contain all of the `colNames` supplied.
func (df *DataFrame) HasCols(colNames ...string) error {
	for _, name := range colNames {
		_, err := findColWithName(name, df.values)
		if err != nil {
			return fmt.Errorf("HasCols(): %v", err)
		}
	}
	return nil
}

// InPlace returns a DataFrameMutator, which contains most of the same methods as DataFrame but never returns a new DataFrame.
// If you want to save memory and improve performance and do not need to preserve the original DataFrame, consider using InPlace().
func (df *DataFrame) InPlace() *DataFrameMutator {
	return &DataFrameMutator{dataframe: df}
}

// Subset returns only the rows specified at the index positions, in the order specified. Returns a new DataFrame.
func (df *DataFrame) Subset(index []int) *DataFrame {
	df = df.Copy()
	df.InPlace().Subset(index)
	return df
}

// Subset returns only the rows specified at the index positions, in the order specified.
// Modifies the underlying DataFrame in place.
func (df *DataFrameMutator) Subset(index []int) {
	if reflect.DeepEqual(index, []int{-999}) {
		df.dataframe.resetWithError(errors.New(
			"Subset(): invalid filter (every filter must have at least one filter function; if ColName is supplied, it must be valid)"))
	}
	for k := range df.dataframe.values {
		err := df.dataframe.values[k].subsetRows(index)
		if err != nil {
			df.dataframe.resetWithError(fmt.Errorf("Subset(): %v", err))
			return
		}
	}
	for j := range df.dataframe.labels {
		df.dataframe.labels[j].subsetRows(index)
	}
	return
}

// SubsetLabels returns only the labels specified at the index positions, in the order specified.
// Returns a new DataFrame.
func (df *DataFrame) SubsetLabels(index []int) *DataFrame {
	df = df.Copy()
	df.InPlace().SubsetLabels(index)
	return df
}

// SubsetLabels returns only the labels specified at the index positions, in the order specified.
// Modifies the underlying DataFrame in place.
func (df *DataFrameMutator) SubsetLabels(index []int) {
	labels, err := subsetCols(df.dataframe.labels, index)
	if err != nil {
		df.dataframe.resetWithError(fmt.Errorf("SubsetLabels(): %v", err))
		return
	}
	df.dataframe.labels = labels
	return
}

// SubsetCols returns only the labels specified at the index positions, in the order specified.
// Returns a new DataFrame.
func (df *DataFrame) SubsetCols(index []int) *DataFrame {
	df = df.Copy()
	df.InPlace().SubsetCols(index)
	return df
}

// SubsetCols returns only the labels specified at the index positions, in the order specified.
// Modifies the underlying DataFrame in place.
func (df *DataFrameMutator) SubsetCols(index []int) {
	cols, err := subsetCols(df.dataframe.values, index)
	if err != nil {
		df.dataframe.resetWithError(fmt.Errorf("SubsetCols(): %v", err))
		return
	}
	df.dataframe.values = cols
	return
}

// Col finds the first column with matching `name` and returns as a Series.
func (df *DataFrame) Col(name string) *Series {
	index, err := findColWithName(name, df.values)
	if err != nil {
		return seriesWithError(fmt.Errorf("Col(): %v", err))
	}
	return &Series{
		values: df.values[index],
		labels: df.labels,
	}
}

// Cols returns all column with matching `names`.
func (df *DataFrame) Cols(names ...string) *DataFrame {
	vals := make([]*valueContainer, len(names))
	for i, name := range names {
		index, err := findColWithName(name, df.values)
		if err != nil {
			return dataFrameWithError(fmt.Errorf("Cols(): %v", err))
		}
		vals[i] = df.values[index]
	}
	return &DataFrame{
		values: vals,
		labels: df.labels,
		name:   df.name,
	}
}

// Head returns the first `n` rows of the Series. If `n` is greater than the length of the Series, returns the entire Series.
// In either case, returns a new Series.
func (df *DataFrame) Head(n int) *DataFrame {
	if df.Len() < n {
		n = df.Len()
	}
	retVals := make([]*valueContainer, len(df.values))
	for k := range df.values {
		retVals[k] = df.values[k].head(n)
	}
	retLabels := make([]*valueContainer, df.numLevels())
	for j := range df.labels {
		retLabels[j] = df.labels[j].head(n)
	}
	return &DataFrame{values: retVals, labels: retLabels, name: df.name, colLevelNames: df.colLevelNames}
}

// Tail returns the last `n` rows of the Series. If `n` is greater than the length of the Series, returns the entire Series.
// In either case, returns a new Series.
func (df *DataFrame) Tail(n int) *DataFrame {
	if df.Len() < n {
		n = df.Len()
	}
	retVals := make([]*valueContainer, len(df.values))
	for k := range df.values {
		retVals[k] = df.values[k].tail(n)
	}
	retLabels := make([]*valueContainer, df.numLevels())
	for j := range df.labels {
		retLabels[j] = df.labels[j].tail(n)
	}
	return &DataFrame{values: retVals, labels: retLabels, name: df.name, colLevelNames: df.colLevelNames}
}

// Range returns the rows of the DataFrame starting at `first` and `ending` with last (inclusive).
// If either `first` or `last` is greater than the length of the DataFrame, a DataFrame error is returned.
// In all cases, returns a new DataFrame.
func (df *DataFrame) Range(first, last int) *DataFrame {
	if first >= df.Len() {
		return dataFrameWithError(fmt.Errorf("Range(): first index out of range (%d > %d)", first, df.Len()-1))
	} else if last >= df.Len() {
		return dataFrameWithError(fmt.Errorf("Range(): last index out of range (%d > %d)", last, df.Len()-1))
	}
	retVals := make([]*valueContainer, len(df.values))
	for k := range df.values {
		retVals[k] = df.values[k].rangeSlice(first, last)
	}
	retLabels := make([]*valueContainer, df.numLevels())
	for j := range df.labels {
		retLabels[j] = df.labels[j].rangeSlice(first, last)
	}
	return &DataFrame{values: retVals, labels: retLabels, name: df.name, colLevelNames: df.colLevelNames}
}

// DropNull removes rows with null values.
// If `subset` is supplied, removes any rows with null values in any of the specified columns.
// Returns a new DataFrame.
func (df *DataFrame) DropNull(subset ...string) *DataFrame {
	df = df.Copy()
	df.InPlace().DropNull(subset...)
	return df
}

// DropNull removes rows with null values.
// If `subset` is supplied, removes any rows with null values in any of the specified columns.
// Modifies the underlying DataFrame.
func (df *DataFrameMutator) DropNull(subset ...string) {
	var index []int
	if len(subset) == 0 {
		index = makeIntRange(0, len(df.dataframe.values))
	} else {
		for _, name := range subset {
			i, err := findColWithName(name, df.dataframe.values)
			if err != nil {
				df.dataframe.resetWithError(fmt.Errorf("DropNull(): %v", err))
			}
			index = append(index, i)
		}

	}

	subIndexes := make([][]int, len(index))
	for k := range index {
		subIndexes[k] = df.dataframe.values[k].valid()
	}
	allValid := intersection(subIndexes)
	df.Subset(allValid)
}

// Null returns all the rows with any null values.
// If `subset` is supplied, returns all the rows with all non-null values in the specified columns.
// Returns a new DataFrame.
func (df *DataFrame) Null(subset ...string) *DataFrame {
	var index []int
	if len(subset) == 0 {
		index = makeIntRange(0, len(df.values))
	} else {
		for _, name := range subset {
			i, err := findColWithName(name, df.values)
			if err != nil {
				return dataFrameWithError(fmt.Errorf("Valid(): %v", err))
			}
			index = append(index, i)
		}
	}

	subIndexes := make([][]int, len(index))
	for k := range index {
		subIndexes[k] = df.values[k].null()
	}
	anyNull := union(subIndexes)
	return df.Subset(anyNull)
}

// FilterCols returns the column positions of all columns (excluding labels) that satisfy `lambda`.
// If a column contains multiple levels, its name is a single pipe-delimited string and may be split within the lambda function.
func (df *DataFrame) FilterCols(lambda func(string) bool) []int {
	var ret []int
	for k := range df.values {
		if lambda(df.values[k].name) {
			ret = append(ret, k)
		}
	}
	return ret
}

// -- SETTERS

// WithLabels resolves as follows:
//
// If a scalar string is supplied as `input` and a column of labels exists that matches `name`: rename the level to match `input`
//
// If a slice is supplied as `input` and a column of labels exists that matches `name`: replace the values at this level to match `input`
//
// If a slice is supplied as `input` and a column of labels does not exist that matches `name`: append a new level with a name matching `name` and values matching `input`
//
// Error conditions: supplying slice of unsupported type, supplying slice with a different length than the underlying DataFrame, or supplying scalar string and `name` that does not match an existing label level.
// In all cases, returns a new DataFrame.
func (df *DataFrame) WithLabels(name string, input interface{}) *DataFrame {
	df.Copy()
	df.InPlace().WithLabels(name, input)
	return df
}

// WithLabels resolves as follows:
//
// If a scalar string is supplied as `input` and a column of labels exists that matches `name`: rename the level to match `input`
//
// If a slice is supplied as `input` and a column of labels exists that matches `name`: replace the values at this level to match `input`
//
// If a slice is supplied as `input` and a column of labels does not exist that matches `name`: append a new level with a name matching `name` and values matching `input`
//
// Error conditions: supplying slice of unsupported type, supplying slice with a different length than the underlying DataFrame, or supplying scalar string and `name` that does not match an existing label level.
// In all cases, modifies the underlying DataFrame in place.
func (df *DataFrameMutator) WithLabels(name string, input interface{}) {
	labels, err := withColumn(df.dataframe.labels, name, input, df.dataframe.Len())
	if err != nil {
		df.dataframe.resetWithError(fmt.Errorf("WithLabels(): %v", err))
	}
	df.dataframe.labels = labels
}

// WithCol resolves as follows:
//
// If a scalar string is supplied as `input` and a column exists that matches `name`: rename the column to match `input`
//
// If a slice is supplied as `input` and a column exists that matches `name`: replace the values at this column to match `input`
//
// If a slice is supplied as `input` and a column does not exist that matches `name`: append a new column with a name matching `name` and values matching `input`
//
// Error conditions: supplying slice of unsupported type, supplying slice with a different length than the underlying DataFrame, or supplying scalar string and `name` that does not match an existing label level.
// In all cases, returns a new DataFrame.
func (df *DataFrame) WithCol(name string, input interface{}) *DataFrame {
	df.Copy()
	df.InPlace().WithCol(name, input)
	return df
}

// WithCol resolves as follows:
//
// If a scalar string is supplied as `input` and a column exists that matches `name`: rename the column to match `input`
//
// If a slice is supplied as `input` and a column exists that matches `name`: replace the values at this column to match `input`
//
// If a slice is supplied as `input` and a column does not exist that matches `name`: append a new column with a name matching `name` and values matching `input`
//
// Error conditions: supplying slice of unsupported type, supplying slice with a different length than the underlying DataFrame, or supplying scalar string and `name` that does not match an existing label level.
// In all cases, modifies the underlying DataFrame in place.
func (df *DataFrameMutator) WithCol(name string, input interface{}) {
	cols, err := withColumn(df.dataframe.values, name, input, df.dataframe.Len())
	if err != nil {
		df.dataframe.resetWithError(fmt.Errorf("WithCol(): %v", err))
	}
	df.dataframe.values = cols
}

// WithRow stub
func (df *DataFrame) WithRow(label string, values []interface{}) *DataFrame {
	return nil
}

// DropCol drops the first column matching `name`
// Returns a new DataFrame.
func (df *DataFrame) DropCol(name string) *DataFrame {
	df.Copy()
	df.InPlace().DropCol(name)
	return df
}

// DropCol drops the first column matching `name`
func (df *DataFrameMutator) DropCol(name string) {
	toExclude, err := findColWithName(name, df.dataframe.values)
	if err != nil {
		df.dataframe.resetWithError(fmt.Errorf("DropCol(): %v", err))
	}
	index := excludeFromIndex(len(df.dataframe.values), toExclude)
	df.SubsetCols(index)
	return
}

// Drop removes the row at the specified index.
// Returns a new DataFrame.
func (df *DataFrame) Drop(index int) *DataFrame {
	df.Copy()
	df.InPlace().Drop(index)
	return df
}

// Drop removes the row at the specified index.
// Modifies the underlying DataFrame in place.
func (df *DataFrameMutator) Drop(index int) {
	for k := range df.dataframe.values {
		err := df.dataframe.values[k].dropRow(index)
		if err != nil {
			df.dataframe.resetWithError(fmt.Errorf("Drop(): %v", err))
			return
		}
	}
	for j := range df.dataframe.labels {
		df.dataframe.labels[j].dropRow(index)
	}
	return
}

// Append adds the `other` values as new rows to the DataFrame.
// Returns a new DataFrame.
func (df *DataFrame) Append(other *DataFrame) *DataFrame {
	df.Copy()
	df.InPlace().Append(other)
	return df
}

// Append adds the `other` values as new rows to the Series by coercing all values to string.
// Returns a new Series.
func (df *DataFrameMutator) Append(other *DataFrame) {
	if len(other.labels) != len(df.dataframe.labels) {
		df.dataframe.resetWithError(
			fmt.Errorf("other DataFrame must have same number of label levels as original DataFrame (%d != %d)",
				len(other.labels), len(df.dataframe.labels)))
	}
	if len(other.values) != len(df.dataframe.values) {
		df.dataframe.resetWithError(
			fmt.Errorf("other DataFrame must have same number of columns as original DataFrame (%d != %d)",
				len(other.values), len(df.dataframe.values)))
	}
	for j := range df.dataframe.labels {
		df.dataframe.labels[j] = df.dataframe.labels[j].append(other.labels[j])
	}
	for k := range df.dataframe.values {
		df.dataframe.values[k] = df.dataframe.values[k].append(other.values[k])
	}
	return
}

// SetLabels removes the row at the specified index.
// Returns a new DataFrame.
func (df *DataFrame) SetLabels(colNames ...string) *DataFrame {
	df.Copy()
	df.InPlace().SetLabels(colNames...)
	return df
}

// SetLabels appends the column(s) supplied as `colNames` as label levels and drops the column(s).
// The number of `colNames` supplied must be less than the number of columns in the Series.
// Modifies the underlying DataFrame in place.
func (df *DataFrameMutator) SetLabels(colNames ...string) {
	if len(colNames) >= len(df.dataframe.values) {
		df.dataframe.resetWithError(fmt.Errorf("SetLabels(): number of colNames must be less than number of columns (%d >= %d)",
			len(colNames), len(df.dataframe.values)))
	}
	for i := 0; i < len(colNames); i++ {
		index, err := findColWithName(colNames[i], df.dataframe.values)
		if err != nil {
			df.dataframe.resetWithError(fmt.Errorf("SetLabels(): %v", err))
		}
		df.dataframe.labels = append(df.dataframe.labels, df.dataframe.values[index])
		df.DropCol(colNames[i])
	}
	return
}

// ResetLabels appends the label level(s) at the supplied index levels as columns and drops the level.
// If no index levels are supplied, all label levels are appended as columns and dropped as levels, and replaced by a default label column.
// Returns a new DataFrame.
func (df *DataFrame) ResetLabels(index ...int) *DataFrame {
	df.Copy()
	df.InPlace().ResetLabels(index...)
	return df
}

// ResetLabels appends the label level(s) at the supplied index levels as columns and drops the level.
// If no index levels are supplied, all label levels are appended as columns and dropped as levels, and replaced by a default label column.
// Modifies the underlying DataFrame in place.
func (df *DataFrameMutator) ResetLabels(index ...int) {
	if len(index) == 0 {
		index = makeIntRange(0, df.dataframe.numLevels())
	}
	for _, i := range index {
		if i >= df.dataframe.numLevels() {
			df.dataframe.resetWithError(fmt.Errorf("ResetLabels(): index out of range (%d > %d)", i, df.dataframe.numLevels()-1))
		}
		newVal := df.dataframe.labels[i]
		// If label level name has default indicator, remove default indicator
		newVal.name = removeDefaultNameIndicator(newVal.name)
		df.dataframe.values = append(df.dataframe.values, newVal)
		df.dataframe.labels, _ = subsetCols(df.dataframe.labels, excludeFromIndex(df.dataframe.numLevels(), i))
	}
	if df.dataframe.numLevels() == 0 {
		defaultLabels := makeDefaultLabels(0, df.dataframe.Len())
		df.dataframe.labels[0] = defaultLabels
	}
	return
}

// SetName sets the name of a DataFrame and returns the entire DataFrame.
func (df *DataFrame) SetName(name string) *DataFrame {
	df.name = name
	return df
}

// Name returns the name of the DataFrame
func (df *DataFrame) Name() string {
	return df.name
}

// SetCols sets the names of all the columns in the DataFrame and returns the entire DataFrame.
func (df *DataFrame) SetCols(colNames []string) *DataFrame {
	if len(colNames) != len(df.values) {
		return dataFrameWithError(
			fmt.Errorf("SetCols(): number of colNames must match number of columns in DataFrame (%d != %d)", len(colNames), len(df.values)))
	}
	for k := range colNames {
		df.values[k].name = colNames[k]
	}
	return df
}

// reshape

func (df *DataFrame) numColLevels() int {
	return len(df.colLevelNames)
}

func (df *DataFrame) numColumns() int {
	return len(df.values)
}

// Transpose stub
func (df *DataFrame) Transpose() *DataFrame {
	// row values become column values: 2 row x 1 col -> 2 col x 1 row
	vals := make([][]string, df.Len())
	valsIsNull := make([][]bool, df.Len())
	// each new column has the same number of rows as prior columns
	for i := range vals {
		vals[i] = make([]string, df.numColumns())
		valsIsNull[i] = make([]bool, df.numColumns())
	}
	// label names become column names: 2 row x 1 level -> 2 col x 1 level
	colNames := make([][]string, df.Len())
	// each new column name has the same number of levels as prior label levels
	for i := range colNames {
		colNames[i] = make([]string, df.numLevels())
	}
	// column levels become label levels: 2 level x 1 col -> 2 level x 1 row
	labels := make([][]string, df.numColLevels())
	labelsIsNull := make([][]bool, df.numColLevels())

	// column level names become label level names
	labelNames := make([]string, df.numColLevels())
	// label level names become column level names
	colLevelNames := make([]string, df.numLevels())

	// each new label level has same number of rows as prior columns
	for l := range labels {
		labels[l] = make([]string, df.numColumns())
		labelsIsNull[l] = make([]bool, df.numColumns())
	}

	// iterate over labels to write column names and column level names
	for j := range df.labels {
		v := df.labels[j].str().slice
		for i := range v {
			colNames[i][j] = v[i]
		}
		colLevelNames[j] = df.labels[j].name
	}
	// iterate over column levels to write label level names
	for l := range df.colLevelNames {
		labelNames[l] = df.colLevelNames[l]
	}
	// iterate over columns
	for k := range df.values {
		// write label values
		splitColName := splitLabelIntoLevels(df.values[k].name, df.numColLevels() > 1)
		for l := range splitColName {
			labels[l][k] = splitColName[l]
			labelsIsNull[l][k] = false
		}
		// write values
		v := df.values[k].str().slice
		for i := range v {
			vals[i][k] = v[i]
			valsIsNull[i][k] = df.values[k].isNull[i]
		}
	}
	// transfer to valueContainers
	retLabels := make([]*valueContainer, df.numColLevels())
	retVals := make([]*valueContainer, df.Len())
	// labels
	for j := range labels {
		retLabels[j] = &valueContainer{
			slice:  labels[j],
			isNull: labelsIsNull[j],
			name:   labelNames[j],
		}
	}
	// values
	for k := range retVals {
		retVals[k] = &valueContainer{
			slice:  vals[k],
			isNull: valsIsNull[k],
			name:   joinLevelsIntoLabel(colNames[k]),
		}
	}
	return &DataFrame{
		values:        retVals,
		labels:        retLabels,
		name:          df.name,
		colLevelNames: colLevelNames,
	}
}

// PromoteToColLevel pivots either a column or label level into a new column level.
// If promoting would use either the last column or index level, it returns an error.
// Adds new columns - each unique value in the stacked column is stacked above each existing column.
// Can remove rows - returns only one row per unique label combination.
func (df *DataFrame) PromoteToColLevel(name string) *DataFrame {
	index, isCol, err := findNameInColumnsOrLabels(name, df.values, df.labels)
	if err != nil {
		return dataFrameWithError(fmt.Errorf("PromoteToColLevel(): %v", err))
	}
	var valsToPromote *valueContainer

	// by default, include all label levels
	residualLabelIndex := makeIntRange(0, len(df.labels))
	if isCol {
		if len(df.values) <= 1 {
			return dataFrameWithError(fmt.Errorf("PromoteToColLevel(): cannot stack only column"))
		}
		valsToPromote = df.values[index]
	} else {
		if len(df.labels) <= 1 {
			return dataFrameWithError(fmt.Errorf("PromoteToColLevel(): cannot stack only label level"))
		}
		valsToPromote = df.labels[index]
		// if a label is selected, remove it from the residual labels
		residualLabelIndex = excludeFromIndex(len(df.labels), index)
	}
	retName := valsToPromote.name
	// lookupSource maps the unique values in the promoted column to the rows with that value
	lookupSource, _, orderedKeys, _ := labelsToMap([]*valueContainer{valsToPromote}, []int{0})
	// rowToUniqueLabels maps each original row index to its matching index in the return container
	// this step consolidates duplicate residual labels
	_, _, uniqueLabels, rowToUniqueLabels := labelsToMap(df.labels, residualLabelIndex)

	// new labels will have as many columns as the residual label index
	newLabelNames := make([]string, len(residualLabelIndex))
	newLabels := make([][]string, len(residualLabelIndex))
	for j, pos := range residualLabelIndex {
		newLabels[j] = make([]string, len(uniqueLabels))
		for i := range uniqueLabels {
			splitNames := splitLabelIntoLevels(uniqueLabels[i], true)
			for _, name := range splitNames {
				newLabels[j][i] = name
			}
		}
		newLabelNames[j] = df.labels[pos].name
	}

	// new values will have as many columns as unique values in the column-to-be-stacked * existing columns
	// (minus the stacked column * existing columns, if a column is selected and not label level)
	newVals := make([][]string, len(lookupSource)*df.numColumns())
	colNames := make([]string, len(lookupSource)*df.numColumns())
	for k := range newVals {
		newVals[k] = make([]string, len(uniqueLabels))
	}
	// iterate over original columns -> unique values of stacked column -> row index of each unique value
	// compare to original value at column and row position
	// write to new row in container for each label value
	for k := 0; k < df.numColumns(); k++ {
		for m, orderedKey := range orderedKeys {
			newColumn := k*len(lookupSource) + m
			// skip column if it is derivative of the original
			if k == index && isCol {
				continue
			}
			newHeader := joinLevelsIntoLabel([]string{orderedKey, df.values[k].name})
			colNames[newColumn] = newHeader
			originalVals := df.values[k].str().slice
			for _, i := range lookupSource[orderedKey] {
				newRow := rowToUniqueLabels[i]
				newVals[newColumn][newRow] = originalVals[i]
			}
		}
	}
	// transfer values to final from
	retVals := make([]*valueContainer, len(newVals))
	for k := range retVals {
		retVals[k] = &valueContainer{
			slice:  newVals[k],
			isNull: setNullsFromInterface(newVals[k]),
			name:   colNames[k],
		}
	}
	// transfer labels to final from
	retLabels := make([]*valueContainer, len(newLabels))
	for j := range retLabels {
		retLabels[j] = &valueContainer{
			slice:  newLabels[j],
			isNull: setNullsFromInterface(newLabels[j]),
			name:   newLabelNames[j],
		}
	}
	// if a column is selected, drop cols that are a derivative of the original
	if isCol {
		retVals = append(retVals[:index*len(lookupSource)],
			retVals[index*len(lookupSource)+len(lookupSource):]...)
	}
	return &DataFrame{
		values:        retVals,
		labels:        retLabels,
		colLevelNames: append([]string{retName}, df.colLevelNames...),
		name:          df.name,
	}
}

// ResetColLevel pivots a column level to be a new label level.
// If unstacking would use the last column level, it fails.
// Removes columns by consolidating all lower level columns with the same name, and adds one label level.
// Unstacking does not change the number of rows.
// func (df *DataFrame) ResetColLevel(level int) *DataFrame {
// 	if len(df.colLevelNames) <= 1 {
// 		return dataFrameWithError(fmt.Errorf("ResetColLevel(): cannot unstack only column level"))
// 	}
// 	if level >= len(df.colLevelNames) {
// 		return dataFrameWithError(
// 			fmt.Errorf("ResetColLevel(): level out of range (%d > %d", level, len(df.colLevelNames)-1))

// 	}
// 	valPositions := make(map[string][]int)
// 	for k := range df.values {
// 		name := splitLabelIntoLevels(df.values[k].name)[level]
// 		// the first time the column header is found, isolate all the rows where it is valid
// 		if _, ok := valPositions[name]; !ok {
// 			valPositions[name] = make([]int, 0)
// 			for i := 0; i < df.Len(); i++ {
// 				if !df.values[k].isNull[i] {
// 					valPositions[name] = append(valPositions[name], i)
// 				}
// 			}
// 		}
// 	}

// 	newVals := make([][]string, )
// 	newLabels := make([]string, df.Len())

// 	return nil
// }

// -- FILTERS

// Filter stub
func (df *DataFrame) Filter(filters ...FilterFn) []int {
	if len(filters) == 0 {
		return makeIntRange(0, df.Len())
	}
	// subIndexes contains the index positions computed across all the filters
	var subIndexes [][]int
	for _, filter := range filters {
		// if ColName is empty, apply filter to all columns
		if filter.ColName == "" {
			var dfWideSubIndexes [][]int
			for k := range df.values {
				subIndex, err := df.values[k].filter(filter)
				if err != nil {
					return []int{-999}
				}
				dfWideSubIndexes = append(dfWideSubIndexes, subIndex)
			}
			subIndexes = append(subIndexes, intersection(dfWideSubIndexes))
			continue
		}
		// if ColName is not empty, find name in either columns or labels
		var data *valueContainer
		mergedLabelsAndCols := append(df.labels, df.values...)
		index, err := findColWithName(filter.ColName, mergedLabelsAndCols)
		if err != nil {
			return []int{-999}
		}
		data = mergedLabelsAndCols[index]

		subIndex, err := data.filter(filter)
		if err != nil {
			return []int{-999}
		}
		subIndexes = append(subIndexes, subIndex)
	}
	// reduce the subindexes to a single index that shares all the values
	index := intersection(subIndexes)
	return index
}

// -- APPLY

// Apply applies a user-defined `lambda` function to every row in a particular column and coerces all values to match the lambda type.
// Apply may be applied to any label level or column by specifying a ColName in `lambda`.
// If no ColName is specified in `lambda`, the function is applied to every column.
// If a value is considered null either prior to or after the lambda function is applied, it is considered null after.
// Returns a new DataFrame.
func (df *DataFrame) Apply(lambda ApplyFn) *DataFrame {
	df.Copy()
	df.InPlace().Apply(lambda)
	return df
}

// Apply applies a user-defined `lambda` function to every row in a particular column and coerces all values to match the lambda type.
// Apply may be applied to any label level or column by specifying a ColName in `lambda`.
// If no ColName is specified in `lambda`, the function is applied to every column.
// If a value is considered null either prior to or after the lambda function is applied, it is considered null after.
// Modifies the underlying DataFrame in place.
func (df *DataFrameMutator) Apply(lambda ApplyFn) {
	err := lambda.validate()
	if err != nil {
		df.dataframe.resetWithError((fmt.Errorf("Apply(): %v", err)))
		return
	}
	// if ColName is empty, apply lambda to all columns
	if lambda.ColName == "" {
		for k := range df.dataframe.values {
			df.dataframe.values[k].slice = df.dataframe.values[k].apply(lambda)
			df.dataframe.values[k].isNull = isEitherNull(
				df.dataframe.values[k].isNull,
				setNullsFromInterface(df.dataframe.values[k].slice))
		}
	} else {
		// if ColName is not empty, find name in either columns or labels
		mergedLabelsAndCols := append(df.dataframe.labels, df.dataframe.values...)
		index, err := findColWithName(lambda.ColName, mergedLabelsAndCols)
		if err != nil {
			df.dataframe.resetWithError((fmt.Errorf("Apply(): %v", err)))
		}
		mergedLabelsAndCols[index].slice = mergedLabelsAndCols[index].apply(lambda)
		mergedLabelsAndCols[index].isNull = isEitherNull(
			mergedLabelsAndCols[index].isNull,
			setNullsFromInterface(mergedLabelsAndCols[index].slice))
	}
	return
}

// ApplyFormat stub
func (df *DataFrame) ApplyFormat(lambda ApplyFormatFn) *DataFrame {
	df.Copy()
	df.InPlace().ApplyFormat(lambda)
	return df
}

// ApplyFormat stub
// Modifies the underlying DataFrame in place.
func (df *DataFrameMutator) ApplyFormat(lambda ApplyFormatFn) {
	err := lambda.validate()
	if err != nil {
		df.dataframe.resetWithError((fmt.Errorf("ApplyFormat(): %v", err)))
		return
	}
	// if ColName is empty, apply lambda to all columns
	if lambda.ColName == "" {
		for k := range df.dataframe.values {
			df.dataframe.values[k].slice = df.dataframe.values[k].applyFormat(lambda)
			df.dataframe.values[k].isNull = isEitherNull(
				df.dataframe.values[k].isNull,
				setNullsFromInterface(df.dataframe.values[k].slice))
		}
	} else {
		// if ColName is not empty, find name in either columns or labels
		mergedLabelsAndCols := append(df.dataframe.labels, df.dataframe.values...)
		index, err := findColWithName(lambda.ColName, mergedLabelsAndCols)
		if err != nil {
			df.dataframe.resetWithError((fmt.Errorf("ApplyFormat(): %v", err)))
		}
		mergedLabelsAndCols[index].slice = mergedLabelsAndCols[index].applyFormat(lambda)
		mergedLabelsAndCols[index].isNull = isEitherNull(
			mergedLabelsAndCols[index].isNull,
			setNullsFromInterface(mergedLabelsAndCols[index].slice))
	}
	return
}

// -- MERGERS

// Merge stub
func (df *DataFrame) Merge(other *DataFrame) *DataFrame {
	df.Copy()
	df.InPlace().Merge(other)
	return df
}

// Merge stub
func (df *DataFrameMutator) Merge(other *DataFrame) {
	lookupDF := df.dataframe.Lookup(other, "left", nil, nil)
	for k := range lookupDF.values {
		df.dataframe.values = append(df.dataframe.values, lookupDF.values[k])
	}
}

// Lookup stub
func (df *DataFrame) Lookup(other *DataFrame, how string, leftOn []string, rightOn []string) *DataFrame {
	mergedLabelsAndCols := append(df.labels, df.values...)
	otherMergedLabelsAndCols := append(other.labels, other.values...)
	var leftKeys, rightKeys []int
	var err error
	if len(leftOn) == 0 || len(rightOn) == 0 {
		if !(len(leftOn) == 0 && len(rightOn) == 0) {
			return dataFrameWithError(
				fmt.Errorf("Lookup(): if either leftOn or rightOn is empty, both must be empty"))
		}
	}
	if len(leftOn) == 0 {
		leftKeys, rightKeys = findMatchingKeysBetweenTwoLabelContainers(
			mergedLabelsAndCols, otherMergedLabelsAndCols)
	} else {
		leftKeys, err = convertColNamesToIndexPositions(leftOn, mergedLabelsAndCols)
		if err != nil {
			return dataFrameWithError(fmt.Errorf("Lookup(): %v", err))
		}
		rightKeys, err = convertColNamesToIndexPositions(rightOn, otherMergedLabelsAndCols)
		if err != nil {
			return dataFrameWithError(fmt.Errorf("Lookup(): %v", err))
		}
	}
	ret, err := lookupDataFrame(
		how, df.name, df.values, df.labels, leftKeys,
		other.values, other.labels, rightKeys, leftOn, rightOn)
	if err != nil {
		return dataFrameWithError(fmt.Errorf("Lookup(): %v", err))
	}
	return ret
}

// -- SORTERS

// Sort stub
func (df *DataFrame) Sort(by ...Sorter) *DataFrame {
	df.Copy()
	df.InPlace().Sort(by...)
	return df
}

// Sort stub
func (df *DataFrameMutator) Sort(by ...Sorter) {
	// original index
	index := makeIntRange(0, df.dataframe.Len())
	var vals *valueContainer
	// no Sorters supplied -> error
	if len(by) == 0 {
		df.dataframe.resetWithError(fmt.Errorf(
			"Sort(): must supply at least one Sorter"))
		return
	}
	for i := len(by) - 1; i >= 0; i-- {
		// Sorter with empty ColName -> error
		if by[i].ColName == "" {
			df.dataframe.resetWithError(fmt.Errorf(
				"Sort(): Sorter (position %d) must have ColName", i))
			return
		}

		mergedLabelsAndCols := append(df.dataframe.labels, df.dataframe.values...)
		colPosition, err := findColWithName(by[i].ColName, mergedLabelsAndCols)
		if err != nil {
			df.dataframe.resetWithError((fmt.Errorf("Sort(): %v", err)))
		}
		vals = mergedLabelsAndCols[colPosition].copy()
		index = vals.sort(by[i].DType, by[i].Descending, index)
	}
	df.Subset(index)
}

// -- GROUPERS

// GroupBy stub
// includes label levels and columns
func (df *DataFrame) GroupBy(names ...string) GroupedDataFrame {
	var index []int
	var err error
	mergedLabelsAndCols := append(df.labels, df.values...)
	// if no names supplied, group by all label levels and use all label level names
	if len(names) == 0 {
		index = makeIntRange(0, df.numLevels())
	} else {
		index, err = convertColNamesToIndexPositions(names, mergedLabelsAndCols)
		if err != nil {
			return GroupedDataFrame{err: fmt.Errorf("GroupBy(): %v", err)}
		}
	}
	return df.groupby(index)
}

// expects index to refer to merged labels and columns
func (df *DataFrame) groupby(index []int) GroupedDataFrame {
	mergedLabelsAndCols := append(df.labels, df.values...)
	g, _, orderedKeys, _ := labelsToMap(mergedLabelsAndCols, index)
	names := make([]string, len(index))
	for i, pos := range index {
		names[i] = mergedLabelsAndCols[pos].name
	}
	return GroupedDataFrame{
		groups:      g,
		orderedKeys: orderedKeys,
		df:          df,
		labelNames:  names,
	}
}

// PivotTable stub
func (df *DataFrame) PivotTable(labels, columns, values, aggFunc string) *DataFrame {

	mergedLabelsAndCols := append(df.labels, df.values...)
	labelIndex, err := findColWithName(labels, mergedLabelsAndCols)
	if err != nil {
		return dataFrameWithError(fmt.Errorf("PivotTable(): invalid labels: %v", err))
	}
	colIndex, err := findColWithName(columns, mergedLabelsAndCols)
	if err != nil {
		return dataFrameWithError(fmt.Errorf("PivotTable(): invalid columns: %v", err))
	}
	grouper := df.groupby([]int{labelIndex, colIndex})
	var ret *DataFrame
	switch aggFunc {
	case "sum":
		ret = grouper.Sum(values)
	case "mean":
		ret = grouper.Mean(values)
	case "median":
		ret = grouper.Median(values)
	case "std":
		ret = grouper.Std(values)
	default:
		return dataFrameWithError(fmt.Errorf("df.Pivot(): unsupported aggFunc (%v)", aggFunc))
	}
	if ret.err != nil {
		return dataFrameWithError(fmt.Errorf("df.Pivot(): %v", err))
	}
	ret = ret.PromoteToColLevel(columns)
	ret.dropColLevel(1)
	return ret
}

// inplace
func (df *DataFrame) dropColLevel(level int) *DataFrame {
	df.colLevelNames = append(df.colLevelNames[:level], df.colLevelNames[level+1:]...)
	for k := range df.values {
		priorNames := splitLabelIntoLevels(df.values[k].name, true)
		newNames := append(priorNames[:level], priorNames[level+1:]...)
		df.values[k].name = joinLevelsIntoLabel(newNames)
	}
	return df
}

// -- ITERATORS

// IterRows returns a slice of maps that return the underlying data for every row in the DataFrame.
// The key in each map is a column header, including label level headers.
// The value in each map is an Element containing an interface value and whether or not the value is null.
// If multiple label levels or columns have the same header, only the Elements of the right-most column are returned.
func (df *DataFrame) IterRows() []map[string]Element {
	ret := make([]map[string]Element, df.Len())
	for i := 0; i < df.Len(); i++ {
		// all label levels + all columns
		ret[i] = make(map[string]Element, df.numLevels()+len(df.values))
		for j := range df.labels {
			key := df.labels[j].name
			ret[i][key] = df.labels[j].iterRow(i)
		}
		for k := range df.values {
			key := df.values[k].name
			ret[i][key] = df.values[k].iterRow(i)
		}
	}
	return ret
}

// -- MATH

func (df *DataFrame) math(name string, mathFunction func([]float64, []bool, []int) (float64, bool)) *Series {
	retVals := make([]float64, len(df.values))
	retIsNull := make([]bool, len(df.values))
	labels := make([]string, len(df.values))
	labelsIsNull := make([]bool, len(df.values))

	for k := range df.values {
		retVals[k], retIsNull[k] = mathFunction(
			df.values[k].float().slice,
			df.values[k].isNull,
			makeIntRange(0, df.Len()))

		labels[k] = df.values[k].name
		labelsIsNull[k] = false
	}
	return &Series{
		values: &valueContainer{slice: retVals, isNull: retIsNull, name: name},
		labels: []*valueContainer{{slice: labels, isNull: labelsIsNull, name: "*0"}},
	}
}

// Sum coerces the values in each column to float64 and sums each column.
func (df *DataFrame) Sum() *Series {
	return df.math("sum", sum)
}

// Mean stub
func (df *DataFrame) Mean() *Series {
	return df.math("mean", mean)
}

// Median stub
func (df *DataFrame) Median() *Series {
	return df.math("median", median)
}

// Std stub
func (df *DataFrame) Std() *Series {
	return df.math("std", std)
}

// Count stub
func (df *DataFrame) Count() *Series {
	return df.math("count", count)
}

// Min stub
func (df *DataFrame) Min() *Series {
	return df.math("min", min)
}

// Max stub
func (df *DataFrame) Max() *Series {
	return df.math("max", max)
}
