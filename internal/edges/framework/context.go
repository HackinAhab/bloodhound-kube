package framework

import (
	"bloodhound-kube/internal/model"
)

type Context struct {
	Core  *model.CoreFacts
	Index model.EdgeIndex
}

func NewContext(core *model.CoreFacts) *Context {
	ctx := &Context{Core: core}
	ctx.Index = model.NewEdgeIndex(core)
	return ctx
}
