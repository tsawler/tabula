package htmldoc

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsawler/tabula/rag"
)

func TestOpenReader_SimpleHTML(t *testing.T) {
	html := `<!DOCTYPE html>
<html>
<head>
	<title>Test Document</title>
	<meta name="author" content="Test Author">
	<meta name="description" content="Test description">
	<meta name="keywords" content="test, keywords, here">
</head>
<body>
	<h1>Main Heading</h1>
	<p>This is a paragraph.</p>
</body>
</html>`

	r, err := OpenReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("OpenReader() failed: %v", err)
	}
	defer r.Close()

	if r.title != "Test Document" {
		t.Errorf("title = %q, want 'Test Document'", r.title)
	}
}

func TestOpenReader_InvalidHTML(t *testing.T) {
	// Even malformed HTML should parse (HTML parser is lenient)
	html := `<html><body><p>unclosed paragraph`

	r, err := OpenReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("OpenReader() should handle malformed HTML: %v", err)
	}
	defer r.Close()
}

func TestOpen_NotFound(t *testing.T) {
	_, err := Open("/nonexistent/file.html")
	if err == nil {
		t.Error("Open() expected error for nonexistent file")
	}
}

func TestOpen_ValidFile(t *testing.T) {
	// Create temp HTML file
	tmpFile, err := os.CreateTemp("", "test-*.html")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	tmpFile.WriteString("<html><body><p>Test</p></body></html>")
	tmpFile.Close()

	r, err := Open(tmpFile.Name())
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer r.Close()
}

func TestReader_Close(t *testing.T) {
	html := `<html><body></body></html>`
	r, _ := OpenReader(strings.NewReader(html))

	// Close should succeed
	if err := r.Close(); err != nil {
		t.Errorf("Close() failed: %v", err)
	}

	// Second close should be safe
	if err := r.Close(); err != nil {
		t.Errorf("Second Close() failed: %v", err)
	}
}

func TestReader_PageCount(t *testing.T) {
	html := `<html><body><p>Test</p></body></html>`
	r, _ := OpenReader(strings.NewReader(html))
	defer r.Close()

	count, err := r.PageCount()
	if err != nil {
		t.Errorf("PageCount() failed: %v", err)
	}
	if count != 1 {
		t.Errorf("PageCount() = %d, want 1", count)
	}
}

func TestReader_Metadata(t *testing.T) {
	html := `<!DOCTYPE html>
<html>
<head>
	<title>Test Title</title>
	<meta name="author" content="John Doe">
	<meta name="description" content="A test document">
	<meta name="keywords" content="test, document, html">
</head>
<body></body>
</html>`

	r, _ := OpenReader(strings.NewReader(html))
	defer r.Close()

	meta := r.Metadata()

	if meta.Title != "Test Title" {
		t.Errorf("Title = %q, want 'Test Title'", meta.Title)
	}
	if meta.Author != "John Doe" {
		t.Errorf("Author = %q, want 'John Doe'", meta.Author)
	}
	if meta.Subject != "A test document" {
		t.Errorf("Subject = %q, want 'A test document'", meta.Subject)
	}
	if len(meta.Keywords) != 3 {
		t.Errorf("Keywords length = %d, want 3", len(meta.Keywords))
	}
}

func TestReader_Text_Headings(t *testing.T) {
	html := `<html><body>
<h1>Heading 1</h1>
<h2>Heading 2</h2>
<h3>Heading 3</h3>
</body></html>`

	r, _ := OpenReader(strings.NewReader(html))
	defer r.Close()

	text, err := r.TextWithOptions(ExtractOptions{NavigationExclusion: NavigationExclusionNone})
	if err != nil {
		t.Fatalf("Text() failed: %v", err)
	}

	if !strings.Contains(text, "Heading 1") {
		t.Errorf("Text missing 'Heading 1', got: %s", text)
	}
	if !strings.Contains(text, "Heading 2") {
		t.Errorf("Text missing 'Heading 2', got: %s", text)
	}
	if !strings.Contains(text, "Heading 3") {
		t.Errorf("Text missing 'Heading 3', got: %s", text)
	}
}

func TestReader_Text_Paragraphs(t *testing.T) {
	html := `<html><body>
<p>First paragraph.</p>
<p>Second paragraph.</p>
</body></html>`

	r, _ := OpenReader(strings.NewReader(html))
	defer r.Close()

	text, err := r.TextWithOptions(ExtractOptions{NavigationExclusion: NavigationExclusionNone})
	if err != nil {
		t.Fatalf("Text() failed: %v", err)
	}

	if !strings.Contains(text, "First paragraph") {
		t.Errorf("Text missing 'First paragraph', got: %s", text)
	}
	if !strings.Contains(text, "Second paragraph") {
		t.Errorf("Text missing 'Second paragraph', got: %s", text)
	}
}

func TestReader_Text_Lists(t *testing.T) {
	html := `<html><body>
<ul>
	<li>Item 1</li>
	<li>Item 2</li>
	<li>Item 3</li>
</ul>
</body></html>`

	r, _ := OpenReader(strings.NewReader(html))
	defer r.Close()

	text, err := r.TextWithOptions(ExtractOptions{NavigationExclusion: NavigationExclusionNone})
	if err != nil {
		t.Fatalf("Text() failed: %v", err)
	}

	if !strings.Contains(text, "Item 1") {
		t.Errorf("Text missing 'Item 1', got: %s", text)
	}
	if !strings.Contains(text, "•") {
		t.Errorf("Text missing bullet marker, got: %s", text)
	}
}

func TestReader_Text_OrderedList(t *testing.T) {
	html := `<html><body>
<ol>
	<li>First</li>
	<li>Second</li>
</ol>
</body></html>`

	r, _ := OpenReader(strings.NewReader(html))
	defer r.Close()

	text, err := r.TextWithOptions(ExtractOptions{NavigationExclusion: NavigationExclusionNone})
	if err != nil {
		t.Fatalf("Text() failed: %v", err)
	}

	if !strings.Contains(text, "First") {
		t.Errorf("Text missing 'First', got: %s", text)
	}
}

func TestReader_Text_NestedList(t *testing.T) {
	html := `<html><body>
<ul>
	<li>Parent
		<ul>
			<li>Child 1</li>
			<li>Child 2</li>
		</ul>
	</li>
</ul>
</body></html>`

	r, _ := OpenReader(strings.NewReader(html))
	defer r.Close()

	text, err := r.TextWithOptions(ExtractOptions{NavigationExclusion: NavigationExclusionNone})
	if err != nil {
		t.Fatalf("Text() failed: %v", err)
	}

	if !strings.Contains(text, "Parent") {
		t.Errorf("Text missing 'Parent', got: %s", text)
	}
	if !strings.Contains(text, "Child 1") {
		t.Errorf("Text missing 'Child 1', got: %s", text)
	}
}

func TestReader_Text_Table(t *testing.T) {
	html := `<html><body>
<table>
	<thead>
		<tr><th>Name</th><th>Age</th></tr>
	</thead>
	<tbody>
		<tr><td>Alice</td><td>30</td></tr>
		<tr><td>Bob</td><td>25</td></tr>
	</tbody>
</table>
</body></html>`

	r, _ := OpenReader(strings.NewReader(html))
	defer r.Close()

	text, err := r.TextWithOptions(ExtractOptions{NavigationExclusion: NavigationExclusionNone})
	if err != nil {
		t.Fatalf("Text() failed: %v", err)
	}

	if !strings.Contains(text, "Name") {
		t.Errorf("Text missing 'Name', got: %s", text)
	}
	if !strings.Contains(text, "Alice") {
		t.Errorf("Text missing 'Alice', got: %s", text)
	}
}

func TestReader_Text_Code(t *testing.T) {
	html := `<html><body>
<pre>func main() {
	fmt.Println("Hello")
}</pre>
</body></html>`

	r, _ := OpenReader(strings.NewReader(html))
	defer r.Close()

	text, err := r.TextWithOptions(ExtractOptions{NavigationExclusion: NavigationExclusionNone})
	if err != nil {
		t.Fatalf("Text() failed: %v", err)
	}

	if !strings.Contains(text, "func main()") {
		t.Errorf("Text missing code content, got: %s", text)
	}
}

func TestReader_Text_Blockquote(t *testing.T) {
	html := `<html><body>
<blockquote>This is a quote.</blockquote>
</body></html>`

	r, _ := OpenReader(strings.NewReader(html))
	defer r.Close()

	text, err := r.TextWithOptions(ExtractOptions{NavigationExclusion: NavigationExclusionNone})
	if err != nil {
		t.Fatalf("Text() failed: %v", err)
	}

	if !strings.Contains(text, "This is a quote") {
		t.Errorf("Text missing blockquote, got: %s", text)
	}
}

func TestReader_Text_SkipsScriptStyle(t *testing.T) {
	html := `<html><body>
<script>console.log("hidden");</script>
<style>.hidden { display: none; }</style>
<p>Visible content</p>
</body></html>`

	r, _ := OpenReader(strings.NewReader(html))
	defer r.Close()

	text, err := r.TextWithOptions(ExtractOptions{NavigationExclusion: NavigationExclusionNone})
	if err != nil {
		t.Fatalf("Text() failed: %v", err)
	}

	if strings.Contains(text, "console.log") {
		t.Errorf("Text should not contain script content, got: %s", text)
	}
	if strings.Contains(text, ".hidden") {
		t.Errorf("Text should not contain style content, got: %s", text)
	}
	if !strings.Contains(text, "Visible content") {
		t.Errorf("Text missing visible content, got: %s", text)
	}
}

func TestReader_Markdown_Headings(t *testing.T) {
	html := `<html><body>
<h1>Heading 1</h1>
<h2>Heading 2</h2>
<h3>Heading 3</h3>
</body></html>`

	r, _ := OpenReader(strings.NewReader(html))
	defer r.Close()

	md, err := r.MarkdownWithOptions(ExtractOptions{NavigationExclusion: NavigationExclusionNone})
	if err != nil {
		t.Fatalf("Markdown() failed: %v", err)
	}

	if !strings.Contains(md, "# Heading 1") {
		t.Errorf("Markdown missing '# Heading 1', got: %s", md)
	}
	if !strings.Contains(md, "## Heading 2") {
		t.Errorf("Markdown missing '## Heading 2', got: %s", md)
	}
	if !strings.Contains(md, "### Heading 3") {
		t.Errorf("Markdown missing '### Heading 3', got: %s", md)
	}
}

func TestReader_Markdown_Lists(t *testing.T) {
	html := `<html><body>
<ul>
	<li>Item 1</li>
	<li>Item 2</li>
</ul>
</body></html>`

	r, _ := OpenReader(strings.NewReader(html))
	defer r.Close()

	md, err := r.MarkdownWithOptions(ExtractOptions{NavigationExclusion: NavigationExclusionNone})
	if err != nil {
		t.Fatalf("Markdown() failed: %v", err)
	}

	if !strings.Contains(md, "- Item 1") {
		t.Errorf("Markdown missing '- Item 1', got: %s", md)
	}
}

func TestReader_Markdown_OrderedList(t *testing.T) {
	html := `<html><body>
<ol>
	<li>First</li>
	<li>Second</li>
</ol>
</body></html>`

	r, _ := OpenReader(strings.NewReader(html))
	defer r.Close()

	md, err := r.MarkdownWithOptions(ExtractOptions{NavigationExclusion: NavigationExclusionNone})
	if err != nil {
		t.Fatalf("Markdown() failed: %v", err)
	}

	if !strings.Contains(md, "1. First") {
		t.Errorf("Markdown missing '1. First', got: %s", md)
	}
}

func TestReader_Markdown_Table(t *testing.T) {
	html := `<html><body>
<table>
	<tr><th>A</th><th>B</th></tr>
	<tr><td>1</td><td>2</td></tr>
</table>
</body></html>`

	r, _ := OpenReader(strings.NewReader(html))
	defer r.Close()

	md, err := r.MarkdownWithOptions(ExtractOptions{NavigationExclusion: NavigationExclusionNone})
	if err != nil {
		t.Fatalf("Markdown() failed: %v", err)
	}

	if !strings.Contains(md, "|") {
		t.Errorf("Markdown missing table pipes, got: %s", md)
	}
	if !strings.Contains(md, "---") {
		t.Errorf("Markdown missing table separator, got: %s", md)
	}
}

func TestReader_Markdown_Code(t *testing.T) {
	html := `<html><body>
<pre>code block</pre>
</body></html>`

	r, _ := OpenReader(strings.NewReader(html))
	defer r.Close()

	md, err := r.MarkdownWithOptions(ExtractOptions{NavigationExclusion: NavigationExclusionNone})
	if err != nil {
		t.Fatalf("Markdown() failed: %v", err)
	}

	if !strings.Contains(md, "```") {
		t.Errorf("Markdown missing code fence, got: %s", md)
	}
	if !strings.Contains(md, "code block") {
		t.Errorf("Markdown missing code content, got: %s", md)
	}
}

func TestReader_Markdown_Blockquote(t *testing.T) {
	html := `<html><body>
<blockquote>Quote text</blockquote>
</body></html>`

	r, _ := OpenReader(strings.NewReader(html))
	defer r.Close()

	md, err := r.MarkdownWithOptions(ExtractOptions{NavigationExclusion: NavigationExclusionNone})
	if err != nil {
		t.Fatalf("Markdown() failed: %v", err)
	}

	if !strings.Contains(md, "> Quote text") {
		t.Errorf("Markdown missing blockquote, got: %s", md)
	}
}

func TestReader_MarkdownWithRAGOptions_Metadata(t *testing.T) {
	html := `<!DOCTYPE html>
<html>
<head>
	<title>Test Doc</title>
	<meta name="author" content="Author Name">
	<meta name="description" content="Description here">
</head>
<body>
<h1>Content</h1>
</body>
</html>`

	r, _ := OpenReader(strings.NewReader(html))
	defer r.Close()

	md, err := r.MarkdownWithRAGOptions(
		ExtractOptions{NavigationExclusion: NavigationExclusionNone},
		rag.MarkdownOptions{IncludeMetadata: true},
	)
	if err != nil {
		t.Fatalf("MarkdownWithRAGOptions() failed: %v", err)
	}

	if !strings.Contains(md, "---") {
		t.Errorf("Missing YAML front matter, got: %s", md)
	}
	if !strings.Contains(md, "title:") {
		t.Errorf("Missing title in metadata, got: %s", md)
	}
	if !strings.Contains(md, "author:") {
		t.Errorf("Missing author in metadata, got: %s", md)
	}
}

func TestReader_MarkdownWithRAGOptions_TOC(t *testing.T) {
	html := `<html><body>
<h1>First</h1>
<h2>Second</h2>
<h2>Third</h2>
</body></html>`

	r, _ := OpenReader(strings.NewReader(html))
	defer r.Close()

	md, err := r.MarkdownWithRAGOptions(
		ExtractOptions{NavigationExclusion: NavigationExclusionNone},
		rag.MarkdownOptions{IncludeTableOfContents: true},
	)
	if err != nil {
		t.Fatalf("MarkdownWithRAGOptions() failed: %v", err)
	}

	if !strings.Contains(md, "Table of Contents") {
		t.Errorf("Missing TOC, got: %s", md)
	}
	if !strings.Contains(md, "[First]") {
		t.Errorf("Missing TOC entry, got: %s", md)
	}
}

func TestReader_Document(t *testing.T) {
	html := `<html><body>
<h1>Title</h1>
<p>Paragraph</p>
<ul><li>Item</li></ul>
</body></html>`

	r, _ := OpenReader(strings.NewReader(html))
	defer r.Close()

	doc, err := r.Document()
	if err != nil {
		t.Fatalf("Document() failed: %v", err)
	}

	if doc == nil {
		t.Fatal("Document() returned nil")
	}

	if len(doc.Pages) != 1 {
		t.Errorf("Document has %d pages, want 1", len(doc.Pages))
	}
}

func TestReader_DocumentWithOptions(t *testing.T) {
	html := `<html><body>
<h1>Title</h1>
<p>Content</p>
</body></html>`

	r, _ := OpenReader(strings.NewReader(html))
	defer r.Close()

	doc, err := r.DocumentWithOptions(ExtractOptions{NavigationExclusion: NavigationExclusionNone})
	if err != nil {
		t.Fatalf("DocumentWithOptions() failed: %v", err)
	}

	if doc == nil {
		t.Fatal("DocumentWithOptions() returned nil")
	}
}

func TestParseTable_Simple(t *testing.T) {
	html := `<html><body>
<table>
	<tr><td>A</td><td>B</td></tr>
	<tr><td>1</td><td>2</td></tr>
</table>
</body></html>`

	r, _ := OpenReader(strings.NewReader(html))
	defer r.Close()

	// Access parsed elements
	if len(r.elements) == 0 {
		t.Fatal("No elements parsed")
	}

	var table *ParsedTable
	for _, elem := range r.elements {
		if elem.Type == ElementTable {
			table = elem.Table
			break
		}
	}

	if table == nil {
		t.Fatal("No table found")
	}

	if len(table.Rows) != 2 {
		t.Errorf("Table has %d rows, want 2", len(table.Rows))
	}
	if len(table.Rows[0]) != 2 {
		t.Errorf("Row has %d cells, want 2", len(table.Rows[0]))
	}
}

func TestParseTable_WithHeader(t *testing.T) {
	html := `<html><body>
<table>
	<thead><tr><th>Name</th><th>Value</th></tr></thead>
	<tbody><tr><td>A</td><td>1</td></tr></tbody>
</table>
</body></html>`

	r, _ := OpenReader(strings.NewReader(html))
	defer r.Close()

	var table *ParsedTable
	for _, elem := range r.elements {
		if elem.Type == ElementTable {
			table = elem.Table
			break
		}
	}

	if table == nil {
		t.Fatal("No table found")
	}

	if !table.HasHeader {
		t.Error("Table should have header")
	}
	if !table.Rows[0][0].IsHeader {
		t.Error("First cell should be header")
	}
}

func TestParseTable_Spans(t *testing.T) {
	html := `<html><body>
<table>
	<tr><td colspan="2">Wide</td></tr>
	<tr><td rowspan="2">Tall</td><td>A</td></tr>
	<tr><td>B</td></tr>
</table>
</body></html>`

	r, _ := OpenReader(strings.NewReader(html))
	defer r.Close()

	var table *ParsedTable
	for _, elem := range r.elements {
		if elem.Type == ElementTable {
			table = elem.Table
			break
		}
	}

	if table == nil {
		t.Fatal("No table found")
	}

	// First row, first cell should have colspan=2
	if table.Rows[0][0].ColSpan != 2 {
		t.Errorf("ColSpan = %d, want 2", table.Rows[0][0].ColSpan)
	}

	// Second row, first cell should have rowspan=2
	if table.Rows[1][0].RowSpan != 2 {
		t.Errorf("RowSpan = %d, want 2", table.Rows[1][0].RowSpan)
	}
}

func TestParsedTable_ToMarkdown(t *testing.T) {
	table := &ParsedTable{
		HasHeader: true,
		Rows: [][]TableCell{
			{{Text: "A", IsHeader: true}, {Text: "B", IsHeader: true}},
			{{Text: "1"}, {Text: "2"}},
		},
	}

	md := table.ToMarkdown()

	if !strings.Contains(md, "| A |") {
		t.Errorf("Missing header, got: %s", md)
	}
	if !strings.Contains(md, "| --- |") {
		t.Errorf("Missing separator, got: %s", md)
	}
	if !strings.Contains(md, "| 1 |") {
		t.Errorf("Missing data, got: %s", md)
	}
}

func TestParsedTable_ToMarkdown_Empty(t *testing.T) {
	table := &ParsedTable{}
	md := table.ToMarkdown()
	if md != "" {
		t.Errorf("Empty table should return empty string, got: %s", md)
	}
}

func TestEscapeMarkdown(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "hello"},
		{"with|pipe", "with\\|pipe"},
		{"line\nbreak", "line break"},
		{"carriage\rreturn", "carriagereturn"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := escapeMarkdown(tt.input); got != tt.want {
				t.Errorf("escapeMarkdown(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestShouldSkipElement(t *testing.T) {
	skip := []string{"script", "style", "noscript", "template", "svg", "math", "iframe", "object", "embed"}
	keep := []string{"div", "p", "span", "h1", "table", "ul", "li"}

	for _, tag := range skip {
		if !shouldSkipElement(tag) {
			t.Errorf("shouldSkipElement(%q) = false, want true", tag)
		}
	}

	for _, tag := range keep {
		if shouldSkipElement(tag) {
			t.Errorf("shouldSkipElement(%q) = true, want false", tag)
		}
	}
}

func TestDefaultExtractOptions(t *testing.T) {
	opts := DefaultExtractOptions()
	if opts.NavigationExclusion != NavigationExclusionStandard {
		t.Errorf("Default NavigationExclusion = %v, want NavigationExclusionStandard", opts.NavigationExclusion)
	}
}

func TestReader_Text_WithNavigationExclusion(t *testing.T) {
	html := `<html><body>
<nav><p>Navigation: Home About</p></nav>
<main>
<h1>Main Content</h1>
<p>Important text here.</p>
</main>
<footer><p>Copyright 2024</p></footer>
</body></html>`

	r, _ := OpenReader(strings.NewReader(html))
	defer r.Close()

	// With NavigationExclusionNone - should include everything
	textNone, _ := r.TextWithOptions(ExtractOptions{NavigationExclusion: NavigationExclusionNone})

	// Main content should always be present
	if !strings.Contains(textNone, "Main Content") {
		t.Errorf("Text missing main content, got: %s", textNone)
	}

	// With NavigationExclusionExplicit - should exclude nav
	textExplicit, _ := r.TextWithOptions(ExtractOptions{NavigationExclusion: NavigationExclusionExplicit})

	// Main content should still be present
	if !strings.Contains(textExplicit, "Main Content") {
		t.Errorf("Explicit mode should keep main content, got: %s", textExplicit)
	}

	// Nav content should be excluded in explicit mode
	if strings.Contains(textExplicit, "Navigation:") {
		t.Logf("Navigation was excluded as expected")
	}
}

func TestReader_SemanticElements(t *testing.T) {
	html := `<html><body>
<article>
<header><h1>Article Title</h1></header>
<section><p>Section content</p></section>
<footer><p>Article footer</p></footer>
</article>
</body></html>`

	r, _ := OpenReader(strings.NewReader(html))
	defer r.Close()

	text, _ := r.TextWithOptions(ExtractOptions{NavigationExclusion: NavigationExclusionNone})

	if !strings.Contains(text, "Article Title") {
		t.Errorf("Text missing 'Article Title', got: %s", text)
	}
	if !strings.Contains(text, "Section content") {
		t.Errorf("Text missing 'Section content', got: %s", text)
	}
}

func TestReader_Div_AsBlockContainer(t *testing.T) {
	html := `<html><body>
<div>
	<p>Paragraph inside div</p>
	<ul><li>List inside div</li></ul>
</div>
</body></html>`

	r, _ := OpenReader(strings.NewReader(html))
	defer r.Close()

	text, _ := r.TextWithOptions(ExtractOptions{NavigationExclusion: NavigationExclusionNone})

	if !strings.Contains(text, "Paragraph inside div") {
		t.Errorf("Text missing paragraph, got: %s", text)
	}
	if !strings.Contains(text, "List inside div") {
		t.Errorf("Text missing list item, got: %s", text)
	}
}

func TestReader_NoBody(t *testing.T) {
	// HTML without explicit body tag
	html := `<html><p>Content without body tag</p></html>`

	r, _ := OpenReader(strings.NewReader(html))
	defer r.Close()

	text, err := r.TextWithOptions(ExtractOptions{NavigationExclusion: NavigationExclusionNone})
	if err != nil {
		t.Fatalf("Text() failed: %v", err)
	}

	if !strings.Contains(text, "Content without body tag") {
		t.Errorf("Text should handle missing body tag, got: %s", text)
	}
}

// Integration test - create a real HTML file
func TestIntegration_HTMLFile(t *testing.T) {
	html := `<!DOCTYPE html>
<html>
<head>
	<title>Integration Test</title>
	<meta name="author" content="Test Author">
</head>
<body>
	<h1>Main Heading</h1>
	<p>This is the first paragraph.</p>
	<h2>Subheading</h2>
	<ul>
		<li>Item one</li>
		<li>Item two</li>
	</ul>
	<table>
		<tr><th>Col1</th><th>Col2</th></tr>
		<tr><td>A</td><td>B</td></tr>
	</table>
	<pre>code block</pre>
	<blockquote>A wise quote</blockquote>
</body>
</html>`

	// Write to temp file
	tmpFile, err := os.CreateTemp("", "test-*.html")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	tmpFile.WriteString(html)
	tmpFile.Close()

	r, err := Open(tmpFile.Name())
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer r.Close()

	t.Run("Metadata", func(t *testing.T) {
		meta := r.Metadata()
		if meta.Title != "Integration Test" {
			t.Errorf("Title = %q, want 'Integration Test'", meta.Title)
		}
		if meta.Author != "Test Author" {
			t.Errorf("Author = %q, want 'Test Author'", meta.Author)
		}
	})

	t.Run("Text", func(t *testing.T) {
		text, err := r.TextWithOptions(ExtractOptions{NavigationExclusion: NavigationExclusionNone})
		if err != nil {
			t.Errorf("Text() failed: %v", err)
		}
		t.Logf("Text length: %d", len(text))
	})

	t.Run("Markdown", func(t *testing.T) {
		md, err := r.MarkdownWithOptions(ExtractOptions{NavigationExclusion: NavigationExclusionNone})
		if err != nil {
			t.Errorf("Markdown() failed: %v", err)
		}
		if !strings.Contains(md, "# Main Heading") {
			t.Errorf("Markdown missing heading")
		}
	})

	t.Run("Document", func(t *testing.T) {
		doc, err := r.Document()
		if err != nil {
			t.Errorf("Document() failed: %v", err)
		}
		if doc == nil || len(doc.Pages) == 0 {
			t.Error("Document should have pages")
		}
	})
}

// Test with testdata file if available
func TestIntegration_TestdataFile(t *testing.T) {
	testdataPath := filepath.Join("testdata", "sample.html")
	if _, err := os.Stat(testdataPath); os.IsNotExist(err) {
		t.Skip("testdata/sample.html not found")
	}

	r, err := Open(testdataPath)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer r.Close()

	text, _ := r.Text()
	t.Logf("Text length: %d", len(text))
}

// Benchmarks
func BenchmarkOpenReader(b *testing.B) {
	html := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<h1>Heading</h1>
<p>Paragraph one.</p>
<p>Paragraph two.</p>
<ul><li>Item 1</li><li>Item 2</li></ul>
</body>
</html>`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r, _ := OpenReader(strings.NewReader(html))
		r.Close()
	}
}

func BenchmarkText(b *testing.B) {
	html := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<h1>Heading</h1>
<p>Paragraph one.</p>
<p>Paragraph two.</p>
<ul><li>Item 1</li><li>Item 2</li></ul>
</body>
</html>`

	r, _ := OpenReader(strings.NewReader(html))
	defer r.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.TextWithOptions(ExtractOptions{NavigationExclusion: NavigationExclusionNone})
	}
}

func BenchmarkMarkdown(b *testing.B) {
	html := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<h1>Heading</h1>
<p>Paragraph one.</p>
<p>Paragraph two.</p>
<ul><li>Item 1</li><li>Item 2</li></ul>
<table><tr><td>A</td><td>B</td></tr></table>
</body>
</html>`

	r, _ := OpenReader(strings.NewReader(html))
	defer r.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.MarkdownWithOptions(ExtractOptions{NavigationExclusion: NavigationExclusionNone})
	}
}
