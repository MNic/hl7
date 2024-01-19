package hl7

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLocation(t *testing.T) {
	tests := []struct {
		tag          string
		wantSeg      string
		wantFieldSeq int
		wantComp     int
		wantSubComp  int
	}{
		{"PID.5.1", "PID", 5, 1, 0},
		{"MSH.0", "MSH", 0, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			loc := NewLocation(tt.tag)
			assert.Equal(t, tt.wantSeg, loc.Segment)
			assert.Equal(t, tt.wantFieldSeq, loc.FieldSeq)
			assert.Equal(t, tt.wantComp, loc.Comp)
			assert.Equal(t, tt.wantSubComp, loc.SubComp)
		})
	}
}
