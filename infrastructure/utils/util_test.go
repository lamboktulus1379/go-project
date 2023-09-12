package utils

import (
	"reflect"
	"testing"
	"time"
)

func TestGetCurrentTime(t *testing.T) {
	tests := []struct {
		name string
		want time.Time
	}{
		{
			name: "TestGetCurrentTime - 1",
			want: time.Now().UTC(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetCurrentTime(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetCurrentTime() = %v, want %v", got, tt.want)
			}
		})
	}
}
