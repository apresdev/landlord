package tinytools

// StringIf returns s if expr is true, otherwise it returns an empty string.
// Useful in templates.
func StringIf(expr bool, s string) string {
	if expr {
		return s
	}
	return ""
}
