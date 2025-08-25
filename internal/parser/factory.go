package parser

import (
	"fmt"
	"path/filepath"
)

// NewParser creates a parser based on file extension or content
func NewParser(filename string) (Parser, error) {
	// First try by extension
	ext := filepath.Ext(filename)
	switch ext {
	case ".fit":
		return NewFITParser(), nil
	case ".tcx":
		return NewTCXParser(), nil // To be implemented
	case ".gpx":
		return NewGPXParser(), nil // To be implemented
	}

	// If extension doesn't match, detect by content
	fileType, err := DetectFileTypeFromFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to detect file type: %w", err)
	}

	switch fileType {
	case FIT:
		return NewFITParser(), nil
	case TCX:
		return NewTCXParser(), nil
	case GPX:
		return NewGPXParser(), nil
	default:
		return nil, fmt.Errorf("unsupported file type: %s", fileType)
	}
}

// NewParserFromData creates a parser based on file content
func NewParserFromData(data []byte) (Parser, error) {
	fileType := DetectFileTypeFromData(data)
	
	switch fileType {
	case FIT:
		return NewFITParser(), nil
	case TCX:
		return NewTCXParser(), nil
	case GPX:
		return NewGPXParser(), nil
	default:
		return nil, fmt.Errorf("unsupported file type: %s", fileType)
	}
}

// Placeholder implementations (will create these next)
func NewTCXParser() Parser { return nil }
func NewGPXParser() Parser { return nil }
