package security

import (
	"testing"

	"bloodhound-kube/internal/nodes/workload"
)

// ---------------------------------------------------------------------------
// podHasPrivilegedContainer
// ---------------------------------------------------------------------------

func TestPodHasPrivilegedContainer(t *testing.T) {
	cases := []struct {
		name string
		pod  *workload.Pod
		want bool
	}{
		{name: "nil pod", pod: nil, want: false},
		{name: "no containers", pod: &workload.Pod{}, want: false},
		{
			name: "single non-priv",
			pod:  &workload.Pod{Containers: []workload.Container{{Privileged: false}}},
			want: false,
		},
		{
			name: "single priv",
			pod:  &workload.Pod{Containers: []workload.Container{{Privileged: true}}},
			want: true,
		},
		{
			name: "any priv among many",
			pod: &workload.Pod{Containers: []workload.Container{
				{Privileged: false}, {Privileged: true}, {Privileged: false},
			}},
			want: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := podHasPrivilegedContainer(tc.pod); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ceNsEnterCheck (priv && hostPID)
// ---------------------------------------------------------------------------

func TestCeNsEnterCheck(t *testing.T) {
	priv := workload.Container{Privileged: true}
	cases := []struct {
		name string
		pod  *workload.Pod
		want bool
	}{
		{name: "nil pod", pod: nil, want: false},
		{
			name: "priv + hostPID -> true",
			pod:  &workload.Pod{Containers: []workload.Container{priv}, HostPID: true},
			want: true,
		},
		{
			name: "priv only",
			pod:  &workload.Pod{Containers: []workload.Container{priv}},
			want: false,
		},
		{
			name: "hostPID only",
			pod:  &workload.Pod{Containers: []workload.Container{{Privileged: false}}, HostPID: true},
			want: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ceNsEnterCheck(tc.pod); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ceSysPtraceCheck (priv OR (hostPID + caps))
// ---------------------------------------------------------------------------

func TestCeSysPtraceCheck(t *testing.T) {
	priv := workload.Container{Privileged: true}
	nonPriv := workload.Container{Privileged: false}
	cases := []struct {
		name string
		pod  *workload.Pod
		want bool
	}{
		{name: "nil pod", pod: nil, want: false},
		{name: "priv shortcut", pod: &workload.Pod{Containers: []workload.Container{priv}}, want: true},
		{
			name: "hostPID + both caps",
			pod: &workload.Pod{
				Containers:      []workload.Container{nonPriv},
				HostPID:         true,
				CapabilitiesAdd: []string{"SYS_PTRACE", "SYS_ADMIN"},
			},
			want: true,
		},
		{
			name: "hostPID + caps but missing SYS_ADMIN",
			pod: &workload.Pod{
				Containers:      []workload.Container{nonPriv},
				HostPID:         true,
				CapabilitiesAdd: []string{"SYS_PTRACE"},
			},
			want: false,
		},
		{
			name: "caps without hostPID",
			pod: &workload.Pod{
				Containers:      []workload.Container{nonPriv},
				CapabilitiesAdd: []string{"SYS_PTRACE", "SYS_ADMIN"},
			},
			want: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ceSysPtraceCheck(tc.pod); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// podHasSocketHostPath
// ---------------------------------------------------------------------------

func TestPodHasSocketHostPath(t *testing.T) {
	cases := []struct {
		name string
		pod  *workload.Pod
		want string
	}{
		{name: "nil pod", pod: nil, want: ""},
		{name: "no volumes", pod: &workload.Pod{}, want: ""},
		{
			name: "single .sock volume",
			pod: &workload.Pod{Volumes: []workload.VolumeDetail{
				{HostPath: "/var/run/docker.sock"},
			}},
			want: "/var/run/docker.sock",
		},
		{
			name: "first match wins",
			pod: &workload.Pod{Volumes: []workload.VolumeDetail{
				{HostPath: "/var/run/containerd.sock"},
				{HostPath: "/var/run/docker.sock"},
			}},
			want: "/var/run/containerd.sock",
		},
		{
			name: "no socket suffix",
			pod: &workload.Pod{Volumes: []workload.VolumeDetail{
				{HostPath: "/var/run"},
				{HostPath: "/etc"},
			}},
			want: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := podHasSocketHostPath(tc.pod); got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ceUmhCorePatternCheck
// ---------------------------------------------------------------------------

func TestCeUmhCorePatternCheck(t *testing.T) {
	rwMount := workload.VolumeMount{Name: "proc-vol", MountPath: "/host/proc", ReadOnly: false}
	roMount := workload.VolumeMount{Name: "proc-vol", MountPath: "/host/proc", ReadOnly: true}

	cases := []struct {
		name      string
		pod       *workload.Pod
		wantPath  string
		wantOK    bool
	}{
		{name: "nil pod", pod: nil, wantPath: "", wantOK: false},
		{name: "no critical volume", pod: &workload.Pod{
			Volumes:    []workload.VolumeDetail{{Name: "v", HostPath: "/etc"}},
			Containers: []workload.Container{{VolumeMounts: []workload.VolumeMount{rwMount}}},
		}, wantPath: "", wantOK: false},
		{
			name: "/proc volume with rw mount",
			pod: &workload.Pod{
				Volumes:    []workload.VolumeDetail{{Name: "proc-vol", HostPath: "/proc"}},
				Containers: []workload.Container{{VolumeMounts: []workload.VolumeMount{rwMount}}},
			},
			wantPath: "/host/proc", wantOK: true,
		},
		{
			name: "/proc/sys volume with rw mount",
			pod: &workload.Pod{
				Volumes:    []workload.VolumeDetail{{Name: "proc-vol", HostPath: "/proc/sys"}},
				Containers: []workload.Container{{VolumeMounts: []workload.VolumeMount{rwMount}}},
			},
			wantPath: "/host/proc", wantOK: true,
		},
		{
			name: "/proc/sys/kernel volume with rw mount",
			pod: &workload.Pod{
				Volumes:    []workload.VolumeDetail{{Name: "proc-vol", HostPath: "/proc/sys/kernel"}},
				Containers: []workload.Container{{VolumeMounts: []workload.VolumeMount{rwMount}}},
			},
			wantPath: "/host/proc", wantOK: true,
		},
		{
			name: "critical volume mounted readonly does not count",
			pod: &workload.Pod{
				Volumes:    []workload.VolumeDetail{{Name: "proc-vol", HostPath: "/proc"}},
				Containers: []workload.Container{{VolumeMounts: []workload.VolumeMount{roMount}}},
			},
			wantPath: "", wantOK: false,
		},
		{
			name: "non-critical /proc/foo not flagged",
			pod: &workload.Pod{
				Volumes:    []workload.VolumeDetail{{Name: "proc-vol", HostPath: "/proc/foo"}},
				Containers: []workload.Container{{VolumeMounts: []workload.VolumeMount{rwMount}}},
			},
			wantPath: "", wantOK: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotPath, gotOK := ceUmhCorePatternCheck(tc.pod)
			if gotPath != tc.wantPath || gotOK != tc.wantOK {
				t.Fatalf("got (%q,%v) want (%q,%v)", gotPath, gotOK, tc.wantPath, tc.wantOK)
			}
		})
	}
}
