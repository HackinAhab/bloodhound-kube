package edges

type capabilityInfo struct {
	Description string
	Reference   string
}

var capabilityDescriptions = map[string]capabilityInfo{
	"CAP_SYS_ADMIN": {
		Description: "Container in pod has CAP_SYS_ADMIN capability which is a powerful capability that can allow for a wide range of actions, including privilege escalation and container escape.",
		Reference:   "https://book.hacktricks.wiki/en/linux-hardening/privilege-escalation/linux-capabilities.html#cap_sys_admin",
	},
	"CAP_NET_ADMIN": {
		Description: "Container in pod has CAP_NET_ADMIN capability which allows for network administration tasks and can be used for intercepting network traffic or modifying network configurations.",
		Reference:   "",
	},
	"CAP_SYS_MODULE": {
		Description: "Container in pod has CAP_SYS_MODULE capability which allows for loading and unloading kernel modules, and can be used for installing custom kernel modules.",
		Reference:   "https://book.hacktricks.wiki/en/linux-hardening/privilege-escalation/linux-capabilities.html#cap_sys_module",
	},
	"CAP_SYS_PTRACE": {
		Description: "Container in pod has CAP_SYS_PTRACE capability which allows for tracing and debugging of processes, and can be used for stealing sensitive information from other processes or performing code injection attacks.",
		Reference:   "https://book.hacktricks.wiki/en/linux-hardening/privilege-escalation/linux-capabilities.html#cap_sys_ptrace",
	},
	"CAP_SYS_RAWIO": {
		Description: "Container in pod has CAP_SYS_RAWIO capability which allows for raw I/O operations, and can be used for malicious purposes such as bypassing security controls or accessing sensitive data on the host.",
		Reference:   "https://book.hacktricks.wiki/en/linux-hardening/privilege-escalation/linux-capabilities.html#cap_sys_rawio",
	},
}
