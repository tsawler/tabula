// Package reader provides high-level PDF file reading and object resolution.
//
// This package orchestrates the lower-level core package to provide a
// convenient API for reading PDF files and extracting content.
//
// # Opening PDF Files
//
// Use [Open] to open a PDF file for reading:
//
//	reader, err := reader.Open("document.pdf")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer reader.Close()
//
// Or use [NewReader] with an existing *os.File.
//
// # Document Information
//
// The Reader provides access to document structure:
//
//   - Version() - PDF version (e.g., 1.7)
//   - PageCount() - number of pages
//   - GetCatalog() - document catalog dictionary
//   - GetInfo() - document info dictionary (metadata)
//   - Trailer() - trailer dictionary
//
// # Page Access
//
// Access pages by index (0-based):
//
//	page, err := reader.GetPage(0)  // First page
//
// # Object Resolution
//
// The Reader resolves indirect object references:
//
//   - GetObject(objNum) - load object by number
//   - ResolveReference(ref) - resolve an IndirectRef
//   - Resolve(obj) - resolve if indirect, otherwise return as-is
//   - ResolveDeep(obj) - recursively resolve all references
//
// # Text Extraction
//
// Convenience methods for text extraction:
//
//   - ExtractText(page) - extract text as a string
//   - ExtractTextFragments(page) - extract positioned text fragments
//
// # Object Caching
//
// The Reader caches loaded objects for efficiency. Use ClearCache() to free
// memory when processing large PDFs.
package reader
