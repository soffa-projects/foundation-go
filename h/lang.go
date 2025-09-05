package h

import "strconv"

func ToInt(input string) int {
	return Safe(strconv.Atoi(input))
}
