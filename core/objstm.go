package core

import (
	"bytes"
	"fmt"
)

// ObjectStream represents a PDF Object Stream (Type /ObjStm), introduced in PDF 1.5.
// Object streams store multiple objects in a single compressed stream, providing
// better compression than storing objects individually.
type ObjectStream struct {
	stream  *Stream              // Underlying stream object
	n       int                  // Number of objects in stream
	first   int                  // Byte offset of first object in decoded data
	extends *IndirectRef         // Optional reference to another ObjStm this one extends
	objects map[int]Object       // Cached parsed objects (index -> object)
	offsets []objectStreamOffset // Parsed offset pairs from header
	decoded []byte               // Decoded stream data (cached)
}

// objectStreamOffset pairs an object number with its byte offset within the decoded data.
type objectStreamOffset struct {
	ObjNum int // Object number
	Offset int // Byte offset within decoded data (relative to First)
}

// NewObjectStream creates an ObjectStream from a Stream object.
// The stream must have Type /ObjStm and required entries /N and /First.
// Returns an error if the stream is not a valid object stream.
func NewObjectStream(stream *Stream) (*ObjectStream, error) {
	if stream == nil {
		return nil, fmt.Errorf("stream is nil")
	}

	// Verify /Type is /ObjStm
	typeObj := stream.Dict.Get("Type")
	if typeObj == nil {
		return nil, fmt.Errorf("object stream missing /Type")
	}
	typeName, ok := typeObj.(Name)
	if !ok || string(typeName) != "ObjStm" {
		return nil, fmt.Errorf("stream is not an object stream, got type: %v", typeObj)
	}

	// Get /N - number of objects
	nObj := stream.Dict.Get("N")
	if nObj == nil {
		return nil, fmt.Errorf("object stream missing /N")
	}
	nInt, ok := nObj.(Int)
	if !ok {
		return nil, fmt.Errorf("invalid /N type: %T", nObj)
	}
	n := int(nInt)
	if n < 0 {
		return nil, fmt.Errorf("invalid /N value: %d", n)
	}

	// Get /First - byte offset to first object data
	firstObj := stream.Dict.Get("First")
	if firstObj == nil {
		return nil, fmt.Errorf("object stream missing /First")
	}
	firstInt, ok := firstObj.(Int)
	if !ok {
		return nil, fmt.Errorf("invalid /First type: %T", firstObj)
	}
	first := int(firstInt)
	if first < 0 {
		return nil, fmt.Errorf("invalid /First value: %d", first)
	}

	// Get optional /Extends - reference to another object stream
	var extends *IndirectRef
	if extendsObj := stream.Dict.Get("Extends"); extendsObj != nil {
		ref, ok := extendsObj.(*IndirectRef)
		if !ok {
			return nil, fmt.Errorf("invalid /Extends type: %T", extendsObj)
		}
		extends = ref
	}

	os := &ObjectStream{
		stream:  stream,
		n:       n,
		first:   first,
		extends: extends,
		objects: make(map[int]Object),
	}

	return os, nil
}

// N returns the number of objects stored in the stream.
func (os *ObjectStream) N() int {
	return os.n
}

// First returns the byte offset to the first object's data in the decoded stream.
// The header (object number/offset pairs) precedes this offset.
func (os *ObjectStream) First() int {
	return os.first
}

// Extends returns the reference to another object stream this one extends, or nil.
func (os *ObjectStream) Extends() *IndirectRef {
	return os.extends
}

// decode decodes the stream data and parses the header. Called lazily on first access.
func (os *ObjectStream) decode() error {
	if os.decoded != nil {
		return nil // Already decoded
	}

	// Decode the stream
	decoded, err := os.stream.Decode()
	if err != nil {
		return fmt.Errorf("failed to decode object stream: %w", err)
	}
	os.decoded = decoded

	// Parse the header: N pairs of (objNum offset)
	// The header is plain text integers separated by whitespace
	if err := os.parseHeader(); err != nil {
		return fmt.Errorf("failed to parse object stream header: %w", err)
	}

	return nil
}

// parseHeader parses the object stream header containing N pairs of integers.
// Format: "objNum1 offset1 objNum2 offset2 ... objNumN offsetN"
func (os *ObjectStream) parseHeader() error {
	if os.first > len(os.decoded) {
		return fmt.Errorf("First offset (%d) exceeds decoded data length (%d)", os.first, len(os.decoded))
	}

	headerData := os.decoded[:os.first]
	parser := NewParser(bytes.NewReader(headerData))

	os.offsets = make([]objectStreamOffset, 0, os.n)

	for i := 0; i < os.n; i++ {
		// Parse object number
		objNumObj, err := parser.ParseObject()
		if err != nil {
			return fmt.Errorf("failed to parse object number %d: %w", i, err)
		}
		objNum, ok := objNumObj.(Int)
		if !ok {
			return fmt.Errorf("object number %d is not an integer: %T", i, objNumObj)
		}

		// Parse offset
		offsetObj, err := parser.ParseObject()
		if err != nil {
			return fmt.Errorf("failed to parse offset %d: %w", i, err)
		}
		offset, ok := offsetObj.(Int)
		if !ok {
			return fmt.Errorf("offset %d is not an integer: %T", i, offsetObj)
		}

		os.offsets = append(os.offsets, objectStreamOffset{
			ObjNum: int(objNum),
			Offset: int(offset),
		})
	}

	return nil
}

// GetObjectByIndex extracts an object by its index within the stream (0-based).
// Returns the object, its object number, and any error. The index corresponds
// to the position in the header, not the object number.
func (os *ObjectStream) GetObjectByIndex(index int) (Object, int, error) {
	// Ensure stream is decoded
	if err := os.decode(); err != nil {
		return nil, 0, err
	}

	if index < 0 || index >= len(os.offsets) {
		return nil, 0, fmt.Errorf("index %d out of range [0, %d)", index, len(os.offsets))
	}

	// Check cache
	if obj, ok := os.objects[index]; ok {
		return obj, os.offsets[index].ObjNum, nil
	}

	// Calculate the actual offset in the decoded data
	offset := os.first + os.offsets[index].Offset

	// Determine the end of this object's data
	// It extends until the next object's offset, or end of data
	var endOffset int
	if index+1 < len(os.offsets) {
		endOffset = os.first + os.offsets[index+1].Offset
	} else {
		endOffset = len(os.decoded)
	}

	if offset >= len(os.decoded) {
		return nil, 0, fmt.Errorf("object offset %d exceeds decoded data length %d", offset, len(os.decoded))
	}
	if endOffset > len(os.decoded) {
		endOffset = len(os.decoded)
	}

	// Parse the object from its data slice
	objectData := os.decoded[offset:endOffset]
	parser := NewParser(bytes.NewReader(objectData))

	obj, err := parser.ParseObject()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to parse object at index %d: %w", index, err)
	}

	// Cache the parsed object
	os.objects[index] = obj

	return obj, os.offsets[index].ObjNum, nil
}

// GetObjectByNumber finds and extracts an object by its object number.
// Returns the object, its index within the stream, and any error.
func (os *ObjectStream) GetObjectByNumber(objNum int) (Object, int, error) {
	// Ensure stream is decoded
	if err := os.decode(); err != nil {
		return nil, 0, err
	}

	// Find the index for this object number
	for i, entry := range os.offsets {
		if entry.ObjNum == objNum {
			obj, _, err := os.GetObjectByIndex(i)
			return obj, i, err
		}
	}

	return nil, 0, fmt.Errorf("object %d not found in object stream", objNum)
}

// ObjectNumbers returns a slice of all object numbers stored in this stream.
func (os *ObjectStream) ObjectNumbers() ([]int, error) {
	// Ensure stream is decoded
	if err := os.decode(); err != nil {
		return nil, err
	}

	nums := make([]int, len(os.offsets))
	for i, entry := range os.offsets {
		nums[i] = entry.ObjNum
	}
	return nums, nil
}

// ContainsObject reports whether the given object number is stored in this stream.
func (os *ObjectStream) ContainsObject(objNum int) (bool, error) {
	// Ensure stream is decoded
	if err := os.decode(); err != nil {
		return false, err
	}

	for _, entry := range os.offsets {
		if entry.ObjNum == objNum {
			return true, nil
		}
	}
	return false, nil
}
