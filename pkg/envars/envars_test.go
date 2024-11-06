package envars

import (
	"reflect"
	"strconv"
	"testing"
)

func TestToMap(t *testing.T) {
	env := []string{"VAR1=value1", "VAR2=value2", "VAR3=value3"}

	expected := map[string]string{"VAR1": "value1", "VAR2": "value2", "VAR3": "value3"}

	result := ToMap(env)

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("ToMap(%v) = %v; expected %v", env, result, expected)
	}
}

func TestToSlice(t *testing.T) {
	m := map[string]string{"VAR1": "value1", "VAR2": "value2"}
	quote := true

	result := ToSlice(quote, m)

	expected1 := []string{"VAR1=" + strconv.Quote("value1"), "VAR2=" + strconv.Quote("value2")}
	expected2 := []string{"VAR2=" + strconv.Quote("value2"), "VAR1=" + strconv.Quote("value1")}

	if !reflect.DeepEqual(result, expected1) && !reflect.DeepEqual(result, expected2) {
		t.Errorf("ToSlice(%v, %v) = %v; expected %v or %v", quote, m, result, expected1, expected2)
	}
}

func TestUniq(t *testing.T) {
	x := map[string]string{"VAR1": "value1", "VAR2": "value2", "VAR3": "value3"}
	y := map[string]string{"VAR2": "value2", "VAR3": "different_value", "VAR4": "value4"}

	// Test duplicates = false (unique vars only)
	expectedUnique := map[string]string{"VAR1": "value1", "VAR3": "value3"}

	resultUnique := Uniq(false, x, y)

	if !reflect.DeepEqual(resultUnique, expectedUnique) {
		t.Errorf("Uniq(false, %v, %v) = %v; expected %v", x, y, resultUnique, expectedUnique)
	}

	// Test duplicates = true (duplicates only)
	expectedDuplicates := map[string]string{"VAR2": "value2"}

	resultDuplicates := Uniq(true, x, y)

	if !reflect.DeepEqual(resultDuplicates, expectedDuplicates) {
		t.Errorf("Uniq(true, %v, %v) = %v; expected %v", x, y, resultDuplicates, expectedDuplicates)
	}
}

func TestUniqKeys(t *testing.T) {
	x := map[string]string{"VAR1": "value1", "VAR2": "value2", "VAR3": "value3"}
	y := map[string]string{"VAR2": "value2", "VAR4": "value4"}

	expected := map[string]string{"VAR1": "value1", "VAR3": "value3"}

	result := UniqKeys(x, y)

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("UniqKeys(%v, %v) = %v; expected %v", x, y, result, expected)
	}
}

func TestMerge(t *testing.T) {
	m1 := map[string]string{"VAR1": "value1", "VAR2": "value2"}
	m2 := map[string]string{"VAR2": "new_value2", "VAR3": "value3"}
	m3 := map[string]string{"VAR4": "value4"}

	expected := map[string]string{"VAR1": "value1", "VAR2": "new_value2", "VAR3": "value3", "VAR4": "value4"}

	result := Merge(m1, m2, m3)

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Merge(%v, %v, %v) = %v; expected %v", m1, m2, m3, result, expected)
	}
}
