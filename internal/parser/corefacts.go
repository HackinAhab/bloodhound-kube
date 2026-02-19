package parser

import "bloodhound-kube/internal/parser/nodes"

type CoreFacts struct {
	Namespaces map[string]map[string][]any
	Cluster    map[string][]any
}

func NewCoreFacts() *CoreFacts {
	return &CoreFacts{
		Namespaces: map[string]map[string][]any{},
		Cluster:    map[string][]any{},
	}
}

func (c *CoreFacts) Add(entry nodes.CoreEntry) {
	if entry.Key == "" || entry.Data == nil {
		return
	}
	if entry.Cluster {
		c.Cluster[entry.Key] = append(c.Cluster[entry.Key], entry.Data)
		return
	}
	ns := entry.Namespace
	if c.Namespaces[ns] == nil {
		c.Namespaces[ns] = map[string][]any{}
	}
	c.Namespaces[ns][entry.Key] = append(c.Namespaces[ns][entry.Key], entry.Data)
}
