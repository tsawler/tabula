package tables

import (
	"github.com/tsawler/tabula/model"
)

// Detector is the interface for table detection algorithms
type Detector interface {
	// Detect finds tables in a page
	Detect(page *model.Page) ([]*model.Table, error)

	// Name returns the detector name
	Name() string

	// Configure sets detector parameters
	Configure(config Config) error
}

// Config holds detector configuration
type Config struct {
	// Minimum rows for a valid table
	MinRows int

	// Minimum columns for a valid table
	MinCols int

	// Minimum confidence threshold (0-1)
	MinConfidence float64

	// Whether to use line-based detection
	UseLines bool

	// Whether to use whitespace-based detection
	UseWhitespace bool

	// Maximum gap between text fragments to consider them in same cell (points)
	MaxCellGap float64

	// Tolerance for row/column alignment (points)
	AlignmentTolerance float64

	// Whether to detect merged cells
	DetectMergedCells bool
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		MinRows:            2,
		MinCols:            2,
		MinConfidence:      0.5,
		UseLines:           true,
		UseWhitespace:      true,
		MaxCellGap:         5.0,
		AlignmentTolerance: 2.0,
		DetectMergedCells:  true,
	}
}

// DetectorRegistry holds registered detectors
type DetectorRegistry struct {
	detectors map[string]Detector
}

// NewRegistry creates a new detector registry
func NewRegistry() *DetectorRegistry {
	return &DetectorRegistry{
		detectors: make(map[string]Detector),
	}
}

// Register registers a detector
func (r *DetectorRegistry) Register(detector Detector) {
	r.detectors[detector.Name()] = detector
}

// Get retrieves a detector by name
func (r *DetectorRegistry) Get(name string) Detector {
	return r.detectors[name]
}

// List returns all registered detector names
func (r *DetectorRegistry) List() []string {
	names := make([]string, 0, len(r.detectors))
	for name := range r.detectors {
		names = append(names, name)
	}
	return names
}

// Global registry
var globalRegistry = NewRegistry()

// RegisterDetector registers a detector globally
func RegisterDetector(detector Detector) {
	globalRegistry.Register(detector)
}

// GetDetector retrieves a detector by name
func GetDetector(name string) Detector {
	return globalRegistry.Get(name)
}

// ListDetectors returns all registered detector names
func ListDetectors() []string {
	return globalRegistry.List()
}

func init() {
	// Register default detectors
	RegisterDetector(NewGeometricDetector())
}
