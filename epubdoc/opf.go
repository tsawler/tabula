package epubdoc

import (
	"archive/zip"
	"encoding/xml"
	"errors"
	"io"
	"path"
	"strings"
	"time"
)

// OPF-related errors.
var (
	ErrNoOPF      = errors.New("epub: missing package document (OPF)")
	ErrInvalidOPF = errors.New("epub: invalid package document")
	ErrEmptySpine = errors.New("epub: no content in spine")
)

// opfPackage represents the OPF package document.
type opfPackage struct {
	XMLName  xml.Name    `xml:"package"`
	Version  string      `xml:"version,attr"`
	Metadata opfMetadata `xml:"metadata"`
	Manifest opfManifest `xml:"manifest"`
	Spine    opfSpine    `xml:"spine"`
}

type opfMetadata struct {
	Title       []dcElement `xml:"title"`
	Creator     []dcElement `xml:"creator"`
	Language    []dcElement `xml:"language"`
	Identifier  []dcElement `xml:"identifier"`
	Publisher   []dcElement `xml:"publisher"`
	Date        []dcElement `xml:"date"`
	Description []dcElement `xml:"description"`
	Subject     []dcElement `xml:"subject"`
	Rights      []dcElement `xml:"rights"`
	Meta        []opfMeta   `xml:"meta"`
}

type dcElement struct {
	ID      string `xml:"id,attr"`
	Content string `xml:",chardata"`
}

type opfMeta struct {
	Property string `xml:"property,attr"`
	Refines  string `xml:"refines,attr"`
	Name     string `xml:"name,attr"`    // EPUB 2 style
	Content  string `xml:"content,attr"` // EPUB 2 style
	Value    string `xml:",chardata"`    // EPUB 3 style
}

type opfManifest struct {
	Items []opfItem `xml:"item"`
}

type opfItem struct {
	ID         string `xml:"id,attr"`
	Href       string `xml:"href,attr"`
	MediaType  string `xml:"media-type,attr"`
	Properties string `xml:"properties,attr"`
}

type opfSpine struct {
	Toc      string       `xml:"toc,attr"` // NCX ID for EPUB 2
	ItemRefs []opfItemRef `xml:"itemref"`
}

type opfItemRef struct {
	IDRef  string `xml:"idref,attr"`
	Linear string `xml:"linear,attr"`
}

// parseOPF parses the OPF file and returns a Package struct.
func parseOPF(zr *zip.Reader, opfPath string) (*Package, string, error) {
	// Find the OPF file
	var opfFile *zip.File
	for _, f := range zr.File {
		if f.Name == opfPath {
			opfFile = f
			break
		}
	}

	if opfFile == nil {
		return nil, "", ErrNoOPF
	}

	// Get base directory for resolving relative paths
	baseDir := path.Dir(opfPath)
	if baseDir == "." {
		baseDir = ""
	}

	// Read and parse
	rc, err := opfFile.Open()
	if err != nil {
		return nil, "", err
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, "", err
	}

	var opf opfPackage
	if err := xml.Unmarshal(data, &opf); err != nil {
		return nil, "", ErrInvalidOPF
	}

	// Convert to our Package struct
	pkg := &Package{
		Version:  opf.Version,
		Metadata: convertMetadata(&opf.Metadata),
		Manifest: convertManifest(&opf.Manifest),
		Spine:    convertSpine(&opf.Spine),
	}

	if len(pkg.Spine) == 0 {
		return nil, "", ErrEmptySpine
	}

	return pkg, baseDir, nil
}

func convertMetadata(m *opfMetadata) Metadata {
	meta := Metadata{}

	// Title - take first
	if len(m.Title) > 0 {
		meta.Title = strings.TrimSpace(m.Title[0].Content)
	}

	// Creators
	for _, c := range m.Creator {
		if s := strings.TrimSpace(c.Content); s != "" {
			meta.Creator = append(meta.Creator, s)
		}
	}

	// Language - take first
	if len(m.Language) > 0 {
		meta.Language = strings.TrimSpace(m.Language[0].Content)
	}

	// Identifier - take first
	if len(m.Identifier) > 0 {
		meta.Identifier = strings.TrimSpace(m.Identifier[0].Content)
	}

	// Publisher - take first
	if len(m.Publisher) > 0 {
		meta.Publisher = strings.TrimSpace(m.Publisher[0].Content)
	}

	// Date - take first
	if len(m.Date) > 0 {
		meta.Date = strings.TrimSpace(m.Date[0].Content)
	}

	// Description - take first
	if len(m.Description) > 0 {
		meta.Description = strings.TrimSpace(m.Description[0].Content)
	}

	// Subjects
	for _, s := range m.Subject {
		if subj := strings.TrimSpace(s.Content); subj != "" {
			meta.Subjects = append(meta.Subjects, subj)
		}
	}

	// Rights - take first
	if len(m.Rights) > 0 {
		meta.Rights = strings.TrimSpace(m.Rights[0].Content)
	}

	// Check meta elements for modified date (EPUB 3)
	for _, mt := range m.Meta {
		if mt.Property == "dcterms:modified" {
			if t, err := time.Parse(time.RFC3339, mt.Value); err == nil {
				meta.Modified = t
			}
		}
	}

	return meta
}

func convertManifest(m *opfManifest) map[string]ManifestItem {
	manifest := make(map[string]ManifestItem, len(m.Items))

	for _, item := range m.Items {
		mi := ManifestItem{
			ID:        item.ID,
			Href:      item.Href,
			MediaType: item.MediaType,
		}

		// Parse properties
		if item.Properties != "" {
			mi.Properties = strings.Fields(item.Properties)
		}

		manifest[item.ID] = mi
	}

	return manifest
}

func convertSpine(s *opfSpine) []SpineItem {
	spine := make([]SpineItem, 0, len(s.ItemRefs))

	for _, ref := range s.ItemRefs {
		si := SpineItem{
			IDRef:  ref.IDRef,
			Linear: ref.Linear != "no", // Default is true
		}
		spine = append(spine, si)
	}

	return spine
}
