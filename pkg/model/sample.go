package model

type Sample struct {
	TimeStamp int64
	Value     float64
}

type Samples []Sample

func (s Samples) Append(sample Sample) Samples {
	return append(s, sample)
}
