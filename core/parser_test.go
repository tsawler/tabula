package core

import (
	"fmt"
	"io"
	"strings"
	"testing"
)

// TestParserNull tests parsing null objects
func TestParserNull(t *testing.T) {
	input := "null"
	parser := NewParser(strings.NewReader(input))
	obj, err := parser.ParseObject()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := obj.(Null); !ok {
		t.Errorf("expected Null, got %T", obj)
	}
}

// TestParserBool tests parsing boolean objects
func TestParserBool(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"true", "true", true},
		{"false", "false", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(strings.NewReader(tt.input))
			obj, err := parser.ParseObject()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			b, ok := obj.(Bool)
			if !ok {
				t.Fatalf("expected Bool, got %T", obj)
			}
			if bool(b) != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, bool(b))
			}
		})
	}
}

// TestParserInt tests parsing integer objects
func TestParserInt(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
	}{
		{"zero", "0", 0},
		{"positive", "123", 123},
		{"negative", "-456", -456},
		{"large", "999999", 999999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(strings.NewReader(tt.input))
			obj, err := parser.ParseObject()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			i, ok := obj.(Int)
			if !ok {
				t.Fatalf("expected Int, got %T", obj)
			}
			if int64(i) != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, int64(i))
			}
		})
	}
}

// TestParserReal tests parsing real number objects
func TestParserReal(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected float64
	}{
		{"simple", "3.14", 3.14},
		{"negative", "-2.5", -2.5},
		{"leading decimal", ".5", 0.5},
		{"trailing decimal", "5.", 5.0},
		{"zero", "0.0", 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(strings.NewReader(tt.input))
			obj, err := parser.ParseObject()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			r, ok := obj.(Real)
			if !ok {
				t.Fatalf("expected Real, got %T", obj)
			}
			if float64(r) != tt.expected {
				t.Errorf("expected %f, got %f", tt.expected, float64(r))
			}
		})
	}
}

// TestParserString tests parsing string objects
func TestParserString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple", "(hello)", "hello"},
		{"empty", "()", ""},
		{"with spaces", "(hello world)", "hello world"},
		{"nested", "(hello (world))", "hello (world)"},
		{"escaped", "(hello\\nworld)", "hello\nworld"},
		{"octal", "(\\101\\102)", "AB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(strings.NewReader(tt.input))
			obj, err := parser.ParseObject()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			s, ok := obj.(String)
			if !ok {
				t.Fatalf("expected String, got %T", obj)
			}
			if string(s) != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, string(s))
			}
		})
	}
}

// TestParserHexString tests parsing hexadecimal string objects
func TestParserHexString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple", "<48656C6C6F>", "Hello"},
		{"empty", "<>", ""},
		{"lowercase", "<68656c6c6f>", "hello"},
		{"uppercase", "<48454C4C4F>", "HELLO"},
		{"with whitespace", "<48 65 6C 6C 6F>", "Hello"},
		{"odd length", "<123>", "\x12\x30"}, // Padded with 0
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(strings.NewReader(tt.input))
			obj, err := parser.ParseObject()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			s, ok := obj.(String)
			if !ok {
				t.Fatalf("expected String, got %T", obj)
			}
			if string(s) != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, string(s))
			}
		})
	}
}

// TestParserName tests parsing name objects
func TestParserName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple", "/Type", "Type"},
		{"empty", "/", ""},
		{"with numbers", "/F1", "F1"},
		{"complex", "/BaseFont", "BaseFont"},
		{"with escape", "/Name#20Test", "Name Test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(strings.NewReader(tt.input))
			obj, err := parser.ParseObject()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			n, ok := obj.(Name)
			if !ok {
				t.Fatalf("expected Name, got %T", obj)
			}
			if string(n) != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, string(n))
			}
		})
	}
}

// TestParserArray tests parsing array objects
func TestParserArray(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int // expected length
	}{
		{"empty", "[]", 0},
		{"integers", "[1 2 3]", 3},
		{"mixed", "[1 /Name (string) true]", 4},
		{"nested", "[[1 2] [3 4]]", 2},
		{"with whitespace", "[ 1 2 3 ]", 3},
		{"with newlines", "[\n1\n2\n3\n]", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(strings.NewReader(tt.input))
			obj, err := parser.ParseObject()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			arr, ok := obj.(Array)
			if !ok {
				t.Fatalf("expected Array, got %T", obj)
			}
			if len(arr) != tt.expected {
				t.Errorf("expected length %d, got %d", tt.expected, len(arr))
			}
		})
	}
}

// TestParserArrayElements tests array element access
func TestParserArrayElements(t *testing.T) {
	input := "[123 3.14 /Name (string) true false null]"
	parser := NewParser(strings.NewReader(input))
	obj, err := parser.ParseObject()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	arr, ok := obj.(Array)
	if !ok {
		t.Fatalf("expected Array, got %T", obj)
	}

	// Check element types
	if _, ok := arr[0].(Int); !ok {
		t.Errorf("element 0: expected Int, got %T", arr[0])
	}
	if _, ok := arr[1].(Real); !ok {
		t.Errorf("element 1: expected Real, got %T", arr[1])
	}
	if _, ok := arr[2].(Name); !ok {
		t.Errorf("element 2: expected Name, got %T", arr[2])
	}
	if _, ok := arr[3].(String); !ok {
		t.Errorf("element 3: expected String, got %T", arr[3])
	}
	if _, ok := arr[4].(Bool); !ok {
		t.Errorf("element 4: expected Bool, got %T", arr[4])
	}
	if _, ok := arr[5].(Bool); !ok {
		t.Errorf("element 5: expected Bool, got %T", arr[5])
	}
	if _, ok := arr[6].(Null); !ok {
		t.Errorf("element 6: expected Null, got %T", arr[6])
	}
}

// TestParserDict tests parsing dictionary objects
func TestParserDict(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int // expected number of keys
	}{
		{"empty", "<<>>", 0},
		{"single entry", "<</Type /Page>>", 1},
		{"multiple entries", "<</Type /Page /Count 1>>", 2},
		{"with whitespace", "<< /Type /Page >>", 1},
		{"with newlines", "<<\n/Type /Page\n>>", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(strings.NewReader(tt.input))
			obj, err := parser.ParseObject()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			dict, ok := obj.(Dict)
			if !ok {
				t.Fatalf("expected Dict, got %T", obj)
			}
			if len(dict) != tt.expected {
				t.Errorf("expected %d keys, got %d", tt.expected, len(dict))
			}
		})
	}
}

// TestParserDictAccess tests dictionary value access
func TestParserDictAccess(t *testing.T) {
	input := "<</Type /Page /Count 10 /Title (Test) /Active true>>"
	parser := NewParser(strings.NewReader(input))
	obj, err := parser.ParseObject()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	dict, ok := obj.(Dict)
	if !ok {
		t.Fatalf("expected Dict, got %T", obj)
	}

	// Check Type
	typeObj := dict.Get("Type")
	if typeObj == nil {
		t.Error("expected Type key")
	} else if typeName, ok := typeObj.(Name); !ok || string(typeName) != "Page" {
		t.Errorf("expected Type=/Page, got %v", typeObj)
	}

	// Check Count
	countObj := dict.Get("Count")
	if countObj == nil {
		t.Error("expected Count key")
	} else if count, ok := countObj.(Int); !ok || int(count) != 10 {
		t.Errorf("expected Count=10, got %v", countObj)
	}

	// Check Title
	titleObj := dict.Get("Title")
	if titleObj == nil {
		t.Error("expected Title key")
	} else if title, ok := titleObj.(String); !ok || string(title) != "Test" {
		t.Errorf("expected Title='Test', got %v", titleObj)
	}

	// Check Active
	activeObj := dict.Get("Active")
	if activeObj == nil {
		t.Error("expected Active key")
	} else if active, ok := activeObj.(Bool); !ok || !bool(active) {
		t.Errorf("expected Active=true, got %v", activeObj)
	}
}

// TestParserIndirectRef tests parsing indirect references
func TestParserIndirectRef(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		number int
		gen    int
	}{
		{"simple", "5 0 R", 5, 0},
		{"with generation", "12 3 R", 12, 3},
		{"large number", "999 0 R", 999, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(strings.NewReader(tt.input))
			obj, err := parser.ParseObject()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			ref, ok := obj.(IndirectRef)
			if !ok {
				t.Fatalf("expected IndirectRef, got %T", obj)
			}
			if ref.Number != tt.number {
				t.Errorf("expected number %d, got %d", tt.number, ref.Number)
			}
			if ref.Generation != tt.gen {
				t.Errorf("expected generation %d, got %d", tt.gen, ref.Generation)
			}
		})
	}
}

// TestParserNestedArray tests nested array parsing
func TestParserNestedArray(t *testing.T) {
	input := "[[1 2] [3 4] [5 6]]"
	parser := NewParser(strings.NewReader(input))
	obj, err := parser.ParseObject()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	arr, ok := obj.(Array)
	if !ok {
		t.Fatalf("expected Array, got %T", obj)
	}
	if len(arr) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(arr))
	}

	// Check nested arrays
	for i := 0; i < 3; i++ {
		nested, ok := arr[i].(Array)
		if !ok {
			t.Errorf("element %d: expected Array, got %T", i, arr[i])
			continue
		}
		if len(nested) != 2 {
			t.Errorf("nested array %d: expected 2 elements, got %d", i, len(nested))
		}
	}
}

// TestParserNestedDict tests nested dictionary parsing
func TestParserNestedDict(t *testing.T) {
	input := "<</Outer <</Inner /Value>>>>"
	parser := NewParser(strings.NewReader(input))
	obj, err := parser.ParseObject()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	dict, ok := obj.(Dict)
	if !ok {
		t.Fatalf("expected Dict, got %T", obj)
	}

	outerObj := dict.Get("Outer")
	if outerObj == nil {
		t.Fatal("expected Outer key")
	}
	innerDict, ok := outerObj.(Dict)
	if !ok {
		t.Fatalf("expected nested Dict, got %T", outerObj)
	}

	innerValue := innerDict.Get("Inner")
	if innerValue == nil {
		t.Fatal("expected Inner key")
	}
	if name, ok := innerValue.(Name); !ok || string(name) != "Value" {
		t.Errorf("expected Inner=/Value, got %v", innerValue)
	}
}

// TestParserComplexStructure tests a complex nested structure
func TestParserComplexStructure(t *testing.T) {
	input := `<<
		/Type /Page
		/MediaBox [0 0 612 792]
		/Resources <<
			/Font << /F1 5 0 R >>
		>>
		/Contents 10 0 R
	>>`

	parser := NewParser(strings.NewReader(input))
	obj, err := parser.ParseObject()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	dict, ok := obj.(Dict)
	if !ok {
		t.Fatalf("expected Dict, got %T", obj)
	}

	// Check Type
	if typeObj, ok := dict.GetName("Type"); !ok || string(typeObj) != "Page" {
		t.Errorf("expected Type=/Page")
	}

	// Check MediaBox
	if mediaBox, ok := dict.GetArray("MediaBox"); !ok || len(mediaBox) != 4 {
		t.Errorf("expected MediaBox array with 4 elements")
	}

	// Check Resources
	resources, ok := dict.GetDict("Resources")
	if !ok {
		t.Fatal("expected Resources dict")
	}

	// Check Font in Resources
	font, ok := resources.GetDict("Font")
	if !ok {
		t.Fatal("expected Font dict in Resources")
	}

	// Check F1 reference
	f1, ok := font.GetIndirectRef("F1")
	if !ok || f1.Number != 5 {
		t.Errorf("expected F1=5 0 R")
	}

	// Check Contents reference
	contents, ok := dict.GetIndirectRef("Contents")
	if !ok || contents.Number != 10 {
		t.Errorf("expected Contents=10 0 R")
	}
}

// TestParserIndirectObject tests parsing indirect objects
func TestParserIndirectObject(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		number int
		gen    int
	}{
		{
			"simple int",
			"5 0 obj\n123\nendobj",
			5, 0,
		},
		{
			"dict",
			"10 0 obj\n<</Type /Page>>\nendobj",
			10, 0,
		},
		{
			"array",
			"3 2 obj\n[1 2 3]\nendobj",
			3, 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(strings.NewReader(tt.input))
			indObj, err := parser.ParseIndirectObject()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if indObj.Ref.Number != tt.number {
				t.Errorf("expected number %d, got %d", tt.number, indObj.Ref.Number)
			}
			if indObj.Ref.Generation != tt.gen {
				t.Errorf("expected generation %d, got %d", tt.gen, indObj.Ref.Generation)
			}
			if indObj.Object == nil {
				t.Error("expected non-nil object")
			}
		})
	}
}

// TestParserMultipleObjects tests parsing multiple objects in sequence
func TestParserMultipleObjects(t *testing.T) {
	input := "123 /Name (string) [1 2 3] << /Key /Value >>"
	parser := NewParser(strings.NewReader(input))

	// Parse integer
	obj1, err := parser.ParseObject()
	if err != nil {
		t.Fatalf("object 1 error: %v", err)
	}
	if _, ok := obj1.(Int); !ok {
		t.Errorf("object 1: expected Int, got %T", obj1)
	}

	// Parse name
	obj2, err := parser.ParseObject()
	if err != nil {
		t.Fatalf("object 2 error: %v", err)
	}
	if _, ok := obj2.(Name); !ok {
		t.Errorf("object 2: expected Name, got %T", obj2)
	}

	// Parse string
	obj3, err := parser.ParseObject()
	if err != nil {
		t.Fatalf("object 3 error: %v", err)
	}
	if _, ok := obj3.(String); !ok {
		t.Errorf("object 3: expected String, got %T", obj3)
	}

	// Parse array
	obj4, err := parser.ParseObject()
	if err != nil {
		t.Fatalf("object 4 error: %v", err)
	}
	if _, ok := obj4.(Array); !ok {
		t.Errorf("object 4: expected Array, got %T", obj4)
	}

	// Parse dict
	obj5, err := parser.ParseObject()
	if err != nil {
		t.Fatalf("object 5 error: %v", err)
	}
	if _, ok := obj5.(Dict); !ok {
		t.Errorf("object 5: expected Dict, got %T", obj5)
	}

	// Should be EOF
	obj6, err := parser.ParseObject()
	if err != io.EOF {
		t.Errorf("expected EOF, got error: %v, obj: %v", err, obj6)
	}
}

// TestParserWithComments tests parsing with comments
func TestParserWithComments(t *testing.T) {
	input := `%Comment before
123
%Comment between
/Name
%Comment after`

	parser := NewParser(strings.NewReader(input))

	// Parse integer (comments should be skipped)
	obj1, err := parser.ParseObject()
	if err != nil {
		t.Fatalf("object 1 error: %v", err)
	}
	if _, ok := obj1.(Int); !ok {
		t.Errorf("object 1: expected Int, got %T", obj1)
	}

	// Parse name
	obj2, err := parser.ParseObject()
	if err != nil {
		t.Fatalf("object 2 error: %v", err)
	}
	if _, ok := obj2.(Name); !ok {
		t.Errorf("object 2: expected Name, got %T", obj2)
	}
}

// TestParserErrors tests error handling
func TestParserErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"unclosed array", "[1 2 3"},
		{"unclosed dict", "<</Key /Value"},
		{"invalid number", "abc"},
		{"dict without key", "<<123>>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(strings.NewReader(tt.input))
			_, err := parser.ParseObject()
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

// TestParserRealPDF tests parsing realistic PDF fragments
func TestParserRealPDF(t *testing.T) {
	input := `1 0 obj
<< /Type /Catalog /Pages 2 0 R >>
endobj`

	parser := NewParser(strings.NewReader(input))
	indObj, err := parser.ParseIndirectObject()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if indObj.Ref.Number != 1 || indObj.Ref.Generation != 0 {
		t.Errorf("expected 1 0 obj, got %d %d obj", indObj.Ref.Number, indObj.Ref.Generation)
	}

	dict, ok := indObj.Object.(Dict)
	if !ok {
		t.Fatalf("expected Dict, got %T", indObj.Object)
	}

	if typeObj, ok := dict.GetName("Type"); !ok || string(typeObj) != "Catalog" {
		t.Errorf("expected Type=/Catalog")
	}

	if pages, ok := dict.GetIndirectRef("Pages"); !ok || pages.Number != 2 {
		t.Errorf("expected Pages=2 0 R")
	}
}

// mockResolver implements ReferenceResolver for testing
type mockResolver struct {
	objects map[int]Object
}

func (m *mockResolver) ResolveReference(ref IndirectRef) (Object, error) {
	if obj, ok := m.objects[ref.Number]; ok {
		return obj, nil
	}
	return nil, fmt.Errorf("object %d not found", ref.Number)
}

func TestParseStreamWithIndirectLength(t *testing.T) {
	// Stream with indirect length reference (5 0 R)
	input := "1 0 obj\n<< /Length 5 0 R >>\nstream\nHello\nendstream\nendobj"
	parser := NewParser(strings.NewReader(input))

	// Set up a mock resolver that returns 6 (length of "Hello\n")
	resolver := &mockResolver{
		objects: map[int]Object{
			5: Int(6),
		},
	}
	parser.SetReferenceResolver(resolver)

	indObj, err := parser.ParseIndirectObject()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stream, ok := indObj.Object.(*Stream)
	if !ok {
		t.Fatalf("expected *Stream, got %T", indObj.Object)
	}

	if string(stream.Data) != "Hello\n" {
		t.Errorf("expected stream data 'Hello\\n', got %q", string(stream.Data))
	}
}

func TestParseStreamWithBinaryData(t *testing.T) {
	// Stream with binary data starting with NULL bytes (whitespace-like characters)
	// This tests that the parser correctly handles binary streams without corrupting
	// data that looks like PDF whitespace (NULL, CR, LF, etc.)
	binaryData := []byte{0x00, 0x16, 0x0a, 0x40, 0x05, 0x82} // starts with NULL, contains LF
	input := fmt.Sprintf("1 0 obj\n<< /Length %d >>\nstream\n%sendstream\nendobj", len(binaryData), string(binaryData))
	parser := NewParser(strings.NewReader(input))

	indObj, err := parser.ParseIndirectObject()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stream, ok := indObj.Object.(*Stream)
	if !ok {
		t.Fatalf("expected *Stream, got %T", indObj.Object)
	}

	if len(stream.Data) != len(binaryData) {
		t.Errorf("expected stream data length %d, got %d", len(binaryData), len(stream.Data))
	}

	for i, b := range binaryData {
		if i >= len(stream.Data) {
			break
		}
		if stream.Data[i] != b {
			t.Errorf("byte %d: expected 0x%02x, got 0x%02x", i, b, stream.Data[i])
		}
	}
}

func TestParseStreamWithIndirectLengthNoResolver(t *testing.T) {
	// Stream with indirect length reference but no resolver set
	input := "1 0 obj\n<< /Length 5 0 R >>\nstream\nHello\nendstream\nendobj"
	parser := NewParser(strings.NewReader(input))

	_, err := parser.ParseIndirectObject()
	if err == nil {
		t.Fatal("expected error when no resolver set")
	}

	expectedMsg := "indirect reference for stream length requires a reference resolver"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Errorf("expected error containing %q, got %q", expectedMsg, err.Error())
	}
}

// Benchmark tests
func BenchmarkParserSimpleObject(b *testing.B) {
	input := "123"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser := NewParser(strings.NewReader(input))
		parser.ParseObject()
	}
}

func BenchmarkParserArray(b *testing.B) {
	input := "[1 2 3 4 5 6 7 8 9 10]"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser := NewParser(strings.NewReader(input))
		parser.ParseObject()
	}
}

func BenchmarkParserDict(b *testing.B) {
	input := "<</Type /Page /MediaBox [0 0 612 792] /Contents 10 0 R>>"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser := NewParser(strings.NewReader(input))
		parser.ParseObject()
	}
}

func BenchmarkParserIndirectObject(b *testing.B) {
	input := "1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser := NewParser(strings.NewReader(input))
		parser.ParseIndirectObject()
	}
}
