package squish

import (
	"strconv"
)

// Compresses the 2nd numeric part of a string to a base-36 encoding to
// shorten the length of an id - so passing offset == 3 will compress leaving
// the Hob part of the id untouched
// This is needed for the loyalty service which must send id.s with relaively
// short max. length
func CompressTail36(offset int, in string) string {
	return in[:offset] + Compress36(in[offset:])
}

// The converse of the CompressTail36(..) that reflates the numeric part
// of an id to base-10 so it can be matched elsewhere in the system
func UncompressTail36(offset int, in string) string {
	return in[:offset] + Uncompress36(in[offset:])
}

// Compresses a numeric id string from base-10 to base-36
func Compress36(in string) string {
	return switchBase(10, 36, in)
}

// Uncompresses a numeric id string from base-36 to base-10
func Uncompress36(in string) string {
	return switchBase(36, 10, in)
}

func switchBase(bi, bo int, in string) string {
	ini, err := strconv.ParseUint(in, bi, 64)
	if err != nil {
		return in
	}

	return strconv.FormatUint(ini, bo)
}
