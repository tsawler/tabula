// Package resolver provides PDF indirect reference resolution.
//
// PDF documents use indirect references (e.g., "5 0 R") to refer to objects
// stored elsewhere in the file. This package resolves these references,
// following chains of references and detecting circular dependencies.
//
// # Basic Usage
//
// Create a resolver with an object reader and resolve references:
//
//	resolver := resolver.NewResolver(reader)
//	obj, err := resolver.Resolve(ref)
//
// # Deep Resolution
//
// For complete expansion of nested references in dictionaries and arrays:
//
//	resolved, err := resolver.ResolveDeep(obj)
//
// This recursively resolves all indirect references within the object tree.
//
// # Cycle Detection
//
// The resolver automatically detects circular references and returns an
// error rather than entering an infinite loop. The maximum recursion depth
// is configurable:
//
//	resolver := resolver.NewResolver(reader, resolver.WithMaxDepth(50))
//
// # Convenience Methods
//
// Several convenience methods simplify common operations:
//   - ResolveDict: Resolve a dictionary and all its values
//   - ResolveArray: Resolve an array and all its elements
//   - GetObjectResolved: Load and resolve an object by number
package resolver
