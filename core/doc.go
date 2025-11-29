// Package core provides low-level PDF parsing primitives and object types.
//
// This package implements the fundamental building blocks for working with PDF files,
// including all eight PDF object types (null, boolean, integer, real, string, name,
// array, and dictionary), as well as streams, indirect references, cross-reference
// tables, and object streams.
//
// # Object Types
//
// PDF defines eight basic object types, all implemented as types satisfying the
// Object interface:
//
//   - [Null] - represents the PDF null object
//   - [Bool] - represents PDF boolean values (true/false)
//   - [Int] - represents PDF integers
//   - [Real] - represents PDF real numbers (floating point)
//   - [String] - represents PDF string objects (literal or hexadecimal)
//   - [Name] - represents PDF name objects (e.g., /Type, /Font)
//   - [Array] - represents PDF arrays
//   - [Dict] - represents PDF dictionaries
//
// Additionally, [Stream] represents a PDF stream (dictionary + binary data),
// and [IndirectRef] represents a reference to an indirect object.
//
// # Parsing
//
// The [Parser] type handles parsing PDF syntax from an io.Reader. It can parse
// individual objects or complete indirect object definitions.
//
// The [Lexer] type provides tokenization of PDF input, converting raw bytes
// into tokens that the parser consumes.
//
// # Cross-Reference Tables
//
// The [XRefTable] type represents a PDF cross-reference table, which maps object
// numbers to their locations in the file. The [XRefParser] type handles parsing
// both traditional xref tables (PDF 1.0-1.4) and xref streams (PDF 1.5+).
//
// # Object Streams
//
// The [ObjectStream] type (PDF 1.5+) handles object streams, which store multiple
// objects in a single compressed stream for better compression.
//
// # Stream Decoding
//
// Streams can be compressed using various filters. The [Stream.Decode] method
// handles decompression, supporting filters like FlateDecode, ASCIIHexDecode,
// and ASCII85Decode.
package core
