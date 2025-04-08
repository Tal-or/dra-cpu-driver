package config

import (
	coreclientset "k8s.io/client-go/kubernetes"

	"github.com/Tal-or/dra-cpu-driver/pkg/flags"
)

const (
	DriverName                   = "manager.cpu.com"
	DriverPluginRegistrationPath = "/var/lib/kubelet/plugins_registry/" + DriverName + ".sock"
	DriverPluginPath             = "/var/lib/kubelet/plugins/" + DriverName
	DriverPluginSocketPath       = DriverPluginPath + "/Plugin.sock"
)

type ProgArgs struct {
	KubeClientConfig flags.KubeClientConfig
	LoggingConfig    *flags.LoggingConfig

	CdiRoot     string
	NodeName    string
	Reserved    string
	Allocatable string
	Shared      string
}

type Config struct {
	ProgArgs   *ProgArgs
	Coreclient coreclientset.Interface
}
