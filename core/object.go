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
