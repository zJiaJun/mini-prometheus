package scrape

type Body struct {
	jobName   string
	targetUrl string
	data      []byte
	labels    map[string]string
}

func NewBody(jobName string, targetUrl string, data []byte, labels map[string]string) *Body {
	return &Body{
		jobName:   jobName,
		targetUrl: targetUrl,
		data:      data,
		labels:    labels,
	}
}
