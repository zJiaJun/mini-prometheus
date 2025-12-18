package scrape

import (
	"context"
	"io"
	"mini-promethues/pkg/config"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"
)

type Scraper struct {
	config     *config.Config
	httpClient *http.Client
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

func NewScraper(config *config.Config) *Scraper {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scraper{
		config:     config,
		httpClient: &http.Client{},
		ctx:        ctx,
		cancel:     cancel,
	}
}

func (s *Scraper) Start() error {
	for _, sc := range s.config.ScrapeConfigs {
		s.wg.Add(1)
		go s.runJob(sc)
	}
	return nil
}

func (s *Scraper) Stop() error {
	s.cancel()
	s.wg.Wait()
	return nil
}

func (s *Scraper) runJob(sc config.ScrapeConfig) {
	defer s.wg.Done()
	interval := sc.ScrapeInterval
	if interval == 0 {
		interval = s.config.Global.ScrapeInterval
	}
	timeout := sc.ScrapeTimeout
	if timeout == 0 {
		timeout = s.config.Global.ScrapeTimeout
	}
	for _, stc := range sc.StaticConfigs {
		for _, target := range stc.Targets {
			s.wg.Add(1)
			go s.runTarget(sc.JobName, target, sc.MetricsPath,
				interval, timeout, stc.Labels)
		}
	}
}

func (s *Scraper) runTarget(jobName, target, metricsPath string,
	interval time.Duration, timeout time.Duration, labels map[string]string) {

	defer s.wg.Done()
	targetUrl, err := s.buildTargetUrl(target, metricsPath)
	if err != nil {
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.scrape(targetUrl, jobName, labels, timeout)
		case <-s.ctx.Done():
			return
		}
	}
}
func (s *Scraper) buildTargetUrl(target, metricPath string) (string, error) {
	if !strings.HasPrefix(target, "http://") || !strings.HasPrefix(target, "https://") {
		target = "http://" + target
	}
	url, err := url.Parse(target)
	if err != nil {
		return "", err
	}
	url.Path = path.Join(url.Path, metricPath)
	return url.String(), nil
}

func (s *Scraper) scrape(targetUrl, jobName string,
	labels map[string]string, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(s.ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", targetUrl, nil)
	if err != nil {
		return
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return
	}
	_, err = io.ReadAll(resp.Body)
	if err != nil {
		return
	}

}
