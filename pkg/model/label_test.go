package model

import "testing"

func TestLabels_String(t *testing.T) {
	tests := []struct {
		name string
		l    Labels
		want string
	}{
		{
			"test.basic.labels",
			Labels{
				{Name: "host", Value: "A"},
				{Name: "region", Value: "us"},
			},
			"host=A,region=us",
		},
		{
			"test.single.labels",
			Labels{
				{Name: "host", Value: "A"},
			},
			"host=A",
		},
		{
			"test.empty.labels",
			Labels{},
			"",
		},
		{
			"test.sample.labels",
			Labels{
				{Name: "host", Value: "X"},
				{Name: "host", Value: "A"},
				{Name: "region", Value: "us"},
			},
			"host=A,host=X,region=us",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.l.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}
