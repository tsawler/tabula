package htmldoc

import (
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/net/html"
)

// navigationPatterns defines class/id patterns that indicate navigation or boilerplate content.
var navigationPatterns = struct {
	nav      *regexp.Regexp
	header   *regexp.Regexp
	footer   *regexp.Regexp
	sidebar  *regexp.Regexp
	excluded *regexp.Regexp // Combined pattern for efficiency
}{
	// Word boundary matching for common navigation patterns
	nav:     regexp.MustCompile(`(?i)(^|[^a-z])(nav|navbar|navigation|menu|topnav|sidenav|breadcrumb|breadcrumbs)([^a-z]|$)`),
	header:  regexp.MustCompile(`(?i)(^|[^a-z])(site-header|page-header|masthead|banner)([^a-z]|$)`),
	footer:  regexp.MustCompile(`(?i)(^|[^a-z])(footer|site-footer|page-footer|colophon)([^a-z]|$)`),
	sidebar: regexp.MustCompile(`(?i)(^|[^a-z])(sidebar|widget-area|widget|aside)([^a-z]|$)`),
}

func init() {
	// Pre-compile combined pattern for efficiency in standard mode
	navigationPatterns.excluded = regexp.MustCompile(
		`(?i)(^|[^a-z])(nav|navbar|navigation|menu|topnav|sidenav|breadcrumb|breadcrumbs|` +
			`site-header|page-header|masthead|banner|` +
			`footer|site-footer|page-footer|colophon|` +
			`sidebar|widget-area|widget|aside)([^a-z]|$)`)
}

// exclusionChecker holds state for determining which elements to exclude.
type exclusionChecker struct {
	mode             NavigationExclusionMode
	bodyNode         *html.Node
	topLevelWrapper  *html.Node // Single wrapper div/main if present
	linkDensityCache map[*html.Node]float64
}

// newExclusionChecker creates a checker for the given mode and document.
func newExclusionChecker(mode NavigationExclusionMode, doc *html.Node) *exclusionChecker {
	checker := &exclusionChecker{
		mode:             mode,
		linkDensityCache: make(map[*html.Node]float64),
	}

	// Find body node
	checker.bodyNode = findElement(doc, "body")
	if checker.bodyNode == nil {
		checker.bodyNode = doc
	}

	// Detect single top-level wrapper
	checker.topLevelWrapper = detectTopLevelWrapper(checker.bodyNode)

	return checker
}

// detectTopLevelWrapper finds a single structural wrapper element if one exists.
// This handles the common pattern of <body><div id="wrapper">...</div></body>
func detectTopLevelWrapper(body *html.Node) *html.Node {
	var structuralChildren []*html.Node

	for c := body.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode {
			switch c.Data {
			case "div", "main":
				structuralChildren = append(structuralChildren, c)
			case "script", "style", "noscript", "template":
				// Ignore these
			default:
				// Any other element means no single wrapper
				return nil
			}
		}
	}

	if len(structuralChildren) == 1 {
		return structuralChildren[0]
	}
	return nil
}

// shouldExclude determines if a node should be excluded based on the exclusion mode.
func (ec *exclusionChecker) shouldExclude(n *html.Node) bool {
	if n.Type != html.ElementNode {
		return false
	}

	if ec.mode == NavigationExclusionNone {
		return false
	}

	// Check explicit semantic elements (all modes except None)
	if ec.shouldExcludeExplicit(n) {
		return true
	}

	// Check class/id patterns (Standard and Aggressive modes)
	if ec.mode >= NavigationExclusionStandard {
		if ec.shouldExcludeByPattern(n) {
			return true
		}
	}

	// Check link density (Aggressive mode only)
	if ec.mode >= NavigationExclusionAggressive {
		if ec.shouldExcludeByLinkDensity(n) {
			return true
		}
	}

	return false
}

// shouldExcludeExplicit checks for explicit semantic HTML5 elements and ARIA roles.
func (ec *exclusionChecker) shouldExcludeExplicit(n *html.Node) bool {
	// Always exclude <nav> and <aside> regardless of position
	switch n.Data {
	case "nav", "aside":
		return true
	}

	// Check ARIA roles
	role := getAttr(n, "role")
	switch role {
	case "navigation", "complementary":
		return true
	case "banner", "contentinfo":
		// These correspond to header/footer - check depth
		return ec.isTopLevel(n)
	}

	// <header> and <footer> - only exclude if top-level
	switch n.Data {
	case "header", "footer":
		return ec.isTopLevel(n)
	}

	return false
}

// isTopLevel returns true if the node is a direct child of body or a single top-level wrapper.
func (ec *exclusionChecker) isTopLevel(n *html.Node) bool {
	parent := n.Parent
	if parent == nil {
		return false
	}

	// Direct child of body
	if parent == ec.bodyNode {
		return true
	}

	// Direct child of a single top-level wrapper
	if ec.topLevelWrapper != nil && parent == ec.topLevelWrapper {
		return true
	}

	return false
}

// shouldExcludeByPattern checks class and id attributes for common navigation patterns.
func (ec *exclusionChecker) shouldExcludeByPattern(n *html.Node) bool {
	class := getAttr(n, "class")
	id := getAttr(n, "id")

	// Check combined pattern against class and id
	if class != "" && navigationPatterns.excluded.MatchString(class) {
		return true
	}
	if id != "" && navigationPatterns.excluded.MatchString(id) {
		return true
	}

	return false
}

// shouldExcludeByLinkDensity checks if an element has an unusually high link-to-text ratio.
// This is used in Aggressive mode to catch navigation sections that lack semantic markup.
func (ec *exclusionChecker) shouldExcludeByLinkDensity(n *html.Node) bool {
	// Only check block-level container elements
	switch n.Data {
	case "div", "section", "ul", "ol":
		// Continue with check
	default:
		return false
	}

	density := ec.calculateLinkDensity(n)

	// Threshold: if more than 60% of text is within links, likely navigation
	// Also require a minimum amount of links to avoid false positives on small elements
	linkCount := countLinks(n)
	return density > 0.6 && linkCount >= 4
}

// calculateLinkDensity returns the ratio of link text to total text (0.0 to 1.0).
func (ec *exclusionChecker) calculateLinkDensity(n *html.Node) float64 {
	if cached, ok := ec.linkDensityCache[n]; ok {
		return cached
	}

	totalLen := textLength(n)
	if totalLen == 0 {
		ec.linkDensityCache[n] = 0
		return 0
	}

	linkLen := linkTextLength(n)
	density := float64(linkLen) / float64(totalLen)

	ec.linkDensityCache[n] = density
	return density
}

// textLength returns the total length of text content in a node.
func textLength(n *html.Node) int {
	if n.Type == html.TextNode {
		return len(strings.TrimSpace(n.Data))
	}

	total := 0
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		total += textLength(c)
	}
	return total
}

// linkTextLength returns the length of text content within <a> tags.
func linkTextLength(n *html.Node) int {
	if n.Type == html.ElementNode && n.Data == "a" {
		return textLength(n)
	}

	total := 0
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		total += linkTextLength(c)
	}
	return total
}

// countLinks returns the number of <a> elements within a node.
func countLinks(n *html.Node) int {
	count := 0
	if n.Type == html.ElementNode && n.Data == "a" {
		count = 1
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		count += countLinks(c)
	}
	return count
}

// getAttr returns the value of an attribute on a node, or empty string if not found.
func getAttr(n *html.Node, key string) string {
	for _, attr := range n.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}

// normalizeClassName normalizes a class name for pattern matching.
// It converts camelCase and various separators to a consistent format.
func normalizeClassName(s string) string {
	var result strings.Builder
	result.Grow(len(s))

	for i, r := range s {
		if unicode.IsUpper(r) && i > 0 {
			result.WriteRune('-')
		}
		result.WriteRune(unicode.ToLower(r))
	}

	return result.String()
}
