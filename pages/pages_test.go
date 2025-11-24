package pages

import (
	"fmt"
	"testing"

	"github.com/tsawler/tabula/core"
)

// mockResolver is a mock ObjectResolver for testing
type mockResolver struct {
	objects map[int]core.Object
}

func newMockResolver() *mockResolver {
	return &mockResolver{
		objects: make(map[int]core.Object),
	}
}

func (m *mockResolver) AddObject(num int, obj core.Object) {
	m.objects[num] = obj
}

func (m *mockResolver) Resolve(obj core.Object) (core.Object, error) {
	if ref, ok := obj.(core.IndirectRef); ok {
		return m.ResolveReference(ref)
	}
	return obj, nil
}

func (m *mockResolver) ResolveDeep(obj core.Object) (core.Object, error) {
	return m.Resolve(obj)
}

func (m *mockResolver) ResolveReference(ref core.IndirectRef) (core.Object, error) {
	obj, ok := m.objects[ref.Number]
	if !ok {
		return nil, fmt.Errorf("object %d not found", ref.Number)
	}
	return obj, nil
}

// TestNewCatalog tests catalog creation
func TestNewCatalog(t *testing.T) {
	resolver := newMockResolver()
	dict := core.Dict{
		"Type": core.Name("Catalog"),
	}

	catalog := NewCatalog(dict, resolver)
	if catalog == nil {
		t.Fatal("expected catalog")
	}

	if catalog.Type() != "Catalog" {
		t.Errorf("expected Type=Catalog, got %s", catalog.Type())
	}
}

// TestCatalogPages tests getting pages from catalog
func TestCatalogPages(t *testing.T) {
	resolver := newMockResolver()

	pagesDict := core.Dict{
		"Type":  core.Name("Pages"),
		"Count": core.Int(1),
		"Kids":  core.Array{},
	}
	resolver.AddObject(2, pagesDict)

	catalogDict := core.Dict{
		"Type":  core.Name("Catalog"),
		"Pages": core.IndirectRef{Number: 2},
	}

	catalog := NewCatalog(catalogDict, resolver)
	pages, err := catalog.Pages()
	if err != nil {
		t.Fatalf("failed to get pages: %v", err)
	}

	if pages == nil {
		t.Fatal("expected pages dict")
	}

	typeObj := pages.Get("Type")
	if typeName, ok := typeObj.(core.Name); !ok || string(typeName) != "Pages" {
		t.Errorf("expected Type=Pages, got %v", typeObj)
	}
}

// TestCatalogVersion tests getting version from catalog
func TestCatalogVersion(t *testing.T) {
	resolver := newMockResolver()

	dict := core.Dict{
		"Type":    core.Name("Catalog"),
		"Version": core.Name("1.7"),
	}

	catalog := NewCatalog(dict, resolver)
	version := catalog.Version()
	if version != "1.7" {
		t.Errorf("expected version 1.7, got %s", version)
	}
}

// TestCatalogMetadata tests getting metadata from catalog
func TestCatalogMetadata(t *testing.T) {
	resolver := newMockResolver()

	metadataStream := &core.Stream{
		Dict: core.Dict{
			"Type": core.Name("Metadata"),
		},
		Data: []byte("metadata"),
	}
	resolver.AddObject(10, metadataStream)

	catalogDict := core.Dict{
		"Type":     core.Name("Catalog"),
		"Metadata": core.IndirectRef{Number: 10},
	}

	catalog := NewCatalog(catalogDict, resolver)
	metadata, err := catalog.Metadata()
	if err != nil {
		t.Fatalf("failed to get metadata: %v", err)
	}

	if metadata == nil {
		t.Fatal("expected metadata stream")
	}

	if string(metadata.Data) != "metadata" {
		t.Errorf("unexpected metadata: %s", metadata.Data)
	}
}

// TestPageTreeFlatStructure tests a flat page tree
func TestPageTreeFlatStructure(t *testing.T) {
	resolver := newMockResolver()

	// Create 3 pages
	page1 := core.Dict{
		"Type":     core.Name("Page"),
		"MediaBox": core.Array{core.Int(0), core.Int(0), core.Int(612), core.Int(792)},
	}
	page2 := core.Dict{
		"Type":     core.Name("Page"),
		"MediaBox": core.Array{core.Int(0), core.Int(0), core.Int(612), core.Int(792)},
	}
	page3 := core.Dict{
		"Type":     core.Name("Page"),
		"MediaBox": core.Array{core.Int(0), core.Int(0), core.Int(612), core.Int(792)},
	}

	resolver.AddObject(10, page1)
	resolver.AddObject(11, page2)
	resolver.AddObject(12, page3)

	// Create page tree root
	pagesRoot := core.Dict{
		"Type":  core.Name("Pages"),
		"Count": core.Int(3),
		"Kids": core.Array{
			core.IndirectRef{Number: 10},
			core.IndirectRef{Number: 11},
			core.IndirectRef{Number: 12},
		},
	}

	tree := NewPageTree(pagesRoot, resolver)

	// Test count
	count, err := tree.Count()
	if err != nil {
		t.Fatalf("failed to get count: %v", err)
	}
	if count != 3 {
		t.Errorf("expected count=3, got %d", count)
	}

	// Test getting all pages
	pages, err := tree.Pages()
	if err != nil {
		t.Fatalf("failed to get pages: %v", err)
	}
	if len(pages) != 3 {
		t.Errorf("expected 3 pages, got %d", len(pages))
	}

	// Test getting page by index
	page, err := tree.GetPage(0)
	if err != nil {
		t.Fatalf("failed to get page 0: %v", err)
	}
	if page == nil {
		t.Fatal("expected page 0")
	}

	page, err = tree.GetPage(2)
	if err != nil {
		t.Fatalf("failed to get page 2: %v", err)
	}
	if page == nil {
		t.Fatal("expected page 2")
	}
}

// TestPageTreeNestedStructure tests a nested page tree
func TestPageTreeNestedStructure(t *testing.T) {
	resolver := newMockResolver()

	// Create 4 pages
	page1 := core.Dict{"Type": core.Name("Page"), "MediaBox": core.Array{core.Int(0), core.Int(0), core.Int(612), core.Int(792)}}
	page2 := core.Dict{"Type": core.Name("Page"), "MediaBox": core.Array{core.Int(0), core.Int(0), core.Int(612), core.Int(792)}}
	page3 := core.Dict{"Type": core.Name("Page"), "MediaBox": core.Array{core.Int(0), core.Int(0), core.Int(612), core.Int(792)}}
	page4 := core.Dict{"Type": core.Name("Page"), "MediaBox": core.Array{core.Int(0), core.Int(0), core.Int(612), core.Int(792)}}

	resolver.AddObject(10, page1)
	resolver.AddObject(11, page2)
	resolver.AddObject(12, page3)
	resolver.AddObject(13, page4)

	// Create intermediate pages nodes
	pages1 := core.Dict{
		"Type":  core.Name("Pages"),
		"Count": core.Int(2),
		"Kids": core.Array{
			core.IndirectRef{Number: 10},
			core.IndirectRef{Number: 11},
		},
	}
	pages2 := core.Dict{
		"Type":  core.Name("Pages"),
		"Count": core.Int(2),
		"Kids": core.Array{
			core.IndirectRef{Number: 12},
			core.IndirectRef{Number: 13},
		},
	}

	resolver.AddObject(20, pages1)
	resolver.AddObject(21, pages2)

	// Create root
	pagesRoot := core.Dict{
		"Type":  core.Name("Pages"),
		"Count": core.Int(4),
		"Kids": core.Array{
			core.IndirectRef{Number: 20},
			core.IndirectRef{Number: 21},
		},
	}

	tree := NewPageTree(pagesRoot, resolver)

	// Test count
	count, err := tree.Count()
	if err != nil {
		t.Fatalf("failed to get count: %v", err)
	}
	if count != 4 {
		t.Errorf("expected count=4, got %d", count)
	}

	// Test getting all pages
	pages, err := tree.Pages()
	if err != nil {
		t.Fatalf("failed to get pages: %v", err)
	}
	if len(pages) != 4 {
		t.Errorf("expected 4 pages, got %d", len(pages))
	}
}

// TestPageMediaBox tests getting MediaBox from page
func TestPageMediaBox(t *testing.T) {
	resolver := newMockResolver()

	pageDict := core.Dict{
		"Type":     core.Name("Page"),
		"MediaBox": core.Array{core.Int(0), core.Int(0), core.Int(612), core.Int(792)},
	}

	page := NewPage(pageDict, nil, resolver)
	mediaBox, err := page.MediaBox()
	if err != nil {
		t.Fatalf("failed to get MediaBox: %v", err)
	}

	expected := []float64{0, 0, 612, 792}
	for i, v := range expected {
		if mediaBox[i] != v {
			t.Errorf("MediaBox[%d] = %f, expected %f", i, mediaBox[i], v)
		}
	}
}

// TestPageInheritableMediaBox tests MediaBox inheritance from parent
func TestPageInheritableMediaBox(t *testing.T) {
	resolver := newMockResolver()

	// Parent has MediaBox
	parent := core.Dict{
		"Type":     core.Name("Pages"),
		"MediaBox": core.Array{core.Int(0), core.Int(0), core.Int(612), core.Int(792)},
	}

	// Page doesn't have MediaBox (should inherit)
	pageDict := core.Dict{
		"Type": core.Name("Page"),
	}

	page := NewPage(pageDict, parent, resolver)
	mediaBox, err := page.MediaBox()
	if err != nil {
		t.Fatalf("failed to get inherited MediaBox: %v", err)
	}

	if len(mediaBox) != 4 {
		t.Errorf("expected MediaBox length 4, got %d", len(mediaBox))
	}
}

// TestPageCropBox tests getting CropBox from page
func TestPageCropBox(t *testing.T) {
	resolver := newMockResolver()

	pageDict := core.Dict{
		"Type":     core.Name("Page"),
		"MediaBox": core.Array{core.Int(0), core.Int(0), core.Int(612), core.Int(792)},
		"CropBox":  core.Array{core.Int(10), core.Int(10), core.Int(602), core.Int(782)},
	}

	page := NewPage(pageDict, nil, resolver)

	cropBox, err := page.CropBox()
	if err != nil {
		t.Fatalf("failed to get CropBox: %v", err)
	}

	if cropBox[0] != 10 {
		t.Errorf("CropBox[0] = %f, expected 10", cropBox[0])
	}
}

// TestPageCropBoxDefaultsToMediaBox tests CropBox defaulting
func TestPageCropBoxDefaultsToMediaBox(t *testing.T) {
	resolver := newMockResolver()

	pageDict := core.Dict{
		"Type":     core.Name("Page"),
		"MediaBox": core.Array{core.Int(0), core.Int(0), core.Int(612), core.Int(792)},
		// No CropBox
	}

	page := NewPage(pageDict, nil, resolver)

	cropBox, err := page.CropBox()
	if err != nil {
		t.Fatalf("failed to get CropBox: %v", err)
	}

	// Should equal MediaBox
	if cropBox[2] != 612 {
		t.Errorf("CropBox should default to MediaBox")
	}
}

// TestPageResources tests getting resources from page
func TestPageResources(t *testing.T) {
	resolver := newMockResolver()

	resourcesDict := core.Dict{
		"Font": core.Dict{
			"F1": core.IndirectRef{Number: 100},
		},
	}

	pageDict := core.Dict{
		"Type":      core.Name("Page"),
		"MediaBox":  core.Array{core.Int(0), core.Int(0), core.Int(612), core.Int(792)},
		"Resources": resourcesDict,
	}

	page := NewPage(pageDict, nil, resolver)
	resources, err := page.Resources()
	if err != nil {
		t.Fatalf("failed to get resources: %v", err)
	}

	if resources == nil {
		t.Fatal("expected resources dict")
	}

	fontDict := resources.Get("Font")
	if fontDict == nil {
		t.Error("expected Font in resources")
	}
}

// TestPageInheritableResources tests Resources inheritance
func TestPageInheritableResources(t *testing.T) {
	resolver := newMockResolver()

	resourcesDict := core.Dict{
		"Font": core.Dict{},
	}

	parent := core.Dict{
		"Type":      core.Name("Pages"),
		"Resources": resourcesDict,
	}

	pageDict := core.Dict{
		"Type":     core.Name("Page"),
		"MediaBox": core.Array{core.Int(0), core.Int(0), core.Int(612), core.Int(792)},
		// No Resources
	}

	page := NewPage(pageDict, parent, resolver)
	resources, err := page.Resources()
	if err != nil {
		t.Fatalf("failed to get inherited resources: %v", err)
	}

	if resources == nil {
		t.Fatal("expected inherited resources")
	}
}

// TestPageContents tests getting contents from page
func TestPageContents(t *testing.T) {
	resolver := newMockResolver()

	contentsStream := &core.Stream{
		Dict: core.Dict{},
		Data: []byte("content data"),
	}

	pageDict := core.Dict{
		"Type":     core.Name("Page"),
		"MediaBox": core.Array{core.Int(0), core.Int(0), core.Int(612), core.Int(792)},
		"Contents": contentsStream,
	}

	page := NewPage(pageDict, nil, resolver)
	contents, err := page.Contents()
	if err != nil {
		t.Fatalf("failed to get contents: %v", err)
	}

	if len(contents) != 1 {
		t.Fatalf("expected 1 content stream, got %d", len(contents))
	}

	stream, ok := contents[0].(*core.Stream)
	if !ok {
		t.Fatal("expected Stream")
	}

	if string(stream.Data) != "content data" {
		t.Errorf("unexpected content data: %s", stream.Data)
	}
}

// TestPageContentsArray tests contents as array
func TestPageContentsArray(t *testing.T) {
	resolver := newMockResolver()

	stream1 := &core.Stream{Dict: core.Dict{}, Data: []byte("part1")}
	stream2 := &core.Stream{Dict: core.Dict{}, Data: []byte("part2")}

	pageDict := core.Dict{
		"Type":     core.Name("Page"),
		"MediaBox": core.Array{core.Int(0), core.Int(0), core.Int(612), core.Int(792)},
		"Contents": core.Array{stream1, stream2},
	}

	page := NewPage(pageDict, nil, resolver)
	contents, err := page.Contents()
	if err != nil {
		t.Fatalf("failed to get contents: %v", err)
	}

	if len(contents) != 2 {
		t.Fatalf("expected 2 content streams, got %d", len(contents))
	}
}

// TestPageRotate tests getting rotation from page
func TestPageRotate(t *testing.T) {
	resolver := newMockResolver()

	pageDict := core.Dict{
		"Type":     core.Name("Page"),
		"MediaBox": core.Array{core.Int(0), core.Int(0), core.Int(612), core.Int(792)},
		"Rotate":   core.Int(90),
	}

	page := NewPage(pageDict, nil, resolver)
	rotate := page.Rotate()
	if rotate != 90 {
		t.Errorf("expected rotation 90, got %d", rotate)
	}
}

// TestPageWidthHeight tests page dimensions
func TestPageWidthHeight(t *testing.T) {
	resolver := newMockResolver()

	pageDict := core.Dict{
		"Type":     core.Name("Page"),
		"MediaBox": core.Array{core.Int(0), core.Int(0), core.Int(612), core.Int(792)},
	}

	page := NewPage(pageDict, nil, resolver)

	width, err := page.Width()
	if err != nil {
		t.Fatalf("failed to get width: %v", err)
	}
	if width != 612 {
		t.Errorf("expected width 612, got %f", width)
	}

	height, err := page.Height()
	if err != nil {
		t.Fatalf("failed to get height: %v", err)
	}
	if height != 792 {
		t.Errorf("expected height 792, got %f", height)
	}
}

// TestPageTreeOutOfBounds tests index out of bounds
func TestPageTreeOutOfBounds(t *testing.T) {
	resolver := newMockResolver()

	page1 := core.Dict{
		"Type":     core.Name("Page"),
		"MediaBox": core.Array{core.Int(0), core.Int(0), core.Int(612), core.Int(792)},
	}
	resolver.AddObject(10, page1)

	pagesRoot := core.Dict{
		"Type":  core.Name("Pages"),
		"Count": core.Int(1),
		"Kids": core.Array{
			core.IndirectRef{Number: 10},
		},
	}

	tree := NewPageTree(pagesRoot, resolver)

	// Index too large
	_, err := tree.GetPage(5)
	if err == nil {
		t.Error("expected error for out of bounds index")
	}

	// Negative index
	_, err = tree.GetPage(-1)
	if err == nil {
		t.Error("expected error for negative index")
	}
}

// TestPageMissingMediaBox tests error when MediaBox missing
func TestPageMissingMediaBox(t *testing.T) {
	resolver := newMockResolver()

	pageDict := core.Dict{
		"Type": core.Name("Page"),
		// No MediaBox
	}

	page := NewPage(pageDict, nil, resolver)
	_, err := page.MediaBox()
	if err == nil {
		t.Error("expected error when MediaBox missing")
	}
}
