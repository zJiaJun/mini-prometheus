package scrape

import (
	"context"
	"io"
	"mini-promethues/pkg/config"
	"net/http"
	"sync"
	"time"
)

type Scraper struct {
	configMap  map[string]config.ScrapeConfig
	httpClient *http.Client
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	parser     *Parser
}

func NewScraper(config *config.Config) *Scraper {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scraper{
		configMap:  config.Process(),
		httpClient: &http.Client{},
		ctx:        ctx,
		cancel:     cancel,
		parser:     NewParser(ctx),
	}
}

func (s *Scraper) Start() error {
	for _, v := range s.configMap {
		s.wg.Add(1)
		go s.runJob(v)
	}
	s.parser.start()
	return nil
}

func (s *Scraper) Stop() error {
	s.cancel()
	s.wg.Wait()
	s.parser.stop()
	return nil
}

func (s *Scraper) runJob(sc config.ScrapeConfig) {
	defer s.wg.Done()
	for _, stc := range sc.StaticConfigs {
		for _, target := range stc.Targets {
			s.wg.Add(1)
			go s.runTarget(sc.JobName, target, sc.ScrapeInterval, sc.ScrapeTimeout, stc.Labels)
		}
	}
}

func (s *Scraper) runTarget(jobName, target string,
	interval time.Duration, timeout time.Duration, labels map[string]string) {

	defer s.wg.Done()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.scrape(target, jobName, labels, timeout)
		case <-s.ctx.Done():
			return
		}
	}
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
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	body := NewBody(jobName, targetUrl, data, labels)
	if err = s.parser.produce(body); err != nil {
		return
	}
}
