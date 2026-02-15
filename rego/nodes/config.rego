package nodes.config

import rego.v1

import future.keywords.in

# Common sensitive keys to check for in resources 
sensitive_keys := ["password", "token", "key", "secret", "cert", "credential", "api_key", "apikey", "private", "pass"]

# Known resource kinds with node policies
known_kinds := {
    "clusterrole",
    "clusterrolebinding",
    "clustersecretstore",
    "configmap",
    "daemonset",
    "deployment",
    "externalsecret",
    "httproute",
    "ingress",
    "namespace",
    "networkpolicy",
    "node",
    "nodelist",
    "persistentvolume",
    "persistentvolumeclaim",
    "pod",
    "role",
    "rolebinding",
    "secret",
    "secretstore",
    "securitycontextconstraints",
    "service",
    "serviceaccount",
    "statefulset",
}