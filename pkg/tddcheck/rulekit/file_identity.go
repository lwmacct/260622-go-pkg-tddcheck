package rulekit

import "strings"

// FileIdentity is the canonical interpretation of a checked Go filename.
// Subject and Namespace are mutually exclusive.
type FileIdentity struct {
	Layer     string `json:"layer"`
	Subject   string `json:"subject,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Kind      string `json:"kind"`
}

// ParseFileIdentity parses base according to the layer's configured filename mode.
func ParseFileIdentity(layer string, base string, mode string) (FileIdentity, bool) {
	name, ok := strings.CutSuffix(base, ".go")
	if !ok {
		return FileIdentity{}, false
	}
	if mode == FileNameModePackageKind {
		if !validFileAtom(name) {
			return FileIdentity{}, false
		}
		return FileIdentity{Layer: layer, Kind: name}, true
	}
	parts := strings.Split(name, ".")
	switch {
	case len(parts) == 2 && parts[0] != "x" && validSnakeComponent(parts[0]) && validFileAtom(parts[1]):
		return FileIdentity{Layer: layer, Subject: parts[0], Kind: parts[1]}, true
	case len(parts) == 3 && parts[0] == "x" && validSnakeComponent(parts[1]) && validFileAtom(parts[2]):
		return FileIdentity{Layer: layer, Namespace: parts[1], Kind: parts[2]}, true
	default:
		return FileIdentity{}, false
	}
}

// Architecture reports whether the identity belongs to an architecture namespace.
func (i FileIdentity) Architecture() bool {
	return i.Namespace != ""
}

// Qualifier returns Namespace for architecture files and Subject otherwise.
func (i FileIdentity) Qualifier() string {
	if i.Namespace != "" {
		return i.Namespace
	}
	return i.Subject
}

func validFileAtom(value string) bool {
	if value == "" || value[0] < 'a' || value[0] > 'z' {
		return false
	}
	for index := 1; index < len(value); index++ {
		char := value[index]
		if (char < 'a' || char > 'z') && (char < '0' || char > '9') {
			return false
		}
	}
	return true
}

func validSnakeComponent(value string) bool {
	if value == "" || value[0] < 'a' || value[0] > 'z' {
		return false
	}
	previousUnderscore := false
	for index := 1; index < len(value); index++ {
		char := value[index]
		switch {
		case char == '_':
			if previousUnderscore {
				return false
			}
			previousUnderscore = true
		case (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9'):
			previousUnderscore = false
		default:
			return false
		}
	}
	return !previousUnderscore
}
