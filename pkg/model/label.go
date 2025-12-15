package model

import (
	"sort"
	"strings"
)

type Label struct {
	Name  string
	Value string
}
type Labels []Label

func (l Labels) Sorted() Labels {
	sorted := make(Labels, len(l))
	copy(sorted, l)
	sort.Sort(sorted)
	return sorted
}

// 返回类似 "host=A,region=us" 的字符串
func (l Labels) String() string {
	sorted := l.Sorted()
	var b strings.Builder
	for i, label := range sorted {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(label.Name)
		b.WriteString("=")
		b.WriteString(label.Value)
	}
	return b.String()
}

// 实现 sort.Interface 接口

func (l Labels) Len() int {
	return len(l)
}

func (l Labels) Less(i, j int) bool {
	if l[i].Name != l[j].Name {
		return l[i].Name < l[j].Name
	}
	return l[i].Value < l[j].Value
}

func (l Labels) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}
