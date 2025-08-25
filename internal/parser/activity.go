package parser

import (
	"time"
	
	"github.com/sstent/garminsync-go/internal/models"
)

// ActivityMetrics is now defined in internal/models

// Parser defines the interface for activity file parsers
type Parser interface {
	ParseFile(filename string) (*models.ActivityMetrics, error)
}

// FileType represents supported file formats
type FileType string

const (
	FIT FileType = "fit"
	TCX FileType = "tcx"
	GPX FileType = "gpx"
)
