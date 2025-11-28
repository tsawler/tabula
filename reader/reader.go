package reader

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/tsawler/tabula/core"
	"github.com/tsawler/tabula/pages"
	"github.com/tsawler/tabula/text"
)

// PDFVersion represents a PDF version
type PDFVersion struct {
	Major int
	Minor int
}

// String returns the version as a string (e.g., "1.7")
func (v PDFVersion) String() string {
	return fmt.Sprintf("%d.%d", v.Major, v.Minor)
}

// Reader represents a PDF file reader
type Reader struct {
	file        *os.File
	xrefTable   *core.XRefTable
	trailer     core.Dict
	version     PDFVersion
	objCache    map[int]core.Object        // Cache for loaded objects
	objStmCache map[int]*core.ObjectStream // Cache for object streams
	fileSize    int64
	pageTree    *pages.PageTree // Cached page tree
}

// Ensure Reader implements pages.ObjectResolver
var _ pages.ObjectResolver = (*Reader)(nil)

// NewReader creates a new PDF reader for the given file
func NewReader(file *os.File) (*Reader, error) {
	// Get file size
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	reader := &Reader{
		file:        file,
		objCache:    make(map[int]core.Object),
		objStmCache: make(map[int]*core.ObjectStream),
		fileSize:    fileInfo.Size(),
	}

	// Parse PDF header
	version, err := reader.parseHeader()
	if err != nil {
		return nil, fmt.Errorf("failed to parse header: %w", err)
	}
	reader.version = version

	// Load XRef table
	xrefTable, err := reader.loadXRef()
	if err != nil {
		return nil, fmt.Errorf("failed to load xref: %w", err)
	}
	reader.xrefTable = xrefTable
	reader.trailer = xrefTable.Trailer

	return reader, nil
}

// Open opens a PDF file and returns a Reader
func Open(filename string) (*Reader, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	reader, err := NewReader(file)
	if err != nil {
		file.Close()
		return nil, err
	}

	return reader, nil
}

// Close closes the PDF file
func (r *Reader) Close() error {
	if r.file != nil {
		return r.file.Close()
	}
	return nil
}

// parseHeader parses the PDF header (%PDF-x.y)
func (r *Reader) parseHeader() (PDFVersion, error) {
	// Read first 8 bytes for header
	_, err := r.file.Seek(0, io.SeekStart)
	if err != nil {
		return PDFVersion{}, fmt.Errorf("failed to seek to start: %w", err)
	}

	header := make([]byte, 8)
	n, err := r.file.Read(header)
	if err != nil {
		return PDFVersion{}, fmt.Errorf("failed to read header: %w", err)
	}
	if n < 8 {
		return PDFVersion{}, fmt.Errorf("header too short: %d bytes", n)
	}

	// Parse header format: %PDF-x.y
	headerStr := string(header)
	if !strings.HasPrefix(headerStr, "%PDF-") {
		return PDFVersion{}, fmt.Errorf("invalid PDF header: %s", headerStr)
	}

	// Extract version
	versionStr := headerStr[5:] // After "%PDF-"
	re := regexp.MustCompile(`(\d+)\.(\d+)`)
	matches := re.FindStringSubmatch(versionStr)
	if len(matches) < 3 {
		return PDFVersion{}, fmt.Errorf("invalid version format: %s", versionStr)
	}

	var major, minor int
	fmt.Sscanf(matches[1], "%d", &major)
	fmt.Sscanf(matches[2], "%d", &minor)

	return PDFVersion{Major: major, Minor: minor}, nil
}

// loadXRef loads the cross-reference table
func (r *Reader) loadXRef() (*core.XRefTable, error) {
	xrefParser := core.NewXRefParser(r.file)
	table, err := xrefParser.ParseXRefFromEOF()
	if err != nil {
		return nil, fmt.Errorf("failed to parse xref: %w", err)
	}

	// Handle incremental updates if present
	if table.Trailer.Get("Prev") != nil {
		tables, err := xrefParser.ParseAllXRefs()
		if err != nil {
			return nil, fmt.Errorf("failed to parse all xrefs: %w", err)
		}
		// Merge all tables
		table = core.MergeXRefTables(tables...)
	}

	return table, nil
}

// Version returns the PDF version
func (r *Reader) Version() PDFVersion {
	return r.version
}

// Trailer returns the trailer dictionary
func (r *Reader) Trailer() core.Dict {
	return r.trailer
}

// GetObject loads an object by its number
// Uses caching to avoid re-reading objects
// Supports both uncompressed objects and objects in object streams (PDF 1.5+)
func (r *Reader) GetObject(objNum int) (core.Object, error) {
	// Check cache first
	if obj, ok := r.objCache[objNum]; ok {
		return obj, nil
	}

	// Look up in XRef table
	entry, ok := r.xrefTable.Get(objNum)
	if !ok {
		return nil, fmt.Errorf("object %d not found in xref table", objNum)
	}

	if !entry.InUse {
		return nil, fmt.Errorf("object %d is not in use", objNum)
	}

	var obj core.Object
	var err error

	// Handle based on entry type
	switch entry.Type {
	case core.XRefEntryCompressed:
		// Object is stored in an object stream (PDF 1.5+)
		obj, err = r.getCompressedObject(objNum, entry)
	case core.XRefEntryUncompressed:
		// Standard uncompressed object
		obj, err = r.getUncompressedObject(objNum, entry)
	default:
		return nil, fmt.Errorf("unexpected entry type for object %d: %v", objNum, entry.Type)
	}

	if err != nil {
		return nil, err
	}

	// Cache the object
	r.objCache[objNum] = obj

	return obj, nil
}

// getUncompressedObject reads an object directly from the file
func (r *Reader) getUncompressedObject(objNum int, entry *core.XRefEntry) (core.Object, error) {
	// Seek to object position
	_, err := r.file.Seek(entry.Offset, io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("failed to seek to object %d: %w", objNum, err)
	}

	// Parse the indirect object
	parser := core.NewParser(r.file)
	indObj, err := parser.ParseIndirectObject()
	if err != nil {
		return nil, fmt.Errorf("failed to parse object %d: %w", objNum, err)
	}

	// Verify object number matches
	if indObj.Ref.Number != objNum {
		return nil, fmt.Errorf("object number mismatch: expected %d, got %d", objNum, indObj.Ref.Number)
	}

	return indObj.Object, nil
}

// getCompressedObject extracts an object from an object stream
func (r *Reader) getCompressedObject(objNum int, entry *core.XRefEntry) (core.Object, error) {
	// For compressed entries:
	// - entry.Offset is the object stream number
	// - entry.Generation is the index within the object stream
	objStmNum := int(entry.Offset)
	index := entry.Generation

	// Get or load the object stream
	objStm, err := r.getObjectStream(objStmNum)
	if err != nil {
		return nil, fmt.Errorf("failed to get object stream %d for object %d: %w", objStmNum, objNum, err)
	}

	// Extract the object by index
	obj, extractedObjNum, err := objStm.GetObjectByIndex(index)
	if err != nil {
		return nil, fmt.Errorf("failed to extract object %d from stream %d at index %d: %w",
			objNum, objStmNum, index, err)
	}

	// Verify the object number matches
	if extractedObjNum != objNum {
		return nil, fmt.Errorf("object number mismatch in stream: expected %d, got %d", objNum, extractedObjNum)
	}

	return obj, nil
}

// getObjectStream loads and caches an object stream
func (r *Reader) getObjectStream(objStmNum int) (*core.ObjectStream, error) {
	// Check cache first
	if objStm, ok := r.objStmCache[objStmNum]; ok {
		return objStm, nil
	}

	// Load the object stream object (must be uncompressed - object streams can't be in other object streams)
	entry, ok := r.xrefTable.Get(objStmNum)
	if !ok {
		return nil, fmt.Errorf("object stream %d not found in xref table", objStmNum)
	}

	if entry.Type == core.XRefEntryCompressed {
		return nil, fmt.Errorf("object stream %d cannot be in another object stream", objStmNum)
	}

	// Load the stream object
	streamObj, err := r.getUncompressedObject(objStmNum, entry)
	if err != nil {
		return nil, fmt.Errorf("failed to load object stream %d: %w", objStmNum, err)
	}

	stream, ok := streamObj.(*core.Stream)
	if !ok {
		return nil, fmt.Errorf("object %d is not a stream (got %T)", objStmNum, streamObj)
	}

	// Create the ObjectStream wrapper
	objStm, err := core.NewObjectStream(stream)
	if err != nil {
		return nil, fmt.Errorf("failed to create object stream from object %d: %w", objStmNum, err)
	}

	// Cache the object stream
	r.objStmCache[objStmNum] = objStm

	return objStm, nil
}

// ResolveReference resolves an indirect reference
func (r *Reader) ResolveReference(ref core.IndirectRef) (core.Object, error) {
	return r.GetObject(ref.Number)
}

// GetCatalog returns the document catalog (root object)
func (r *Reader) GetCatalog() (core.Dict, error) {
	rootRef := r.trailer.Get("Root")
	if rootRef == nil {
		return nil, fmt.Errorf("trailer missing /Root entry")
	}

	ref, ok := rootRef.(core.IndirectRef)
	if !ok {
		return nil, fmt.Errorf("invalid /Root type: %T", rootRef)
	}

	obj, err := r.ResolveReference(ref)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve catalog: %w", err)
	}

	catalog, ok := obj.(core.Dict)
	if !ok {
		return nil, fmt.Errorf("catalog is not a dictionary: %T", obj)
	}

	return catalog, nil
}

// GetInfo returns the document info dictionary (metadata)
func (r *Reader) GetInfo() (core.Dict, error) {
	infoRef := r.trailer.Get("Info")
	if infoRef == nil {
		return nil, nil // Info is optional
	}

	ref, ok := infoRef.(core.IndirectRef)
	if !ok {
		return nil, fmt.Errorf("invalid /Info type: %T", infoRef)
	}

	obj, err := r.ResolveReference(ref)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve info: %w", err)
	}

	info, ok := obj.(core.Dict)
	if !ok {
		return nil, fmt.Errorf("info is not a dictionary: %T", obj)
	}

	return info, nil
}

// NumObjects returns the total number of objects in the PDF
func (r *Reader) NumObjects() int {
	sizeObj := r.trailer.Get("Size")
	if sizeObj == nil {
		return 0
	}

	size, ok := sizeObj.(core.Int)
	if !ok {
		return 0
	}

	return int(size)
}

// FileSize returns the size of the PDF file in bytes
func (r *Reader) FileSize() int64 {
	return r.fileSize
}

// XRefTable returns the cross-reference table
// Exposed for debugging/inspection
func (r *Reader) XRefTable() *core.XRefTable {
	return r.xrefTable
}

// ClearCache clears the object cache and object stream cache
// Useful for freeing memory when processing large PDFs
func (r *Reader) ClearCache() {
	r.objCache = make(map[int]core.Object)
	r.objStmCache = make(map[int]*core.ObjectStream)
}

// CacheSize returns the number of cached objects
func (r *Reader) CacheSize() int {
	return len(r.objCache)
}

// ObjectStreamCacheSize returns the number of cached object streams
func (r *Reader) ObjectStreamCacheSize() int {
	return len(r.objStmCache)
}

// Resolve resolves an object if it's an indirect reference, otherwise returns it as-is
// Implements pages.ObjectResolver interface
func (r *Reader) Resolve(obj core.Object) (core.Object, error) {
	if ref, ok := obj.(core.IndirectRef); ok {
		return r.ResolveReference(ref)
	}
	return obj, nil
}

// ResolveDeep recursively resolves all indirect references in an object
// Implements pages.ObjectResolver interface
func (r *Reader) ResolveDeep(obj core.Object) (core.Object, error) {
	// First resolve if it's a reference
	resolved, err := r.Resolve(obj)
	if err != nil {
		return nil, err
	}

	// Recursively resolve based on type
	switch v := resolved.(type) {
	case core.Array:
		result := make(core.Array, len(v))
		for i, elem := range v {
			resolvedElem, err := r.ResolveDeep(elem)
			if err != nil {
				return nil, err
			}
			result[i] = resolvedElem
		}
		return result, nil

	case core.Dict:
		result := make(core.Dict)
		for key, val := range v {
			resolvedVal, err := r.ResolveDeep(val)
			if err != nil {
				return nil, err
			}
			result[key] = resolvedVal
		}
		return result, nil

	default:
		return resolved, nil
	}
}

// PageCount returns the number of pages in the PDF
func (r *Reader) PageCount() (int, error) {
	if err := r.ensurePageTree(); err != nil {
		return 0, err
	}
	return r.pageTree.Count()
}

// GetPage returns the page at the given index (0-based)
func (r *Reader) GetPage(index int) (*pages.Page, error) {
	if err := r.ensurePageTree(); err != nil {
		return nil, err
	}
	return r.pageTree.GetPage(index)
}

// ensurePageTree loads the page tree if not already loaded
func (r *Reader) ensurePageTree() error {
	if r.pageTree != nil {
		return nil
	}

	// Get catalog
	catalog, err := r.GetCatalog()
	if err != nil {
		return fmt.Errorf("failed to get catalog: %w", err)
	}

	// Get pages dict
	pagesRef := catalog.Get("Pages")
	if pagesRef == nil {
		return fmt.Errorf("catalog missing /Pages entry")
	}

	pagesObj, err := r.Resolve(pagesRef)
	if err != nil {
		return fmt.Errorf("failed to resolve pages: %w", err)
	}

	pagesDict, ok := pagesObj.(core.Dict)
	if !ok {
		return fmt.Errorf("pages is not a dictionary: %T", pagesObj)
	}

	// Create page tree
	r.pageTree = pages.NewPageTree(pagesDict, r)
	return nil
}

// ExtractTextFragments extracts text fragments from a page
// This is a convenience method that handles content stream decoding and font registration
func (r *Reader) ExtractTextFragments(page *pages.Page) ([]text.TextFragment, error) {
	_, fragments, err := r.extractTextWithFragments(page)
	return fragments, err
}

// ExtractText extracts text from a page and returns it as a string
// This is a convenience method for simple text extraction
func (r *Reader) ExtractText(page *pages.Page) (string, error) {
	extractor, _, err := r.extractTextWithFragments(page)
	if err != nil {
		return "", err
	}
	if extractor == nil {
		return "", nil
	}
	return extractor.GetText(), nil
}

// extractTextWithFragments is the internal implementation that returns both
// the extractor (for GetText) and the fragments
func (r *Reader) extractTextWithFragments(page *pages.Page) (*text.Extractor, []text.TextFragment, error) {
	// Get content streams
	contents, err := page.Contents()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get contents: %w", err)
	}
	if contents == nil {
		return nil, nil, nil // Empty page
	}

	// Decode and concatenate all content streams
	var allData []byte
	for _, contentObj := range contents {
		stream, ok := contentObj.(*core.Stream)
		if !ok {
			continue
		}
		data, err := stream.Decode()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to decode content stream: %w", err)
		}
		allData = append(allData, data...)
	}

	if len(allData) == 0 {
		return nil, nil, nil
	}

	// Create extractor and register fonts
	extractor := text.NewExtractor()

	// Register fonts from page resources
	resolverFunc := func(ref core.IndirectRef) (core.Object, error) {
		return r.ResolveReference(ref)
	}
	if err := extractor.RegisterFontsFromPage(page, resolverFunc); err != nil {
		// Non-fatal - continue with default font handling
	}

	// Extract text fragments
	fragments, err := extractor.ExtractFromBytes(allData)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to extract text: %w", err)
	}

	return extractor, fragments, nil
}
