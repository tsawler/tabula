package filters

import (
	"testing"
)

func TestGetBoolParam(t *testing.T) {
	tests := []struct {
		name         string
		params       Params
		key          string
		defaultValue bool
		want         bool
	}{
		{
			name:         "nil params",
			params:       nil,
			key:          "BlackIs1",
			defaultValue: false,
			want:         false,
		},
		{
			name:         "missing key",
			params:       Params{"Columns": 1728},
			key:          "BlackIs1",
			defaultValue: false,
			want:         false,
		},
		{
			name:         "true value",
			params:       Params{"BlackIs1": true},
			key:          "BlackIs1",
			defaultValue: false,
			want:         true,
		},
		{
			name:         "false value",
			params:       Params{"BlackIs1": false},
			key:          "BlackIs1",
			defaultValue: true,
			want:         false,
		},
		{
			name:         "invalid type returns default",
			params:       Params{"BlackIs1": "true"},
			key:          "BlackIs1",
			defaultValue: false,
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getBoolParam(tt.params, tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("getBoolParam() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCCITTFaxDecodeParams(t *testing.T) {
	// Test that parameter extraction works correctly
	// Note: We can't easily test actual CCITT decoding without sample data,
	// but we can verify the parameter handling logic

	// Test with default params
	params := Params{
		"K":        -1, // Group 4
		"Columns":  100,
		"Rows":     50,
		"BlackIs1": true,
	}

	// Verify parameter extraction helpers work
	if getIntParam(params, "K", 0) != -1 {
		t.Error("K should be -1")
	}
	if getIntParam(params, "Columns", 1728) != 100 {
		t.Error("Columns should be 100")
	}
	if getIntParam(params, "Rows", 0) != 50 {
		t.Error("Rows should be 50")
	}
	if getBoolParam(params, "BlackIs1", false) != true {
		t.Error("BlackIs1 should be true")
	}
}
