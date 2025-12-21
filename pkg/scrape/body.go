package scrape

type Body struct {
	JobName   string
	TargetUrl string
	Data      []byte
	Labels    map[string]string
}

func NewBody(jobName string, targetUrl string, data []byte, labels map[string]string) *Body {
	return &Body{
		JobName:   jobName,
		TargetUrl: targetUrl,
		Data:      data,
		Labels:    labels,
	}
}
