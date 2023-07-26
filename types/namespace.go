package types

import ()

const (
	HostProcDirectory = "/host/proc"
	ProcDirectory     = "/proc"
)

const NsBinary = "nsenter"

type Namespace string

const (
	NamespaceIpc = Namespace("ipc")
	NamespaceMnt = Namespace("mnt")
	NamespaceNet = Namespace("net")
)

func (ns Namespace) String() string {
	return string(ns)
}
