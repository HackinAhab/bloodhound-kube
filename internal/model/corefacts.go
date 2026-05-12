package model

import (
	"bloodhound-kube/internal/nodes"
	"reflect"
)

type CoreFacts struct {
	Namespaces map[string]*Namespace
	Cluster    *Cluster
}

func NewCoreFacts() *CoreFacts {
	return &CoreFacts{
		Namespaces: map[string]*Namespace{},
		Cluster:    &Cluster{},
	}
}

func (c *CoreFacts) Add(entry nodes.CoreEntry) {
	if entry.Data == nil {
		return
	}
	if entry.Cluster {
		c.addClusterFact(entry.Data)
		return
	}

	ns := entry.Namespace
	if c.Namespaces[ns] == nil {
		c.Namespaces[ns] = &Namespace{}
	}
	c.addNamespacedFact(c.Namespaces[ns], entry.Data)
}

func (c *CoreFacts) addClusterFact(data any) {
	if add, ok := clusterFactAdders[reflect.TypeOf(data)]; ok {
		add(c, data)
	}
}

func (c *CoreFacts) addNamespacedFact(space *Namespace, data any) {
	if add, ok := namespacedFactAdders[reflect.TypeOf(data)]; ok {
		add(space, data)
	}
}
