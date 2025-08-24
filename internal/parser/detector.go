// internal/parser/detector.go
package parser

import (
    "bytes"
    "os"
)

type FileType string

const (
    FileTypeFIT     FileType = "fit"
    FileTypeTCX     FileType = "tcx"
    FileTypeGPX     FileType = "gpx"
    FileTypeUnknown FileType = "unknown"
)

func DetectFileType(filepath string) (FileType, error) {
    file, err := os.Open(filepath)
    if err != nil {
        return FileTypeUnknown, err
    }
    defer file.Close()
    
    // Read first 512 bytes for detection
    header := make([]byte, 512)
    n, err := file.Read(header)
    if err != nil && n == 0 {
        return FileTypeUnknown, err
    }
    
    header = header[:n]
    
    return DetectFileTypeFromData(header), nil
}

func DetectFileTypeFromData(data []byte) FileType {
    // Check for FIT file signature
    if len(data) >= 8 && bytes.Equal(data[8:12], []byte(".FIT")) {
        return FileTypeFIT
    }
    
    // Check for XML-based formats
    if bytes.HasPrefix(data, []byte("<?xml")) {
        if bytes.Contains(data[:200], []byte("<gpx")) ||
           bytes.Contains(data[:200], []byte("topografix.com/GPX")) {
            return FileTypeGPX
        }
        if bytes.Contains(data[:500], []byte("TrainingCenterDatabase")) {
            return FileTypeTCX
        }
    }
    
    return FileTypeUnknown
}
