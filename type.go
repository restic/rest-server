package resticserver

import "errors"

// ValidTypes is a list of valid types
var ValidTypes = []string{"data", "index", "keys", "locks", "snapshots", "config"}

// IsValidType checks wether a type is valid
func IsValidType(name string) error {
	for _, valid := range ValidTypes {
		if name == valid {
			return nil
		}
	}
	return errors.New("invalid file type")
}

func isHashed(name string) bool {
	return name == "data"
}
