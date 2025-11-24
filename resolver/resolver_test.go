package resolver

import (
	"fmt"
	"strings"
	"testing"

	"github.com/tsawler/tabula/core"
)

// mockReader is a mock ObjectReader for testing
type mockReader struct {
	objects map[int]core.Object
}

func newMockReader() *mockReader {
	return &mockReader{
		objects: make(map[int]core.Object),
	}
}

func (m *mockReader) AddObject(num int, obj core.Object) {
	m.objects[num] = obj
}

func (m *mockReader) GetObject(objNum int) (core.Object, error) {
	obj, ok := m.objects[objNum]
	if !ok {
		return nil, fmt.Errorf("object %d not found", objNum)
	}
	return obj, nil
}

func (m *mockReader) ResolveReference(ref core.IndirectRef) (core.Object, error) {
	return m.GetObject(ref.Number)
}

// TestResolveIndirectRef tests resolving a simple indirect reference
func TestResolveIndirectRef(t *testing.T) {
	reader := newMockReader()
	reader.AddObject(5, core.Int(42))

	resolver := NewResolver(reader)
	ref := core.IndirectRef{Number: 5, Generation: 0}

	resolved, err := resolver.Resolve(ref)
	if err != nil {
		t.Fatalf("failed to resolve: %v", err)
	}

	val, ok := resolved.(core.Int)
	if !ok {
		t.Fatalf("expected Int, got %T", resolved)
	}
	if val != 42 {
		t.Errorf("expected 42, got %d", val)
	}
}

// TestResolvePrimitive tests that primitives pass through unchanged
func TestResolvePrimitive(t *testing.T) {
	reader := newMockReader()
	resolver := NewResolver(reader)

	tests := []struct {
		name string
		obj  core.Object
	}{
		{"Bool", core.Bool(true)},
		{"Int", core.Int(123)},
		{"Real", core.Real(3.14)},
		{"String", core.String("hello")},
		{"Name", core.Name("Test")},
		{"Null", core.Null{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved, err := resolver.Resolve(tt.obj)
			if err != nil {
				t.Fatalf("failed to resolve: %v", err)
			}
			if resolved != tt.obj {
				t.Errorf("primitive changed: %v -> %v", tt.obj, resolved)
			}
		})
	}
}

// TestResolveDict tests dictionary resolution
func TestResolveDict(t *testing.T) {
	reader := newMockReader()
	reader.AddObject(10, core.String("Value"))

	dict := core.Dict{
		"Direct": core.Int(123),
		"Ref":    core.IndirectRef{Number: 10},
	}

	resolver := NewResolver(reader)

	// Shallow resolution - references not resolved
	shallow, err := resolver.Resolve(dict)
	if err != nil {
		t.Fatalf("shallow resolve failed: %v", err)
	}
	shallowDict := shallow.(core.Dict)
	if _, ok := shallowDict["Ref"].(core.IndirectRef); !ok {
		t.Error("shallow resolve should not resolve references in dict")
	}

	// Deep resolution - references resolved
	resolver.Reset()
	deep, err := resolver.ResolveDeep(dict)
	if err != nil {
		t.Fatalf("deep resolve failed: %v", err)
	}
	deepDict := deep.(core.Dict)
	if str, ok := deepDict["Ref"].(core.String); !ok || string(str) != "Value" {
		t.Error("deep resolve should resolve references in dict")
	}
}

// TestResolveArray tests array resolution
func TestResolveArray(t *testing.T) {
	reader := newMockReader()
	reader.AddObject(20, core.String("Element"))

	arr := core.Array{
		core.Int(1),
		core.IndirectRef{Number: 20},
		core.String("Direct"),
	}

	resolver := NewResolver(reader)

	// Shallow resolution
	shallow, err := resolver.Resolve(arr)
	if err != nil {
		t.Fatalf("shallow resolve failed: %v", err)
	}
	shallowArr := shallow.(core.Array)
	if _, ok := shallowArr[1].(core.IndirectRef); !ok {
		t.Error("shallow resolve should not resolve references in array")
	}

	// Deep resolution
	resolver.Reset()
	deep, err := resolver.ResolveDeep(arr)
	if err != nil {
		t.Fatalf("deep resolve failed: %v", err)
	}
	deepArr := deep.(core.Array)
	if str, ok := deepArr[1].(core.String); !ok || string(str) != "Element" {
		t.Error("deep resolve should resolve references in array")
	}
}

// TestResolveNestedDict tests nested dictionary resolution
func TestResolveNestedDict(t *testing.T) {
	reader := newMockReader()
	reader.AddObject(30, core.String("Nested Value"))
	reader.AddObject(31, core.Dict{
		"Value": core.IndirectRef{Number: 30},
	})

	topDict := core.Dict{
		"Nested": core.IndirectRef{Number: 31},
	}

	resolver := NewResolver(reader)
	resolved, err := resolver.ResolveDeep(topDict)
	if err != nil {
		t.Fatalf("failed to resolve nested dict: %v", err)
	}

	resolvedDict := resolved.(core.Dict)
	nestedDict, ok := resolvedDict["Nested"].(core.Dict)
	if !ok {
		t.Fatal("nested dict not resolved")
	}

	str, ok := nestedDict["Value"].(core.String)
	if !ok || string(str) != "Nested Value" {
		t.Error("deeply nested reference not resolved")
	}
}

// TestResolveNestedArray tests nested array resolution
func TestResolveNestedArray(t *testing.T) {
	reader := newMockReader()
	reader.AddObject(40, core.String("Inner"))
	reader.AddObject(41, core.Array{
		core.IndirectRef{Number: 40},
	})

	topArray := core.Array{
		core.IndirectRef{Number: 41},
	}

	resolver := NewResolver(reader)
	resolved, err := resolver.ResolveDeep(topArray)
	if err != nil {
		t.Fatalf("failed to resolve nested array: %v", err)
	}

	resolvedArr := resolved.(core.Array)
	innerArr, ok := resolvedArr[0].(core.Array)
	if !ok {
		t.Fatal("nested array not resolved")
	}

	str, ok := innerArr[0].(core.String)
	if !ok || string(str) != "Inner" {
		t.Error("deeply nested reference not resolved")
	}
}

// TestCycleDetection tests that circular references are detected
func TestCycleDetection(t *testing.T) {
	reader := newMockReader()

	// Create circular reference: 50 -> 51 -> 50
	reader.AddObject(50, core.Dict{
		"Next": core.IndirectRef{Number: 51},
	})
	reader.AddObject(51, core.Dict{
		"Next": core.IndirectRef{Number: 50},
	})

	resolver := NewResolver(reader)
	ref := core.IndirectRef{Number: 50}

	_, err := resolver.ResolveDeep(ref)
	if err == nil {
		t.Error("expected error for circular reference")
	}
	// Check that error contains "circular reference detected for object 50"
	if err != nil {
		errMsg := err.Error()
		if !strings.Contains(errMsg, "circular reference detected for object 50") {
			t.Errorf("expected circular reference error, got: %v", err)
		}
	}
}

// TestMaxDepth tests depth limiting
func TestMaxDepth(t *testing.T) {
	reader := newMockReader()

	// Create deep nesting chain: 60 -> 61 -> 62 -> ... -> 70
	for i := 60; i < 70; i++ {
		reader.AddObject(i, core.Dict{
			"Next": core.IndirectRef{Number: i + 1},
		})
	}
	reader.AddObject(70, core.String("End"))

	// Set max depth to 5
	resolver := NewResolver(reader, WithMaxDepth(5))
	ref := core.IndirectRef{Number: 60}

	_, err := resolver.ResolveDeep(ref)
	if err == nil {
		t.Error("expected error for exceeding max depth")
	}
}

// TestResolveDict convenience method
func TestResolveDictConvenience(t *testing.T) {
	reader := newMockReader()
	reader.AddObject(80, core.String("Value"))

	dict := core.Dict{
		"Key": core.IndirectRef{Number: 80},
	}

	resolver := NewResolver(reader)
	resolved, err := resolver.ResolveDict(dict)
	if err != nil {
		t.Fatalf("ResolveDict failed: %v", err)
	}

	str, ok := resolved["Key"].(core.String)
	if !ok || string(str) != "Value" {
		t.Error("ResolveDict did not resolve reference")
	}
}

// TestResolveArray convenience method
func TestResolveArrayConvenience(t *testing.T) {
	reader := newMockReader()
	reader.AddObject(90, core.String("Element"))

	arr := core.Array{
		core.IndirectRef{Number: 90},
	}

	resolver := NewResolver(reader)
	resolved, err := resolver.ResolveArray(arr)
	if err != nil {
		t.Fatalf("ResolveArray failed: %v", err)
	}

	str, ok := resolved[0].(core.String)
	if !ok || string(str) != "Element" {
		t.Error("ResolveArray did not resolve reference")
	}
}

// TestResolveStream tests stream resolution
func TestResolveStream(t *testing.T) {
	reader := newMockReader()
	reader.AddObject(100, core.Name("FlateDecode"))

	stream := &core.Stream{
		Dict: core.Dict{
			"Filter": core.IndirectRef{Number: 100},
			"Length": core.Int(100),
		},
		Data: []byte("stream data"),
	}

	resolver := NewResolver(reader)

	// Shallow - dict not resolved
	shallow, err := resolver.Resolve(stream)
	if err != nil {
		t.Fatalf("shallow resolve failed: %v", err)
	}
	shallowStream := shallow.(*core.Stream)
	if _, ok := shallowStream.Dict["Filter"].(core.IndirectRef); !ok {
		t.Error("shallow resolve should not resolve stream dict")
	}

	// Deep - dict resolved
	resolver.Reset()
	deep, err := resolver.ResolveDeep(stream)
	if err != nil {
		t.Fatalf("deep resolve failed: %v", err)
	}
	deepStream := deep.(*core.Stream)
	name, ok := deepStream.Dict["Filter"].(core.Name)
	if !ok || string(name) != "FlateDecode" {
		t.Error("deep resolve should resolve stream dict")
	}

	// Verify stream data unchanged
	if string(deepStream.Data) != "stream data" {
		t.Error("stream data should not change")
	}
}

// TestGetObjectResolved tests the convenience method
func TestGetObjectResolved(t *testing.T) {
	reader := newMockReader()
	reader.AddObject(110, core.Int(123))

	resolver := NewResolver(reader)

	obj, err := resolver.GetObjectResolved(110)
	if err != nil {
		t.Fatalf("GetObjectResolved failed: %v", err)
	}

	val, ok := obj.(core.Int)
	if !ok || val != 123 {
		t.Error("GetObjectResolved returned wrong value")
	}
}

// TestGetObjectResolvedDeep tests deep object loading
func TestGetObjectResolvedDeep(t *testing.T) {
	reader := newMockReader()
	reader.AddObject(120, core.String("Value"))
	reader.AddObject(121, core.Dict{
		"Ref": core.IndirectRef{Number: 120},
	})

	resolver := NewResolver(reader)

	obj, err := resolver.GetObjectResolvedDeep(121)
	if err != nil {
		t.Fatalf("GetObjectResolvedDeep failed: %v", err)
	}

	dict, ok := obj.(core.Dict)
	if !ok {
		t.Fatal("expected Dict")
	}

	str, ok := dict["Ref"].(core.String)
	if !ok || string(str) != "Value" {
		t.Error("reference not deeply resolved")
	}
}

// TestReset tests that Reset clears visited map
func TestReset(t *testing.T) {
	reader := newMockReader()
	reader.AddObject(130, core.String("Value"))

	resolver := NewResolver(reader)
	ref := core.IndirectRef{Number: 130}

	// First resolution
	_, err := resolver.Resolve(ref)
	if err != nil {
		t.Fatalf("first resolve failed: %v", err)
	}

	if !resolver.visited[130] {
		t.Error("object should be marked as visited")
	}

	// Reset
	resolver.Reset()

	if resolver.visited[130] {
		t.Error("Reset should clear visited map")
	}

	if resolver.currentDepth != 0 {
		t.Error("Reset should clear depth")
	}
}

// TestObjectNotFound tests error handling for missing objects
func TestObjectNotFound(t *testing.T) {
	reader := newMockReader()
	resolver := NewResolver(reader)

	ref := core.IndirectRef{Number: 999}

	_, err := resolver.Resolve(ref)
	if err == nil {
		t.Error("expected error for missing object")
	}
}

// TestComplexStructure tests a complex nested structure
func TestComplexStructure(t *testing.T) {
	reader := newMockReader()

	// Build complex structure
	reader.AddObject(200, core.String("Leaf1"))
	reader.AddObject(201, core.String("Leaf2"))
	reader.AddObject(202, core.Array{
		core.IndirectRef{Number: 200},
		core.IndirectRef{Number: 201},
	})
	reader.AddObject(203, core.Dict{
		"Array": core.IndirectRef{Number: 202},
		"Value": core.Int(42),
	})

	root := core.Dict{
		"Top": core.IndirectRef{Number: 203},
		"Direct": core.String("DirectValue"),
	}

	resolver := NewResolver(reader)
	resolved, err := resolver.ResolveDeep(root)
	if err != nil {
		t.Fatalf("failed to resolve complex structure: %v", err)
	}

	// Verify structure
	resolvedDict := resolved.(core.Dict)

	// Check direct value unchanged
	if str, ok := resolvedDict["Direct"].(core.String); !ok || string(str) != "DirectValue" {
		t.Error("direct value changed")
	}

	// Check nested dict resolved
	topDict, ok := resolvedDict["Top"].(core.Dict)
	if !ok {
		t.Fatal("top dict not resolved")
	}

	// Check value in nested dict
	if val, ok := topDict["Value"].(core.Int); !ok || val != 42 {
		t.Error("nested dict value incorrect")
	}

	// Check nested array resolved
	arr, ok := topDict["Array"].(core.Array)
	if !ok {
		t.Fatal("nested array not resolved")
	}

	// Check array elements resolved
	if str, ok := arr[0].(core.String); !ok || string(str) != "Leaf1" {
		t.Error("array element 0 not resolved")
	}
	if str, ok := arr[1].(core.String); !ok || string(str) != "Leaf2" {
		t.Error("array element 1 not resolved")
	}
}
