package workload

import (
	. "bloodhound-kube/internal/nodes/framework"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type VolumeMount struct {
	Name      string
	MountPath string
	ReadOnly  bool
	Raw       map[string]any
}

type HostPort struct {
	ContainerPort int
	HostPort      int
	HostIP        string
	Protocol      string
	Raw           map[string]any
}

type NamedObjectRef struct {
	Name string
}

type EnvFromSource struct {
	SecretRef    *NamedObjectRef
	ConfigMapRef *NamedObjectRef
	Raw          map[string]any
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
	Raw                    map[string]any
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
	EnvFrom                []EnvFromSource
	HostPorts              []HostPort
	VolumeMounts           []VolumeMount
	Raw                    map[string]any
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
		"runAsUser":                 runAsUser,
		"runAsGroup":                runAsGroup,
		"runAsNonRoot":              runAsNonRoot,
		"fsGroup":                   fsGroup,
		"supplementalGroups":        supplementalGroups,
		"seccompProfile":            seccompProfile,
		"appArmorProfile":           appArmorPod,
		"seLinuxOptions":            SeLinuxSummary(seLinuxRaw),
		"volumes":                   volumeSummaries,
	}

	hostPID := pod.Spec.HostPID
	automountSAToken := pod.Spec.AutomountServiceAccountToken
	shareProcNs := pod.Spec.ShareProcessNamespace
	core := CoreEntry{
		Namespace: namespace,
		Cluster:   false,
		Data: Pod{
			GraphNodeBase: GraphNodeBase{
				ID:             BuildID("Pod", namespace, name),
				Kinds:          []string{"Pod"},
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
		},
	}

	return BuildResult{
		Node: NodeResult{
			ID:         BuildID("Pod", namespace, name),
			Kinds:      []string{"Pod"},
			Properties: properties,
		},
		Core: []CoreEntry{core},
	}, true
}
