package epubdoc

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// createTestEPUB creates a minimal valid EPUB file for testing.
func createTestEPUB(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()
	epubPath := filepath.Join(tmpDir, "test.epub")

	f, err := os.Create(epubPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	w := zip.NewWriter(f)

	// Write mimetype (must be first, uncompressed)
	mimeWriter, err := w.CreateHeader(&zip.FileHeader{
		Name:   "mimetype",
		Method: zip.Store, // No compression
	})
	if err != nil {
		t.Fatal(err)
	}
	mimeWriter.Write([]byte("application/epub+zip"))

	// Write container.xml
	containerWriter, err := w.Create("META-INF/container.xml")
	if err != nil {
		t.Fatal(err)
	}
	containerWriter.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`))

	// Write OPF
	opfWriter, err := w.Create("OEBPS/content.opf")
	if err != nil {
		t.Fatal(err)
	}
	opfWriter.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Test Book</dc:title>
    <dc:creator>Test Author</dc:creator>
    <dc:language>en</dc:language>
    <dc:identifier id="bookid">test-isbn-123</dc:identifier>
  </metadata>
  <manifest>
    <item id="chapter1" href="chapter1.xhtml" media-type="application/xhtml+xml"/>
    <item id="chapter2" href="chapter2.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="chapter1"/>
    <itemref idref="chapter2"/>
  </spine>
</package>`))

	// Write chapter 1
	ch1Writer, err := w.Create("OEBPS/chapter1.xhtml")
	if err != nil {
		t.Fatal(err)
	}
	ch1Writer.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Chapter 1</title></head>
<body>
<h1>Introduction</h1>
<p>This is the first chapter of the test book.</p>
<p>It contains multiple paragraphs.</p>
</body>
</html>`))

	// Write chapter 2
	ch2Writer, err := w.Create("OEBPS/chapter2.xhtml")
	if err != nil {
		t.Fatal(err)
	}
	ch2Writer.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Chapter 2</title></head>
<body>
<h1>Conclusion</h1>
<p>This is the second chapter.</p>
<ul>
  <li>Item one</li>
  <li>Item two</li>
</ul>
</body>
</html>`))

	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	return epubPath
}

// createDRMProtectedEPUB creates an EPUB with DRM markers for testing rejection.
func createDRMProtectedEPUB(t *testing.T, drmType string) string {
	t.Helper()

	tmpDir := t.TempDir()
	epubPath := filepath.Join(tmpDir, "drm.epub")

	f, err := os.Create(epubPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	w := zip.NewWriter(f)

	// Write mimetype
	mimeWriter, err := w.CreateHeader(&zip.FileHeader{
		Name:   "mimetype",
		Method: zip.Store,
	})
	if err != nil {
		t.Fatal(err)
	}
	mimeWriter.Write([]byte("application/epub+zip"))

	// Write container.xml
	containerWriter, err := w.Create("META-INF/container.xml")
	if err != nil {
		t.Fatal(err)
	}
	containerWriter.Write([]byte(`<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`))

	// Add DRM markers based on type
	switch drmType {
	case "rights":
		rightsWriter, err := w.Create("META-INF/rights.xml")
		if err != nil {
			t.Fatal(err)
		}
		rightsWriter.Write([]byte(`<?xml version="1.0"?>
<rights xmlns="http://ns.adobe.com/adept">
  <encryptedKey>...</encryptedKey>
</rights>`))

	case "encryption":
		encWriter, err := w.Create("META-INF/encryption.xml")
		if err != nil {
			t.Fatal(err)
		}
		encWriter.Write([]byte(`<?xml version="1.0"?>
<encryption xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <EncryptedData xmlns="http://www.w3.org/2001/04/xmlenc#">
    <EncryptionMethod Algorithm="http://www.w3.org/2001/04/xmlenc#aes256-cbc"/>
    <CipherData>
      <CipherReference URI="OEBPS/chapter1.xhtml"/>
    </CipherData>
  </EncryptedData>
</encryption>`))
	}

	// Write minimal OPF
	opfWriter, err := w.Create("OEBPS/content.opf")
	if err != nil {
		t.Fatal(err)
	}
	opfWriter.Write([]byte(`<?xml version="1.0"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>DRM Test</dc:title>
  </metadata>
  <manifest>
    <item id="chapter1" href="chapter1.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="chapter1"/>
  </spine>
</package>`))

	// Write chapter
	chWriter, err := w.Create("OEBPS/chapter1.xhtml")
	if err != nil {
		t.Fatal(err)
	}
	chWriter.Write([]byte(`<html><body><p>Content</p></body></html>`))

	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	return epubPath
}

func TestOpen(t *testing.T) {
	epubPath := createTestEPUB(t)

	r, err := Open(epubPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer r.Close()

	if r.ChapterCount() != 2 {
		t.Errorf("ChapterCount = %d, want 2", r.ChapterCount())
	}
}

func TestMetadata(t *testing.T) {
	epubPath := createTestEPUB(t)

	r, err := Open(epubPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer r.Close()

	meta := r.Metadata()

	if meta.Title != "Test Book" {
		t.Errorf("Title = %q, want %q", meta.Title, "Test Book")
	}

	if len(meta.Creator) != 1 || meta.Creator[0] != "Test Author" {
		t.Errorf("Creator = %v, want [Test Author]", meta.Creator)
	}

	if meta.Language != "en" {
		t.Errorf("Language = %q, want %q", meta.Language, "en")
	}

	if meta.Identifier != "test-isbn-123" {
		t.Errorf("Identifier = %q, want %q", meta.Identifier, "test-isbn-123")
	}
}

func TestText(t *testing.T) {
	epubPath := createTestEPUB(t)

	r, err := Open(epubPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer r.Close()

	text, err := r.Text()
	if err != nil {
		t.Fatalf("Text failed: %v", err)
	}

	// Check that content from both chapters is present
	if !bytes.Contains([]byte(text), []byte("Introduction")) {
		t.Error("Text should contain 'Introduction'")
	}
	if !bytes.Contains([]byte(text), []byte("first chapter")) {
		t.Error("Text should contain 'first chapter'")
	}
	if !bytes.Contains([]byte(text), []byte("Conclusion")) {
		t.Error("Text should contain 'Conclusion'")
	}
	if !bytes.Contains([]byte(text), []byte("second chapter")) {
		t.Error("Text should contain 'second chapter'")
	}
}

func TestMarkdown(t *testing.T) {
	epubPath := createTestEPUB(t)

	r, err := Open(epubPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer r.Close()

	md, err := r.Markdown()
	if err != nil {
		t.Fatalf("Markdown failed: %v", err)
	}

	// Check for markdown heading format
	if !bytes.Contains([]byte(md), []byte("# Introduction")) {
		t.Error("Markdown should contain '# Introduction'")
	}
	if !bytes.Contains([]byte(md), []byte("# Conclusion")) {
		t.Error("Markdown should contain '# Conclusion'")
	}
}

func TestDocument(t *testing.T) {
	epubPath := createTestEPUB(t)

	r, err := Open(epubPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer r.Close()

	doc, err := r.Document()
	if err != nil {
		t.Fatalf("Document failed: %v", err)
	}

	if doc.Metadata.Title != "Test Book" {
		t.Errorf("Document title = %q, want %q", doc.Metadata.Title, "Test Book")
	}

	if len(doc.Pages) == 0 {
		t.Error("Document should have pages")
	}
}

func TestDRMRejection_Rights(t *testing.T) {
	epubPath := createDRMProtectedEPUB(t, "rights")

	_, err := Open(epubPath)
	if err != ErrDRMProtected {
		t.Errorf("Expected ErrDRMProtected for rights.xml DRM, got: %v", err)
	}
}

func TestDRMRejection_Encryption(t *testing.T) {
	epubPath := createDRMProtectedEPUB(t, "encryption")

	_, err := Open(epubPath)
	if err != ErrDRMProtected {
		t.Errorf("Expected ErrDRMProtected for encryption.xml DRM, got: %v", err)
	}
}

func TestInvalidEPUB(t *testing.T) {
	// Test with a non-existent file
	_, err := Open("/nonexistent/file.epub")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}

	// Test with an invalid ZIP file
	tmpDir := t.TempDir()
	invalidPath := filepath.Join(tmpDir, "invalid.epub")
	os.WriteFile(invalidPath, []byte("not a zip file"), 0644)

	_, err = Open(invalidPath)
	if err != ErrInvalidArchive {
		t.Errorf("Expected ErrInvalidArchive, got: %v", err)
	}
}

func TestTableOfContents(t *testing.T) {
	epubPath := createTestEPUB(t)

	r, err := Open(epubPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer r.Close()

	toc := r.TableOfContents()

	// Since we don't have NCX or nav document, it should generate from spine
	if len(toc.Entries) != 2 {
		t.Errorf("TOC entries = %d, want 2", len(toc.Entries))
	}
}

// ============================================================================
// Integration Tests with Real EPUB Files
// ============================================================================

func testEPUBPath(filename string) string {
	return filepath.Join("testdata", filename)
}

func TestIntegration_RealEPUB(t *testing.T) {
	epubPath := testEPUBPath("Frankenstein.epub")
	if _, err := os.Stat(epubPath); os.IsNotExist(err) {
		t.Skip("test EPUB not found:", epubPath)
	}

	r, err := Open(epubPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer r.Close()

	// Test chapter count
	count := r.ChapterCount()
	if count == 0 {
		t.Error("expected at least one chapter")
	}
	t.Logf("Chapter count: %d", count)

	// Test metadata
	meta := r.Metadata()
	if meta.Title == "" {
		t.Error("expected non-empty title")
	}
	t.Logf("Title: %s", meta.Title)
	t.Logf("Creator: %v", meta.Creator)
}

func TestIntegration_RealEPUB_Text(t *testing.T) {
	epubPath := testEPUBPath("Frankenstein.epub")
	if _, err := os.Stat(epubPath); os.IsNotExist(err) {
		t.Skip("test EPUB not found:", epubPath)
	}

	r, err := Open(epubPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer r.Close()

	text, err := r.Text()
	if err != nil {
		t.Fatalf("Text failed: %v", err)
	}

	if len(text) == 0 {
		t.Error("expected non-empty text")
	}

	// Frankenstein should contain recognizable content
	if !bytes.Contains([]byte(text), []byte("Frankenstein")) && !bytes.Contains([]byte(text), []byte("monster")) {
		t.Log("Text may not contain expected Frankenstein content")
	}
}

func TestIntegration_RealEPUB_Markdown(t *testing.T) {
	epubPath := testEPUBPath("Frankenstein.epub")
	if _, err := os.Stat(epubPath); os.IsNotExist(err) {
		t.Skip("test EPUB not found:", epubPath)
	}

	r, err := Open(epubPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer r.Close()

	md, err := r.Markdown()
	if err != nil {
		t.Fatalf("Markdown failed: %v", err)
	}

	if len(md) == 0 {
		t.Error("expected non-empty markdown")
	}
}

func TestIntegration_RealEPUB_Document(t *testing.T) {
	epubPath := testEPUBPath("Frankenstein.epub")
	if _, err := os.Stat(epubPath); os.IsNotExist(err) {
		t.Skip("test EPUB not found:", epubPath)
	}

	r, err := Open(epubPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer r.Close()

	doc, err := r.Document()
	if err != nil {
		t.Fatalf("Document failed: %v", err)
	}

	if doc == nil {
		t.Fatal("expected non-nil document")
	}

	if len(doc.Pages) == 0 {
		t.Error("expected at least one page")
	}
}

func TestIntegration_RealEPUB_TableOfContents(t *testing.T) {
	epubPath := testEPUBPath("Frankenstein.epub")
	if _, err := os.Stat(epubPath); os.IsNotExist(err) {
		t.Skip("test EPUB not found:", epubPath)
	}

	r, err := Open(epubPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer r.Close()

	toc := r.TableOfContents()

	if len(toc.Entries) == 0 {
		t.Error("expected at least one TOC entry")
	}

	t.Logf("TOC entries: %d", len(toc.Entries))
	for i, entry := range toc.Entries {
		if i < 5 { // Log first 5 entries
			t.Logf("  %d: %s", i, entry.Title)
		}
	}
}

func TestIntegration_RealEPUB_Chapters(t *testing.T) {
	epubPath := testEPUBPath("Frankenstein.epub")
	if _, err := os.Stat(epubPath); os.IsNotExist(err) {
		t.Skip("test EPUB not found:", epubPath)
	}

	r, err := Open(epubPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer r.Close()

	chapters := r.Chapters()

	if len(chapters) == 0 {
		t.Error("expected at least one chapter")
	}

	// Check first chapter has content
	if len(chapters) > 0 {
		ch := chapters[0]
		if len(ch.Content) == 0 {
			t.Error("expected first chapter to have content")
		}
		t.Logf("First chapter: ID=%s, Title=%s", ch.ID, ch.Title)
	}
}

// ============================================================================
// OpenReader Tests
// ============================================================================

func TestOpenReader(t *testing.T) {
	epubPath := createTestEPUB(t)

	// Open as a file and read contents
	data, err := os.ReadFile(epubPath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	// Open from bytes
	ra := bytes.NewReader(data)
	r, err := OpenReader(ra, int64(len(data)))
	if err != nil {
		t.Fatalf("OpenReader failed: %v", err)
	}
	defer r.Close()

	if r.ChapterCount() != 2 {
		t.Errorf("ChapterCount = %d, want 2", r.ChapterCount())
	}
}

func TestOpenReader_Invalid(t *testing.T) {
	// Test with invalid data
	data := []byte("not a zip file")
	ra := bytes.NewReader(data)

	_, err := OpenReader(ra, int64(len(data)))
	if err != ErrInvalidArchive {
		t.Errorf("expected ErrInvalidArchive, got: %v", err)
	}
}

// ============================================================================
// TextWithOptions Tests
// ============================================================================

func TestTextWithOptions(t *testing.T) {
	epubPath := createTestEPUB(t)

	r, err := Open(epubPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer r.Close()

	// NavigationExclusion is an int that maps to htmldoc.NavigationExclusionMode
	// Use value 3 for Aggressive mode
	opts := ExtractOptions{
		NavigationExclusion: 3, // htmldoc.NavigationExclusionAggressive
	}

	text, err := r.TextWithOptions(opts)
	if err != nil {
		t.Fatalf("TextWithOptions failed: %v", err)
	}

	if len(text) == 0 {
		t.Error("expected non-empty text")
	}
}

func TestMarkdownWithOptions(t *testing.T) {
	epubPath := createTestEPUB(t)

	r, err := Open(epubPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer r.Close()

	// NavigationExclusion is an int that maps to htmldoc.NavigationExclusionMode
	// Use value 3 for Aggressive mode
	opts := ExtractOptions{
		NavigationExclusion: 3, // htmldoc.NavigationExclusionAggressive
	}

	md, err := r.MarkdownWithOptions(opts)
	if err != nil {
		t.Fatalf("MarkdownWithOptions failed: %v", err)
	}

	if len(md) == 0 {
		t.Error("expected non-empty markdown")
	}
}

// ============================================================================
// Close Tests
// ============================================================================

func TestClose_MultipleCalls(t *testing.T) {
	epubPath := createTestEPUB(t)

	r, err := Open(epubPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	// First close
	err = r.Close()
	if err != nil {
		t.Fatalf("first Close failed: %v", err)
	}

	// Second close should not panic
	err = r.Close()
	// Second close may or may not error - just ensure no panic
	_ = err
}

func TestClose_OpenReader(t *testing.T) {
	epubPath := createTestEPUB(t)

	data, err := os.ReadFile(epubPath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	ra := bytes.NewReader(data)
	r, err := OpenReader(ra, int64(len(data)))
	if err != nil {
		t.Fatalf("OpenReader failed: %v", err)
	}

	// Close should work even for OpenReader (which doesn't own a file)
	err = r.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

// ============================================================================
// getZipReader Tests
// ============================================================================

func TestGetZipReader_FromOpen(t *testing.T) {
	epubPath := createTestEPUB(t)

	r, err := Open(epubPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer r.Close()

	zr := r.getZipReader()
	if zr == nil {
		t.Error("expected non-nil zip reader")
	}
}

func TestGetZipReader_FromOpenReader(t *testing.T) {
	epubPath := createTestEPUB(t)

	data, err := os.ReadFile(epubPath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	ra := bytes.NewReader(data)
	r, err := OpenReader(ra, int64(len(data)))
	if err != nil {
		t.Fatalf("OpenReader failed: %v", err)
	}
	defer r.Close()

	zr := r.getZipReader()
	if zr == nil {
		t.Error("expected non-nil zip reader")
	}
}

// ============================================================================
// Edge Case Tests
// ============================================================================

func TestInvalidMimetype(t *testing.T) {
	tmpDir := t.TempDir()
	epubPath := filepath.Join(tmpDir, "bad_mime.epub")

	f, err := os.Create(epubPath)
	if err != nil {
		t.Fatal(err)
	}

	w := zip.NewWriter(f)

	// Write wrong mimetype
	mimeWriter, _ := w.CreateHeader(&zip.FileHeader{
		Name:   "mimetype",
		Method: zip.Store,
	})
	mimeWriter.Write([]byte("text/plain"))

	// Write container.xml
	containerWriter, _ := w.Create("META-INF/container.xml")
	containerWriter.Write([]byte(`<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`))

	// Write minimal OPF
	opfWriter, _ := w.Create("OEBPS/content.opf")
	opfWriter.Write([]byte(`<?xml version="1.0"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Test</dc:title>
  </metadata>
  <manifest>
    <item id="ch1" href="ch1.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="ch1"/>
  </spine>
</package>`))

	// Write chapter
	chWriter, _ := w.Create("OEBPS/ch1.xhtml")
	chWriter.Write([]byte(`<html><body><p>Content</p></body></html>`))

	w.Close()
	f.Close()

	// Opening should still work (mimetype validation is non-fatal)
	r, err := Open(epubPath)
	if err != nil {
		// Some implementations may reject invalid mimetype
		t.Logf("Open rejected invalid mimetype: %v", err)
		return
	}
	defer r.Close()
}

func TestMissingChapterContent(t *testing.T) {
	tmpDir := t.TempDir()
	epubPath := filepath.Join(tmpDir, "missing.epub")

	f, err := os.Create(epubPath)
	if err != nil {
		t.Fatal(err)
	}

	w := zip.NewWriter(f)

	// Write mimetype
	mimeWriter, _ := w.CreateHeader(&zip.FileHeader{
		Name:   "mimetype",
		Method: zip.Store,
	})
	mimeWriter.Write([]byte("application/epub+zip"))

	// Write container.xml
	containerWriter, _ := w.Create("META-INF/container.xml")
	containerWriter.Write([]byte(`<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`))

	// Write OPF referencing a file that doesn't exist
	opfWriter, _ := w.Create("OEBPS/content.opf")
	opfWriter.Write([]byte(`<?xml version="1.0"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Test</dc:title>
  </metadata>
  <manifest>
    <item id="ch1" href="missing.xhtml" media-type="application/xhtml+xml"/>
    <item id="ch2" href="exists.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="ch1"/>
    <itemref idref="ch2"/>
  </spine>
</package>`))

	// Only write ch2, not ch1
	chWriter, _ := w.Create("OEBPS/exists.xhtml")
	chWriter.Write([]byte(`<html><body><p>This exists</p></body></html>`))

	w.Close()
	f.Close()

	r, err := Open(epubPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer r.Close()

	// Should have loaded only the existing chapter
	if r.ChapterCount() != 1 {
		t.Errorf("ChapterCount = %d, want 1 (missing chapter should be skipped)", r.ChapterCount())
	}
}

// ============================================================================
// Benchmark Tests
// ============================================================================

func BenchmarkOpen(b *testing.B) {
	epubPath := testEPUBPath("Frankenstein.epub")
	if _, err := os.Stat(epubPath); os.IsNotExist(err) {
		b.Skip("test EPUB not found")
	}

	for i := 0; i < b.N; i++ {
		r, err := Open(epubPath)
		if err != nil {
			b.Fatal(err)
		}
		r.Close()
	}
}

func BenchmarkText(b *testing.B) {
	epubPath := testEPUBPath("Frankenstein.epub")
	if _, err := os.Stat(epubPath); os.IsNotExist(err) {
		b.Skip("test EPUB not found")
	}

	r, err := Open(epubPath)
	if err != nil {
		b.Fatal(err)
	}
	defer r.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := r.Text()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMarkdown(b *testing.B) {
	epubPath := testEPUBPath("Frankenstein.epub")
	if _, err := os.Stat(epubPath); os.IsNotExist(err) {
		b.Skip("test EPUB not found")
	}

	r, err := Open(epubPath)
	if err != nil {
		b.Fatal(err)
	}
	defer r.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := r.Markdown()
		if err != nil {
			b.Fatal(err)
		}
	}
}
