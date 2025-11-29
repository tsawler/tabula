// Package pages provides PDF page tree traversal and page access.
//
// This package handles the hierarchical page tree structure in PDFs,
// providing efficient access to individual pages and their resources.
//
// # Page Tree
//
// PDF documents organize pages in a tree structure. The [PageTree] type
// navigates this hierarchy:
//
//	tree := pages.NewPageTree(pagesDict, resolver)
//	count, _ := tree.Count()
//	page, _ := tree.GetPage(0)  // 0-indexed
//
// # Page Access
//
// The [Page] type represents a single PDF page with:
//
//   - MediaBox - page dimensions
//   - CropBox - visible area (optional)
//   - Rotation - page rotation (0, 90, 180, 270)
//   - Resources - fonts, images, etc.
//   - Contents - content streams
//
// # Resources
//
// Page resources provide access to:
//
//   - Fonts (/Font dictionary)
//   - Images (/XObject dictionary)
//   - Color spaces (/ColorSpace dictionary)
//   - Extended graphics states (/ExtGState dictionary)
//
// Resources can be inherited from parent page tree nodes.
//
// # Object Resolution
//
// The [ObjectResolver] interface abstracts object lookup:
//
//	type ObjectResolver interface {
//	    Resolve(obj core.Object) (core.Object, error)
//	    ResolveDeep(obj core.Object) (core.Object, error)
//	}
//
// This allows the page tree to resolve indirect references without
// depending on the full reader implementation.
package pages
