package files

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation"
)

// FileTypeLabelPrefix is the label prefix that records a file's user-defined
// type ("alerts", "metrics", …). It matches the key earlier operator versions
// used, so files typed before the feature was removed keep their type.
const FileTypeLabelPrefix = "file-type.krkn.krkn-chaos.dev/"

// ValidateFileType checks that a file type can be stored as a Kubernetes label
// value. An empty type is valid and means "untyped".
func ValidateFileType(fileType string) error {
	if fileType == "" {
		return nil
	}
	if problems := validation.IsValidLabelValue(fileType); len(problems) > 0 {
		return fmt.Errorf("invalid file type %q: %s", fileType, strings.Join(problems, "; "))
	}
	return nil
}

// ExtractFileTypeFromLabels returns the file type recorded on a file ConfigMap,
// or an empty string when the file is untyped.
func ExtractFileTypeFromLabels(labels map[string]string) string {
	for key, value := range labels {
		if value != "true" || !strings.HasPrefix(key, FileTypeLabelPrefix) {
			continue
		}
		return strings.TrimPrefix(key, FileTypeLabelPrefix)
	}
	return ""
}

// ApplyFileType replaces any file type label with the supplied type. An empty
// type removes the association.
func ApplyFileType(labels map[string]string, fileType string) {
	for key := range labels {
		if strings.HasPrefix(key, FileTypeLabelPrefix) {
			delete(labels, key)
		}
	}
	if fileType == "" {
		return
	}
	labels[FileTypeLabelPrefix+fileType] = "true"
}
