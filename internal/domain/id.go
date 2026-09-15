package domain

// ValidID reports whether s is a UUID in the canonical 36-character form, the
// only form the API issues. Every id column is a UUID, and Postgres rejects
// anything else with an error the repositories report as a server fault, so
// callers check an id before it reaches a query.
func ValidID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i := 0; i < len(s); i++ {
		switch i {
		case 8, 13, 18, 23:
			if s[i] != '-' {
				return false
			}
		default:
			if !isHex(s[i]) {
				return false
			}
		}
	}
	return true
}

func isHex(c byte) bool {
	return '0' <= c && c <= '9' || 'a' <= c && c <= 'f' || 'A' <= c && c <= 'F'
}
