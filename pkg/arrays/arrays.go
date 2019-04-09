package arrays

import (
	"regexp"
)

// ContainsString - returns true if an array contains a string, otherwise false
func ContainsString(array []string, value string) bool {
	for _, v := range array {
		if v == value {
			return true
		}
	}
	return false
}

// FindRegexpIndexesString - finds all indexes of RegExp matching values in the array
func FindRegexpIndexesString(array []string, re *regexp.Regexp) (indexes []int) {
	for i, v := range array {
		if re.MatchString(v) {
			indexes = append(indexes, i)
		}
	}
	return indexes
}

// AppendIfRegexpNotExistString - returns new array with inserted new value if RegExp doesn't match any existing value
func AppendIfRegexpNotExistString(array []string, re *regexp.Regexp, value string) []string {
	indexes := FindRegexpIndexesString(array, re)
	if len(indexes) == 0 {
		return append(array, value)
	}
	return array
}
