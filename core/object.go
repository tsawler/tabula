package core

import (
	"fmt"
	"strconv"
	"strings"
)

// Object represents a PDF object
type Object interface {
	Type() ObjectType
	String() string
}

// ObjectType represents the type of PDF object
type ObjectType int

const (
	ObjNull ObjectType = iota
	ObjBool
	ObjInt
	ObjReal
	ObjString
	ObjName
	ObjArray
	ObjDict
	ObjStream
	ObjIndirect
)

// String returns the string representation of the object type
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

// Null represents a PDF null object
type Null struct{}

func (n Null) Type() ObjectType { return ObjNull }
func (n Null) String() string   { return "null" }

// Bool represents a PDF boolean
type Bool bool

func (b Bool) Type() ObjectType { return ObjBool }
func (b Bool) String() string {
	if b {
		return "true"
	}
	return "false"
}

// Int represents a PDF integer
type Int int64

func (i Int) Type() ObjectType { return ObjInt }
func (i Int) String() string   { return strconv.FormatInt(int64(i), 10) }

// Real represents a PDF real number
type Real float64

func (r Real) Type() ObjectType { return ObjReal }
func (r Real) String() string   { return strconv.FormatFloat(float64(r), 'f', -1, 64) }

// String represents a PDF string
type String string

func (s String) Type() ObjectType { return ObjString }
func (s String) String() string   { return string(s) }

// Name represents a PDF name
type Name string

func (n Name) Type() ObjectType { return ObjName }
func (n Name) String() string   { return "/" + string(n) }

// Array represents a PDF array
type Array []Object

func (a Array) Type() ObjectType { return ObjArray }
func (a Array) String() string {
	var parts []string
	for _, obj := range a {
		parts = append(parts, obj.String())
	}
	return "[" + strings.Join(parts, " ") + "]"
}

// Len returns the length of the array
func (a Array) Len() int {
	return len(a)
}

// Get retrieves an element at the given index
func (a Array) Get(index int) Object {
	if index < 0 || index >= len(a) {
		return nil
	}
	return a[index]
}

// GetInt retrieves an integer at the given index
func (a Array) GetInt(index int) (Int, bool) {
	obj := a.Get(index)
	if obj == nil {
		return 0, false
	}
	i, ok := obj.(Int)
	return i, ok
}

// GetReal retrieves a real number at the given index
func (a Array) GetReal(index int) (Real, bool) {
	obj := a.Get(index)
	if obj == nil {
		return 0, false
	}
	r, ok := obj.(Real)
	return r, ok
}

// GetName retrieves a name at the given index
func (a Array) GetName(index int) (Name, bool) {
	obj := a.Get(index)
	if obj == nil {
		return "", false
	}
	n, ok := obj.(Name)
	return n, ok
}

// Dict represents a PDF dictionary
type Dict map[string]Object

func (d Dict) Type() ObjectType { return ObjDict }
func (d Dict) String() string {
	var parts []string
	for key, val := range d {
		parts = append(parts, fmt.Sprintf("/%s %s", key, val.String()))
	}
	return "<<" + strings.Join(parts, " ") + ">>"
}

// Get retrieves a value from the dictionary
func (d Dict) Get(key string) Object {
	return d[key]
}

// GetName retrieves a name value
func (d Dict) GetName(key string) (Name, bool) {
	obj, ok := d[key]
	if !ok {
		return "", false
	}
	name, ok := obj.(Name)
	return name, ok
}

// GetInt retrieves an integer value
func (d Dict) GetInt(key string) (Int, bool) {
	obj, ok := d[key]
	if !ok {
		return 0, false
	}
	i, ok := obj.(Int)
	return i, ok
}

// GetDict retrieves a dictionary value
func (d Dict) GetDict(key string) (Dict, bool) {
	obj, ok := d[key]
	if !ok {
		return nil, false
	}
	dict, ok := obj.(Dict)
	return dict, ok
}

// GetArray retrieves an array value
func (d Dict) GetArray(key string) (Array, bool) {
	obj, ok := d[key]
	if !ok {
		return nil, false
	}
	arr, ok := obj.(Array)
	return arr, ok
}

// GetReal retrieves a real number value
func (d Dict) GetReal(key string) (Real, bool) {
	obj, ok := d[key]
	if !ok {
		return 0, false
	}
	r, ok := obj.(Real)
	return r, ok
}

// GetString retrieves a string value
func (d Dict) GetString(key string) (String, bool) {
	obj, ok := d[key]
	if !ok {
		return "", false
	}
	s, ok := obj.(String)
	return s, ok
}

// GetBool retrieves a boolean value
func (d Dict) GetBool(key string) (Bool, bool) {
	obj, ok := d[key]
	if !ok {
		return false, false
	}
	b, ok := obj.(Bool)
	return b, ok
}

// GetStream retrieves a stream value
func (d Dict) GetStream(key string) (*Stream, bool) {
	obj, ok := d[key]
	if !ok {
		return nil, false
	}
	s, ok := obj.(*Stream)
	return s, ok
}

// GetIndirectRef retrieves an indirect reference
func (d Dict) GetIndirectRef(key string) (IndirectRef, bool) {
	obj, ok := d[key]
	if !ok {
		return IndirectRef{}, false
	}
	ref, ok := obj.(IndirectRef)
	return ref, ok
}

// Has checks if a key exists in the dictionary
func (d Dict) Has(key string) bool {
	_, ok := d[key]
	return ok
}

// Set sets a value in the dictionary
func (d Dict) Set(key string, value Object) {
	d[key] = value
}

// Delete removes a key from the dictionary
func (d Dict) Delete(key string) {
	delete(d, key)
}

// Keys returns all keys in the dictionary
func (d Dict) Keys() []string {
	keys := make([]string, 0, len(d))
	for k := range d {
		keys = append(keys, k)
	}
	return keys
}

// Stream represents a PDF stream object
type Stream struct {
	Dict   Dict
	Data   []byte
	decoded []byte
}

func (s *Stream) Type() ObjectType { return ObjStream }
func (s *Stream) String() string {
	return fmt.Sprintf("stream %s (%d bytes)", s.Dict.String(), len(s.Data))
}

// Decoded returns the decoded stream data
func (s *Stream) Decoded() ([]byte, error) {
	if s.decoded != nil {
		return s.decoded, nil
	}
	// TODO: implement stream decoding based on filters
	return s.Data, nil
}

// IndirectRef represents an indirect object reference
type IndirectRef struct {
	Number     int
	Generation int
}

func (r IndirectRef) Type() ObjectType { return ObjIndirect }
func (r IndirectRef) String() string {
	return fmt.Sprintf("%d %d R", r.Number, r.Generation)
}

// IndirectObject represents an indirect object with its reference
type IndirectObject struct {
	Ref    IndirectRef
	Object Object
}
