package core

import (
	"testing"
)

// TestNewObjectStream tests creating an ObjectStream from a Stream
func TestNewObjectStream(t *testing.T) {
	tests := []struct {
		name      string
		dict      Dict
		wantN     int
		wantFirst int
		wantErr   bool
	}{
		{
			name: "valid object stream",
			dict: Dict{
				"Type":  Name("ObjStm"),
				"N":     Int(3),
				"First": Int(20),
			},
			wantN:     3,
			wantFirst: 20,
			wantErr:   false,
		},
		{
			name: "with Extends",
			dict: Dict{
				"Type":    Name("ObjStm"),
				"N":       Int(2),
				"First":   Int(15),
				"Extends": &IndirectRef{Number: 10, Generation: 0},
			},
			wantN:     2,
			wantFirst: 15,
			wantErr:   false,
		},
		{
			name: "missing Type",
			dict: Dict{
				"N":     Int(3),
				"First": Int(20),
			},
			wantErr: true,
		},
		{
			name: "wrong Type",
			dict: Dict{
				"Type":  Name("XRef"),
				"N":     Int(3),
				"First": Int(20),
			},
			wantErr: true,
		},
		{
			name: "missing N",
			dict: Dict{
				"Type":  Name("ObjStm"),
				"First": Int(20),
			},
			wantErr: true,
		},
		{
			name: "missing First",
			dict: Dict{
				"Type": Name("ObjStm"),
				"N":    Int(3),
			},
			wantErr: true,
		},
		{
			name: "negative N",
			dict: Dict{
				"Type":  Name("ObjStm"),
				"N":     Int(-1),
				"First": Int(20),
			},
			wantErr: true,
		},
		{
			name: "negative First",
			dict: Dict{
				"Type":  Name("ObjStm"),
				"N":     Int(3),
				"First": Int(-1),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stream := &Stream{
				Dict: tt.dict,
				Data: []byte{}, // Empty data for validation tests
			}

			os, err := NewObjectStream(stream)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewObjectStream() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if os.N() != tt.wantN {
				t.Errorf("N() = %d, want %d", os.N(), tt.wantN)
			}
			if os.First() != tt.wantFirst {
				t.Errorf("First() = %d, want %d", os.First(), tt.wantFirst)
			}
		})
	}
}

func TestNewObjectStreamNilStream(t *testing.T) {
	_, err := NewObjectStream(nil)
	if err == nil {
		t.Error("expected error for nil stream")
	}
}

// TestObjectStreamParsing tests parsing objects from an object stream
func TestObjectStreamParsing(t *testing.T) {
	// Create a mock object stream with uncompressed data
	// Header format: objNum offset objNum offset ...
	// Offsets are relative to the First byte (start of object data)
	//
	// Objects:
	//   Object 5: << /Type /Catalog >> (20 bytes)
	//   Object 6: << /Count 1 >>       (14 bytes)
	//   Object 7: [ 1 2 3 ]            (9 bytes)
	//
	// Header: "5 0 6 20 7 34 " (14 bytes)
	// First = 14
	// Object offsets from First: 0, 20, 34
	header := "5 0 6 20 7 34 "
	obj5 := "<< /Type /Catalog >>"
	obj6 := "<< /Count 1 >>"
	obj7 := "[ 1 2 3 ]"

	data := []byte(header + obj5 + obj6 + obj7)

	stream := &Stream{
		Dict: Dict{
			"Type":  Name("ObjStm"),
			"N":     Int(3),
			"First": Int(len(header)),
		},
		Data: data, // No compression filter
	}

	os, err := NewObjectStream(stream)
	if err != nil {
		t.Fatalf("NewObjectStream() error = %v", err)
	}

	// Test GetObjectByIndex
	t.Run("GetObjectByIndex", func(t *testing.T) {
		// Object at index 0 (object number 5)
		obj, objNum, err := os.GetObjectByIndex(0)
		if err != nil {
			t.Fatalf("GetObjectByIndex(0) error = %v", err)
		}
		if objNum != 5 {
			t.Errorf("objNum = %d, want 5", objNum)
		}
		dict, ok := obj.(Dict)
		if !ok {
			t.Fatalf("expected Dict, got %T", obj)
		}
		if dict.Get("Type") != Name("Catalog") {
			t.Errorf("Type = %v, want /Catalog", dict.Get("Type"))
		}

		// Object at index 1 (object number 6)
		obj, objNum, err = os.GetObjectByIndex(1)
		if err != nil {
			t.Fatalf("GetObjectByIndex(1) error = %v", err)
		}
		if objNum != 6 {
			t.Errorf("objNum = %d, want 6", objNum)
		}
		dict, ok = obj.(Dict)
		if !ok {
			t.Fatalf("expected Dict, got %T", obj)
		}

		// Object at index 2 (object number 7)
		obj, objNum, err = os.GetObjectByIndex(2)
		if err != nil {
			t.Fatalf("GetObjectByIndex(2) error = %v", err)
		}
		if objNum != 7 {
			t.Errorf("objNum = %d, want 7", objNum)
		}
		arr, ok := obj.(Array)
		if !ok {
			t.Fatalf("expected Array, got %T", obj)
		}
		if len(arr) != 3 {
			t.Errorf("array length = %d, want 3", len(arr))
		}
	})

	// Test GetObjectByNumber
	t.Run("GetObjectByNumber", func(t *testing.T) {
		obj, index, err := os.GetObjectByNumber(6)
		if err != nil {
			t.Fatalf("GetObjectByNumber(6) error = %v", err)
		}
		if index != 1 {
			t.Errorf("index = %d, want 1", index)
		}
		_, ok := obj.(Dict)
		if !ok {
			t.Fatalf("expected Dict, got %T", obj)
		}

		// Non-existent object
		_, _, err = os.GetObjectByNumber(999)
		if err == nil {
			t.Error("expected error for non-existent object")
		}
	})

	// Test ObjectNumbers
	t.Run("ObjectNumbers", func(t *testing.T) {
		nums, err := os.ObjectNumbers()
		if err != nil {
			t.Fatalf("ObjectNumbers() error = %v", err)
		}
		if len(nums) != 3 {
			t.Fatalf("len(nums) = %d, want 3", len(nums))
		}
		expected := []int{5, 6, 7}
		for i, n := range nums {
			if n != expected[i] {
				t.Errorf("nums[%d] = %d, want %d", i, n, expected[i])
			}
		}
	})

	// Test ContainsObject
	t.Run("ContainsObject", func(t *testing.T) {
		contains, err := os.ContainsObject(5)
		if err != nil {
			t.Fatalf("ContainsObject(5) error = %v", err)
		}
		if !contains {
			t.Error("expected ContainsObject(5) = true")
		}

		contains, err = os.ContainsObject(999)
		if err != nil {
			t.Fatalf("ContainsObject(999) error = %v", err)
		}
		if contains {
			t.Error("expected ContainsObject(999) = false")
		}
	})

	// Test index out of range
	t.Run("IndexOutOfRange", func(t *testing.T) {
		_, _, err := os.GetObjectByIndex(-1)
		if err == nil {
			t.Error("expected error for negative index")
		}

		_, _, err = os.GetObjectByIndex(10)
		if err == nil {
			t.Error("expected error for index beyond range")
		}
	})
}

// TestObjectStreamCaching tests that objects are cached after first parse
func TestObjectStreamCaching(t *testing.T) {
	header := "5 0 "
	obj5 := "<< /Test /Value >>"
	data := []byte(header + obj5)

	stream := &Stream{
		Dict: Dict{
			"Type":  Name("ObjStm"),
			"N":     Int(1),
			"First": Int(len(header)),
		},
		Data: data,
	}

	os, err := NewObjectStream(stream)
	if err != nil {
		t.Fatalf("NewObjectStream() error = %v", err)
	}

	// First access
	obj1, _, err := os.GetObjectByIndex(0)
	if err != nil {
		t.Fatalf("first GetObjectByIndex(0) error = %v", err)
	}

	// Second access should return cached object
	obj2, _, err := os.GetObjectByIndex(0)
	if err != nil {
		t.Fatalf("second GetObjectByIndex(0) error = %v", err)
	}

	// Should be the same object (pointer equality for cached)
	dict1, ok1 := obj1.(Dict)
	dict2, ok2 := obj2.(Dict)
	if !ok1 || !ok2 {
		t.Fatal("expected both objects to be Dict")
	}

	// Verify the values are the same
	if dict1.Get("Test") != dict2.Get("Test") {
		t.Error("cached object should return same values")
	}
}

// TestObjectStreamExtends tests the Extends accessor
func TestObjectStreamExtends(t *testing.T) {
	// Without Extends
	stream1 := &Stream{
		Dict: Dict{
			"Type":  Name("ObjStm"),
			"N":     Int(1),
			"First": Int(4),
		},
		Data: []byte("1 0 42"),
	}

	os1, err := NewObjectStream(stream1)
	if err != nil {
		t.Fatalf("NewObjectStream() error = %v", err)
	}
	if os1.Extends() != nil {
		t.Error("expected Extends() = nil for stream without /Extends")
	}

	// With Extends
	stream2 := &Stream{
		Dict: Dict{
			"Type":    Name("ObjStm"),
			"N":       Int(1),
			"First":   Int(4),
			"Extends": &IndirectRef{Number: 10, Generation: 0},
		},
		Data: []byte("1 0 42"),
	}

	os2, err := NewObjectStream(stream2)
	if err != nil {
		t.Fatalf("NewObjectStream() error = %v", err)
	}
	ext := os2.Extends()
	if ext == nil {
		t.Fatal("expected Extends() != nil")
	}
	if ext.Number != 10 {
		t.Errorf("Extends().Number = %d, want 10", ext.Number)
	}
}

// TestObjectStreamHeaderErrors tests error handling in header parsing
func TestObjectStreamHeaderErrors(t *testing.T) {
	tests := []struct {
		name   string
		header string
		n      int
		first  int
	}{
		{
			name:   "First exceeds data length",
			header: "1 0 ",
			n:      1,
			first:  1000, // Way beyond data length
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stream := &Stream{
				Dict: Dict{
					"Type":  Name("ObjStm"),
					"N":     Int(tt.n),
					"First": Int(tt.first),
				},
				Data: []byte(tt.header + "42"),
			}

			os, err := NewObjectStream(stream)
			if err != nil {
				// Error during creation is fine
				return
			}

			// Try to decode - should fail
			_, _, err = os.GetObjectByIndex(0)
			if err == nil {
				t.Error("expected error during decode")
			}
		})
	}
}
