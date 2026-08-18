package workload

import (
	"fmt"

	. "bloodhound-kube/internal/nodes/framework"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type VolumeMount struct {
	Name      string
	MountPath string
	ReadOnly  bool
}

type HostPort struct {
	ContainerPort int
	HostPort      int
	HostIP        string
	Protocol      string
}

type NamedObjectRef struct {
	Name string
}

type EnvFromSource struct {
	SecretRef    *NamedObjectRef
	ConfigMapRef *NamedObjectRef
}

type EnvVarValueRef struct {
	SecretRef    *NamedObjectRef
	ConfigMapRef *NamedObjectRef
	Key          string
}

type EnvVar struct {
	Name      string
	Value     string
	ValueRef  *EnvVarValueRef
	IsLiteral bool
}

type EnvDefinition struct {
	Container       string
	InitContainer   bool
	EnvName         string
	Value           string
	ValueSourceType string
	RefName         string
	RefKey          string
	SourceKind      string
	SourceName      string
	SourcePath      string
}

type ContainerSecurityContext struct {
	RunAsUser              *int64
	RunAsGroup             *int64
	RunAsNonRoot           bool
	SeccompProfile         string
	AppArmorProfile        string
	SeLinuxOptions         map[string]any
	ReadOnlyRootFilesystem bool
	Privileged             bool
}

type Container struct {
	Name                   string
	Image                  string
	Privileged             bool
	RunAsUser              *int64
	RunAsGroup             *int64
	RunAsNonRoot           bool
	ReadOnlyRootFilesystem bool
	SecurityContext        ContainerSecurityContext
	Env                    []EnvVar
	EnvFrom                []EnvFromSource
	HostPorts              []HostPort
	VolumeMounts           []VolumeMount
}

type VolumeDetail struct {
	Name          string
	Type          string
	SecretName    string
	ConfigMapName string
	PVCName       string
	HostPath      string
}

type Pod struct {
	GraphNodeBase
	NodeName         string
	ServiceAccount   string
	AutomountSAToken *bool
	ShareProcNs      *bool
	Containers       []Container
	InitContainers   []Container
	Volumes          []VolumeDetail
	CapabilitiesAdd  []string
	CapabilitiesDrop []string
	SeLinuxOptions   map[string]any
	HostPID          bool
	HostIPC          bool
	HostNetwork      bool
	EnvDefinitions   []EnvDefinition
}

func BuildPodNode(obj runtime.Object) (BuildResult, bool) {
	pod, ok := obj.(*corev1.Pod)
	if !ok || pod == nil {
		return BuildResult{}, false
	}
	name := pod.Name
	if name == "" {
		return BuildResult{}, false
	}

	namespace := pod.Namespace
	labelsMap := StringMapToAnyMap(pod.Labels)
	annotationsMap := StringMapToAnyMap(pod.Annotations)

	var podSec *corev1.PodSecurityContext
	if pod.Spec.SecurityContext != nil {
		podSec = pod.Spec.SecurityContext
	}
	seccompProfile := ""
	seLinuxRaw := map[string]any{}
	if podSec != nil {
		if podSec.SeccompProfile != nil {
			seccompProfile = string(podSec.SeccompProfile.Type)
		}
		seLinuxRaw = SeLinuxOptionsToMap(podSec.SELinuxOptions)
	}

	capAdd, capDrop := extractCapabilitiesFromContainers(pod.Spec.Containers)
	containerImages := extractContainerImages(pod.Spec.Containers)
	initContainerImages := extractContainerImages(pod.Spec.InitContainers)

	privateContainers := extractContainersDetail(pod.Spec.Containers, podSec)
	privateInitContainers := extractInitContainersDetail(pod.Spec.InitContainers)
	privateVolumes := extractVolumesDetail(pod.Spec.Volumes)
	podEnvDefinitions := buildEnvDefinitionsFromContainers(pod.Spec.Containers, false, "Pod", name)
	podEnvDefinitions = append(podEnvDefinitions, buildEnvDefinitionsFromContainers(pod.Spec.InitContainers, true, "Pod", name)...)

	containerSummaries := summarizeContainers(privateContainers, false)
	initContainerSummaries := summarizeContainers(privateInitContainers, true)
	volumeSummaries := summarizeVolumes(privateVolumes)

	runAsUser := inferPodRunAsUser(pod.Spec.Containers, podSec)
	runAsGroup := 0
	fsGroup := 0
	var supplementalGroups []int64
	runAsNonRoot := inferPodRunAsNonRoot(pod.Spec.Containers, podSec)
	if podSec != nil {
		if podSec.RunAsGroup != nil {
			runAsGroup = int(*podSec.RunAsGroup)
		}
		if podSec.FSGroup != nil {
			fsGroup = int(*podSec.FSGroup)
		}
		supplementalGroups = podSec.SupplementalGroups
	}
	runAsUserStr := fmt.Sprintf("%v", runAsUser) // handles both int and "unset"
	runAsGroupStr := fmt.Sprintf("%d", runAsGroup)
	fsGroupStr := fmt.Sprintf("%d", fsGroup)
	supplementalGroupStrs := make([]string, len(supplementalGroups))
	for i, g := range supplementalGroups {
		supplementalGroupStrs[i] = fmt.Sprintf("%d", g)
	}

	appArmorPod := ""
	if podSec != nil {
		appArmorPod = AppArmorProfileValue(podSec.AppArmorProfile)
	}

	shareProcessNamespace := false
	if pod.Spec.ShareProcessNamespace != nil {
		shareProcessNamespace = *pod.Spec.ShareProcessNamespace
	}
	properties := map[string]any{
		"name":                      name,
		"namespace":                 namespace,
		"labels":                    MapToSortedList(labelsMap),
		"annotations":               MapToSortedList(annotationsMap),
		"securityContextConstraint": annotationsMap["openshift.io/scc"],
		"nodeName":                  pod.Spec.NodeName,
		"serviceAccount":            pod.Spec.ServiceAccountName,
		"containers":                containerSummaries,
		"initContainers":            initContainerSummaries,
		"containerImages":           containerImages,
		"initContainerImages":       initContainerImages,
		"capabilitiesAdd":           capAdd,
		"capabilitiesDrop":          capDrop,
		"hostNetwork":               pod.Spec.HostNetwork,
		"hostPid":                   pod.Spec.HostPID,
		"hostIpc":                   pod.Spec.HostIPC,
		"shareProcessNamespace":     shareProcessNamespace,
		"runAsUser":                 runAsUserStr,
		"runAsGroup":                runAsGroupStr,
		"runAsNonRoot":              runAsNonRoot,
		"fsGroup":                   fsGroupStr,
		"supplementalGroups":        supplementalGroupStrs,
		"seccompProfile":            seccompProfile,
		"appArmorProfile":           appArmorPod,
		"seLinuxOptions":            SeLinuxSummary(seLinuxRaw),
		"volumes":                   volumeSummaries,
	}

	hostPID := pod.Spec.HostPID
	hostIPC := pod.Spec.HostIPC
	hostNetwork := pod.Spec.HostNetwork
	automountSAToken := pod.Spec.AutomountServiceAccountToken
	shareProcNs := pod.Spec.ShareProcessNamespace
	core := CoreEntry{
		Namespace: namespace,
		Cluster:   false,
		Data: Pod{
			GraphNodeBase: GraphNodeBase{
				ID:             BuildID("BHK_Pod", namespace, name),
				Kinds:          []string{"BHK_Pod"},
				Name:           name,
				Namespace:      namespace,
				LabelsMap:      labelsMap,
				AnnotationsMap: annotationsMap,
			},
			NodeName:         pod.Spec.NodeName,
			ServiceAccount:   pod.Spec.ServiceAccountName,
			AutomountSAToken: automountSAToken,
			ShareProcNs:      shareProcNs,
			Containers:       privateContainers,
			InitContainers:   privateInitContainers,
			Volumes:          privateVolumes,
			CapabilitiesAdd:  capAdd,
			CapabilitiesDrop: capDrop,
			SeLinuxOptions:   seLinuxRaw,
			HostPID:          hostPID,
			HostIPC:          hostIPC,
			HostNetwork:      hostNetwork,
			EnvDefinitions:   podEnvDefinitions,
		},
	}

	return BuildResult{
		Node: NodeResult{
			ID:         BuildID("BHK_Pod", namespace, name),
			Kinds:      []string{"BHK_Pod"},
			Properties: properties,
		},
		Core: []CoreEntry{core},
	}, true
}
