package nodes

type EdgeNode interface {
	EdgeID() string
	EdgeKind() string
	EdgeName() string
	EdgeNamespace() string
}

type CoreNode struct {
	ID             string
	Kinds          []string
	Name           string
	Namespace      string
	LabelsMap      map[string]any
	AnnotationsMap map[string]any
}

func (n CoreNode) EdgeID() string {
	return n.ID
}

func (n CoreNode) EdgeKind() string {
	if len(n.Kinds) == 0 {
		return ""
	}
	return n.Kinds[0]
}

func (n CoreNode) EdgeName() string {
	return n.Name
}

func (n CoreNode) EdgeNamespace() string {
	return n.Namespace
}

type PodCore struct {
	CoreNode
	NodeName         string
	ServiceAccount   string
	Containers       []map[string]any
	InitContainers   []map[string]any
	Volumes          []map[string]any
	CapabilitiesAdd  []string
	CapabilitiesDrop []string
	SeLinuxOptions   map[string]any
	HostPID          bool
}

type ServiceAccountCore struct {
	CoreNode
	Secrets []string
}

type SecretCore struct {
	CoreNode
	SecretType string
	Data       map[string]any
}

type ConfigMapCore struct {
	CoreNode
	Data map[string]any
}

type ServiceCore struct {
	CoreNode
	SelectorMap map[string]any
	Ports       []any
	ServiceType string
}

type DeploymentCore struct {
	CoreNode
	SelectorMap       map[string]any
	PodTemplateLabels map[string]any
	ServiceAccount    string
}

type DaemonSetCore struct {
	CoreNode
	SelectorMap map[string]any
}

type StatefulSetCore struct {
	CoreNode
	SelectorMap map[string]any
}

type NetworkPolicyCore struct {
	CoreNode
	PodSelector map[string]any
}

type IngressCore struct {
	CoreNode
	BackendServices []string
	TLS             []any
}

type HTTPRouteCore struct {
	CoreNode
	BackendRefKeys []string
}

type PersistentVolumeCore struct {
	CoreNode
	ClaimRef map[string]any
}

type PersistentVolumeClaimCore struct {
	CoreNode
}

type NodeCore struct {
	CoreNode
}

type RoleCore struct {
	CoreNode
	Perms []string
}

type ClusterRoleCore struct {
	CoreNode
	Perms []string
}

type SubjectCore struct {
	Kind      string
	Name      string
	Namespace string
}

type RoleBindingCore struct {
	CoreNode
	RoleName string
	RoleKind string
	Subjects []SubjectCore
}

type ClusterRoleBindingCore struct {
	CoreNode
	RoleName string
	RoleKind string
	Subjects []SubjectCore
}

type SecretStoreCore struct {
	CoreNode
	ProviderType string
}

type ClusterSecretStoreCore struct {
	CoreNode
	ProviderType string
}

type ExternalSecretCore struct {
	CoreNode
	StoreName     string
	StoreKind     string
	TargetName    string
	DataKeys      []string
	DataFromTypes []string
}

type SecurityContextConstraintsCore struct {
	CoreNode
}

type ExternalCore struct {
	CoreNode
}
