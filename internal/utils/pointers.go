package utils

// StringOrNil returns nil if the input string is empty.
// Otherwise, it returns a pointer to the string.
// This is extremely useful for sqlc nullable string columns.
func StringOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// Ptr is an optional helper for direct pointer creation without the empty check
func Ptr[T any](v T) *T {
	return &v
}
