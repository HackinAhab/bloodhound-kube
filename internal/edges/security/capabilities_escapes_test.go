package security

import (
	"testing"

	"bloodhound-kube/internal/nodes/workload"
)

func TestCeVarLogSymlinkCheck(t *testing.T) {
	privilegedRoot := workload.Container{
		Name:         "privileged-root",
		Privileged:   true,
		RunAsNonRoot: false,
	}
	privilegedNonRoot := workload.Container{
		Name:         "privileged-nonroot",
		Privileged:   true,
		RunAsNonRoot: true,
	}
	unprivileged := workload.Container{
		Name:         "unprivileged",
		Privileged:   false,
		RunAsNonRoot: false,
	}

	varLogVolume := workload.VolumeDetail{Name: "varlog", HostPath: "/var/log"}
	varVolume := workload.VolumeDetail{Name: "var", HostPath: "/var"}
	otherVolume := workload.VolumeDetail{Name: "etc", HostPath: "/etc"}

	tests := []struct {
		name         string
		pod          *workload.Pod
		expectedPath string
		expectedOK   bool
	}{
		{
			name:         "nil pod",
			pod:          nil,
			expectedPath: "",
			expectedOK:   false,
		},
		{
			name: "single privileged root container with /var/log hostPath",
			pod: &workload.Pod{
				Containers: []workload.Container{privilegedRoot},
				Volumes:    []workload.VolumeDetail{varLogVolume},
			},
			expectedPath: "/var/log",
			expectedOK:   true,
		},
		{
			name: "single privileged root container with /var hostPath",
			pod: &workload.Pod{
				Containers: []workload.Container{privilegedRoot},
				Volumes:    []workload.VolumeDetail{varVolume},
			},
			expectedPath: "/var",
			expectedOK:   true,
		},
		{
			// Regression test for the bug: previously the loop returned
			// false on the first non-qualifying container, missing the
			// privileged root container that follows.
			name: "regression: first unprivileged, second privileged+root, /var/log",
			pod: &workload.Pod{
				Containers: []workload.Container{unprivileged, privilegedRoot},
				Volumes:    []workload.VolumeDetail{varLogVolume},
			},
			expectedPath: "/var/log",
			expectedOK:   true,
		},
		{
			// Symmetric regression: ordering should not matter; any
			// qualifying container suffices.
			name: "regression: first privileged+root, second unprivileged, /var/log",
			pod: &workload.Pod{
				Containers: []workload.Container{privilegedRoot, unprivileged},
				Volumes:    []workload.VolumeDetail{varLogVolume},
			},
			expectedPath: "/var/log",
			expectedOK:   true,
		},
		{
			name: "all containers unprivileged",
			pod: &workload.Pod{
				Containers: []workload.Container{unprivileged, unprivileged},
				Volumes:    []workload.VolumeDetail{varLogVolume},
			},
			expectedPath: "",
			expectedOK:   false,
		},
		{
			name: "privileged but RunAsNonRoot=true",
			pod: &workload.Pod{
				Containers: []workload.Container{privilegedNonRoot},
				Volumes:    []workload.VolumeDetail{varLogVolume},
			},
			expectedPath: "",
			expectedOK:   false,
		},
		{
			name: "qualifying container but no matching hostPath",
			pod: &workload.Pod{
				Containers: []workload.Container{privilegedRoot},
				Volumes:    []workload.VolumeDetail{otherVolume},
			},
			expectedPath: "",
			expectedOK:   false,
		},
		{
			name: "qualifying container, no volumes at all",
			pod: &workload.Pod{
				Containers: []workload.Container{privilegedRoot},
				Volumes:    nil,
			},
			expectedPath: "",
			expectedOK:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPath, gotOK := ceVarLogSymlinkCheck(tt.pod)
			if gotPath != tt.expectedPath || gotOK != tt.expectedOK {
				t.Errorf("ceVarLogSymlinkCheck() = (%q, %v), want (%q, %v)",
					gotPath, gotOK, tt.expectedPath, tt.expectedOK)
			}
		})
	}
}
