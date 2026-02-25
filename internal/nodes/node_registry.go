package nodes

type NodeResult struct {
	ID         string
	Kinds      []string
	Properties map[string]any
}

type CoreEntry struct {
	Namespace string
	Cluster   bool
	Data      any
}

type BuildResult struct {
	Node NodeResult
	Core []CoreEntry
}

type Builder func(resource map[string]any) (BuildResult, bool)

var builders = map[string]Builder{}

func Register(kind string, builder Builder) {
	builders[kind] = builder
}

func Build(resource map[string]any) (BuildResult, bool) {
	kind, _ := resource["kind"].(string)
	if builder, ok := builders[kind]; ok {
		return builder(resource)
	}
	return BuildResult{}, false
}
