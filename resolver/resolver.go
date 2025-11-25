package resolver

import (
	"fmt"

	"github.com/tsawler/tabula/core"
)

// ObjectResolver resolves indirect references in PDF objects
// It can recursively resolve references in dictionaries and arrays
type ObjectResolver struct {
	reader      ObjectReader
	visited     map[int]bool // Cycle detection
	maxDepth    int          // Maximum recursion depth
	currentDepth int         // Current recursion depth
}

// ObjectReader interface allows the resolver to work with any reader
type ObjectReader interface {
	GetObject(objNum int) (core.Object, error)
	ResolveReference(ref core.IndirectRef) (core.Object, error)
}

// Option configures the resolver
type Option func(*ObjectResolver)

// WithMaxDepth sets the maximum recursion depth (default: 100)
func WithMaxDepth(depth int) Option {
	return func(r *ObjectResolver) {
		r.maxDepth = depth
	}
}

// NewResolver creates a new object resolver
func NewResolver(reader ObjectReader, opts ...Option) *ObjectResolver {
	r := &ObjectResolver{
		reader:   reader,
		visited:  make(map[int]bool),
		maxDepth: 100, // Reasonable default
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// Resolve resolves an object, following indirect references
// If the object contains nested references (in dicts/arrays), they are also resolved
func (r *ObjectResolver) Resolve(obj core.Object) (core.Object, error) {
	return r.resolve(obj, false)
}

// ResolveDeep recursively resolves all indirect references in dictionaries and arrays
// This will fully expand the object tree
func (r *ObjectResolver) ResolveDeep(obj core.Object) (core.Object, error) {
	return r.resolve(obj, true)
}

// resolve is the internal resolution method
func (r *ObjectResolver) resolve(obj core.Object, deep bool) (core.Object, error) {
	// Reset visited map at top level (depth 0)
	// This allows the same objects to be resolved in different top-level calls
	// while still detecting circular references within a single resolution tree
	if r.currentDepth == 0 {
		r.visited = make(map[int]bool)
	}

	// Check depth limit
	if r.currentDepth >= r.maxDepth {
		return nil, fmt.Errorf("maximum recursion depth (%d) exceeded", r.maxDepth)
	}

	switch v := obj.(type) {
	case core.IndirectRef:
		// Check for cycles
		if r.visited[v.Number] {
			return nil, fmt.Errorf("circular reference detected for object %d", v.Number)
		}

		// Mark as visited
		r.visited[v.Number] = true
		// Unmark after we're done (allows the same object to be visited in different branches)
		defer func() {
			delete(r.visited, v.Number)
		}()

		// Resolve the reference
		resolved, err := r.reader.ResolveReference(v)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve reference %d %d R: %w", v.Number, v.Generation, err)
		}

		// If deep resolution, recursively resolve the resolved object
		if deep {
			r.currentDepth++
			resolved, err = r.resolve(resolved, deep)
			r.currentDepth--
			if err != nil {
				return nil, err
			}
		}

		return resolved, nil

	case core.Dict:
		if !deep {
			return v, nil
		}

		// Resolve all dictionary values
		resolved := make(core.Dict)
		for key, value := range v {
			r.currentDepth++
			resolvedValue, err := r.resolve(value, deep)
			r.currentDepth--
			if err != nil {
				return nil, fmt.Errorf("failed to resolve dict key %s: %w", key, err)
			}
			resolved[key] = resolvedValue
		}
		return resolved, nil

	case core.Array:
		if !deep {
			return v, nil
		}

		// Resolve all array elements
		resolved := make(core.Array, len(v))
		for i, elem := range v {
			r.currentDepth++
			resolvedElem, err := r.resolve(elem, deep)
			r.currentDepth--
			if err != nil {
				return nil, fmt.Errorf("failed to resolve array element %d: %w", i, err)
			}
			resolved[i] = resolvedElem
		}
		return resolved, nil

	case *core.Stream:
		if !deep {
			return v, nil
		}

		// Resolve the stream dictionary
		r.currentDepth++
		resolvedDict, err := r.resolve(v.Dict, deep)
		r.currentDepth--
		if err != nil {
			return nil, fmt.Errorf("failed to resolve stream dict: %w", err)
		}

		// Return new stream with resolved dictionary
		return &core.Stream{
			Dict: resolvedDict.(core.Dict),
			Data: v.Data,
		}, nil

	default:
		// Primitive types don't need resolution
		return obj, nil
	}
}

// Reset clears the visited map and depth counter
// Call this between independent resolution operations
func (r *ObjectResolver) Reset() {
	r.visited = make(map[int]bool)
	r.currentDepth = 0
}

// ResolveDict is a convenience method for resolving dictionaries
// It resolves the dictionary and all its values (deep resolution)
func (r *ObjectResolver) ResolveDict(dict core.Dict) (core.Dict, error) {
	defer r.Reset()
	resolved, err := r.ResolveDeep(dict)
	if err != nil {
		return nil, err
	}
	return resolved.(core.Dict), nil
}

// ResolveArray is a convenience method for resolving arrays
// It resolves all elements in the array (deep resolution)
func (r *ObjectResolver) ResolveArray(arr core.Array) (core.Array, error) {
	defer r.Reset()
	resolved, err := r.ResolveDeep(arr)
	if err != nil {
		return nil, err
	}
	return resolved.(core.Array), nil
}

// ResolveReference resolves a single indirect reference
// This is a shallow resolution - it returns the referenced object but doesn't recurse
func (r *ObjectResolver) ResolveReference(ref core.IndirectRef) (core.Object, error) {
	defer r.Reset()
	return r.reader.ResolveReference(ref)
}

// ResolveReferenceDeep resolves a reference and all nested references
func (r *ObjectResolver) ResolveReferenceDeep(ref core.IndirectRef) (core.Object, error) {
	defer r.Reset()
	return r.ResolveDeep(ref)
}

// GetObject loads an object by number (convenience method)
func (r *ObjectResolver) GetObject(objNum int) (core.Object, error) {
	return r.reader.GetObject(objNum)
}

// GetObjectResolved loads and resolves an object by number (shallow)
func (r *ObjectResolver) GetObjectResolved(objNum int) (core.Object, error) {
	obj, err := r.reader.GetObject(objNum)
	if err != nil {
		return nil, err
	}
	defer r.Reset()
	return r.Resolve(obj)
}

// GetObjectResolvedDeep loads and fully resolves an object by number (deep)
func (r *ObjectResolver) GetObjectResolvedDeep(objNum int) (core.Object, error) {
	obj, err := r.reader.GetObject(objNum)
	if err != nil {
		return nil, err
	}
	defer r.Reset()
	return r.ResolveDeep(obj)
}
