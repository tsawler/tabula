package epubdoc

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"net/url"
	"path"
	"strings"

	"github.com/tsawler/tabula/htmldoc"
	"github.com/tsawler/tabula/model"
)

// Reader-related errors.
var (
	ErrInvalidArchive  = errors.New("epub: invalid or corrupted archive")
	ErrInvalidMimetype = errors.New("epub: invalid mimetype (not an EPUB)")
	ErrMissingContent  = errors.New("epub: referenced content file not found")
)

// Reader provides access to EPUB content.
type Reader struct {
	zr       *zip.ReadCloser
	zrReader *zip.Reader // For when opened from io.ReaderAt
	pkg      *Package
	baseDir  string // Directory containing OPF (for resolving relative paths)
	chapters []*Chapter
	toc      *TableOfContents
}

// Open opens an EPUB file from a path.
func Open(filePath string) (*Reader, error) {
	zr, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, ErrInvalidArchive
	}

	r := &Reader{zr: zr}
	if err := r.init(&zr.Reader); err != nil {
		zr.Close()
		return nil, err
	}

	return r, nil
}

// OpenReader opens an EPUB from an io.ReaderAt.
func OpenReader(ra io.ReaderAt, size int64) (*Reader, error) {
	zr, err := zip.NewReader(ra, size)
	if err != nil {
		return nil, ErrInvalidArchive
	}

	r := &Reader{zrReader: zr}
	if err := r.init(zr); err != nil {
		return nil, err
	}

	return r, nil
}

// init initializes the reader by parsing the EPUB structure.
func (r *Reader) init(zr *zip.Reader) error {
	// Validate mimetype (optional but recommended)
	if err := r.validateMimetype(zr); err != nil {
		// Some EPUBs don't have mimetype file - continue anyway
	}

	// Check for DRM - REJECT if found
	if err := checkForDRM(zr); err != nil {
		return err
	}

	// Parse container.xml to find OPF
	opfPath, err := parseContainer(zr)
	if err != nil {
		return err
	}

	// Parse OPF for metadata, manifest, and spine
	pkg, baseDir, err := parseOPF(zr, opfPath)
	if err != nil {
		return err
	}

	r.pkg = pkg
	r.baseDir = baseDir

	// Load chapters
	if err := r.loadChapters(zr); err != nil {
		return err
	}

	return nil
}

// validateMimetype checks that the mimetype file is correct.
func (r *Reader) validateMimetype(zr *zip.Reader) error {
	for _, f := range zr.File {
		if f.Name == "mimetype" {
			rc, err := f.Open()
			if err != nil {
				return err
			}
			defer rc.Close()

			data, err := io.ReadAll(rc)
			if err != nil {
				return err
			}

			if strings.TrimSpace(string(data)) != "application/epub+zip" {
				return ErrInvalidMimetype
			}
			return nil
		}
	}
	return ErrInvalidMimetype
}

// loadChapters loads all spine items as chapters.
func (r *Reader) loadChapters(zr *zip.Reader) error {
	r.chapters = make([]*Chapter, 0, len(r.pkg.Spine))

	for i, spineItem := range r.pkg.Spine {
		// Look up in manifest
		item, ok := r.pkg.Manifest[spineItem.IDRef]
		if !ok {
			continue // Skip missing items
		}

		// Resolve href relative to OPF location
		href := r.resolveHref(item.Href)

		// Read the content file
		content, err := r.readFile(zr, href)
		if err != nil {
			// Skip missing files but continue
			continue
		}

		chapter := &Chapter{
			ID:      item.ID,
			Index:   i,
			Href:    href,
			Content: content,
		}

		// Extract title from content
		chapter.Title = r.extractChapterTitle(content, i)

		r.chapters = append(r.chapters, chapter)
	}

	if len(r.chapters) == 0 {
		return ErrEmptySpine
	}

	return nil
}

// resolveHref resolves a relative href against the OPF base directory.
func (r *Reader) resolveHref(href string) string {
	// URL-decode the href
	if decoded, err := url.QueryUnescape(href); err == nil {
		href = decoded
	}

	if r.baseDir == "" {
		return href
	}
	return path.Join(r.baseDir, href)
}

// readFile reads a file from the ZIP archive.
func (r *Reader) readFile(zr *zip.Reader, name string) ([]byte, error) {
	for _, f := range zr.File {
		if f.Name == name {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}
	return nil, ErrMissingContent
}

// extractChapterTitle extracts a title from the chapter content.
func (r *Reader) extractChapterTitle(content []byte, index int) string {
	// Try to parse and find the title
	htmlReader, err := htmldoc.OpenReader(bytes.NewReader(content))
	if err != nil {
		return ""
	}

	// Get title from HTML metadata
	meta := htmlReader.Metadata()
	if meta.Title != "" {
		return meta.Title
	}

	// Try to get first heading
	doc, err := htmlReader.Document()
	if err != nil {
		return ""
	}

	for _, page := range doc.Pages {
		for _, elem := range page.Elements {
			if h, ok := elem.(*model.Heading); ok {
				return h.Text
			}
		}
	}

	return ""
}

// Close closes the reader and releases resources.
func (r *Reader) Close() error {
	if r.zr != nil {
		return r.zr.Close()
	}
	return nil
}

// Metadata returns the EPUB metadata.
func (r *Reader) Metadata() Metadata {
	return r.pkg.Metadata
}

// ChapterCount returns the number of chapters.
func (r *Reader) ChapterCount() int {
	return len(r.chapters)
}

// Chapters returns all chapters.
func (r *Reader) Chapters() []*Chapter {
	return r.chapters
}

// Text extracts plain text from all chapters.
func (r *Reader) Text() (string, error) {
	return r.TextWithOptions(ExtractOptions{})
}

// TextWithOptions extracts plain text with the given options.
func (r *Reader) TextWithOptions(opts ExtractOptions) (string, error) {
	htmlOpts := htmldoc.ExtractOptions{
		NavigationExclusion: htmldoc.NavigationExclusionMode(opts.NavigationExclusion),
	}

	var parts []string
	for _, chapter := range r.chapters {
		htmlReader, err := htmldoc.OpenReader(bytes.NewReader(chapter.Content))
		if err != nil {
			continue
		}

		text, err := htmlReader.TextWithOptions(htmlOpts)
		if err != nil {
			continue
		}

		if text = strings.TrimSpace(text); text != "" {
			parts = append(parts, text)
		}
	}

	return strings.Join(parts, "\n\n"), nil
}

// Markdown extracts content as markdown from all chapters.
func (r *Reader) Markdown() (string, error) {
	return r.MarkdownWithOptions(ExtractOptions{})
}

// MarkdownWithOptions extracts content as markdown with the given options.
func (r *Reader) MarkdownWithOptions(opts ExtractOptions) (string, error) {
	htmlOpts := htmldoc.ExtractOptions{
		NavigationExclusion: htmldoc.NavigationExclusionMode(opts.NavigationExclusion),
	}

	var parts []string
	for _, chapter := range r.chapters {
		htmlReader, err := htmldoc.OpenReader(bytes.NewReader(chapter.Content))
		if err != nil {
			continue
		}

		md, err := htmlReader.MarkdownWithOptions(htmlOpts)
		if err != nil {
			continue
		}

		if md = strings.TrimSpace(md); md != "" {
			parts = append(parts, md)
		}
	}

	return strings.Join(parts, "\n\n---\n\n"), nil
}

// Document returns the document model.
func (r *Reader) Document() (*model.Document, error) {
	doc := &model.Document{
		Metadata: model.Metadata{
			Title:   r.pkg.Metadata.Title,
			Author:  strings.Join(r.pkg.Metadata.Creator, ", "),
			Subject: strings.Join(r.pkg.Metadata.Subjects, ", "),
		},
		Pages: make([]*model.Page, 0, len(r.chapters)),
	}

	for i, chapter := range r.chapters {
		htmlReader, err := htmldoc.OpenReader(bytes.NewReader(chapter.Content))
		if err != nil {
			continue
		}

		chapterDoc, err := htmlReader.Document()
		if err != nil {
			continue
		}

		// Use the chapter's content as a page
		for _, page := range chapterDoc.Pages {
			page.Number = i + 1
			doc.Pages = append(doc.Pages, page)
		}
	}

	return doc, nil
}

// getZipReader returns the appropriate zip.Reader.
func (r *Reader) getZipReader() *zip.Reader {
	if r.zr != nil {
		return &r.zr.Reader
	}
	return r.zrReader
}
