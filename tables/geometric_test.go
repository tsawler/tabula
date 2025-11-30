package tables

import (
	"testing"

	"github.com/tsawler/tabula/model"
)

func TestNewGeometricDetector(t *testing.T) {
	d := NewGeometricDetector()
	if d == nil {
		t.Fatal("NewGeometricDetector() returned nil")
	}
}

func TestGeometricDetector_Name(t *testing.T) {
	d := NewGeometricDetector()
	if name := d.Name(); name != "geometric" {
		t.Errorf("Name() = %q, want 'geometric'", name)
	}
}

func TestGeometricDetector_Configure(t *testing.T) {
	d := NewGeometricDetector()

	config := Config{
		MinRows:            3,
		MinCols:            3,
		MinConfidence:      0.7,
		AlignmentTolerance: 5.0,
	}

	err := d.Configure(config)
	if err != nil {
		t.Errorf("Configure() failed: %v", err)
	}

	if d.config.MinRows != 3 {
		t.Errorf("MinRows = %d, want 3", d.config.MinRows)
	}
	if d.config.MinConfidence != 0.7 {
		t.Errorf("MinConfidence = %f, want 0.7", d.config.MinConfidence)
	}
}

func TestGeometricDetector_Detect_EmptyPage(t *testing.T) {
	d := NewGeometricDetector()
	page := &model.Page{}

	tables, err := d.Detect(page)
	if err != nil {
		t.Errorf("Detect() failed: %v", err)
	}
	if tables != nil {
		t.Errorf("Detect() on empty page should return nil, got %d tables", len(tables))
	}
}

func TestGeometricDetector_Detect_WithFragments(t *testing.T) {
	d := NewGeometricDetector()

	// Create a page with text fragments arranged in a grid pattern
	page := &model.Page{
		Width:  612,
		Height: 792,
		RawText: []model.TextFragment{
			// Row 1
			{Text: "A1", BBox: model.BBox{X: 100, Y: 700, Width: 50, Height: 15}},
			{Text: "B1", BBox: model.BBox{X: 200, Y: 700, Width: 50, Height: 15}},
			{Text: "C1", BBox: model.BBox{X: 300, Y: 700, Width: 50, Height: 15}},
			// Row 2
			{Text: "A2", BBox: model.BBox{X: 100, Y: 680, Width: 50, Height: 15}},
			{Text: "B2", BBox: model.BBox{X: 200, Y: 680, Width: 50, Height: 15}},
			{Text: "C2", BBox: model.BBox{X: 300, Y: 680, Width: 50, Height: 15}},
			// Row 3
			{Text: "A3", BBox: model.BBox{X: 100, Y: 660, Width: 50, Height: 15}},
			{Text: "B3", BBox: model.BBox{X: 200, Y: 660, Width: 50, Height: 15}},
			{Text: "C3", BBox: model.BBox{X: 300, Y: 660, Width: 50, Height: 15}},
		},
	}

	tables, err := d.Detect(page)
	if err != nil {
		t.Errorf("Detect() failed: %v", err)
	}

	// The detector should find a table pattern
	t.Logf("Found %d tables", len(tables))
}

func TestGeometricDetector_Detect_TooFewFragments(t *testing.T) {
	d := NewGeometricDetector()

	// Only 2 fragments - not enough for a 2x2 table
	page := &model.Page{
		Width:  612,
		Height: 792,
		RawText: []model.TextFragment{
			{Text: "A", BBox: model.BBox{X: 100, Y: 700, Width: 50, Height: 15}},
			{Text: "B", BBox: model.BBox{X: 200, Y: 700, Width: 50, Height: 15}},
		},
	}

	tables, err := d.Detect(page)
	if err != nil {
		t.Errorf("Detect() failed: %v", err)
	}

	// Should not find a table with so few fragments
	if len(tables) > 0 {
		t.Logf("Found %d tables with only 2 fragments", len(tables))
	}
}

func TestClusterFragments(t *testing.T) {
	d := NewGeometricDetector()

	fragments := []model.TextFragment{
		// Cluster 1 - top
		{Text: "A", BBox: model.BBox{X: 100, Y: 700, Width: 50, Height: 15}},
		{Text: "B", BBox: model.BBox{X: 200, Y: 700, Width: 50, Height: 15}},
		{Text: "C", BBox: model.BBox{X: 100, Y: 680, Width: 50, Height: 15}},
		// Cluster 2 - separated by large gap
		{Text: "D", BBox: model.BBox{X: 100, Y: 500, Width: 50, Height: 15}},
		{Text: "E", BBox: model.BBox{X: 200, Y: 500, Width: 50, Height: 15}},
	}

	clusters := d.clusterFragments(fragments)

	if len(clusters) != 2 {
		t.Errorf("Expected 2 clusters, got %d", len(clusters))
	}

	if len(clusters) >= 2 {
		if len(clusters[0]) != 3 {
			t.Errorf("First cluster should have 3 fragments, got %d", len(clusters[0]))
		}
		if len(clusters[1]) != 2 {
			t.Errorf("Second cluster should have 2 fragments, got %d", len(clusters[1]))
		}
	}
}

func TestClusterFragments_Empty(t *testing.T) {
	d := NewGeometricDetector()

	clusters := d.clusterFragments(nil)
	if clusters != nil {
		t.Errorf("Expected nil for empty input, got %v", clusters)
	}

	clusters = d.clusterFragments([]model.TextFragment{})
	if clusters != nil {
		t.Errorf("Expected nil for empty slice, got %v", clusters)
	}
}

func TestClusterFragments_Single(t *testing.T) {
	d := NewGeometricDetector()

	fragments := []model.TextFragment{
		{Text: "A", BBox: model.BBox{X: 100, Y: 700, Width: 50, Height: 15}},
	}

	clusters := d.clusterFragments(fragments)

	if len(clusters) != 1 {
		t.Errorf("Expected 1 cluster, got %d", len(clusters))
	}
}

func TestExtractRowBoundaries(t *testing.T) {
	d := NewGeometricDetector()

	fragments := []model.TextFragment{
		{Text: "A", BBox: model.BBox{X: 100, Y: 100, Width: 50, Height: 20}},
		{Text: "B", BBox: model.BBox{X: 100, Y: 70, Width: 50, Height: 20}},
		{Text: "C", BBox: model.BBox{X: 100, Y: 40, Width: 50, Height: 20}},
	}

	boundaries := d.extractRowBoundaries(fragments)

	// Should have boundaries for top and bottom of each row
	if len(boundaries) < 3 {
		t.Errorf("Expected at least 3 row boundaries, got %d", len(boundaries))
	}

	// Boundaries should be sorted descending (PDF coordinates)
	for i := 1; i < len(boundaries); i++ {
		if boundaries[i] > boundaries[i-1] {
			t.Errorf("Row boundaries should be sorted descending")
			break
		}
	}
}

func TestExtractRowBoundaries_Empty(t *testing.T) {
	d := NewGeometricDetector()

	boundaries := d.extractRowBoundaries(nil)
	if boundaries != nil {
		t.Errorf("Expected nil for empty input")
	}
}

func TestExtractColumnBoundaries(t *testing.T) {
	d := NewGeometricDetector()

	fragments := []model.TextFragment{
		{Text: "A", BBox: model.BBox{X: 100, Y: 100, Width: 50, Height: 20}},
		{Text: "B", BBox: model.BBox{X: 200, Y: 100, Width: 50, Height: 20}},
		{Text: "C", BBox: model.BBox{X: 300, Y: 100, Width: 50, Height: 20}},
	}

	boundaries := d.extractColumnBoundaries(fragments)

	// Should have boundaries for left and right of each column
	if len(boundaries) < 3 {
		t.Errorf("Expected at least 3 column boundaries, got %d", len(boundaries))
	}

	// Boundaries should be sorted ascending
	for i := 1; i < len(boundaries); i++ {
		if boundaries[i] < boundaries[i-1] {
			t.Errorf("Column boundaries should be sorted ascending")
			break
		}
	}
}

func TestExtractColumnBoundaries_Empty(t *testing.T) {
	d := NewGeometricDetector()

	boundaries := d.extractColumnBoundaries(nil)
	if boundaries != nil {
		t.Errorf("Expected nil for empty input")
	}
}

func TestClusterValues(t *testing.T) {
	d := NewGeometricDetector()

	tests := []struct {
		name      string
		values    []float64
		tolerance float64
		minLen    int
	}{
		{
			name:      "distinct values",
			values:    []float64{10, 20, 30, 40},
			tolerance: 2.0,
			minLen:    4,
		},
		{
			name:      "clustered values",
			values:    []float64{10, 10.5, 11, 20, 20.5, 21},
			tolerance: 2.0,
			minLen:    2, // Should cluster into ~2 groups
		},
		{
			name:      "single value",
			values:    []float64{10},
			tolerance: 2.0,
			minLen:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := d.clusterValues(tt.values, tt.tolerance)
			if len(result) < tt.minLen {
				t.Errorf("Expected at least %d clusters, got %d", tt.minLen, len(result))
			}
		})
	}
}

func TestClusterValues_Empty(t *testing.T) {
	d := NewGeometricDetector()

	result := d.clusterValues(nil, 2.0)
	if result != nil {
		t.Errorf("Expected nil for empty input")
	}
}

func TestDetectHorizontalLines(t *testing.T) {
	d := NewGeometricDetector()

	yCoords := []float64{100, 80, 60, 40}

	lines := []model.Line{
		{Start: model.Point{X: 0, Y: 100}, End: model.Point{X: 200, Y: 100}}, // At y=100
		{Start: model.Point{X: 0, Y: 60}, End: model.Point{X: 200, Y: 60}},   // At y=60
	}

	hasLines := d.detectHorizontalLines(yCoords, lines)

	if len(hasLines) != len(yCoords) {
		t.Fatalf("hasLines length = %d, want %d", len(hasLines), len(yCoords))
	}

	if !hasLines[0] { // y=100
		t.Error("Expected hasLines[0] = true for y=100")
	}
	if hasLines[1] { // y=80 - no line
		t.Error("Expected hasLines[1] = false for y=80")
	}
	if !hasLines[2] { // y=60
		t.Error("Expected hasLines[2] = true for y=60")
	}
}

func TestDetectVerticalLines(t *testing.T) {
	d := NewGeometricDetector()

	xCoords := []float64{100, 200, 300}

	lines := []model.Line{
		{Start: model.Point{X: 100, Y: 0}, End: model.Point{X: 100, Y: 200}}, // At x=100
		{Start: model.Point{X: 300, Y: 0}, End: model.Point{X: 300, Y: 200}}, // At x=300
	}

	hasLines := d.detectVerticalLines(xCoords, lines)

	if len(hasLines) != len(xCoords) {
		t.Fatalf("hasLines length = %d, want %d", len(hasLines), len(xCoords))
	}

	if !hasLines[0] { // x=100
		t.Error("Expected hasLines[0] = true for x=100")
	}
	if hasLines[1] { // x=200 - no line
		t.Error("Expected hasLines[1] = false for x=200")
	}
	if !hasLines[2] { // x=300
		t.Error("Expected hasLines[2] = true for x=300")
	}
}

func TestCalculateGridRegularity(t *testing.T) {
	d := NewGeometricDetector()

	// Regular grid
	regularGrid := &model.TableGrid{
		Rows: []float64{100, 80, 60, 40}, // Equal spacing of 20
		Cols: []float64{100, 150, 200},   // Equal spacing of 50
	}

	regularScore := d.calculateGridRegularity(regularGrid)
	if regularScore < 0.8 {
		t.Errorf("Regular grid should have high score, got %f", regularScore)
	}

	// Irregular grid
	irregularGrid := &model.TableGrid{
		Rows: []float64{100, 90, 30, 20}, // Varying spacing: 10, 60, 10
		Cols: []float64{100, 110, 200},   // Varying spacing: 10, 90
	}

	irregularScore := d.calculateGridRegularity(irregularGrid)
	if irregularScore >= regularScore {
		t.Errorf("Irregular grid should have lower score than regular, got %f vs %f",
			irregularScore, regularScore)
	}
}

func TestCalculateGridRegularity_TooSmall(t *testing.T) {
	d := NewGeometricDetector()

	// Grid too small
	smallGrid := &model.TableGrid{
		Rows: []float64{100}, // Only 1 row boundary
		Cols: []float64{100},
	}

	score := d.calculateGridRegularity(smallGrid)
	if score != 0 {
		t.Errorf("Too small grid should have score 0, got %f", score)
	}
}

func TestIsNearGridLine(t *testing.T) {
	d := NewGeometricDetector()

	gridLines := []float64{100, 200, 300}

	tests := []struct {
		value float64
		want  bool
	}{
		{100, true},  // Exact match
		{101, true},  // Within tolerance
		{103, true},  // Within 2x tolerance
		{110, false}, // Too far
		{200, true},
		{250, false},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := d.isNearGridLine(tt.value, gridLines)
			if got != tt.want {
				t.Errorf("isNearGridLine(%f) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestFindCell(t *testing.T) {
	d := NewGeometricDetector()

	grid := &model.TableGrid{
		Rows: []float64{100, 80, 60}, // 2 rows
		Cols: []float64{100, 150, 200}, // 2 cols
	}

	tests := []struct {
		point   model.Point
		wantRow int
		wantCol int
	}{
		{model.Point{X: 125, Y: 90}, 0, 0},  // Center of (0,0)
		{model.Point{X: 175, Y: 90}, 0, 1},  // Center of (0,1)
		{model.Point{X: 125, Y: 70}, 1, 0},  // Center of (1,0)
		{model.Point{X: 175, Y: 70}, 1, 1},  // Center of (1,1)
		{model.Point{X: 50, Y: 90}, 0, -1},  // X outside, Y inside
		{model.Point{X: 125, Y: 50}, -1, 0}, // Below grid
	}

	for _, tt := range tests {
		row, col := d.findCell(tt.point, grid)
		if row != tt.wantRow || col != tt.wantCol {
			t.Errorf("findCell(%v) = (%d, %d), want (%d, %d)",
				tt.point, row, col, tt.wantRow, tt.wantCol)
		}
	}
}

func TestCalculateLineScore(t *testing.T) {
	d := NewGeometricDetector()

	tests := []struct {
		name     string
		grid     *model.TableGrid
		minScore float64
		maxScore float64
	}{
		{
			name: "all lines",
			grid: &model.TableGrid{
				HasHLines: []bool{true, true, true},
				HasVLines: []bool{true, true, true},
			},
			minScore: 0.9,
			maxScore: 1.0,
		},
		{
			name: "no lines",
			grid: &model.TableGrid{
				HasHLines: []bool{false, false, false},
				HasVLines: []bool{false, false, false},
			},
			minScore: 0.0,
			maxScore: 0.1,
		},
		{
			name: "half lines",
			grid: &model.TableGrid{
				HasHLines: []bool{true, false, true, false},
				HasVLines: []bool{true, false, true, false},
			},
			minScore: 0.4,
			maxScore: 0.6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := d.calculateLineScore(tt.grid)
			if score < tt.minScore || score > tt.maxScore {
				t.Errorf("calculateLineScore() = %f, want [%f, %f]",
					score, tt.minScore, tt.maxScore)
			}
		})
	}
}

func TestCalculateLineScore_Empty(t *testing.T) {
	d := NewGeometricDetector()

	grid := &model.TableGrid{
		HasHLines: []bool{},
		HasVLines: []bool{},
	}

	score := d.calculateLineScore(grid)
	if score != 0 {
		t.Errorf("Empty grid should have score 0, got %f", score)
	}
}

func TestCalculateCellOccupancy(t *testing.T) {
	d := NewGeometricDetector()

	grid := &model.TableGrid{
		Rows: []float64{100, 80, 60}, // 2 rows
		Cols: []float64{100, 150, 200}, // 2 cols
	}

	// All cells occupied
	allOccupied := []model.TextFragment{
		{Text: "A", BBox: model.BBox{X: 120, Y: 85, Width: 20, Height: 10}}, // (0,0)
		{Text: "B", BBox: model.BBox{X: 170, Y: 85, Width: 20, Height: 10}}, // (0,1)
		{Text: "C", BBox: model.BBox{X: 120, Y: 65, Width: 20, Height: 10}}, // (1,0)
		{Text: "D", BBox: model.BBox{X: 170, Y: 65, Width: 20, Height: 10}}, // (1,1)
	}

	fullOccupancy := d.calculateCellOccupancy(allOccupied, grid)
	if fullOccupancy < 0.9 {
		t.Errorf("Full occupancy should be ~1.0, got %f", fullOccupancy)
	}

	// Half cells occupied
	halfOccupied := []model.TextFragment{
		{Text: "A", BBox: model.BBox{X: 120, Y: 85, Width: 20, Height: 10}}, // (0,0)
		{Text: "D", BBox: model.BBox{X: 170, Y: 65, Width: 20, Height: 10}}, // (1,1)
	}

	halfOccupancyScore := d.calculateCellOccupancy(halfOccupied, grid)
	if halfOccupancyScore < 0.4 || halfOccupancyScore > 0.6 {
		t.Errorf("Half occupancy should be ~0.5, got %f", halfOccupancyScore)
	}
}

func TestCalculateTableBBox(t *testing.T) {
	d := NewGeometricDetector()

	grid := &model.TableGrid{
		Rows: []float64{100, 80, 60},
		Cols: []float64{50, 100, 150},
	}

	bbox := d.calculateTableBBox(grid)

	if bbox.X != 50 {
		t.Errorf("BBox.X = %f, want 50", bbox.X)
	}
	if bbox.Y != 60 {
		t.Errorf("BBox.Y = %f, want 60", bbox.Y)
	}
	if bbox.Width != 100 { // 150 - 50
		t.Errorf("BBox.Width = %f, want 100", bbox.Width)
	}
	if bbox.Height != 40 { // 100 - 60
		t.Errorf("BBox.Height = %f, want 40", bbox.Height)
	}
}

func TestCalculateTableBBox_Empty(t *testing.T) {
	d := NewGeometricDetector()

	grid := &model.TableGrid{
		Rows: []float64{},
		Cols: []float64{},
	}

	bbox := d.calculateTableBBox(grid)

	if !bbox.IsEmpty() {
		t.Errorf("Empty grid should produce empty BBox")
	}
}

func TestHasVisibleGrid(t *testing.T) {
	d := NewGeometricDetector()

	tests := []struct {
		name     string
		grid     *model.TableGrid
		lines    []model.Line
		expected bool
	}{
		{
			name: "many visible lines",
			grid: &model.TableGrid{
				HasHLines: []bool{true, true, true},
				HasVLines: []bool{true, true, true},
			},
			lines:    nil, // Already counted in HasHLines/HasVLines
			expected: true,
		},
		{
			name: "few visible lines",
			grid: &model.TableGrid{
				HasHLines: []bool{true, false, false, false},
				HasVLines: []bool{false, false, false, false},
			},
			lines:    nil,
			expected: false,
		},
		{
			name: "exactly 50%",
			grid: &model.TableGrid{
				HasHLines: []bool{true, true},
				HasVLines: []bool{false, false},
			},
			lines:    nil,
			expected: true, // >= 50%
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := d.hasVisibleGrid(tt.grid, tt.lines)
			if result != tt.expected {
				t.Errorf("hasVisibleGrid() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestHasVisibleGrid_Empty(t *testing.T) {
	d := NewGeometricDetector()

	grid := &model.TableGrid{
		HasHLines: []bool{},
		HasVLines: []bool{},
	}

	result := d.hasVisibleGrid(grid, nil)
	if result {
		t.Error("Empty grid should not have visible grid")
	}
}

func TestMean(t *testing.T) {
	tests := []struct {
		values []float64
		want   float64
	}{
		{[]float64{1, 2, 3, 4, 5}, 3.0},
		{[]float64{10}, 10.0},
		{[]float64{0, 0, 0}, 0.0},
		{nil, 0.0},
		{[]float64{}, 0.0},
	}

	for _, tt := range tests {
		got := mean(tt.values)
		if got != tt.want {
			t.Errorf("mean(%v) = %f, want %f", tt.values, got, tt.want)
		}
	}
}

func TestVariance(t *testing.T) {
	tests := []struct {
		name   string
		values []float64
		want   float64
	}{
		{"uniform", []float64{5, 5, 5, 5}, 0.0},
		{"nil", nil, 0.0},
		{"empty", []float64{}, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := variance(tt.values)
			if got != tt.want {
				t.Errorf("variance(%v) = %f, want %f", tt.values, got, tt.want)
			}
		})
	}

	// Test that varied values have positive variance
	varied := []float64{1, 2, 3, 4, 5}
	v := variance(varied)
	if v <= 0 {
		t.Errorf("variance of varied values should be positive, got %f", v)
	}
}

// Test detector registry
func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.MinRows != 2 {
		t.Errorf("DefaultConfig MinRows = %d, want 2", config.MinRows)
	}
	if config.MinCols != 2 {
		t.Errorf("DefaultConfig MinCols = %d, want 2", config.MinCols)
	}
	if config.MinConfidence != 0.5 {
		t.Errorf("DefaultConfig MinConfidence = %f, want 0.5", config.MinConfidence)
	}
	if config.AlignmentTolerance != 2.0 {
		t.Errorf("DefaultConfig AlignmentTolerance = %f, want 2.0", config.AlignmentTolerance)
	}
}

func TestNewRegistry(t *testing.T) {
	reg := NewRegistry()
	if reg == nil {
		t.Fatal("NewRegistry() returned nil")
	}
	if reg.detectors == nil {
		t.Error("Registry detectors map should be initialized")
	}
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	reg := NewRegistry()

	d := NewGeometricDetector()
	reg.Register(d)

	got := reg.Get("geometric")
	if got == nil {
		t.Error("Get() returned nil for registered detector")
	}
	if got.Name() != "geometric" {
		t.Errorf("Got detector name = %q, want 'geometric'", got.Name())
	}

	// Get non-existent
	notFound := reg.Get("nonexistent")
	if notFound != nil {
		t.Error("Get() should return nil for non-existent detector")
	}
}

func TestRegistry_List(t *testing.T) {
	reg := NewRegistry()

	// Empty registry
	if len(reg.List()) != 0 {
		t.Error("Empty registry should return empty list")
	}

	// Add detector
	d := NewGeometricDetector()
	reg.Register(d)

	list := reg.List()
	if len(list) != 1 {
		t.Errorf("List() returned %d items, want 1", len(list))
	}
	if list[0] != "geometric" {
		t.Errorf("List()[0] = %q, want 'geometric'", list[0])
	}
}

func TestGlobalRegistry(t *testing.T) {
	// Global registry should have geometric detector registered by init()
	d := GetDetector("geometric")
	if d == nil {
		t.Error("Global registry should have 'geometric' detector")
	}

	list := ListDetectors()
	found := false
	for _, name := range list {
		if name == "geometric" {
			found = true
			break
		}
	}
	if !found {
		t.Error("ListDetectors() should include 'geometric'")
	}
}

// Benchmarks
func BenchmarkClusterFragments(b *testing.B) {
	d := NewGeometricDetector()

	fragments := make([]model.TextFragment, 100)
	for i := 0; i < 100; i++ {
		fragments[i] = model.TextFragment{
			Text: "Text",
			BBox: model.BBox{X: float64(i%10) * 50, Y: float64(700 - (i/10)*20), Width: 40, Height: 15},
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d.clusterFragments(fragments)
	}
}

func BenchmarkDetect(b *testing.B) {
	d := NewGeometricDetector()

	page := &model.Page{
		Width:  612,
		Height: 792,
	}

	// Create 50 fragments in a grid pattern
	for i := 0; i < 50; i++ {
		page.RawText = append(page.RawText, model.TextFragment{
			Text: "Text",
			BBox: model.BBox{X: float64(i%5) * 100, Y: float64(700 - (i/5)*20), Width: 80, Height: 15},
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d.Detect(page)
	}
}
