package core

import (
	"fmt"
	"strconv"
	"strings"
)

// Object is the interface implemented by all PDF object types.
// Every PDF object can report its type and provide a string representation.
type Object interface {
	// Type returns the ObjectType identifying this object's type.
	Type() ObjectType
	// String returns a PDF-syntax string representation of the object.
	String() string
}

// ObjectType identifies the type of a PDF object.
type ObjectType int

// PDF object type constants.
const (
	ObjNull     ObjectType = iota // Null object
	ObjBool                       // Boolean (true/false)
	ObjInt                        // Integer
	ObjReal                       // Real number (floating point)
	ObjString                     // String (literal or hexadecimal)
	ObjName                       // Name object (e.g., /Type)
	ObjArray                      // Array
	ObjDict                       // Dictionary
	ObjStream                     // Stream (dictionary + data)
	ObjIndirect                   // Indirect reference (e.g., "5 0 R")
)

// String returns a human-readable name for the object type.
func (t ObjectType) String() string {
	switch t {
	case ObjNull:
		return "Null"
	case ObjBool:
		return "Bool"
	case ObjInt:
		return "Int"
	case ObjReal:
		return "Real"
	case ObjString:
		return "String"
	case ObjName:
		return "Name"
	case ObjArray:
		return "Array"
	case ObjDict:
		return "Dict"
	case ObjStream:
		return "Stream"
	case ObjIndirect:
		return "IndirectRef"
	default:
		return "Unknown"
	}
}

// Null represents the PDF null object, which denotes the absence of a value.
type Null struct{}

func (n Null) Type() ObjectType { return ObjNull }
func (n Null) String() string   { return "null" }

// Bool represents a PDF boolean value (true or false).
type Bool bool

func (b Bool) Type() ObjectType { return ObjBool }
func (b Bool) String() string {
	if b {
		return "true"
	}
	return "false"
}

// Int represents a PDF integer object, stored as a 64-bit signed integer.
type Int int64

func (i Int) Type() ObjectType { return ObjInt }
func (i Int) String() string   { return strconv.FormatInt(int64(i), 10) }

// Real represents a PDF real number (floating-point), stored as float64.
type Real float64

func (r Real) Type() ObjectType { return ObjReal }
func (r Real) String() string   { return strconv.FormatFloat(float64(r), 'f', -1, 64) }

// String represents a PDF string object (either literal or hexadecimal encoded).
type String string

func (s String) Type() ObjectType { return ObjString }
func (s String) String() string   { return string(s) }

// Name represents a PDF name object, used as identifiers (e.g., /Type, /Font).
// The leading slash is not stored; it is added in String() output.
type Name string

func (n Name) Type() ObjectType { return ObjName }
func (n Name) String() string   { return "/" + string(n) }

// Array represents a PDF array, an ordered collection of PDF objects.
type Array []Object

func (a Array) Type() ObjectType { return ObjArray }
func (a Array) String() string {
	var parts []string
	for _, obj := range a {
		parts = append(parts, obj.String())
	}
	return "[" + strings.Join(parts, " ") + "]"
}

// Len returns the number of elements in the array.
func (a Array) Len() int {
	return len(a)
}

// Get returns the element at the given index, or nil if out of bounds.
func (a Array) Get(index int) Object {
	if index < 0 || index >= len(a) {
		return nil
	}
	return a[index]
}

// GetInt returns the integer at the given index, with a boolean indicating success.
func (a Array) GetInt(index int) (Int, bool) {
	obj := a.Get(index)
	if obj == nil {
		return 0, false
	}
	i, ok := obj.(Int)
	return i, ok
}

// GetReal returns the real number at the given index, with a boolean indicating success.
func (a Array) GetReal(index int) (Real, bool) {
	obj := a.Get(index)
	if obj == nil {
		return 0, false
	}
	r, ok := obj.(Real)
	return r, ok
}

// GetName returns the name at the given index, with a boolean indicating success.
func (a Array) GetName(index int) (Name, bool) {
	obj := a.Get(index)
	if obj == nil {
		return "", false
	}
	n, ok := obj.(Name)
	return n, ok
}

// Dict represents a PDF dictionary, a collection of key-value pairs where keys
// are names (strings) and values are arbitrary PDF objects.
type Dict map[string]Object

func (d Dict) Type() ObjectType { return ObjDict }
func (d Dict) String() string {
	var parts []string
	for key, val := range d {
		parts = append(parts, fmt.Sprintf("/%s %s", key, val.String()))
	}
	return "<<" + strings.Join(parts, " ") + ">>"
}

// Get returns the value associated with the key, or nil if not present.
func (d Dict) Get(key string) Object {
	return d[key]
}

// GetName returns the Name value for the key, with a boolean indicating success.
func (d Dict) GetName(key string) (Name, bool) {
	obj, ok := d[key]
	if !ok {
		return "", false
	}
	name, ok := obj.(Name)
	return name, ok
}

// GetInt returns the Int value for the key, with a boolean indicating success.
func (d Dict) GetInt(key string) (Int, bool) {
	obj, ok := d[key]
	if !ok {
		return 0, false
	}
	i, ok := obj.(Int)
	return i, ok
}

// GetDict returns the Dict value for the key, with a boolean indicating success.
func (d Dict) GetDict(key string) (Dict, bool) {
	obj, ok := d[key]
	if !ok {
		return nil, false
	}
	dict, ok := obj.(Dict)
	return dict, ok
}

// GetArray returns the Array value for the key, with a boolean indicating success.
func (d Dict) GetArray(key string) (Array, bool) {
	obj, ok := d[key]
	if !ok {
		return nil, false
	}
	arr, ok := obj.(Array)
	return arr, ok
}

// GetReal returns the Real value for the key, with a boolean indicating success.
func (d Dict) GetReal(key string) (Real, bool) {
	obj, ok := d[key]
	if !ok {
		return 0, false
	}
	r, ok := obj.(Real)
	return r, ok
}

// GetString returns the String value for the key, with a boolean indicating success.
func (d Dict) GetString(key string) (String, bool) {
	obj, ok := d[key]
	if !ok {
		return "", false
	}
	s, ok := obj.(String)
	return s, ok
}

// GetBool returns the Bool value for the key, with a boolean indicating success.
func (d Dict) GetBool(key string) (Bool, bool) {
	obj, ok := d[key]
	if !ok {
		return false, false
	}
	b, ok := obj.(Bool)
	return b, ok
}

// GetStream returns the Stream value for the key, with a boolean indicating success.
func (d Dict) GetStream(key string) (*Stream, bool) {
	obj, ok := d[key]
	if !ok {
		return nil, false
	}
	s, ok := obj.(*Stream)
	return s, ok
}

// GetIndirectRef returns the IndirectRef value for the key, with a boolean indicating success.
func (d Dict) GetIndirectRef(key string) (IndirectRef, bool) {
	obj, ok := d[key]
	if !ok {
		return IndirectRef{}, false
	}
	ref, ok := obj.(IndirectRef)
	return ref, ok
}

// Has reports whether the key exists in the dictionary.
func (d Dict) Has(key string) bool {
	_, ok := d[key]
	return ok
}

// Set associates a value with the key in the dictionary.
func (d Dict) Set(key string, value Object) {
	d[key] = value
}

// Delete removes the key and its value from the dictionary.
func (d Dict) Delete(key string) {
	delete(d, key)
}

// Keys returns all keys in the dictionary in an arbitrary order.
func (d Dict) Keys() []string {
	keys := make([]string, 0, len(d))
	for k := range d {
		keys = append(keys, k)
	}
	return keys
}

// Stream represents a PDF stream object, consisting of a dictionary and binary data.
// Streams are used for content that may be compressed or filtered, such as page
// content, images, and fonts.
type Stream struct {
	Dict    Dict   // Stream dictionary containing metadata and filter information
	Data    []byte // Raw (possibly compressed) stream data
	decoded []byte // Cached decoded data
}

func (s *Stream) Type() ObjectType { return ObjStream }
func (s *Stream) String() string {
	return fmt.Sprintf("stream %s (%d bytes)", s.Dict.String(), len(s.Data))
}

// Decoded returns the decoded (decompressed) stream data.
// Results are cached for subsequent calls. Use Stream.Decode for full decoding.
func (s *Stream) Decoded() ([]byte, error) {
	if s.decoded != nil {
		return s.decoded, nil
	}
	// TODO: implement stream decoding based on filters
	return s.Data, nil
}

// IndirectRef represents an indirect object reference in PDF syntax (e.g., "5 0 R").
// It references an object by its object number and generation number.
type IndirectRef struct {
	Number     int // Object number
	Generation int // Generation number (usually 0)
}

func (r IndirectRef) Type() ObjectType { return ObjIndirect }
func (r IndirectRef) String() string {
	return fmt.Sprintf("%d %d R", r.Number, r.Generation)
}

// IndirectObject represents an indirect object definition (e.g., "5 0 obj ... endobj").
// It pairs an IndirectRef with the actual object value.
type IndirectObject struct {
	Ref    IndirectRef // Reference identifying this object
	Object Object      // The actual object value
}
