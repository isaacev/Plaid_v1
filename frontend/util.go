package frontend

import (
    "strconv"
)

func toOrdinal(n int) string {
	suffix := "th"

	switch n % 10 {
	case 1:
		if n%100 != 11 {
			suffix = "st"
		}
	case 2:
		if n%100 != 11 {
			suffix = "nd"
		}
	case 3:
		if n%100 != 11 {
			suffix = "rd"
		}
	}

	return strconv.Itoa(n) + suffix
}
