package pages

import (
	"fmt"

	"github.com/tsawler/tabula/core"
)

// ObjectResolver interface for resolving indirect references
type ObjectResolver interface {
	Resolve(obj core.Object) (core.Object, error)
	ResolveDeep(obj core.Object) (core.Object, error)
	ResolveReference(ref core.IndirectRef) (core.Object, error)
}

// Catalog represents the PDF document catalog (root of document structure)
type Catalog struct {
	dict     core.Dict
	resolver ObjectResolver
}

// NewCatalog creates a new catalog from a dictionary
func NewCatalog(dict core.Dict, resolver ObjectResolver) *Catalog {
	return &Catalog{
		dict:     dict,
		resolver: resolver,
	}
}

// Type returns the catalog type (should be "Catalog")
func (c *Catalog) Type() string {
	if typeObj := c.dict.Get("Type"); typeObj != nil {
		if name, ok := typeObj.(core.Name); ok {
			return string(name)
		}
	}
	return ""
}

// Pages returns the page tree root
func (c *Catalog) Pages() (core.Dict, error) {
	pagesRef := c.dict.Get("Pages")
	if pagesRef == nil {
		return nil, fmt.Errorf("catalog missing /Pages entry")
	}

	// Resolve reference if needed
	pagesObj, err := c.resolver.Resolve(pagesRef)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve /Pages: %w", err)
	}

	pagesDict, ok := pagesObj.(core.Dict)
	if !ok {
		return nil, fmt.Errorf("invalid /Pages type: %T", pagesObj)
	}

	return pagesDict, nil
}

// Metadata returns the metadata stream if present
func (c *Catalog) Metadata() (*core.Stream, error) {
	metadataRef := c.dict.Get("Metadata")
	if metadataRef == nil {
		return nil, nil // Optional
	}

	metadataObj, err := c.resolver.Resolve(metadataRef)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve /Metadata: %w", err)
	}

	stream, ok := metadataObj.(*core.Stream)
	if !ok {
		return nil, fmt.Errorf("invalid /Metadata type: %T", metadataObj)
	}

	return stream, nil
}

// Version returns the version entry if present
func (c *Catalog) Version() string {
	if versionObj := c.dict.Get("Version"); versionObj != nil {
		if name, ok := versionObj.(core.Name); ok {
			return string(name)
		}
	}
	return ""
}

// PageTree represents the PDF page tree
type PageTree struct {
	root     core.Dict
	resolver ObjectResolver
	pages    []*Page // Cached flattened page list
}

// NewPageTree creates a new page tree from the root pages dictionary
func NewPageTree(root core.Dict, resolver ObjectResolver) *PageTree {
	return &PageTree{
		root:     root,
		resolver: resolver,
	}
}

// Count returns the total number of pages
func (t *PageTree) Count() (int, error) {
	countObj := t.root.Get("Count")
	if countObj == nil {
		return 0, fmt.Errorf("page tree missing /Count entry")
	}

	count, ok := countObj.(core.Int)
	if !ok {
		return 0, fmt.Errorf("invalid /Count type: %T", countObj)
	}

	return int(count), nil
}

// GetPage returns the page at the given index (0-based)
func (t *PageTree) GetPage(index int) (*Page, error) {
	// Ensure pages are loaded
	if t.pages == nil {
		if err := t.loadPages(); err != nil {
			return nil, err
		}
	}

	if index < 0 || index >= len(t.pages) {
		return nil, fmt.Errorf("page index %d out of range [0, %d)", index, len(t.pages))
	}

	return t.pages[index], nil
}

// Pages returns all pages as a slice
func (t *PageTree) Pages() ([]*Page, error) {
	// Ensure pages are loaded
	if t.pages == nil {
		if err := t.loadPages(); err != nil {
			return nil, err
		}
	}

	return t.pages, nil
}

// loadPages traverses the page tree and builds the flattened page list
func (t *PageTree) loadPages() error {
	t.pages = make([]*Page, 0)

	// Start recursive traversal from root
	if err := t.traversePageNode(t.root, nil); err != nil {
		return fmt.Errorf("failed to traverse page tree: %w", err)
	}

	return nil
}

// traversePageNode recursively traverses a page tree node
// parent is the parent Pages dictionary for inheritable attributes
func (t *PageTree) traversePageNode(node core.Dict, parent core.Dict) error {
	// Get the type to determine if this is a Pages node or Page leaf
	typeObj := node.Get("Type")
	if typeObj == nil {
		return fmt.Errorf("page node missing /Type entry")
	}

	typeName, ok := typeObj.(core.Name)
	if !ok {
		return fmt.Errorf("invalid /Type: %T", typeObj)
	}

	switch string(typeName) {
	case "Pages":
		// Intermediate node - traverse children
		kidsObj := node.Get("Kids")
		if kidsObj == nil {
			return fmt.Errorf("Pages node missing /Kids entry")
		}

		// Resolve Kids if it's a reference
		kidsResolved, err := t.resolver.Resolve(kidsObj)
		if err != nil {
			return fmt.Errorf("failed to resolve /Kids: %w", err)
		}

		kids, ok := kidsResolved.(core.Array)
		if !ok {
			return fmt.Errorf("invalid /Kids type: %T", kidsResolved)
		}

		// Traverse each child
		for i, kidObj := range kids {
			// Resolve child reference
			kidResolved, err := t.resolver.Resolve(kidObj)
			if err != nil {
				return fmt.Errorf("failed to resolve kid %d: %w", i, err)
			}

			kidDict, ok := kidResolved.(core.Dict)
			if !ok {
				return fmt.Errorf("invalid kid type: %T", kidResolved)
			}

			// Recursively traverse child (passing current node as parent)
			if err := t.traversePageNode(kidDict, node); err != nil {
				return err
			}
		}

	case "Page":
		// Leaf node - create Page object
		page := NewPage(node, parent, t.resolver)
		t.pages = append(t.pages, page)

	default:
		return fmt.Errorf("unexpected page node type: %s", typeName)
	}

	return nil
}

// Page represents a single PDF page
type Page struct {
	dict     core.Dict
	parent   core.Dict // Parent Pages node (for inheritable attributes)
	resolver ObjectResolver
}

// NewPage creates a new page from a dictionary
func NewPage(dict core.Dict, parent core.Dict, resolver ObjectResolver) *Page {
	return &Page{
		dict:     dict,
		parent:   parent,
		resolver: resolver,
	}
}

// Type returns the page type (should be "Page")
func (p *Page) Type() string {
	if typeObj := p.dict.Get("Type"); typeObj != nil {
		if name, ok := typeObj.(core.Name); ok {
			return string(name)
		}
	}
	return ""
}

// MediaBox returns the page media box [x1 y1 x2 y2]
// This is inheritable, so checks parent if not present
func (p *Page) MediaBox() ([]float64, error) {
	return p.getBox("MediaBox")
}

// CropBox returns the page crop box [x1 y1 x2 y2]
// This is inheritable, defaults to MediaBox if not present
func (p *Page) CropBox() ([]float64, error) {
	box, err := p.getBox("CropBox")
	if err != nil {
		// CropBox defaults to MediaBox
		return p.MediaBox()
	}
	return box, nil
}

// getBox retrieves a box attribute (inheritable)
func (p *Page) getBox(name string) ([]float64, error) {
	// Try page dict first
	boxObj := p.dict.Get(name)

	// If not found, try parent (inheritable)
	if boxObj == nil && p.parent != nil {
		boxObj = p.parent.Get(name)
	}

	if boxObj == nil {
		return nil, fmt.Errorf("%s not found", name)
	}

	// Resolve if reference
	boxResolved, err := p.resolver.Resolve(boxObj)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve %s: %w", name, err)
	}

	// Parse array
	boxArr, ok := boxResolved.(core.Array)
	if !ok {
		return nil, fmt.Errorf("invalid %s type: %T", name, boxResolved)
	}

	if len(boxArr) != 4 {
		return nil, fmt.Errorf("invalid %s length: %d (expected 4)", name, len(boxArr))
	}

	// Convert to float64 slice
	box := make([]float64, 4)
	for i, elem := range boxArr {
		switch v := elem.(type) {
		case core.Int:
			box[i] = float64(v)
		case core.Real:
			box[i] = float64(v)
		default:
			return nil, fmt.Errorf("invalid %s element type: %T", name, elem)
		}
	}

	return box, nil
}

// Resources returns the page resources dictionary
// This is inheritable
func (p *Page) Resources() (core.Dict, error) {
	// Try page dict first
	resourcesObj := p.dict.Get("Resources")

	// If not found, try parent (inheritable)
	if resourcesObj == nil && p.parent != nil {
		resourcesObj = p.parent.Get("Resources")
	}

	if resourcesObj == nil {
		return nil, fmt.Errorf("resources not found")
	}

	// Resolve if reference
	resourcesResolved, err := p.resolver.Resolve(resourcesObj)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve Resources: %w", err)
	}

	resourcesDict, ok := resourcesResolved.(core.Dict)
	if !ok {
		return nil, fmt.Errorf("invalid Resources type: %T", resourcesResolved)
	}

	return resourcesDict, nil
}

// Contents returns the page content stream(s)
func (p *Page) Contents() ([]core.Object, error) {
	contentsObj := p.dict.Get("Contents")
	if contentsObj == nil {
		return nil, nil // Contents is optional
	}

	// Resolve if reference
	contentsResolved, err := p.resolver.Resolve(contentsObj)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve Contents: %w", err)
	}

	// Contents can be a single stream or array of streams
	switch v := contentsResolved.(type) {
	case *core.Stream:
		return []core.Object{v}, nil
	case core.Array:
		// Resolve each element in the array
		streams := make([]core.Object, len(v))
		for i, elem := range v {
			resolved, err := p.resolver.Resolve(elem)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve contents[%d]: %w", i, err)
			}
			streams[i] = resolved
		}
		return streams, nil
	default:
		return nil, fmt.Errorf("invalid Contents type: %T", contentsResolved)
	}
}

// Rotate returns the page rotation (0, 90, 180, or 270)
// This is inheritable
func (p *Page) Rotate() int {
	// Try page dict first
	rotateObj := p.dict.Get("Rotate")

	// If not found, try parent (inheritable)
	if rotateObj == nil && p.parent != nil {
		rotateObj = p.parent.Get("Rotate")
	}

	if rotateObj == nil {
		return 0 // Default
	}

	if rotate, ok := rotateObj.(core.Int); ok {
		return int(rotate)
	}

	return 0
}

// Width returns the page width (from MediaBox)
func (p *Page) Width() (float64, error) {
	box, err := p.MediaBox()
	if err != nil {
		return 0, err
	}
	return box[2] - box[0], nil
}

// Height returns the page height (from MediaBox)
func (p *Page) Height() (float64, error) {
	box, err := p.MediaBox()
	if err != nil {
		return 0, err
	}
	return box[3] - box[1], nil
}
