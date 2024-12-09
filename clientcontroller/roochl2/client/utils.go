package client

import "strconv"

func Str2Uint64(str string) (uint64, error) {
	i, err := strconv.ParseInt(str, 10, 64)
	return uint64(i), err
}
