package framework

import "bloodhound-kube/internal/model"

type Rule interface {
	Name() string
	Apply(ctx *Context) []model.BloodHoundEdge
}

type Registry struct {
	rules []Rule
}

func NewRegistry() *Registry {
	return &Registry{rules: make([]Rule, 0, 32)}
}

func (r *Registry) Register(rule Rule) {
	if r == nil || rule == nil {
		return
	}
	r.rules = append(r.rules, rule)
}

func (r *Registry) Rules() []Rule {
	if r == nil {
		return nil
	}
	return r.rules
}
