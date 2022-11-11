package repo

// Windows is not macOS.
func isMacENOTTY(err error) bool { return false }
