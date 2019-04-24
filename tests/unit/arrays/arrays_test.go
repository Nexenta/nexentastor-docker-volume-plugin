package arrays_test

import (
	"regexp"
	"testing"

	"github.com/Nexenta/nexentastor-docker-volume-plugin/pkg/arrays"
)

func TestContainsString(t *testing.T) {
	a := []string{"a", "b", "c"}

	if !arrays.ContainsString(a, "a") {
		t.Error("should return true if value exists")
	}

	if arrays.ContainsString(a, "e") {
		t.Error("should return false if value doesn't exist")
	}
}

func TestFindRegexpIndexesString(t *testing.T) {
	a := []string{"aaa", "b11", "b22"}

	indexes := arrays.FindRegexpIndexesString(a, regexp.MustCompile("^b.*$"))
	if len(indexes) != 2 {
		t.Error("should have two elements")
	} else if indexes[0] != 1 || indexes[1] != 2 {
		t.Errorf("should have two elements [1, 2], but got: %v", indexes)
	}

	indexes = arrays.FindRegexpIndexesString(a, regexp.MustCompile("^c$"))
	if len(indexes) != 0 {
		t.Errorf("should have no elements, instead got: %v", indexes)
	}
}

func TestAppendIfRegexpNotExistString(t *testing.T) {
	a := []string{"aaa", "bbb", "ccc"}

	newA := arrays.AppendIfRegexpNotExistString(a, regexp.MustCompile("^e*$"), "eee")
	if len(newA) != 4 {
		t.Error("should append non-existing value")
	}

	newA = arrays.AppendIfRegexpNotExistString(a, regexp.MustCompile("^c*$"), "ccc")
	if len(newA) != 3 {
		t.Errorf("should not append/remove value from array if it already exists, got: %v", newA)
	}
}
