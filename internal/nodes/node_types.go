package nodes

type EdgeNode interface {
	EdgeID() string
	EdgeKind() string
	EdgeName() string
	EdgeNamespace() string
}

type GraphNodeBase struct {
	ID             string
	Kinds          []string
	Name           string
	Namespace      string
	LabelsMap      map[string]any
	AnnotationsMap map[string]any
}

func (n GraphNodeBase) EdgeID() string {
	return n.ID
}

func (n GraphNodeBase) EdgeKind() string {
	if len(n.Kinds) == 0 {
		return ""
	}
	return n.Kinds[0]
}

func (n GraphNodeBase) EdgeName() string {
	return n.Name
}

func (n GraphNodeBase) EdgeNamespace() string {
	return n.Namespace
}
