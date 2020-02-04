package tada

import (
	"math"
	"reflect"
	"strconv"
	"time"

	"github.com/araddon/dateparse"
)

// Less stub
func (vc FloatValueContainer) Less(i, j int) bool {
	if vc.Slice[i] < vc.Slice[j] {
		return true
	}
	return false
}

// Len stub
func (vc FloatValueContainer) Len() int {
	return len(vc.Slice)
}

// Swap stub
func (vc FloatValueContainer) Swap(i, j int) {
	vc.Slice[i], vc.Slice[j] = vc.Slice[j], vc.Slice[i]
	vc.IsNull[i], vc.IsNull[j] = vc.IsNull[j], vc.IsNull[i]
	vc.index[i], vc.index[j] = vc.index[j], vc.index[i]
}

// Less stub
func (vc StringValueContainer) Less(i, j int) bool {
	if vc.Slice[i] < vc.Slice[j] {
		return true
	}
	return false
}

// Len stub
func (vc StringValueContainer) Len() int {
	return len(vc.Slice)
}

// Swap stub
func (vc StringValueContainer) Swap(i, j int) {
	vc.Slice[i], vc.Slice[j] = vc.Slice[j], vc.Slice[i]
	vc.IsNull[i], vc.IsNull[j] = vc.IsNull[j], vc.IsNull[i]
	vc.index[i], vc.index[j] = vc.index[j], vc.index[i]
}

// Less stub
func (vc DateTimeValueContainer) Less(i, j int) bool {
	if vc.Slice[i].Before(vc.Slice[j]) {
		return true
	}
	return false
}

// Len stub
func (vc DateTimeValueContainer) Len() int {
	return len(vc.Slice)
}

// Swap stub
func (vc DateTimeValueContainer) Swap(i, j int) {
	vc.Slice[i], vc.Slice[j] = vc.Slice[j], vc.Slice[i]
	vc.IsNull[i], vc.IsNull[j] = vc.IsNull[j], vc.IsNull[i]
	vc.index[i], vc.index[j] = vc.index[j], vc.index[i]
}

// converters

func convertStringToFloat(val string, originalBool bool) (float64, bool) {
	parsedVal, err := strconv.ParseFloat(val, 64)
	if err == nil {
		return parsedVal, originalBool
	}
	return math.NaN(), true
}

func convertBoolToFloat(val bool) float64 {
	if val {
		return 1
	}
	return 0
}

func (vc *valueContainer) Float() FloatValueContainer {
	newVals := make([]float64, reflect.ValueOf(vc.slice).Len())
	isNull := vc.isNull
	switch vc.slice.(type) {
	case []float64:
		newVals = vc.slice.([]float64)

	case []string:
		arr := vc.slice.([]string)
		for i := range arr {
			newVals[i], isNull[i] = convertStringToFloat(arr[i], isNull[i])
		}

	case []time.Time:
		arr := vc.slice.([]time.Time)
		for i := range arr {
			// newVals[i] = float64(arr[i].UnixNano())
			newVals[i] = math.NaN()
			isNull[i] = true
		}

	case []bool:
		arr := vc.slice.([]bool)
		for i := range arr {
			newVals[i] = convertBoolToFloat(arr[i])
		}

	case []interface{}:
		arr := vc.slice.([]interface{})
		for i := range arr {
			switch arr[i].(type) {
			case string:
				newVals[i], isNull[i] = convertStringToFloat(arr[i].(string), isNull[i])
			case float32, float64:
				newVals[i] = reflect.ValueOf(arr[i]).Float()
			case int, int8, int16, int32, int64:
				newVals[i] = float64(reflect.ValueOf(arr[i]).Int())
			case uint, uint8, uint16, uint32, uint64:
				newVals[i] = float64(reflect.ValueOf(arr[i]).Uint())
			case time.Time:
				newVals[i] = math.NaN()
				isNull[i] = true
			case bool:
				newVals[i] = convertBoolToFloat(arr[i].(bool))
			}
		}

	case []uint, []uint8, []uint16, []uint32, []uint64, []int, []int8, []int16, []int32, []int64, []float32:
		d := reflect.ValueOf(vc.slice)
		for i := 0; i < d.Len(); i++ {
			v := d.Index(i).String()
			newVals[i], isNull[i] = convertStringToFloat(v, isNull[i])
		}
	}

	ret := FloatValueContainer{
		IsNull: isNull,
		Slice:  newVals,
		index:  makeIntRange(0, len(newVals)),
	}
	return ret
}

func (vc *valueContainer) Str() StringValueContainer {
	newVals := make([]string, reflect.ValueOf(vc.slice).Len())
	isNull := vc.isNull
	d := reflect.ValueOf(vc.slice)
	for i := 0; i < d.Len(); i++ {
		newVals[i] = d.Index(i).String()
	}
	ret := StringValueContainer{
		IsNull: isNull,
		Slice:  newVals,
		index:  makeIntRange(0, len(newVals)),
	}
	return ret
}

func convertStringToDateTime(val string, originalBool bool) (time.Time, bool) {
	parsedVal, err := dateparse.ParseAny(val)
	if err == nil {
		return parsedVal, originalBool
	}
	return time.Time{}, true
}

func (vc *valueContainer) DateTime() DateTimeValueContainer {
	newVals := make([]time.Time, reflect.ValueOf(vc.slice).Len())
	isNull := vc.isNull
	switch vc.slice.(type) {
	case []string:
		arr := vc.slice.([]string)
		for i := range arr {
			newVals[i], isNull[i] = convertStringToDateTime(arr[i], isNull[i])
		}
	case []time.Time:
		newVals = vc.slice.([]time.Time)
	case []interface{}:
		arr := vc.slice.([]interface{})
		for i := range arr {
			switch arr[i].(type) {
			case string:
				newVals[i], isNull[i] = convertStringToDateTime(arr[i].(string), isNull[i])
			case time.Time:
				newVals[i] = arr[i].(time.Time)
			default:
				newVals[i] = time.Time{}
				isNull[i] = true
			}
		}
	default:
		for i := range newVals {
			newVals[i] = time.Time{}
			isNull[i] = true
		}
	}
	ret := DateTimeValueContainer{
		IsNull: isNull,
		Slice:  newVals,
		index:  makeIntRange(0, len(newVals)),
	}
	return ret

}
