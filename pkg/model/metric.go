package model

import (
	"fmt"
	"hash/fnv"
)

type Metric struct {
	Name   string
	Labels Labels
}

// 返回类似 "cpu{host=A,region=us}" 的字符串
func (m *Metric) String() string {
	return fmt.Sprintf("%s{%s}", m.Name, m.Labels.String())
}

// FIXME 重复调用的话, 会重复计算和排序
func (m *Metric) Fingerprint() uint64 {
	h := fnv.New64a()
	h.Write([]byte(m.String()))
	return h.Sum64()
}
