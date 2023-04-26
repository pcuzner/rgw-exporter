// utils provides generic utility functions
package utils

import (
	"errors"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// TextToBytes receives a human readable string representing a capacity, and returns
// it as bytes (uint64)
func TextToBytes(textCapacity string) (uint64, error) {
	var err error = nil
	var size uint64 = 0
	var multiplier float64 = 0

	capacityRegex := regexp.MustCompile("([0-9]+)([A-Za-z]+)")
	matches := capacityRegex.FindStringSubmatch(textCapacity)

	if len(matches) == 3 {
		suffix := strings.ToLower(matches[2])
		switch suffix {
		case "k", "kb":
			multiplier = 1000
		case "kib":
			multiplier = 1024
		case "m", "mb":
			multiplier = math.Pow(1000, 2)
		case "mib":
			multiplier = math.Pow(1024, 2)
		case "g", "gb":
			multiplier = math.Pow(1000, 3)
		case "gib":
			multiplier = math.Pow(1024, 3)
		case "t", "tb":
			multiplier = math.Pow(1000, 4)
		case "tib":
			multiplier = math.Pow(1024, 4)
		default:
			err = errors.New("invalid capacity suffix - must be one of: k,kb,kib,m,mb,mib,g,gb,gib,t,tb,tib")
		}

		num, parseErr := strconv.ParseFloat(matches[1], 64)
		if parseErr != nil {
			err = parseErr
		}

		if err == nil {
			size = uint64(num * multiplier)
		}

	} else {
		err = errors.New("invalid size - expecting <int><suffix> format")
	}

	return size, err
}
