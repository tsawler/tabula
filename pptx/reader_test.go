package pptx

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsawler/tabula/rag"
)

// writeZipFile writes a file into a zip archive.
func writeZipFile(t *testing.T, zw *zip.Writer, name, content string) {
	t.Helper()
	w, err := zw.Create(name)
	if err != nil {
		t.Fatalf("Failed to create %s in zip: %v", name, err)
	}
	if _, err := w.Write([]byte(content)); err != nil {
		t.Fatalf("Failed to write %s: %v", name, err)
	}
}

// createMinimalPPTX creates a minimal valid PPTX file for testing.
func createMinimalPPTX(t *testing.T) string {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "test-*.pptx")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()

	f, err := os.Create(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)

	// [Content_Types].xml
	contentTypes := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/ppt/presentation.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.presentation.main+xml"/>
  <Override PartName="/ppt/slides/slide1.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slide+xml"/>
</Types>`
	writeZipFile(t, zw, "[Content_Types].xml", contentTypes)

	// _rels/.rels
	rels := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="ppt/presentation.xml"/>
</Relationships>`
	writeZipFile(t, zw, "_rels/.rels", rels)

	// ppt/_rels/presentation.xml.rels
	presRels := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slide" Target="slides/slide1.xml"/>
</Relationships>`
	writeZipFile(t, zw, "ppt/_rels/presentation.xml.rels", presRels)

	// ppt/presentation.xml
	presentation := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:presentation xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <p:sldIdLst>
    <p:sldId id="256" r:id="rId1"/>
  </p:sldIdLst>
  <p:sldSz cx="9144000" cy="6858000"/>
</p:presentation>`
	writeZipFile(t, zw, "ppt/presentation.xml", presentation)

	// ppt/slides/slide1.xml
	slide := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
  <p:cSld>
    <p:spTree>
      <p:nvGrpSpPr>
        <p:cNvPr id="1" name=""/>
      </p:nvGrpSpPr>
      <p:sp>
        <p:nvSpPr>
          <p:cNvPr id="2" name="Title 1"/>
          <p:nvPr>
            <p:ph type="title"/>
          </p:nvPr>
        </p:nvSpPr>
        <p:spPr>
          <a:xfrm>
            <a:off x="457200" y="274638"/>
            <a:ext cx="8229600" cy="1143000"/>
          </a:xfrm>
        </p:spPr>
        <p:txBody>
          <a:bodyPr/>
          <a:p>
            <a:r>
              <a:t>Test Title</a:t>
            </a:r>
          </a:p>
        </p:txBody>
      </p:sp>
      <p:sp>
        <p:nvSpPr>
          <p:cNvPr id="3" name="Content 1"/>
          <p:nvPr>
            <p:ph type="body" idx="1"/>
          </p:nvPr>
        </p:nvSpPr>
        <p:spPr/>
        <p:txBody>
          <a:bodyPr/>
          <a:p>
            <a:pPr lvl="0"/>
            <a:r>
              <a:t>First bullet point</a:t>
            </a:r>
          </a:p>
          <a:p>
            <a:pPr lvl="0"/>
            <a:r>
              <a:t>Second bullet point</a:t>
            </a:r>
          </a:p>
          <a:p>
            <a:pPr lvl="1"/>
            <a:r>
              <a:t>Nested point</a:t>
            </a:r>
          </a:p>
        </p:txBody>
      </p:sp>
    </p:spTree>
  </p:cSld>
</p:sld>`
	writeZipFile(t, zw, "ppt/slides/slide1.xml", slide)

	if err := zw.Close(); err != nil {
		t.Fatalf("Failed to close zip writer: %v", err)
	}

	return tmpFile.Name()
}

// createPPTXWithTable creates a PPTX with a table for testing.
func createPPTXWithTable(t *testing.T) string {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "test-table-*.pptx")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()

	f, err := os.Create(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)

	// [Content_Types].xml
	contentTypes := `<?xml version="1.0" encoding="UTF-8"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/ppt/presentation.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.presentation.main+xml"/>
  <Override PartName="/ppt/slides/slide1.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slide+xml"/>
</Types>`
	writeZipFile(t, zw, "[Content_Types].xml", contentTypes)

	writeZipFile(t, zw, "_rels/.rels", `<?xml version="1.0"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="ppt/presentation.xml"/></Relationships>`)

	writeZipFile(t, zw, "ppt/_rels/presentation.xml.rels", `<?xml version="1.0"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slide" Target="slides/slide1.xml"/></Relationships>`)

	writeZipFile(t, zw, "ppt/presentation.xml", `<?xml version="1.0"?><p:presentation xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><p:sldIdLst><p:sldId id="256" r:id="rId1"/></p:sldIdLst></p:presentation>`)

	// Slide with table
	slide := `<?xml version="1.0" encoding="UTF-8"?>
<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
  <p:cSld>
    <p:spTree>
      <p:nvGrpSpPr>
        <p:cNvPr id="1" name=""/>
      </p:nvGrpSpPr>
      <p:sp>
        <p:nvSpPr>
          <p:cNvPr id="2" name="Title"/>
          <p:nvPr><p:ph type="title"/></p:nvPr>
        </p:nvSpPr>
        <p:spPr/>
        <p:txBody>
          <a:bodyPr/>
          <a:p><a:r><a:t>Table Slide</a:t></a:r></a:p>
        </p:txBody>
      </p:sp>
      <p:graphicFrame>
        <p:nvGraphicFramePr>
          <p:cNvPr id="4" name="Table"/>
        </p:nvGraphicFramePr>
        <a:graphic>
          <a:graphicData uri="http://schemas.openxmlformats.org/drawingml/2006/table">
            <a:tbl>
              <a:tblGrid>
                <a:gridCol w="1000000"/>
                <a:gridCol w="1000000"/>
              </a:tblGrid>
              <a:tr h="500000">
                <a:tc>
                  <a:txBody>
                    <a:bodyPr/>
                    <a:p><a:r><a:t>Header 1</a:t></a:r></a:p>
                  </a:txBody>
                </a:tc>
                <a:tc>
                  <a:txBody>
                    <a:bodyPr/>
                    <a:p><a:r><a:t>Header 2</a:t></a:r></a:p>
                  </a:txBody>
                </a:tc>
              </a:tr>
              <a:tr h="500000">
                <a:tc>
                  <a:txBody>
                    <a:bodyPr/>
                    <a:p><a:r><a:t>Cell 1</a:t></a:r></a:p>
                  </a:txBody>
                </a:tc>
                <a:tc>
                  <a:txBody>
                    <a:bodyPr/>
                    <a:p><a:r><a:t>Cell 2</a:t></a:r></a:p>
                  </a:txBody>
                </a:tc>
              </a:tr>
            </a:tbl>
          </a:graphicData>
        </a:graphic>
      </p:graphicFrame>
    </p:spTree>
  </p:cSld>
</p:sld>`
	writeZipFile(t, zw, "ppt/slides/slide1.xml", slide)

	if err := zw.Close(); err != nil {
		t.Fatalf("Failed to close zip writer: %v", err)
	}

	return tmpFile.Name()
}

// createMultiSlidePPTX creates a PPTX with multiple slides.
func createMultiSlidePPTX(t *testing.T, numSlides int) string {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "test-multi-*.pptx")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()

	f, err := os.Create(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)

	// Content types
	var contentTypes strings.Builder
	contentTypes.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/ppt/presentation.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.presentation.main+xml"/>`)
	for i := 1; i <= numSlides; i++ {
		contentTypes.WriteString("\n  <Override PartName=\"/ppt/slides/slide")
		contentTypes.WriteString(itoa(i))
		contentTypes.WriteString(".xml\" ContentType=\"application/vnd.openxmlformats-officedocument.presentationml.slide+xml\"/>")
	}
	contentTypes.WriteString("\n</Types>")
	writeZipFile(t, zw, "[Content_Types].xml", contentTypes.String())

	writeZipFile(t, zw, "_rels/.rels", `<?xml version="1.0"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="ppt/presentation.xml"/></Relationships>`)

	// Presentation relationships
	var presRels strings.Builder
	presRels.WriteString(`<?xml version="1.0"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">`)
	for i := 1; i <= numSlides; i++ {
		presRels.WriteString("\n  <Relationship Id=\"rId")
		presRels.WriteString(itoa(i))
		presRels.WriteString("\" Type=\"http://schemas.openxmlformats.org/officeDocument/2006/relationships/slide\" Target=\"slides/slide")
		presRels.WriteString(itoa(i))
		presRels.WriteString(".xml\"/>")
	}
	presRels.WriteString("\n</Relationships>")
	writeZipFile(t, zw, "ppt/_rels/presentation.xml.rels", presRels.String())

	// Presentation
	var pres strings.Builder
	pres.WriteString(`<?xml version="1.0"?><p:presentation xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><p:sldIdLst>`)
	for i := 1; i <= numSlides; i++ {
		pres.WriteString("\n  <p:sldId id=\"")
		pres.WriteString(itoa(255 + i))
		pres.WriteString("\" r:id=\"rId")
		pres.WriteString(itoa(i))
		pres.WriteString("\"/>")
	}
	pres.WriteString("\n</p:sldIdLst></p:presentation>")
	writeZipFile(t, zw, "ppt/presentation.xml", pres.String())

	// Slides
	for i := 1; i <= numSlides; i++ {
		slide := `<?xml version="1.0"?>
<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
  <p:cSld>
    <p:spTree>
      <p:nvGrpSpPr><p:cNvPr id="1" name=""/></p:nvGrpSpPr>
      <p:sp>
        <p:nvSpPr>
          <p:cNvPr id="2" name="Title"/>
          <p:nvPr><p:ph type="title"/></p:nvPr>
        </p:nvSpPr>
        <p:spPr/>
        <p:txBody>
          <a:bodyPr/>
          <a:p><a:r><a:t>Slide ` + itoa(i) + `</a:t></a:r></a:p>
        </p:txBody>
      </p:sp>
      <p:sp>
        <p:nvSpPr>
          <p:cNvPr id="3" name="Body"/>
          <p:nvPr><p:ph type="body"/></p:nvPr>
        </p:nvSpPr>
        <p:spPr/>
        <p:txBody>
          <a:bodyPr/>
          <a:p><a:r><a:t>Content for slide ` + itoa(i) + `</a:t></a:r></a:p>
        </p:txBody>
      </p:sp>
    </p:spTree>
  </p:cSld>
</p:sld>`
		writeZipFile(t, zw, "ppt/slides/slide"+itoa(i)+".xml", slide)
	}

	if err := zw.Close(); err != nil {
		t.Fatalf("Failed to close zip writer: %v", err)
	}

	return tmpFile.Name()
}

func itoa(n int) string {
	if n < 0 {
		return "-" + itoa(-n)
	}
	if n < 10 {
		return string(rune('0' + n))
	}
	return itoa(n/10) + string(rune('0'+n%10))
}

func TestOpen(t *testing.T) {
	path := createMinimalPPTX(t)
	defer os.Remove(path)

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer r.Close()

	if r.SlideCount() != 1 {
		t.Errorf("SlideCount() = %d, want 1", r.SlideCount())
	}
}

func TestOpen_NotFound(t *testing.T) {
	_, err := Open("/nonexistent/file.pptx")
	if err == nil {
		t.Error("Open() expected error for nonexistent file")
	}
}

func TestOpen_InvalidZip(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-*.pptx")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.WriteString("not a zip file")
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	_, err = Open(tmpFile.Name())
	if err == nil {
		t.Error("Open() expected error for invalid zip")
	}
}

func TestOpen_MissingPresentation(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-*.pptx")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()

	f, _ := os.Create(tmpFile.Name())
	zw := zip.NewWriter(f)
	writeZipFile(t, zw, "[Content_Types].xml", "<Types/>")
	zw.Close()
	f.Close()
	defer os.Remove(tmpFile.Name())

	_, err = Open(tmpFile.Name())
	if err == nil {
		t.Error("Open() expected error for missing presentation.xml")
	}
}

func TestOpen_NoSlides(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-*.pptx")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()

	f, _ := os.Create(tmpFile.Name())
	zw := zip.NewWriter(f)
	writeZipFile(t, zw, "[Content_Types].xml", "<Types/>")
	writeZipFile(t, zw, "ppt/presentation.xml", "<presentation/>")
	zw.Close()
	f.Close()
	defer os.Remove(tmpFile.Name())

	_, err = Open(tmpFile.Name())
	if err == nil {
		t.Error("Open() expected error for missing slides")
	}
}

func TestReader_Close(t *testing.T) {
	path := createMinimalPPTX(t)
	defer os.Remove(path)

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}

	if err := r.Close(); err != nil {
		t.Errorf("Close() failed: %v", err)
	}

	// Second close should be safe
	if err := r.Close(); err != nil {
		t.Errorf("Second Close() failed: %v", err)
	}
}

func TestReader_SlideCount(t *testing.T) {
	tests := []struct {
		name   string
		slides int
	}{
		{"single slide", 1},
		{"multiple slides", 3},
		{"five slides", 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := createMultiSlidePPTX(t, tt.slides)
			defer os.Remove(path)

			r, err := Open(path)
			if err != nil {
				t.Fatalf("Open() failed: %v", err)
			}
			defer r.Close()

			if got := r.SlideCount(); got != tt.slides {
				t.Errorf("SlideCount() = %d, want %d", got, tt.slides)
			}
		})
	}
}

func TestReader_Slide(t *testing.T) {
	path := createMultiSlidePPTX(t, 3)
	defer os.Remove(path)

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer r.Close()

	// Valid indices
	for i := 0; i < 3; i++ {
		slide, err := r.Slide(i)
		if err != nil {
			t.Errorf("Slide(%d) failed: %v", i, err)
		}
		if slide == nil {
			t.Errorf("Slide(%d) returned nil", i)
		}
	}

	// Invalid indices
	_, err = r.Slide(-1)
	if err == nil {
		t.Error("Slide(-1) expected error")
	}

	_, err = r.Slide(100)
	if err == nil {
		t.Error("Slide(100) expected error")
	}
}

func TestReader_PageCount(t *testing.T) {
	path := createMultiSlidePPTX(t, 5)
	defer os.Remove(path)

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer r.Close()

	count, err := r.PageCount()
	if err != nil {
		t.Errorf("PageCount() failed: %v", err)
	}
	if count != 5 {
		t.Errorf("PageCount() = %d, want 5", count)
	}
}

func TestReader_Text(t *testing.T) {
	path := createMinimalPPTX(t)
	defer os.Remove(path)

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer r.Close()

	text, err := r.Text()
	if err != nil {
		t.Errorf("Text() failed: %v", err)
	}

	// Check title is included
	if !strings.Contains(text, "Test Title") {
		t.Errorf("Text() missing title, got: %s", text)
	}

	// Check content is included
	if !strings.Contains(text, "First bullet point") {
		t.Errorf("Text() missing content, got: %s", text)
	}
}

func TestReader_TextWithOptions(t *testing.T) {
	path := createMultiSlidePPTX(t, 3)
	defer os.Remove(path)

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer r.Close()

	// Test slide selection
	text, err := r.TextWithOptions(ExtractOptions{
		SlideNumbers:  []int{0, 2},
		IncludeTitles: true,
	})
	if err != nil {
		t.Errorf("TextWithOptions() failed: %v", err)
	}

	if !strings.Contains(text, "Slide 1") {
		t.Errorf("Text should contain Slide 1, got: %s", text)
	}
	if !strings.Contains(text, "Slide 3") {
		t.Errorf("Text should contain Slide 3, got: %s", text)
	}
	if strings.Contains(text, "Slide 2") {
		t.Errorf("Text should NOT contain Slide 2, got: %s", text)
	}
}

func TestReader_Markdown(t *testing.T) {
	path := createMinimalPPTX(t)
	defer os.Remove(path)

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer r.Close()

	md, err := r.Markdown()
	if err != nil {
		t.Errorf("Markdown() failed: %v", err)
	}

	// Should have title as H1
	if !strings.Contains(md, "# Test Title") {
		t.Errorf("Markdown() missing title heading, got: %s", md)
	}

	// Should have bullet points
	if !strings.Contains(md, "First bullet point") {
		t.Errorf("Markdown() missing bullet content, got: %s", md)
	}
}

func TestReader_MarkdownWithOptions(t *testing.T) {
	path := createMultiSlidePPTX(t, 2)
	defer os.Remove(path)

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer r.Close()

	md, err := r.MarkdownWithOptions(ExtractOptions{IncludeTitles: true})
	if err != nil {
		t.Errorf("MarkdownWithOptions() failed: %v", err)
	}

	// Should have slide separator
	if !strings.Contains(md, "---") {
		t.Errorf("Markdown() missing slide separator, got: %s", md)
	}
}

func TestReader_MarkdownWithRAGOptions(t *testing.T) {
	path := createMultiSlidePPTX(t, 3)
	defer os.Remove(path)

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer r.Close()

	// Test with metadata
	md, err := r.MarkdownWithRAGOptions(
		ExtractOptions{},
		rag.MarkdownOptions{IncludeMetadata: true},
	)
	if err != nil {
		t.Errorf("MarkdownWithRAGOptions() failed: %v", err)
	}

	if !strings.Contains(md, "---") {
		t.Errorf("Missing YAML front matter, got: %s", md)
	}
	if !strings.Contains(md, "slides:") {
		t.Errorf("Missing slides metadata, got: %s", md)
	}

	// Test with table of contents
	md2, err := r.MarkdownWithRAGOptions(
		ExtractOptions{},
		rag.MarkdownOptions{IncludeTableOfContents: true},
	)
	if err != nil {
		t.Errorf("MarkdownWithRAGOptions() with TOC failed: %v", err)
	}

	if !strings.Contains(md2, "Table of Contents") {
		t.Errorf("Missing TOC, got: %s", md2)
	}
}

func TestReader_Metadata(t *testing.T) {
	path := createMinimalPPTX(t)
	defer os.Remove(path)

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer r.Close()

	meta := r.Metadata()
	// Basic metadata structure should exist
	_ = meta.Title
	_ = meta.Author
}

func TestReader_Document(t *testing.T) {
	path := createMinimalPPTX(t)
	defer os.Remove(path)

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer r.Close()

	doc, err := r.Document()
	if err != nil {
		t.Errorf("Document() failed: %v", err)
	}

	if doc == nil {
		t.Fatal("Document() returned nil")
	}

	if len(doc.Pages) != 1 {
		t.Errorf("Document has %d pages, want 1", len(doc.Pages))
	}
}

func TestReader_Table(t *testing.T) {
	path := createPPTXWithTable(t)
	defer os.Remove(path)

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer r.Close()

	slide, _ := r.Slide(0)
	if len(slide.Tables) == 0 {
		t.Fatal("Slide has no tables")
	}

	table := slide.Tables[0]
	if len(table.Rows) != 2 {
		t.Errorf("Table has %d rows, want 2", len(table.Rows))
	}
	if table.Columns != 2 {
		t.Errorf("Table has %d columns, want 2", table.Columns)
	}

	// Check header row
	if table.Rows[0][0].Text != "Header 1" {
		t.Errorf("Table header = %q, want 'Header 1'", table.Rows[0][0].Text)
	}
}

func TestSlide_GetText(t *testing.T) {
	path := createMinimalPPTX(t)
	defer os.Remove(path)

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer r.Close()

	slide, _ := r.Slide(0)
	text := slide.GetText()

	if !strings.Contains(text, "Test Title") {
		t.Errorf("GetText() missing title, got: %s", text)
	}
	if !strings.Contains(text, "First bullet point") {
		t.Errorf("GetText() missing content, got: %s", text)
	}
}

func TestSlide_GetMarkdown(t *testing.T) {
	path := createMinimalPPTX(t)
	defer os.Remove(path)

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer r.Close()

	slide, _ := r.Slide(0)
	md := slide.GetMarkdown()

	if !strings.Contains(md, "# Test Title") {
		t.Errorf("GetMarkdown() missing title, got: %s", md)
	}
}

func TestTable_ToMarkdown(t *testing.T) {
	table := Table{
		Columns: 2,
		Rows: [][]TableCell{
			{{Text: "A"}, {Text: "B"}},
			{{Text: "1"}, {Text: "2"}},
		},
	}

	md := table.ToMarkdown()

	if !strings.Contains(md, "| A |") {
		t.Errorf("ToMarkdown() missing header, got: %s", md)
	}
	if !strings.Contains(md, "|---|") {
		t.Errorf("ToMarkdown() missing separator, got: %s", md)
	}
	if !strings.Contains(md, "| 1 |") {
		t.Errorf("ToMarkdown() missing data, got: %s", md)
	}
}

func TestTable_ToMarkdown_Empty(t *testing.T) {
	table := Table{Rows: nil}
	md := table.ToMarkdown()
	if md != "" {
		t.Errorf("ToMarkdown() for empty table = %q, want empty", md)
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
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := escapeMarkdown(tt.input); got != tt.want {
				t.Errorf("escapeMarkdown(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestExtractSlideNumber(t *testing.T) {
	tests := []struct {
		path string
		want int
	}{
		{"ppt/slides/slide1.xml", 1},
		{"ppt/slides/slide10.xml", 10},
		{"ppt/slides/slide123.xml", 123},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := extractSlideNumber(tt.path); got != tt.want {
				t.Errorf("extractSlideNumber(%q) = %d, want %d", tt.path, got, tt.want)
			}
		})
	}
}

func TestIsFooterPlaceholder(t *testing.T) {
	tests := []struct {
		phType string
		want   bool
	}{
		{"ftr", true},
		{"dt", true},
		{"sldNum", true},
		{"title", false},
		{"body", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.phType, func(t *testing.T) {
			if got := isFooterPlaceholder(tt.phType); got != tt.want {
				t.Errorf("isFooterPlaceholder(%q) = %v, want %v", tt.phType, got, tt.want)
			}
		})
	}
}

func TestIsHeaderPlaceholder(t *testing.T) {
	tests := []struct {
		phType string
		want   bool
	}{
		{"hdr", true},
		{"title", false},
		{"body", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.phType, func(t *testing.T) {
			if got := isHeaderPlaceholder(tt.phType); got != tt.want {
				t.Errorf("isHeaderPlaceholder(%q) = %v, want %v", tt.phType, got, tt.want)
			}
		})
	}
}

// Integration test with real PPTX file
func TestIntegration_RealPPTX(t *testing.T) {
	samplePath := filepath.Join("testdata", "test.pptx")

	r, err := Open(samplePath)
	if err != nil {
		t.Fatalf("Open(%s) failed: %v", samplePath, err)
	}
	defer r.Close()

	t.Run("SlideCount", func(t *testing.T) {
		count := r.SlideCount()
		if count < 1 {
			t.Errorf("SlideCount() = %d, want >= 1", count)
		}
		t.Logf("Slide count: %d", count)
	})

	t.Run("Text", func(t *testing.T) {
		text, err := r.Text()
		if err != nil {
			t.Errorf("Text() failed: %v", err)
		}
		if text == "" {
			t.Error("Text() returned empty string")
		}
		t.Logf("Text length: %d chars", len(text))
		if len(text) > 500 {
			t.Logf("Text preview: %s...", text[:500])
		} else {
			t.Logf("Text: %s", text)
		}
	})

	t.Run("Markdown", func(t *testing.T) {
		md, err := r.Markdown()
		if err != nil {
			t.Errorf("Markdown() failed: %v", err)
		}
		if md == "" {
			t.Error("Markdown() returned empty string")
		}
		t.Logf("Markdown length: %d chars", len(md))
	})

	t.Run("Metadata", func(t *testing.T) {
		meta := r.Metadata()
		t.Logf("Metadata: Title=%q, Author=%q, Subject=%q",
			meta.Title, meta.Author, meta.Subject)
	})

	t.Run("Document", func(t *testing.T) {
		doc, err := r.Document()
		if err != nil {
			t.Errorf("Document() failed: %v", err)
		}
		if doc == nil {
			t.Fatal("Document() returned nil")
		}
		t.Logf("Document pages: %d", len(doc.Pages))
	})

	t.Run("SlideAccess", func(t *testing.T) {
		for i := 0; i < r.SlideCount(); i++ {
			slide, err := r.Slide(i)
			if err != nil {
				t.Errorf("Slide(%d) failed: %v", i, err)
				continue
			}
			t.Logf("Slide %d: Title=%q, Content blocks=%d, Tables=%d",
				i+1, slide.Title, len(slide.Content), len(slide.Tables))
		}
	})
}

// Benchmark tests
func BenchmarkOpen(b *testing.B) {
	tmpFile, _ := os.CreateTemp("", "bench-*.pptx")
	tmpFile.Close()
	path := tmpFile.Name()
	defer os.Remove(path)

	f, _ := os.Create(path)
	zw := zip.NewWriter(f)

	writeZipFileBench(zw, "[Content_Types].xml", `<?xml version="1.0"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/ppt/presentation.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.presentation.main+xml"/><Override PartName="/ppt/slides/slide1.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slide+xml"/></Types>`)

	writeZipFileBench(zw, "_rels/.rels", `<?xml version="1.0"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="ppt/presentation.xml"/></Relationships>`)

	writeZipFileBench(zw, "ppt/_rels/presentation.xml.rels", `<?xml version="1.0"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slide" Target="slides/slide1.xml"/></Relationships>`)

	writeZipFileBench(zw, "ppt/presentation.xml", `<?xml version="1.0"?><p:presentation xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><p:sldIdLst><p:sldId id="256" r:id="rId1"/></p:sldIdLst></p:presentation>`)

	writeZipFileBench(zw, "ppt/slides/slide1.xml", `<?xml version="1.0"?><p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"><p:cSld><p:spTree><p:nvGrpSpPr><p:cNvPr id="1" name=""/></p:nvGrpSpPr><p:sp><p:nvSpPr><p:cNvPr id="2" name="Title"/><p:nvPr><p:ph type="title"/></p:nvPr></p:nvSpPr><p:spPr/><p:txBody><a:bodyPr/><a:p><a:r><a:t>Test</a:t></a:r></a:p></p:txBody></p:sp></p:spTree></p:cSld></p:sld>`)

	zw.Close()
	f.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r, err := Open(path)
		if err != nil {
			b.Fatalf("Open failed: %v", err)
		}
		r.Close()
	}
}

func writeZipFileBench(zw *zip.Writer, name, content string) {
	w, _ := zw.Create(name)
	w.Write([]byte(content))
}

func BenchmarkText(b *testing.B) {
	tmpFile, _ := os.CreateTemp("", "bench-*.pptx")
	tmpFile.Close()
	path := tmpFile.Name()
	defer os.Remove(path)

	f, _ := os.Create(path)
	zw := zip.NewWriter(f)

	writeZipFileBench(zw, "[Content_Types].xml", `<?xml version="1.0"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/ppt/presentation.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.presentation.main+xml"/><Override PartName="/ppt/slides/slide1.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slide+xml"/></Types>`)

	writeZipFileBench(zw, "_rels/.rels", `<?xml version="1.0"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="ppt/presentation.xml"/></Relationships>`)

	writeZipFileBench(zw, "ppt/_rels/presentation.xml.rels", `<?xml version="1.0"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slide" Target="slides/slide1.xml"/></Relationships>`)

	writeZipFileBench(zw, "ppt/presentation.xml", `<?xml version="1.0"?><p:presentation xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><p:sldIdLst><p:sldId id="256" r:id="rId1"/></p:sldIdLst></p:presentation>`)

	writeZipFileBench(zw, "ppt/slides/slide1.xml", `<?xml version="1.0"?><p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"><p:cSld><p:spTree><p:nvGrpSpPr><p:cNvPr id="1" name=""/></p:nvGrpSpPr><p:sp><p:nvSpPr><p:cNvPr id="2" name="Title"/><p:nvPr><p:ph type="title"/></p:nvPr></p:nvSpPr><p:spPr/><p:txBody><a:bodyPr/><a:p><a:r><a:t>Test Title</a:t></a:r></a:p></p:txBody></p:sp><p:sp><p:nvSpPr><p:cNvPr id="3" name="Body"/><p:nvPr><p:ph type="body"/></p:nvPr></p:nvSpPr><p:spPr/><p:txBody><a:bodyPr/><a:p><a:r><a:t>Content paragraph one</a:t></a:r></a:p><a:p><a:r><a:t>Content paragraph two</a:t></a:r></a:p><a:p><a:r><a:t>Content paragraph three</a:t></a:r></a:p></p:txBody></p:sp></p:spTree></p:cSld></p:sld>`)

	zw.Close()
	f.Close()

	r, err := Open(path)
	if err != nil {
		b.Fatalf("Open failed: %v", err)
	}
	defer r.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := r.Text()
		if err != nil {
			b.Fatalf("Text failed: %v", err)
		}
	}
}
