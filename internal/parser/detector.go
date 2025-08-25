package parser

import (
	"bytes"
	"errors"
)

var (
	// FIT file signature
	fitSignature = []byte{0x0E, 0x10} // .FIT files start with 0x0E 0x10
)

// DetectFileType detects the file type based on its content
func DetectFileType(data []byte) (string, error) {
	// Check FIT file signature
	if len(data) >= 2 && bytes.Equal(data[:2], fitSignature) {
		return ".fit", nil
	}

	// Check TCX file signature (XML with TrainingCenterDatabase root)
	if bytes.Contains(data, []byte("<TrainingCenterDatabase")) {
		return ".tcx", nil
	}

	// Check GPX file signature (XML with <gpx> root)
	if bytes.Contains(data, []byte("<gpx")) {
		return ".gpx", nil
	}

	return "", errors.New("unrecognized file format")
}
