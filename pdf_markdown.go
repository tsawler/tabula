package tabula

import (
	"fmt"
	"github.com/tsawler/tabula/model"
	"github.com/tsawler/tabula/rag"
	"strings"
)

// A document export preserves elements, rather than flattening them through
// overlapping RAG chunks. Chunk-specific options retain the chunk export path.
func pdfMarkdown(doc *model.Document, opts rag.MarkdownOptions) string {
	var parts []string
	title := strings.TrimSpace(doc.Metadata.Title)
	hasTitle := false
	for _, p := range doc.Pages {
		for _, e := range p.Elements {
			if h, ok := e.(*model.Heading); ok && strings.TrimSpace(h.Text) == title {
				hasTitle = true
			}
		}
	}
	if opts.IncludeMetadata {
		parts = append(parts, fmt.Sprintf("---\ntitle: %q\npages: %d\n---", title, len(doc.Pages)))
	}
	if title != "" && !hasTitle && !opts.IncludeMetadata {
		parts = append(parts, "# "+title)
	}
	if opts.IncludeTableOfContents {
		var toc []string
		for _, p := range doc.Pages {
			for _, e := range p.Elements {
				if h, ok := e.(*model.Heading); ok {
					toc = append(toc, "- "+h.Text)
				}
			}
		}
		if len(toc) > 0 {
			parts = append(parts, "## Table of Contents\n\n"+strings.Join(toc, "\n"))
		}
	}
	for _, p := range doc.Pages {
		var pageParts []string
		for _, e := range p.Elements {
			switch v := e.(type) {
			case *model.Heading:
				level := v.Level + opts.HeadingLevelOffset
				if level < 1 {
					level = 1
				}
				if opts.MaxHeadingLevel > 0 && level > opts.MaxHeadingLevel {
					level = opts.MaxHeadingLevel
				}
				pageParts = append(pageParts, strings.Repeat("#", level)+" "+v.Text)
			case *model.Paragraph:
				value := v.Text
				if v.Preformatted {
					value = "```text\n" + value + "\n```"
				}
				pageParts = append(pageParts, value)
			case *model.List:
				var items []string
				for _, item := range v.Items {
					prefix := "-"
					if v.Ordered {
						prefix = item.Bullet
						if prefix == "" {
							prefix = "1."
						}
					}
					items = append(items, prefix+" "+item.Text)
				}
				pageParts = append(pageParts, strings.Join(items, "\n"))
			case *model.Table:
				pageParts = append(pageParts, v.ToMarkdown())
			}
		}
		if len(pageParts) > 0 {
			if opts.IncludePageNumbers {
				pageParts = append(pageParts, fmt.Sprintf("*Page %d*", p.Number))
			}
			parts = append(parts, strings.Join(pageParts, "\n\n"))
		}
	}
	return strings.Join(parts, "\n\n")
}
