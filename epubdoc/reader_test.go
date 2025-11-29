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
