package model

import "testing"

func TestMetric_String(t *testing.T) {
	type fields struct {
		Name   string
		Labels Labels
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "test.basic.metric",
			fields: fields{
				Name: "cpu_total",
				Labels: Labels{
					{Name: "host", Value: "A"},
					{Name: "region", Value: "us"},
				},
			},
			want: "cpu_total{host=A,region=us}",
		},
		{
			name: "test.empty.metric",
			fields: fields{
				Name:   "cpu_total",
				Labels: Labels{},
			},
			want: "cpu_total{}",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Metric{
				Name:   tt.fields.Name,
				Labels: tt.fields.Labels,
			}
			if got := m.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMetric_Fingerprint(t *testing.T) {
	m1 := &Metric{Name: "cpu_total", Labels: Labels{{Name: "host", Value: "A"}}}
	m2 := &Metric{Name: "cpu_total", Labels: Labels{{Name: "host", Value: "A"}}}
	if m1.Fingerprint() != m2.Fingerprint() {
		t.Errorf("Fingerprint not match %v != %v", m1.Fingerprint(), m2.Fingerprint())
	}

	m3 := &Metric{Name: "cpu_total", Labels: Labels{
		{Name: "region", Value: "us"}, {Name: "host", Value: "A"}},
	}
	m4 := &Metric{Name: "cpu_total", Labels: Labels{
		{Name: "host", Value: "A"}, {Name: "region", Value: "us"}},
	}
	if m3.Fingerprint() != m4.Fingerprint() {
		t.Errorf("Fingerprint not match %v != %v", m3.Fingerprint(), m4.Fingerprint())
	}
	m5 := &Metric{Name: "cpu_total", Labels: Labels{
		{Name: "region", Value: "tw"}, {Name: "host", Value: "A"}},
	}
	m6 := &Metric{Name: "cpu_total", Labels: Labels{
		{Name: "region", Value: "en"}, {Name: "host", Value: "A"}},
	}
	if m5.Fingerprint() == m6.Fingerprint() {
		t.Errorf("Fingerprint not match %v != %v", m5.Fingerprint(), m6.Fingerprint())
	}

}
