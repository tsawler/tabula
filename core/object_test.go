package core

import (
	"testing"
)

// TestObjectType tests the ObjectType String() method
func TestObjectType(t *testing.T) {
	tests := []struct {
		name string
		typ  ObjectType
		want string
	}{
		{"Null", ObjNull, "Null"},
		{"Bool", ObjBool, "Bool"},
		{"Int", ObjInt, "Int"},
		{"Real", ObjReal, "Real"},
		{"String", ObjString, "String"},
		{"Name", ObjName, "Name"},
		{"Array", ObjArray, "Array"},
		{"Dict", ObjDict, "Dict"},
		{"Stream", ObjStream, "Stream"},
		{"IndirectRef", ObjIndirect, "IndirectRef"},
		{"Unknown", ObjectType(99), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.typ.String(); got != tt.want {
				t.Errorf("ObjectType.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestNull tests the Null object
func TestNull(t *testing.T) {
	n := Null{}

	if n.Type() != ObjNull {
		t.Errorf("Null.Type() = %v, want %v", n.Type(), ObjNull)
	}

	if n.String() != "null" {
		t.Errorf("Null.String() = %v, want %v", n.String(), "null")
	}
}

// TestBool tests the Bool object
func TestBool(t *testing.T) {
	tests := []struct {
		name  string
		value Bool
		wantS string
		wantT ObjectType
	}{
		{"true", Bool(true), "true", ObjBool},
		{"false", Bool(false), "false", ObjBool},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value.Type() != tt.wantT {
				t.Errorf("Bool.Type() = %v, want %v", tt.value.Type(), tt.wantT)
			}
			if tt.value.String() != tt.wantS {
				t.Errorf("Bool.String() = %v, want %v", tt.value.String(), tt.wantS)
			}
		})
	}
}

// TestInt tests the Int object
func TestInt(t *testing.T) {
	tests := []struct {
		name  string
		value Int
		want  string
	}{
		{"zero", Int(0), "0"},
		{"positive", Int(42), "42"},
		{"negative", Int(-17), "-17"},
		{"large", Int(9223372036854775807), "9223372036854775807"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value.Type() != ObjInt {
				t.Errorf("Int.Type() = %v, want %v", tt.value.Type(), ObjInt)
			}
			if tt.value.String() != tt.want {
				t.Errorf("Int.String() = %v, want %v", tt.value.String(), tt.want)
			}
		})
	}
}

// TestReal tests the Real object
func TestReal(t *testing.T) {
	tests := []struct {
		name  string
		value Real
		want  string
	}{
		{"zero", Real(0.0), "0"},
		{"positive", Real(3.14), "3.14"},
		{"negative", Real(-2.5), "-2.5"},
		{"integer", Real(42.0), "42"},
		{"small", Real(0.001), "0.001"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value.Type() != ObjReal {
				t.Errorf("Real.Type() = %v, want %v", tt.value.Type(), ObjReal)
			}
			if tt.value.String() != tt.want {
				t.Errorf("Real.String() = %v, want %v", tt.value.String(), tt.want)
			}
		})
	}
}

// TestString tests the String object
func TestString(t *testing.T) {
	tests := []struct {
		name  string
		value String
		want  string
	}{
		{"empty", String(""), ""},
		{"simple", String("hello"), "hello"},
		{"with spaces", String("hello world"), "hello world"},
		{"special chars", String("test\n\r\t"), "test\n\r\t"},
		{"unicode", String("Hello 世界"), "Hello 世界"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value.Type() != ObjString {
				t.Errorf("String.Type() = %v, want %v", tt.value.Type(), ObjString)
			}
			if tt.value.String() != tt.want {
				t.Errorf("String.String() = %v, want %v", tt.value.String(), tt.want)
			}
		})
	}
}

// TestName tests the Name object
func TestName(t *testing.T) {
	tests := []struct {
		name  string
		value Name
		want  string
	}{
		{"simple", Name("Type"), "/Type"},
		{"with number", Name("Page1"), "/Page1"},
		{"with underscore", Name("Parent_Page"), "/Parent_Page"},
		{"empty", Name(""), "/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value.Type() != ObjName {
				t.Errorf("Name.Type() = %v, want %v", tt.value.Type(), ObjName)
			}
			if tt.value.String() != tt.want {
				t.Errorf("Name.String() = %v, want %v", tt.value.String(), tt.want)
			}
		})
	}
}

// TestArray tests the Array object
func TestArray(t *testing.T) {
	t.Run("basic operations", func(t *testing.T) {
		arr := Array{Int(1), Int(2), Int(3)}

		if arr.Type() != ObjArray {
			t.Errorf("Array.Type() = %v, want %v", arr.Type(), ObjArray)
		}

		if arr.String() != "[1 2 3]" {
			t.Errorf("Array.String() = %v, want %v", arr.String(), "[1 2 3]")
		}

		if arr.Len() != 3 {
			t.Errorf("Array.Len() = %v, want %v", arr.Len(), 3)
		}
	})

	t.Run("Get", func(t *testing.T) {
		arr := Array{Int(10), Int(20), Int(30)}

		// Valid indices
		if obj := arr.Get(0); obj != Int(10) {
			t.Errorf("Array.Get(0) = %v, want %v", obj, Int(10))
		}
		if obj := arr.Get(2); obj != Int(30) {
			t.Errorf("Array.Get(2) = %v, want %v", obj, Int(30))
		}

		// Invalid indices
		if obj := arr.Get(-1); obj != nil {
			t.Errorf("Array.Get(-1) = %v, want nil", obj)
		}
		if obj := arr.Get(3); obj != nil {
			t.Errorf("Array.Get(3) = %v, want nil", obj)
		}
	})

	t.Run("GetInt", func(t *testing.T) {
		arr := Array{Int(42), Name("Test"), Real(3.14)}

		// Valid int
		if val, ok := arr.GetInt(0); !ok || val != Int(42) {
			t.Errorf("Array.GetInt(0) = %v, %v; want 42, true", val, ok)
		}

		// Wrong type
		if _, ok := arr.GetInt(1); ok {
			t.Error("Array.GetInt(1) should fail for Name type")
		}

		// Out of bounds
		if _, ok := arr.GetInt(10); ok {
			t.Error("Array.GetInt(10) should fail for out of bounds")
		}
	})

	t.Run("GetReal", func(t *testing.T) {
		arr := Array{Real(3.14), Int(42)}

		if val, ok := arr.GetReal(0); !ok || val != Real(3.14) {
			t.Errorf("Array.GetReal(0) = %v, %v; want 3.14, true", val, ok)
		}

		if _, ok := arr.GetReal(1); ok {
			t.Error("Array.GetReal(1) should fail for Int type")
		}
	})

	t.Run("GetName", func(t *testing.T) {
		arr := Array{Name("Type"), Int(42)}

		if val, ok := arr.GetName(0); !ok || val != Name("Type") {
			t.Errorf("Array.GetName(0) = %v, %v; want Type, true", val, ok)
		}

		if _, ok := arr.GetName(1); ok {
			t.Error("Array.GetName(1) should fail for Int type")
		}
	})

	t.Run("empty array", func(t *testing.T) {
		arr := Array{}

		if arr.Len() != 0 {
			t.Errorf("Empty Array.Len() = %v, want 0", arr.Len())
		}

		if arr.String() != "[]" {
			t.Errorf("Empty Array.String() = %v, want []", arr.String())
		}
	})

	t.Run("nested array", func(t *testing.T) {
		inner := Array{Int(1), Int(2)}
		outer := Array{inner, Int(3)}

		if outer.String() != "[[1 2] 3]" {
			t.Errorf("Nested Array.String() = %v, want [[1 2] 3]", outer.String())
		}
	})
}

// TestDict tests the Dict object
func TestDict(t *testing.T) {
	t.Run("basic operations", func(t *testing.T) {
		dict := Dict{
			"Type":  Name("Page"),
			"Count": Int(10),
		}

		if dict.Type() != ObjDict {
			t.Errorf("Dict.Type() = %v, want %v", dict.Type(), ObjDict)
		}

		// String() output order is not guaranteed, so just check it contains expected parts
		str := dict.String()
		if !contains(str, "/Type /Page") {
			t.Errorf("Dict.String() missing /Type /Page")
		}
		if !contains(str, "/Count 10") {
			t.Errorf("Dict.String() missing /Count 10")
		}
	})

	t.Run("Get", func(t *testing.T) {
		dict := Dict{"Key": Int(42)}

		if obj := dict.Get("Key"); obj != Int(42) {
			t.Errorf("Dict.Get(Key) = %v, want 42", obj)
		}

		if obj := dict.Get("Missing"); obj != nil {
			t.Errorf("Dict.Get(Missing) = %v, want nil", obj)
		}
	})

	t.Run("GetInt", func(t *testing.T) {
		dict := Dict{
			"Count": Int(42),
			"Type":  Name("Page"),
		}

		if val, ok := dict.GetInt("Count"); !ok || val != Int(42) {
			t.Errorf("Dict.GetInt(Count) = %v, %v; want 42, true", val, ok)
		}

		if _, ok := dict.GetInt("Type"); ok {
			t.Error("Dict.GetInt(Type) should fail for Name type")
		}

		if _, ok := dict.GetInt("Missing"); ok {
			t.Error("Dict.GetInt(Missing) should fail for missing key")
		}
	})

	t.Run("GetReal", func(t *testing.T) {
		dict := Dict{"Width": Real(612.0)}

		if val, ok := dict.GetReal("Width"); !ok || val != Real(612.0) {
			t.Errorf("Dict.GetReal(Width) = %v, %v; want 612.0, true", val, ok)
		}
	})

	t.Run("GetString", func(t *testing.T) {
		dict := Dict{"Title": String("Test")}

		if val, ok := dict.GetString("Title"); !ok || val != String("Test") {
			t.Errorf("Dict.GetString(Title) = %v, %v; want Test, true", val, ok)
		}
	})

	t.Run("GetBool", func(t *testing.T) {
		dict := Dict{"Visible": Bool(true)}

		if val, ok := dict.GetBool("Visible"); !ok || val != Bool(true) {
			t.Errorf("Dict.GetBool(Visible) = %v, %v; want true, true", val, ok)
		}
	})

	t.Run("GetName", func(t *testing.T) {
		dict := Dict{"Type": Name("Page")}

		if val, ok := dict.GetName("Type"); !ok || val != Name("Page") {
			t.Errorf("Dict.GetName(Type) = %v, %v; want Page, true", val, ok)
		}
	})

	t.Run("GetArray", func(t *testing.T) {
		arr := Array{Int(1), Int(2), Int(3)}
		dict := Dict{"Items": arr}

		if val, ok := dict.GetArray("Items"); !ok || val.Len() != 3 {
			t.Errorf("Dict.GetArray(Items) failed")
		}
	})

	t.Run("GetDict", func(t *testing.T) {
		inner := Dict{"Key": Int(42)}
		outer := Dict{"Inner": inner}

		if val, ok := outer.GetDict("Inner"); !ok {
			t.Error("Dict.GetDict(Inner) failed")
		} else {
			if v, ok := val.GetInt("Key"); !ok || v != Int(42) {
				t.Error("Nested dict access failed")
			}
		}
	})

	t.Run("GetIndirectRef", func(t *testing.T) {
		ref := IndirectRef{Number: 5, Generation: 0}
		dict := Dict{"Parent": ref}

		if val, ok := dict.GetIndirectRef("Parent"); !ok || val.Number != 5 {
			t.Errorf("Dict.GetIndirectRef(Parent) = %v, %v; want ref 5 0, true", val, ok)
		}
	})

	t.Run("GetStream", func(t *testing.T) {
		stream := &Stream{
			Dict: make(Dict),
			Data: []byte("test"),
		}
		dict := Dict{"Contents": stream}

		if val, ok := dict.GetStream("Contents"); !ok {
			t.Error("Dict.GetStream(Contents) failed")
		} else if string(val.Data) != "test" {
			t.Errorf("Stream data = %v, want test", string(val.Data))
		}

		// Test missing key
		if _, ok := dict.GetStream("Missing"); ok {
			t.Error("Dict.GetStream(Missing) should fail")
		}

		// Test wrong type
		dict["Wrong"] = Int(42)
		if _, ok := dict.GetStream("Wrong"); ok {
			t.Error("Dict.GetStream(Wrong) should fail for non-stream")
		}
	})

	t.Run("Has", func(t *testing.T) {
		dict := Dict{"Key": Int(42)}

		if !dict.Has("Key") {
			t.Error("Dict.Has(Key) = false, want true")
		}

		if dict.Has("Missing") {
			t.Error("Dict.Has(Missing) = true, want false")
		}
	})

	t.Run("Set", func(t *testing.T) {
		dict := make(Dict)
		dict.Set("Key", Int(42))

		if val, ok := dict.GetInt("Key"); !ok || val != Int(42) {
			t.Error("Dict.Set failed")
		}

		// Overwrite
		dict.Set("Key", Int(99))
		if val, ok := dict.GetInt("Key"); !ok || val != Int(99) {
			t.Error("Dict.Set overwrite failed")
		}
	})

	t.Run("Delete", func(t *testing.T) {
		dict := Dict{"Key": Int(42)}
		dict.Delete("Key")

		if dict.Has("Key") {
			t.Error("Dict.Delete failed, key still exists")
		}

		// Deleting non-existent key should not panic
		dict.Delete("Missing")
	})

	t.Run("Keys", func(t *testing.T) {
		dict := Dict{
			"A": Int(1),
			"B": Int(2),
			"C": Int(3),
		}

		keys := dict.Keys()
		if len(keys) != 3 {
			t.Errorf("Dict.Keys() returned %d keys, want 3", len(keys))
		}

		// Check all keys are present
		keyMap := make(map[string]bool)
		for _, k := range keys {
			keyMap[k] = true
		}
		if !keyMap["A"] || !keyMap["B"] || !keyMap["C"] {
			t.Error("Dict.Keys() missing expected keys")
		}
	})

	t.Run("empty dict", func(t *testing.T) {
		dict := make(Dict)

		if dict.String() != "<<>>" {
			t.Errorf("Empty Dict.String() = %v, want <<>>", dict.String())
		}

		if len(dict.Keys()) != 0 {
			t.Error("Empty Dict.Keys() should return empty slice")
		}
	})
}

// TestStream tests the Stream object
func TestStream(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		dict := Dict{"Length": Int(5)}
		data := []byte("hello")
		stream := &Stream{
			Dict: dict,
			Data: data,
		}

		if stream.Type() != ObjStream {
			t.Errorf("Stream.Type() = %v, want %v", stream.Type(), ObjStream)
		}

		str := stream.String()
		if !contains(str, "stream") || !contains(str, "5 bytes") {
			t.Errorf("Stream.String() = %v, want to contain 'stream' and '5 bytes'", str)
		}
	})

	t.Run("Decoded", func(t *testing.T) {
		data := []byte("test data")
		stream := &Stream{
			Dict: make(Dict),
			Data: data,
		}

		// Without filters, should return raw data
		decoded, err := stream.Decoded()
		if err != nil {
			t.Errorf("Stream.Decoded() error = %v", err)
		}
		if string(decoded) != "test data" {
			t.Errorf("Stream.Decoded() = %v, want %v", string(decoded), "test data")
		}

		// Should cache decoded data
		decoded2, _ := stream.Decoded()
		if &decoded[0] != &decoded2[0] {
			t.Error("Stream.Decoded() should cache result")
		}
	})
}

// TestIndirectRef tests the IndirectRef object
func TestIndirectRef(t *testing.T) {
	tests := []struct {
		name       string
		ref        IndirectRef
		wantString string
	}{
		{"simple", IndirectRef{5, 0}, "5 0 R"},
		{"with generation", IndirectRef{10, 2}, "10 2 R"},
		{"large number", IndirectRef{999999, 0}, "999999 0 R"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.ref.Type() != ObjIndirect {
				t.Errorf("IndirectRef.Type() = %v, want %v", tt.ref.Type(), ObjIndirect)
			}
			if tt.ref.String() != tt.wantString {
				t.Errorf("IndirectRef.String() = %v, want %v", tt.ref.String(), tt.wantString)
			}
		})
	}
}

// TestIndirectObject tests the IndirectObject wrapper
func TestIndirectObject(t *testing.T) {
	ref := IndirectRef{Number: 5, Generation: 0}
	obj := Int(42)
	indirect := IndirectObject{
		Ref:    ref,
		Object: obj,
	}

	if indirect.Ref.Number != 5 {
		t.Error("IndirectObject.Ref incorrect")
	}
	if indirect.Object != Int(42) {
		t.Error("IndirectObject.Object incorrect")
	}
}

// TestComplexStructures tests complex nested structures
func TestComplexStructures(t *testing.T) {
	t.Run("nested dicts and arrays", func(t *testing.T) {
		// Create a structure like: <</Kids [1 0 R 2 0 R] /Count 2>>
		kids := Array{
			IndirectRef{1, 0},
			IndirectRef{2, 0},
		}
		dict := Dict{
			"Kids":  kids,
			"Count": Int(2),
		}

		// Test retrieval
		if arr, ok := dict.GetArray("Kids"); ok {
			if arr.Len() != 2 {
				t.Error("Nested array has wrong length")
			}
		} else {
			t.Error("Failed to get Kids array")
		}
	})

	t.Run("deeply nested", func(t *testing.T) {
		// <</Level1 <</Level2 <</Level3 42>>>>>>
		level3 := Dict{"Level3": Int(42)}
		level2 := Dict{"Level2": level3}
		level1 := Dict{"Level1": level2}

		if l2, ok := level1.GetDict("Level1"); ok {
			if l3, ok := l2.GetDict("Level2"); ok {
				if val, ok := l3.GetInt("Level3"); !ok || val != Int(42) {
					t.Error("Deep nesting retrieval failed")
				}
			} else {
				t.Error("Failed to get Level2")
			}
		} else {
			t.Error("Failed to get Level1")
		}
	})
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
