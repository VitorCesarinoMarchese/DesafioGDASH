package utils

// Truncate truncates a byte slice to max length and adds truncation indicator
func Truncate(b []byte, max int) []byte {
	if len(b) <= max {
		return b
	}
	return append(b[:max], []byte("...[truncated]")...)
}