package mounts

import (
	"testing"

	"bloodhound-kube/internal/nodes/workload"
)

// ---------------------------------------------------------------------------
// podHostMountReadCheck
// ---------------------------------------------------------------------------

func TestPodHostMountReadCheck(t *testing.T) {
	cases := []struct {
		name      string
		pod       *workload.Pod
		wantPath  string
		wantOK    bool
	}{
		{name: "nil pod", pod: nil, wantPath: "", wantOK: false},
		{name: "no volumes", pod: &workload.Pod{}, wantPath: "", wantOK: false},
		{
			name: "matches /etc exact",
			pod: &workload.Pod{
				Volumes: []workload.VolumeDetail{{Name: "etc", HostPath: "/etc"}},
				Containers: []workload.Container{{
					VolumeMounts: []workload.VolumeMount{{Name: "etc", MountPath: "/host-etc"}},
				}},
			},
			wantPath: "/host-etc", wantOK: true,
		},
		{
			name: "matches /etc/passwd via prefix",
			pod: &workload.Pod{
				Volumes: []workload.VolumeDetail{{Name: "etc", HostPath: "/etc/kubernetes"}},
				Containers: []workload.Container{{
					VolumeMounts: []workload.VolumeMount{{Name: "etc", MountPath: "/host-etc"}},
				}},
			},
			wantPath: "/host-etc", wantOK: true,
		},
		{
			name: "matches /var/lib/kubelet/pods",
			pod: &workload.Pod{
				Volumes: []workload.VolumeDetail{{Name: "kp", HostPath: "/var/lib/kubelet/pods"}},
				Containers: []workload.Container{{
					VolumeMounts: []workload.VolumeMount{{Name: "kp", MountPath: "/h"}},
				}},
			},
			wantPath: "/h", wantOK: true,
		},
		{
			name: "non-sensitive path",
			pod: &workload.Pod{
				Volumes: []workload.VolumeDetail{{Name: "v", HostPath: "/var/lib/foo"}},
				Containers: []workload.Container{{
					VolumeMounts: []workload.VolumeMount{{Name: "v", MountPath: "/h"}},
				}},
			},
			wantPath: "", wantOK: false,
		},
		{
			name: "false-prefix sibling /etcd does not match /etc",
			pod: &workload.Pod{
				Volumes: []workload.VolumeDetail{{Name: "v", HostPath: "/etcd"}},
				Containers: []workload.Container{{
					VolumeMounts: []workload.VolumeMount{{Name: "v", MountPath: "/h"}},
				}},
			},
			wantPath: "", wantOK: false,
		},
		{
			name: "matching volume but no container mount references it",
			pod: &workload.Pod{
				Volumes: []workload.VolumeDetail{{Name: "etc", HostPath: "/etc"}},
				Containers: []workload.Container{{
					VolumeMounts: []workload.VolumeMount{{Name: "other", MountPath: "/h"}},
				}},
			},
			wantPath: "", wantOK: false,
		},
		{
			name: "matching volume but mount has empty MountPath",
			pod: &workload.Pod{
				Volumes: []workload.VolumeDetail{{Name: "etc", HostPath: "/etc"}},
				Containers: []workload.Container{{
					VolumeMounts: []workload.VolumeMount{{Name: "etc", MountPath: ""}},
				}},
			},
			wantPath: "", wantOK: false,
		},
		{
			name: "volume with empty Name is ignored",
			pod: &workload.Pod{
				Volumes: []workload.VolumeDetail{{Name: "", HostPath: "/etc"}},
				Containers: []workload.Container{{
					VolumeMounts: []workload.VolumeMount{{Name: "etc", MountPath: "/h"}},
				}},
			},
			wantPath: "", wantOK: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotPath, gotOK := podHostMountReadCheck(tc.pod)
			if gotPath != tc.wantPath || gotOK != tc.wantOK {
				t.Fatalf("got (%q,%v) want (%q,%v)", gotPath, gotOK, tc.wantPath, tc.wantOK)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// podHostMountKubeletCheck
// ---------------------------------------------------------------------------

func TestPodHostMountKubeletCheck(t *testing.T) {
	cases := []struct {
		name     string
		pod      *workload.Pod
		wantPath string
		wantOK   bool
	}{
		{name: "nil pod", pod: nil, wantPath: "", wantOK: false},
		{
			name: "matches /var/lib/kubelet exact",
			pod: &workload.Pod{
				Volumes: []workload.VolumeDetail{{Name: "kub", HostPath: "/var/lib/kubelet"}},
				Containers: []workload.Container{{
					VolumeMounts: []workload.VolumeMount{{Name: "kub", MountPath: "/host"}},
				}},
			},
			wantPath: "/host", wantOK: true,
		},
		{
			name: "matches /etc/kubernetes prefix",
			pod: &workload.Pod{
				Volumes: []workload.VolumeDetail{{Name: "k8s", HostPath: "/etc/kubernetes/manifests"}},
				Containers: []workload.Container{{
					VolumeMounts: []workload.VolumeMount{{Name: "k8s", MountPath: "/host"}},
				}},
			},
			wantPath: "/host", wantOK: true,
		},
		{
			name: "non-kubelet path",
			pod: &workload.Pod{
				Volumes: []workload.VolumeDetail{{Name: "v", HostPath: "/etc"}},
				Containers: []workload.Container{{
					VolumeMounts: []workload.VolumeMount{{Name: "v", MountPath: "/host"}},
				}},
			},
			wantPath: "", wantOK: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotPath, gotOK := podHostMountKubeletCheck(tc.pod)
			if gotPath != tc.wantPath || gotOK != tc.wantOK {
				t.Fatalf("got (%q,%v) want (%q,%v)", gotPath, gotOK, tc.wantPath, tc.wantOK)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// podMountsServiceAccountToken
// ---------------------------------------------------------------------------

func TestPodMountsServiceAccountToken(t *testing.T) {
	bp := func(b bool) *bool { return &b }
	cases := []struct {
		name string
		pod  *workload.Pod
		want bool
	}{
		{name: "nil pod", pod: nil, want: false},
		{
			name: "default SA, automount nil",
			pod:  &workload.Pod{ServiceAccount: "default"},
			want: false,
		},
		{
			name: "default SA, automount=false",
			pod:  &workload.Pod{ServiceAccount: "default", AutomountSAToken: bp(false)},
			want: false,
		},
		{
			name: "default SA, automount=true",
			pod:  &workload.Pod{ServiceAccount: "default", AutomountSAToken: bp(true)},
			want: true,
		},
		{
			name: "non-default SA, automount nil (defaults to mounted)",
			pod:  &workload.Pod{ServiceAccount: "myapp"},
			want: true,
		},
		{
			name: "non-default SA, automount=true",
			pod:  &workload.Pod{ServiceAccount: "myapp", AutomountSAToken: bp(true)},
			want: true,
		},
		{
			name: "non-default SA, automount=false",
			pod:  &workload.Pod{ServiceAccount: "myapp", AutomountSAToken: bp(false)},
			want: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := podMountsServiceAccountToken(tc.pod); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}
