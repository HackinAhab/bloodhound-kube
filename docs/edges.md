# Relationship definitions
I found KubeHound's relationship naming scheme to be intuitive and have adopted it where it made sense. I have also included links to the source used as a reference for each relationship policy in the individual policy files. 

## General Relationships for Contextual Awareness
- ScheduledOn: Pod -> Node
- Managedby:
- AppliesTo:
- RoutesTo:
- ExternalRoutesTo
- MountedBy
- EnvVars
- BoundTo
- PermissionsFromRole
- SAToken

## Container Escapes
- CE_NSENTER: Pod -> Node
    - 
- CE_PRIV_MOUNT: Pod -> Node
- CE_SYS_PTRACE: Pod -> Node
- CE_UMH_CORE_PATTERN: Pod -> Node

## Lateral Movement
- LM_HOST_MOUNT_KUBELET: Pod -> Node -> Node

## RBAC
- SAImpersonate
- WorkloadCreate
- RBACCreate
- PodExec
- PodDebug
- WorkloadPatch
- NodeProxy
- SAReadSecret
