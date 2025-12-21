package scrape

import (
	"context"
	"sync"
)

type Parser struct {
	ch     chan *Body
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewParser(ctx context.Context) *Parser {
	ctx, cancel := context.WithCancel(ctx)
	return &Parser{
		ch:     make(chan *Body, 10000),
		ctx:    ctx,
		cancel: cancel,
	}
}

func (p *Parser) start() {
	p.wg.Add(1)
	go p.consume()
}

func (p *Parser) produce(body *Body) error {
	if body == nil {
		return nil
	}
	select {
	case p.ch <- body:
		return nil
	case <-p.ctx.Done():
		return p.ctx.Err()
	}
}

func (p *Parser) consume() {
	defer p.wg.Done()
	for {
		select {
		case body := <-p.ch:
			p.parser(body)
		case <-p.ctx.Done():
			return
		}
	}
}

func (p *Parser) stop() {
	p.cancel()
	close(p.ch)
	p.wg.Wait()
	return
}

func (p *Parser) parser(body *Body) {

}
