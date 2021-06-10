module sigs.k8s.io/cosi-driver-sample

go 1.15

require (
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.1.3
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.7.1
	google.golang.org/grpc v1.38.0
	k8s.io/klog/v2 v2.9.0
	sigs.k8s.io/container-object-storage-interface-provisioner-sidecar v0.0.0-20210528161624-b46634c30d14
	sigs.k8s.io/container-object-storage-interface-spec v0.0.0-20210507203703-a97f2e98ac90
)
