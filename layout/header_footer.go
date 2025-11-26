package layout

import (
	"regexp"
	"sort"
	"strings"

	"github.com/tsawler/tabula/model"
	"github.com/tsawler/tabula/text"
)

// HeaderFooterRegion represents a detected header or footer region
type HeaderFooterRegion struct {
	// Type indicates if this is a header or footer
	Type RegionType

	// BBox is the bounding box of the region
	BBox model.BBox

	// Text is the typical text content (may include page number placeholder)
	Text string

	// IsPageNumber indicates if this region contains page numbers
	IsPageNumber bool

	// Confidence is the detection confidence (0.0 to 1.0)
	Confidence float64

	// PageIndices lists which pages have this header/footer
	PageIndices []int
}

// RegionType indicates whether a region is a header or footer
type RegionType int

const (
	Header RegionType = iota
	Footer
)

func (r RegionType) String() string {
	if r == Header {
		return "header"
	}
	return "footer"
}

// HeaderFooterConfig holds configuration for header/footer detection
type HeaderFooterConfig struct {
	// HeaderRegionHeight is the height from top of page to consider as header zone
	// Default: 72 points (1 inch)
	HeaderRegionHeight float64

	// FooterRegionHeight is the height from bottom of page to consider as footer zone
	// Default: 72 points (1 inch)
	FooterRegionHeight float64

	// MinOccurrenceRatio is the minimum fraction of pages a text must appear on
	// to be considered a header/footer (0.0 to 1.0)
	// Default: 0.5 (50% of pages)
	MinOccurrenceRatio float64

	// PositionTolerance is the maximum Y difference for text to be considered same position
	// Default: 5 points
	PositionTolerance float64

	// XPositionTolerance is the maximum X difference for text to be considered same position
	// Default: 10 points
	XPositionTolerance float64

	// MinPages is the minimum number of pages required for header/footer detection
	// Default: 2
	MinPages int
}

// DefaultHeaderFooterConfig returns sensible default configuration
func DefaultHeaderFooterConfig() HeaderFooterConfig {
	return HeaderFooterConfig{
		HeaderRegionHeight: 72.0,  // 1 inch
		FooterRegionHeight: 72.0,  // 1 inch
		MinOccurrenceRatio: 0.5,   // 50% of pages
		PositionTolerance:  5.0,   // 5 points
		XPositionTolerance: 10.0,  // 10 points
		MinPages:           2,
	}
}

// PageFragments represents text fragments from a single page
type PageFragments struct {
	PageIndex  int
	PageHeight float64
	PageWidth  float64
	Fragments  []text.TextFragment
}

// HeaderFooterDetector detects headers and footers across pages
type HeaderFooterDetector struct {
	config HeaderFooterConfig
}

// NewHeaderFooterDetector creates a new detector with default configuration
func NewHeaderFooterDetector() *HeaderFooterDetector {
	return &HeaderFooterDetector{
		config: DefaultHeaderFooterConfig(),
	}
}

// NewHeaderFooterDetectorWithConfig creates a detector with custom configuration
func NewHeaderFooterDetectorWithConfig(config HeaderFooterConfig) *HeaderFooterDetector {
	return &HeaderFooterDetector{
		config: config,
	}
}

// HeaderFooterResult contains the detection results
type HeaderFooterResult struct {
	// Headers contains detected header regions
	Headers []HeaderFooterRegion

	// Footers contains detected footer regions
	Footers []HeaderFooterRegion

	// Config used for detection
	Config HeaderFooterConfig
}

// Detect analyzes fragments from multiple pages to find headers and footers
func (d *HeaderFooterDetector) Detect(pages []PageFragments) *HeaderFooterResult {
	if len(pages) < d.config.MinPages {
		return &HeaderFooterResult{Config: d.config}
	}

	// Preprocess pages to handle character-level PDFs
	// Assemble character fragments into line-based fragments for better pattern matching
	processedPages := d.preprocessPages(pages)

	// Extract header and footer candidates from each page
	headerCandidates := d.extractCandidates(processedPages, Header)
	footerCandidates := d.extractCandidates(processedPages, Footer)

	// Find repeating patterns
	headers := d.findRepeatingPatterns(headerCandidates, processedPages, Header)
	footers := d.findRepeatingPatterns(footerCandidates, processedPages, Footer)

	return &HeaderFooterResult{
		Headers: headers,
		Footers: footers,
		Config:  d.config,
	}
}

// preprocessPages assembles character-level fragments into line-based fragments
// This is necessary for character-level PDFs (like Google Docs) where each character
// is a separate fragment, making pattern detection impossible without assembly.
func (d *HeaderFooterDetector) preprocessPages(pages []PageFragments) []PageFragments {
	processed := make([]PageFragments, len(pages))

	for i, page := range pages {
		// Check if this is a character-level page
		if isCharacterLevel(page.Fragments) {
			// Assemble fragments into lines
			assembled := assembleFragmentsIntoLines(page.Fragments)
			processed[i] = PageFragments{
				PageIndex:  page.PageIndex,
				PageHeight: page.PageHeight,
				PageWidth:  page.PageWidth,
				Fragments:  assembled,
			}
		} else {
			// Use original fragments
			processed[i] = page
		}
	}

	return processed
}

// isCharacterLevel returns true if fragments appear to be character-level
// (average fragment length <= 2 characters)
func isCharacterLevel(fragments []text.TextFragment) bool {
	if len(fragments) == 0 {
		return false
	}

	totalChars := 0
	for _, f := range fragments {
		totalChars += len([]rune(f.Text))
	}

	avgLen := float64(totalChars) / float64(len(fragments))
	return avgLen <= 2.0
}

// assembleFragmentsIntoLines groups character fragments into line-based fragments
func assembleFragmentsIntoLines(fragments []text.TextFragment) []text.TextFragment {
	if len(fragments) == 0 {
		return nil
	}

	// Sort by Y (descending for typical PDF coords) then by X
	sorted := make([]text.TextFragment, len(fragments))
	copy(sorted, fragments)
	sort.Slice(sorted, func(i, j int) bool {
		yDiff := sorted[i].Y - sorted[j].Y
		if absFloat(yDiff) > sorted[i].Height*0.5 {
			return yDiff > 0 // Higher Y first
		}
		return sorted[i].X < sorted[j].X
	})

	// Group into lines by Y proximity
	var lines [][]text.TextFragment
	var currentLine []text.TextFragment

	for _, frag := range sorted {
		if len(currentLine) == 0 {
			currentLine = append(currentLine, frag)
			continue
		}

		// Check if same line (Y within tolerance)
		lastFrag := currentLine[len(currentLine)-1]
		yDiff := absFloat(frag.Y - lastFrag.Y)

		if yDiff <= lastFrag.Height*0.5 {
			currentLine = append(currentLine, frag)
		} else {
			lines = append(lines, currentLine)
			currentLine = []text.TextFragment{frag}
		}
	}
	if len(currentLine) > 0 {
		lines = append(lines, currentLine)
	}

	// Assemble each line into a single fragment
	var assembled []text.TextFragment
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		// Sort line by X
		sort.Slice(line, func(i, j int) bool {
			return line[i].X < line[j].X
		})

		// Build text with smart spacing
		var textBuilder strings.Builder
		var lastEndX float64

		for i, frag := range line {
			if i > 0 {
				gap := frag.X - lastEndX
				// Add space if gap is significant (> 30% of font size)
				if gap > frag.FontSize*0.3 {
					textBuilder.WriteString(" ")
				}
			}
			textBuilder.WriteString(frag.Text)
			lastEndX = frag.X + frag.Width
		}

		// Compute bounding box
		first, last := line[0], line[len(line)-1]
		minY, maxY := first.Y, first.Y
		for _, f := range line {
			if f.Y < minY {
				minY = f.Y
			}
			if f.Y+f.Height > maxY {
				maxY = f.Y + f.Height
			}
		}

		assembled = append(assembled, text.TextFragment{
			Text:     textBuilder.String(),
			X:        first.X,
			Y:        first.Y,
			Width:    (last.X + last.Width) - first.X,
			Height:   maxY - minY,
			FontSize: first.FontSize,
			FontName: first.FontName,
		})
	}

	return assembled
}

// candidate represents a potential header/footer text
type candidate struct {
	Text      string
	X         float64
	Y         float64 // Normalized Y (distance from top for headers, from bottom for footers)
	Width     float64
	Height    float64
	PageIndex int
}

// extractCandidates extracts header or footer candidates from pages
func (d *HeaderFooterDetector) extractCandidates(pages []PageFragments, regionType RegionType) []candidate {
	var candidates []candidate

	for _, page := range pages {
		if len(page.Fragments) == 0 {
			continue
		}

		// Compute actual content bounds
		minY, maxY := page.Fragments[0].Y, page.Fragments[0].Y
		for _, frag := range page.Fragments {
			if frag.Y < minY {
				minY = frag.Y
			}
			if frag.Y+frag.Height > maxY {
				maxY = frag.Y + frag.Height
			}
		}
		contentHeight := maxY - minY
		if contentHeight <= 0 {
			contentHeight = page.PageHeight
		}

		// Detect coordinate system:
		// - Standard PDF: Y increases upward (high Y = top of page)
		// - Google Docs style: Y increases downward and may exceed page height
		// Heuristic: if maxY > pageHeight, assume inverted coordinates
		invertedCoords := maxY > page.PageHeight

		// Determine reference bounds for header/footer detection
		// Use page bounds when content is within page, otherwise use content bounds
		refMinY, refMaxY := 0.0, page.PageHeight
		headerRegion := d.config.HeaderRegionHeight
		footerRegion := d.config.FooterRegionHeight

		if invertedCoords {
			// Content extends beyond page - use content bounds with scaled regions
			refMinY, refMaxY = minY, maxY
			scale := contentHeight / page.PageHeight
			headerRegion *= scale
			footerRegion *= scale
		}

		for _, frag := range page.Fragments {
			var inRegion bool
			var normalizedY float64

			if regionType == Header {
				if invertedCoords {
					// Inverted: low Y = top of page, so header is near refMinY
					distFromTop := frag.Y - refMinY
					inRegion = distFromTop < headerRegion
					normalizedY = distFromTop
				} else {
					// Standard: high Y = top of page, so header is near refMaxY
					distFromTop := refMaxY - (frag.Y + frag.Height)
					inRegion = distFromTop < headerRegion
					normalizedY = distFromTop
				}
			} else {
				if invertedCoords {
					// Inverted: high Y = bottom of page, so footer is near refMaxY
					distFromBottom := refMaxY - (frag.Y + frag.Height)
					inRegion = distFromBottom < footerRegion
					normalizedY = distFromBottom
				} else {
					// Standard: low Y = bottom of page, so footer is near refMinY
					distFromBottom := frag.Y - refMinY
					inRegion = distFromBottom < footerRegion
					normalizedY = distFromBottom
				}
			}

			if inRegion {
				candidates = append(candidates, candidate{
					Text:      strings.TrimSpace(frag.Text),
					X:         frag.X,
					Y:         normalizedY,
					Width:     frag.Width,
					Height:    frag.Height,
					PageIndex: page.PageIndex,
				})
			}
		}
	}

	return candidates
}

// findRepeatingPatterns finds text that repeats across pages
func (d *HeaderFooterDetector) findRepeatingPatterns(candidates []candidate, pages []PageFragments, regionType RegionType) []HeaderFooterRegion {
	if len(candidates) == 0 {
		return nil
	}

	// Group candidates by normalized text (ignoring page numbers)
	groups := make(map[string][]candidate)

	for _, c := range candidates {
		// Normalize text by replacing page numbers with placeholder
		normalized := normalizeForComparison(c.Text)
		groups[normalized] = append(groups[normalized], c)
	}

	var regions []HeaderFooterRegion
	minOccurrences := int(float64(len(pages)) * d.config.MinOccurrenceRatio)
	if minOccurrences < 2 {
		minOccurrences = 2
	}

	for normalizedText, group := range groups {
		// Skip very short text that isn't a page number
		// Single letters/characters are likely fragments of larger text
		if len(normalizedText) <= 2 && !isPageNumberPattern(normalizedText) {
			continue
		}

		// Check if this text appears on enough pages
		pageSet := make(map[int]bool)
		for _, c := range group {
			pageSet[c.PageIndex] = true
		}

		if len(pageSet) < minOccurrences {
			continue
		}

		// Check position consistency
		if !d.hasConsistentPosition(group) {
			continue
		}

		// Calculate bounding box and confidence
		bbox := d.calculateGroupBBox(group)
		confidence := d.calculateConfidence(group, len(pages))

		// Determine if this is a page number
		isPageNum := isPageNumberPattern(normalizedText) || containsPageNumberPattern(group)

		// Get representative text
		representativeText := group[0].Text
		if isPageNum {
			representativeText = "[Page Number]"
		}

		// Collect page indices
		var pageIndices []int
		for idx := range pageSet {
			pageIndices = append(pageIndices, idx)
		}
		sort.Ints(pageIndices)

		regions = append(regions, HeaderFooterRegion{
			Type:         regionType,
			BBox:         bbox,
			Text:         representativeText,
			IsPageNumber: isPageNum,
			Confidence:   confidence,
			PageIndices:  pageIndices,
		})
	}

	// Sort by confidence (highest first)
	sort.Slice(regions, func(i, j int) bool {
		return regions[i].Confidence > regions[j].Confidence
	})

	return regions
}

// hasConsistentPosition checks if candidates appear at consistent positions
func (d *HeaderFooterDetector) hasConsistentPosition(group []candidate) bool {
	if len(group) < 2 {
		return false
	}

	// Check Y position consistency
	refY := group[0].Y
	refX := group[0].X

	for _, c := range group[1:] {
		yDiff := absFloat(c.Y - refY)
		xDiff := absFloat(c.X - refX)

		if yDiff > d.config.PositionTolerance {
			return false
		}
		if xDiff > d.config.XPositionTolerance {
			return false
		}
	}

	return true
}

// calculateGroupBBox calculates the bounding box for a group of candidates
func (d *HeaderFooterDetector) calculateGroupBBox(group []candidate) model.BBox {
	if len(group) == 0 {
		return model.BBox{}
	}

	minX := group[0].X
	maxX := group[0].X + group[0].Width
	minY := group[0].Y
	maxY := group[0].Y + group[0].Height

	for _, c := range group[1:] {
		if c.X < minX {
			minX = c.X
		}
		if c.X+c.Width > maxX {
			maxX = c.X + c.Width
		}
		if c.Y < minY {
			minY = c.Y
		}
		if c.Y+c.Height > maxY {
			maxY = c.Y + c.Height
		}
	}

	return model.BBox{
		X:      minX,
		Y:      minY,
		Width:  maxX - minX,
		Height: maxY - minY,
	}
}

// calculateConfidence calculates detection confidence
func (d *HeaderFooterDetector) calculateConfidence(group []candidate, totalPages int) float64 {
	if totalPages == 0 {
		return 0
	}

	// Base confidence on occurrence ratio
	pageSet := make(map[int]bool)
	for _, c := range group {
		pageSet[c.PageIndex] = true
	}

	occurrenceRatio := float64(len(pageSet)) / float64(totalPages)

	// Boost confidence for position consistency
	positionBonus := 0.0
	if d.hasConsistentPosition(group) {
		positionBonus = 0.1
	}

	confidence := occurrenceRatio*0.9 + positionBonus
	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

// normalizeForComparison normalizes text for comparison by replacing numbers
func normalizeForComparison(text string) string {
	// Replace sequences of digits with a placeholder
	re := regexp.MustCompile(`\d+`)
	return re.ReplaceAllString(text, "#")
}

// isPageNumberPattern checks if normalized text looks like a page number
func isPageNumberPattern(normalizedText string) bool {
	// Common page number patterns (after normalization)
	patterns := []string{
		"#",                  // Just a number
		"Page #",             // "Page 1"
		"page #",             // "page 1"
		"- # -",              // "- 1 -"
		"# of #",             // "1 of 10"
		"Page # of #",        // "Page 1 of 10"
		"#/#",                // "1/10"
		"p. #",               // "p. 1"
		"p.#",                // "p.1"
		"pg #",               // "pg 1"
		"pg. #",              // "pg. 1"
	}

	trimmed := strings.TrimSpace(normalizedText)
	for _, pattern := range patterns {
		if strings.EqualFold(trimmed, pattern) {
			return true
		}
	}

	return false
}

// containsPageNumberPattern checks if any candidate in the group contains page numbers
func containsPageNumberPattern(group []candidate) bool {
	if len(group) < 2 {
		return false
	}

	// Extract just the numbers from each candidate
	re := regexp.MustCompile(`\d+`)

	var numbers []int
	for _, c := range group {
		matches := re.FindAllString(c.Text, -1)
		for _, match := range matches {
			var num int
			if _, err := parsePageNumber(match, &num); err == nil {
				numbers = append(numbers, num)
			}
		}
	}

	if len(numbers) < 2 {
		return false
	}

	// Sort and check if they form a sequence
	sort.Ints(numbers)

	// Check for sequential or near-sequential pattern
	sequential := 0
	for i := 1; i < len(numbers); i++ {
		diff := numbers[i] - numbers[i-1]
		if diff == 1 {
			sequential++
		}
	}

	// If more than half are sequential, it's likely page numbers
	return sequential >= len(numbers)/2
}

// parsePageNumber parses a string as a page number
func parsePageNumber(s string, result *int) (bool, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return false, nil
	}

	num := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return false, nil
		}
		num = num*10 + int(c-'0')
	}

	*result = num
	return true, nil
}

// FilterFragments removes header/footer fragments from a page
func (r *HeaderFooterResult) FilterFragments(pageIndex int, fragments []text.TextFragment, pageHeight float64) []text.TextFragment {
	if r == nil || len(fragments) == 0 {
		return fragments
	}

	// Check if this is a character-level PDF
	charLevel := isCharacterLevel(fragments)

	// Compute content bounds for position checking
	minY, maxY := fragments[0].Y, fragments[0].Y
	for _, frag := range fragments {
		if frag.Y < minY {
			minY = frag.Y
		}
		if frag.Y+frag.Height > maxY {
			maxY = frag.Y + frag.Height
		}
	}
	contentHeight := maxY - minY
	if contentHeight <= 0 {
		contentHeight = pageHeight
	}

	// Detect coordinate system
	invertedCoords := maxY > pageHeight

	// Scale regions if content extends beyond page
	headerRegion := r.Config.HeaderRegionHeight
	footerRegion := r.Config.FooterRegionHeight
	if contentHeight > pageHeight {
		scale := contentHeight / pageHeight
		headerRegion *= scale
		footerRegion *= scale
	}

	var filtered []text.TextFragment

	for _, frag := range fragments {
		if r.isInHeaderFooter(pageIndex, frag, minY, maxY, headerRegion, footerRegion, invertedCoords, charLevel) {
			continue
		}
		filtered = append(filtered, frag)
	}

	return filtered
}

// isInHeaderFooter checks if a fragment is in a detected header/footer region
func (r *HeaderFooterResult) isInHeaderFooter(pageIndex int, frag text.TextFragment, minY, maxY, headerRegion, footerRegion float64, invertedCoords, charLevel bool) bool {
	// Check headers
	for _, header := range r.Headers {
		if !containsPage(header.PageIndices, pageIndex) {
			continue
		}

		var distFromTop float64
		if invertedCoords {
			distFromTop = frag.Y - minY
		} else {
			distFromTop = maxY - (frag.Y + frag.Height)
		}
		if distFromTop < headerRegion {
			// For character-level PDFs, use position-only filtering since
			// individual characters won't match the assembled header text
			if charLevel {
				return true
			}
			if textsMatch(frag.Text, header.Text, header.IsPageNumber) {
				return true
			}
		}
	}

	// Check footers
	for _, footer := range r.Footers {
		if !containsPage(footer.PageIndices, pageIndex) {
			continue
		}

		var distFromBottom float64
		if invertedCoords {
			distFromBottom = maxY - (frag.Y + frag.Height)
		} else {
			distFromBottom = frag.Y - minY
		}
		if distFromBottom < footerRegion {
			// For character-level PDFs, use position-only filtering
			if charLevel {
				return true
			}
			if textsMatch(frag.Text, footer.Text, footer.IsPageNumber) {
				return true
			}
		}
	}

	return false
}

// containsPage checks if a page index is in the list
func containsPage(pages []int, pageIndex int) bool {
	for _, p := range pages {
		if p == pageIndex {
			return true
		}
	}
	return false
}

// textsMatch checks if two texts match (considering page numbers)
func textsMatch(fragText, regionText string, isPageNumber bool) bool {
	fragText = strings.TrimSpace(fragText)
	regionText = strings.TrimSpace(regionText)

	if isPageNumber {
		// For page numbers, just check if it's a number or page number pattern
		normalized := normalizeForComparison(fragText)
		return isPageNumberPattern(normalized)
	}

	// For regular text, check exact match or normalized match
	if fragText == regionText {
		return true
	}

	// Try normalized comparison
	return normalizeForComparison(fragText) == normalizeForComparison(regionText)
}

// HasHeaders returns true if any headers were detected
func (r *HeaderFooterResult) HasHeaders() bool {
	return r != nil && len(r.Headers) > 0
}

// HasFooters returns true if any footers were detected
func (r *HeaderFooterResult) HasFooters() bool {
	return r != nil && len(r.Footers) > 0
}

// HasHeadersOrFooters returns true if any headers or footers were detected
func (r *HeaderFooterResult) HasHeadersOrFooters() bool {
	return r.HasHeaders() || r.HasFooters()
}

// GetHeaderTexts returns all detected header texts
func (r *HeaderFooterResult) GetHeaderTexts() []string {
	if r == nil {
		return nil
	}

	var texts []string
	for _, h := range r.Headers {
		texts = append(texts, h.Text)
	}
	return texts
}

// GetFooterTexts returns all detected footer texts
func (r *HeaderFooterResult) GetFooterTexts() []string {
	if r == nil {
		return nil
	}

	var texts []string
	for _, f := range r.Footers {
		texts = append(texts, f.Text)
	}
	return texts
}

// abs returns the absolute value of a float64
func absFloat(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// Summary returns a human-readable summary of detection results
func (r *HeaderFooterResult) Summary() string {
	if r == nil || !r.HasHeadersOrFooters() {
		return "No headers or footers detected"
	}

	var parts []string

	if len(r.Headers) > 0 {
		headerTexts := make([]string, len(r.Headers))
		for i, h := range r.Headers {
			headerTexts[i] = h.Text
		}
		parts = append(parts, "Headers: "+strings.Join(headerTexts, ", "))
	}

	if len(r.Footers) > 0 {
		footerTexts := make([]string, len(r.Footers))
		for i, f := range r.Footers {
			footerTexts[i] = f.Text
		}
		parts = append(parts, "Footers: "+strings.Join(footerTexts, ", "))
	}

	return strings.Join(parts, "; ")
}
