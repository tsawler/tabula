package htmldoc

import (
	"strings"
	"testing"
)

func TestNavigationExclusionModes(t *testing.T) {
	tests := []struct {
		name           string
		html           string
		mode           NavigationExclusionMode
		wantContains   []string
		wantNotContain []string
	}{
		{
			name: "None mode includes everything",
			html: `<html><body>
				<nav><p>Home | About</p></nav>
				<main><h1>Title</h1><p>Content</p></main>
				<footer><p>Copyright 2024</p></footer>
			</body></html>`,
			mode:           NavigationExclusionNone,
			wantContains:   []string{"Title", "Content", "Home", "About", "Copyright"},
			wantNotContain: []string{},
		},
		{
			name: "Explicit mode excludes nav element",
			html: `<html><body>
				<nav><a href="/">Home</a><a href="/about">About</a></nav>
				<main><h1>Title</h1><p>Content</p></main>
			</body></html>`,
			mode:           NavigationExclusionExplicit,
			wantContains:   []string{"Title", "Content"},
			wantNotContain: []string{"Home", "About"},
		},
		{
			name: "Explicit mode excludes aside element",
			html: `<html><body>
				<aside><p>Sidebar content</p></aside>
				<main><h1>Title</h1><p>Main content</p></main>
			</body></html>`,
			mode:           NavigationExclusionExplicit,
			wantContains:   []string{"Title", "Main content"},
			wantNotContain: []string{"Sidebar content"},
		},
		{
			name: "Explicit mode excludes top-level header but not article header",
			html: `<html><body>
				<header><h1>Site Header</h1></header>
				<article>
					<header><h2>Article Header</h2></header>
					<p>Article content</p>
				</article>
			</body></html>`,
			mode:           NavigationExclusionExplicit,
			wantContains:   []string{"Article Header", "Article content"},
			wantNotContain: []string{"Site Header"},
		},
		{
			name: "Explicit mode excludes top-level footer but not article footer",
			html: `<html><body>
				<article>
					<p>Article content</p>
					<footer><p>Article author info</p></footer>
				</article>
				<footer><p>Site footer copyright</p></footer>
			</body></html>`,
			mode:           NavigationExclusionExplicit,
			wantContains:   []string{"Article content", "Article author info"},
			wantNotContain: []string{"Site footer copyright"},
		},
		{
			name: "Explicit mode excludes elements with ARIA navigation role",
			html: `<html><body>
				<div role="navigation"><a href="/">Home</a></div>
				<main><h1>Title</h1><p>Content</p></main>
			</body></html>`,
			mode:           NavigationExclusionExplicit,
			wantContains:   []string{"Title", "Content"},
			wantNotContain: []string{"Home"},
		},
		{
			name: "Explicit mode excludes elements with ARIA banner role at top level",
			html: `<html><body>
				<div role="banner"><h1>Site Banner</h1></div>
				<main><h1>Main Title</h1><p>Content</p></main>
			</body></html>`,
			mode:           NavigationExclusionExplicit,
			wantContains:   []string{"Main Title", "Content"},
			wantNotContain: []string{"Site Banner"},
		},
		{
			name: "Standard mode excludes div with nav class",
			html: `<html><body>
				<div class="main-navigation"><a href="/">Home</a></div>
				<main><h1>Title</h1><p>Content</p></main>
			</body></html>`,
			mode:           NavigationExclusionStandard,
			wantContains:   []string{"Title", "Content"},
			wantNotContain: []string{"Home"},
		},
		{
			name: "Standard mode excludes div with navbar class",
			html: `<html><body>
				<div class="navbar"><a href="/">Home</a></div>
				<main><h1>Title</h1><p>Content</p></main>
			</body></html>`,
			mode:           NavigationExclusionStandard,
			wantContains:   []string{"Title", "Content"},
			wantNotContain: []string{"Home"},
		},
		{
			name: "Standard mode excludes div with footer id",
			html: `<html><body>
				<main><h1>Title</h1><p>Content</p></main>
				<div id="footer"><p>Copyright 2024</p></div>
			</body></html>`,
			mode:           NavigationExclusionStandard,
			wantContains:   []string{"Title", "Content"},
			wantNotContain: []string{"Copyright"},
		},
		{
			name: "Standard mode excludes div with sidebar class",
			html: `<html><body>
				<div class="sidebar"><p>Widget content</p></div>
				<main><h1>Title</h1><p>Content</p></main>
			</body></html>`,
			mode:           NavigationExclusionStandard,
			wantContains:   []string{"Title", "Content"},
			wantNotContain: []string{"Widget content"},
		},
		{
			name: "Standard mode excludes breadcrumb navigation",
			html: `<html><body>
				<div class="breadcrumb"><a href="/">Home</a> > <a href="/cat">Category</a></div>
				<main><h1>Title</h1><p>Content</p></main>
			</body></html>`,
			mode:           NavigationExclusionStandard,
			wantContains:   []string{"Title", "Content"},
			wantNotContain: []string{"Category"},
		},
		{
			name: "Standard mode does not exclude content with similar but non-matching names",
			html: `<html><body>
				<div class="navigator-results"><p>Search navigator</p></div>
				<main><h1>Title</h1><p>Content</p></main>
			</body></html>`,
			mode:           NavigationExclusionStandard,
			wantContains:   []string{"Title", "Content", "Search navigator"},
			wantNotContain: []string{},
		},
		{
			name: "Standard mode with wrapper div still detects top-level header/footer",
			html: `<html><body>
				<div id="wrapper">
					<header><h1>Site Header</h1></header>
					<main><h1>Title</h1><p>Content</p></main>
					<footer><p>Copyright</p></footer>
				</div>
			</body></html>`,
			mode:           NavigationExclusionStandard,
			wantContains:   []string{"Title", "Content"},
			wantNotContain: []string{"Site Header", "Copyright"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader, err := OpenReader(strings.NewReader(tt.html))
			if err != nil {
				t.Fatalf("OpenReader failed: %v", err)
			}

			opts := ExtractOptions{NavigationExclusion: tt.mode}
			text, err := reader.TextWithOptions(opts)
			if err != nil {
				t.Fatalf("TextWithOptions failed: %v", err)
			}

			for _, want := range tt.wantContains {
				if !strings.Contains(text, want) {
					t.Errorf("expected text to contain %q, got:\n%s", want, text)
				}
			}

			for _, notWant := range tt.wantNotContain {
				if strings.Contains(text, notWant) {
					t.Errorf("expected text NOT to contain %q, got:\n%s", notWant, text)
				}
			}
		})
	}
}

func TestAggressiveModeWithLinkDensity(t *testing.T) {
	// HTML with a high link density section that should be excluded in aggressive mode
	html := `<html><body>
		<div class="related-links">
			<a href="/1">Link 1</a>
			<a href="/2">Link 2</a>
			<a href="/3">Link 3</a>
			<a href="/4">Link 4</a>
			<a href="/5">Link 5</a>
		</div>
		<main>
			<h1>Main Article</h1>
			<p>This is the main content with some text that is not links.</p>
		</main>
	</body></html>`

	reader, err := OpenReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("OpenReader failed: %v", err)
	}

	// In standard mode, the related-links div should be included (no nav/footer pattern)
	standardOpts := ExtractOptions{NavigationExclusion: NavigationExclusionStandard}
	standardText, _ := reader.TextWithOptions(standardOpts)
	if !strings.Contains(standardText, "Link 1") {
		t.Error("Standard mode should include the link section")
	}

	// In aggressive mode, the high link density section should be excluded
	aggressiveOpts := ExtractOptions{NavigationExclusion: NavigationExclusionAggressive}
	aggressiveText, _ := reader.TextWithOptions(aggressiveOpts)
	if strings.Contains(aggressiveText, "Link 1") {
		t.Error("Aggressive mode should exclude high link density sections")
	}
	if !strings.Contains(aggressiveText, "Main Article") {
		t.Error("Aggressive mode should still include main content")
	}
}

func TestDefaultExtractOptionsUsesStandard(t *testing.T) {
	opts := DefaultExtractOptions()
	if opts.NavigationExclusion != NavigationExclusionStandard {
		t.Errorf("expected default NavigationExclusion to be Standard, got %v", opts.NavigationExclusion)
	}
}

func TestTextMethodUsesDefaultOptions(t *testing.T) {
	html := `<html><body>
		<nav><a href="/">Home</a></nav>
		<main><h1>Title</h1><p>Content</p></main>
	</body></html>`

	reader, err := OpenReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("OpenReader failed: %v", err)
	}

	// Text() should use default options which includes Standard exclusion
	text, err := reader.Text()
	if err != nil {
		t.Fatalf("Text failed: %v", err)
	}

	if strings.Contains(text, "Home") {
		t.Error("Text() should exclude navigation by default")
	}
	if !strings.Contains(text, "Title") {
		t.Error("Text() should include main content")
	}
}

func TestMarkdownMethodUsesDefaultOptions(t *testing.T) {
	html := `<html><body>
		<nav><a href="/">Home</a></nav>
		<main><h1>Title</h1><p>Content</p></main>
	</body></html>`

	reader, err := OpenReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("OpenReader failed: %v", err)
	}

	// Markdown() should use default options which includes Standard exclusion
	md, err := reader.Markdown()
	if err != nil {
		t.Fatalf("Markdown failed: %v", err)
	}

	if strings.Contains(md, "Home") {
		t.Error("Markdown() should exclude navigation by default")
	}
	if !strings.Contains(md, "# Title") {
		t.Error("Markdown() should include main content")
	}
}

func TestCachingOfFilteredElements(t *testing.T) {
	html := `<html><body>
		<nav><a href="/">Home</a></nav>
		<main><h1>Title</h1><p>Content</p></main>
	</body></html>`

	reader, err := OpenReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("OpenReader failed: %v", err)
	}

	// Call twice with same mode - should use cache
	opts := ExtractOptions{NavigationExclusion: NavigationExclusionStandard}
	text1, _ := reader.TextWithOptions(opts)
	text2, _ := reader.TextWithOptions(opts)

	if text1 != text2 {
		t.Error("Cached results should be identical")
	}

	// Verify cache was populated
	if len(reader.filteredCache) == 0 {
		t.Error("Cache should be populated after filtering")
	}
}

func TestPatternMatchingWordBoundaries(t *testing.T) {
	tests := []struct {
		name       string
		html       string
		shouldSkip bool
	}{
		{
			name:       "nav as exact match",
			html:       `<div class="nav">Skip me</div>`,
			shouldSkip: true,
		},
		{
			name:       "nav with prefix",
			html:       `<div class="top-nav">Skip me</div>`,
			shouldSkip: true,
		},
		{
			name:       "nav with suffix",
			html:       `<div class="nav-bar">Skip me</div>`,
			shouldSkip: true,
		},
		{
			name:       "navbar as word",
			html:       `<div class="navbar">Skip me</div>`,
			shouldSkip: true,
		},
		{
			name:       "navigator should not match",
			html:       `<div class="navigator">Keep me</div>`,
			shouldSkip: false,
		},
		{
			name:       "navigation embedded in longer word should not match",
			html:       `<div class="mynavigationsystem">Keep me</div>`,
			shouldSkip: false,
		},
		{
			name:       "footer exact match",
			html:       `<div id="footer">Skip me</div>`,
			shouldSkip: true,
		},
		{
			name:       "site-footer",
			html:       `<div class="site-footer">Skip me</div>`,
			shouldSkip: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fullHTML := "<html><body>" + tt.html + "<main><p>Main content</p></main></body></html>"
			reader, err := OpenReader(strings.NewReader(fullHTML))
			if err != nil {
				t.Fatalf("OpenReader failed: %v", err)
			}

			opts := ExtractOptions{NavigationExclusion: NavigationExclusionStandard}
			text, _ := reader.TextWithOptions(opts)

			if tt.shouldSkip {
				if strings.Contains(text, "Skip me") {
					t.Errorf("expected content to be skipped, but found in output")
				}
			} else {
				if !strings.Contains(text, "Keep me") {
					t.Errorf("expected content to be kept, but not found in output")
				}
			}
		})
	}
}
