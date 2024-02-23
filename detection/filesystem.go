package detection

import (
	"errors"
	"io/fs"
	"os"
	"slices"

	"github.com/anchordotdev/cli/anchorcli"
)

// FileDetector is a generic file-based project detector
type FileDetector struct {
	Title             string
	Paths             []string
	RequiredFiles     []string
	FollowUpDetectors []Detector
	AnchorCategory    *anchorcli.Category
}

// GetTitle returns the name of the detector
func (fd FileDetector) GetTitle() string {
	return fd.Title
}

// Detect checks if the directory contains any of the specified files
func (fd FileDetector) Detect(dirFS FS) (Match, error) {
	var matchedPaths []string

	for _, path := range fd.Paths {
		if _, err := dirFS.Stat(path); err == nil {
			matchedPaths = append(matchedPaths, path)
		} else if !os.IsNotExist(err) {
			return Match{}, errors.Join(err, errors.New("project file detection failure"))
		}
	}

	// Calculate the match confidence based on the percentage of matched paths
	percentage := float64(len(matchedPaths)) / float64(len(fd.Paths))
	var confidence Confidence

	// Assume a 25% window for each confidence level, anything less than 30% is None
	// Completely arbitrary, but it's a start.
	switch {
	case percentage >= 0.80:
		confidence = High
	case percentage >= 0.55:
		confidence = Medium
	case percentage >= 0.30:
		confidence = Low
	default:
		confidence = None
	}

	var missingRequiredFiles []string
	if len(matchedPaths) > 0 && fd.RequiredFiles != nil && len(fd.RequiredFiles) > 0 {
		for _, reqPath := range fd.RequiredFiles {
			if !slices.Contains(matchedPaths, reqPath) {
				missingRequiredFiles = append(missingRequiredFiles, reqPath)
				// Only lower confidence
				if confidence != None && confidence != Low {
					confidence = Low // Force confidence to low when required files are missing
				}
				continue
			}
		}
	}

	match := Match{
		Detector:             fd,
		Detected:             len(matchedPaths) > 0,
		Confidence:           confidence,
		FollowUpDetectors:    fd.FollowUpDetectors,
		MissingRequiredFiles: missingRequiredFiles,
	}

	if fd.AnchorCategory != nil {
		match.AnchorCategory = fd.AnchorCategory
	} else {
		// Default to Custom Category if not specified by the detector
		match.AnchorCategory = anchorcli.CategoryCustom
	}

	// Return a Match with the calculated confidence, follow-ups and category
	return match, nil
}

// FollowUp returns additional detectors
func (fd FileDetector) FollowUp() []Detector {
	return fd.FollowUpDetectors
}

// osFS is a simple wrapper around the os package's file system functions
// so that we can mock them out for testing.
type osFS struct{}

// Stat wraps os.Stat
func (osFS) Stat(path string) (fs.FileInfo, error) { return os.Stat(path) }
func (osFS) Open(path string) (fs.File, error)     { return os.Open(path) }

var (
	_ fs.FS = (*osFS)(nil)
)
