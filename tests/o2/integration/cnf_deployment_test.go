package o2_integration_tests_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeClient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o2"
)

type CNFDeploymentTestSuite struct {
	suite.Suite
	o2Adaptor     *o2.O2Adaptor
	o2Manager     *o2.O2Manager
	k8sClient     client.Client
	k8sClientset  *fake.Clientset
	testLogger    *logging.StructuredLogger
	testNamespace string
}

func (suite *CNFDeploymentTestSuite) SetupSuite() {
	scheme := runtime.NewScheme()
	corev1.AddToScheme(scheme)
	appsv1.AddToScheme(scheme)

	suite.k8sClient = fakeClient.NewClientBuilder().WithScheme(scheme).Build()
	suite.k8sClientset = fake.NewSimpleClientset()
	suite.testLogger = logging.NewLogger("cnf-deployment-test", "debug")
	suite.testNamespace = "o-ran-cnf-test"

	config := &o2.O2Config{
		Namespace: suite.testNamespace,
		DefaultResources: &corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("100m"),
				corev1.ResourceMemory: resource.MustParse("128Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("1"),
				corev1.ResourceMemory: resource.MustParse("1Gi"),
			},
		},
	}

	suite.o2Adaptor = o2.NewO2Adaptor(suite.k8sClient, suite.k8sClientset, config)
	suite.o2Manager = o2.NewO2Manager(suite.o2Adaptor)

	// Create test namespace
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: suite.testNamespace,
		},
	}
	suite.Require().NoError(suite.k8sClient.Create(context.Background(), ns))
}

func (suite *CNFDeploymentTestSuite) TestAMFCNFDeployment() {
	suite.Run("Deploy 5G AMF CNF with Standard Configuration", func() {
		ctx := context.Background()

		// Define AMF CNF descriptor
		amfDescriptor := &o2.VNFDescriptor{
			Name:        "test-amf-cnf",
			Type:        "amf",
			Version:     "1.2.0",
			Vendor:      "Nephoran",
			Description: "5G Access and Mobility Management Function",
			Image:       "registry.nephoran.com/5g/amf:1.2.0",
			Replicas:    2,
			Resources: &corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("1"),
					corev1.ResourceMemory: resource.MustParse("2Gi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("2"),
					corev1.ResourceMemory: resource.MustParse("4Gi"),
				},
			},
			Environment: []corev1.EnvVar{
				{Name: "AMF_CONFIG", Value: "/etc/amf/amf.yaml"},
				{Name: "LOG_LEVEL", Value: "INFO"},
				{Name: "PLMN_ID", Value: "00101"},
				{Name: "AMF_REGION_ID", Value: "128"},
				{Name: "AMF_SET_ID", Value: "1"},
			},
			Ports: []corev1.ContainerPort{
				{Name: "sbi", ContainerPort: 8080, Protocol: corev1.ProtocolTCP},
				{Name: "ngap", ContainerPort: 38412, Protocol: corev1.ProtocolSCTP},
				{Name: "metrics", ContainerPort: 9090, Protocol: corev1.ProtocolTCP},
			},
			VolumeConfig: []o2.VolumeConfig{
				{
					Name:      "amf-config",
					MountPath: "/etc/amf",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "amf-config",
							},
						},
					},
				},
				{
					Name:      "amf-certs",
					MountPath: "/etc/ssl/certs/amf",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: "amf-tls-certs",
						},
					},
				},
			},
			NetworkConfig: &o2.NetworkConfig{
				ServiceType: corev1.ServiceTypeClusterIP,
				Ports: []corev1.ServicePort{
					{Name: "sbi", Port: 8080, TargetPort: intstr.FromInt(8080)},
					{Name: "ngap", Port: 38412, TargetPort: intstr.FromInt(38412), Protocol: corev1.ProtocolSCTP},
					{Name: "metrics", Port: 9090, TargetPort: intstr.FromInt(9090)},
				},
			},
			SecurityContext: &corev1.SecurityContext{
				RunAsNonRoot:             &[]bool{true}[0],
				RunAsUser:                &[]int64{1001}[0],
				AllowPrivilegeEscalation: &[]bool{false}[0],
				ReadOnlyRootFilesystem:   &[]bool{true}[0],
				Capabilities: &corev1.Capabilities{
					Drop: []corev1.Capability{"ALL"},
					Add:  []corev1.Capability{"NET_BIND_SERVICE"},
				},
			},
			HealthCheck: &o2.HealthCheckConfig{
				LivenessProbe: &corev1.Probe{
					ProbeHandler: corev1.ProbeHandler{
						HTTPGet: &corev1.HTTPGetAction{
							Path: "/health",
							Port: intstr.FromInt(8080),
						},
					},
					InitialDelaySeconds: 30,
					PeriodSeconds:       10,
					TimeoutSeconds:      5,
					SuccessThreshold:    1,
					FailureThreshold:    3,
				},
				ReadinessProbe: &corev1.Probe{
					ProbeHandler: corev1.ProbeHandler{
						HTTPGet: &corev1.HTTPGetAction{
							Path: "/ready",
							Port: intstr.FromInt(8080),
						},
					},
					InitialDelaySeconds: 10,
					PeriodSeconds:       5,
					TimeoutSeconds:      3,
					SuccessThreshold:    1,
					FailureThreshold:    3,
				},
				StartupProbe: &corev1.Probe{
					ProbeHandler: corev1.ProbeHandler{
						HTTPGet: &corev1.HTTPGetAction{
							Path: "/startup",
							Port: intstr.FromInt(8080),
						},
					},
					InitialDelaySeconds: 10,
					PeriodSeconds:       10,
					TimeoutSeconds:      5,
					SuccessThreshold:    1,
					FailureThreshold:    10,
				},
			},
			Affinity: &corev1.Affinity{
				PodAntiAffinity: &corev1.PodAntiAffinity{
					PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
						{
							Weight: 100,
							PodAffinityTerm: corev1.PodAffinityTerm{
								LabelSelector: &metav1.LabelSelector{
									MatchExpressions: []metav1.LabelSelectorRequirement{
										{
											Key:      "app",
											Operator: metav1.LabelSelectorOpIn,
											Values:   []string{"test-amf-cnf"},
										},
									},
								},
								TopologyKey: "kubernetes.io/hostname",
							},
						},
					},
				},
			},
			Metadata: map[string]string{
				"cnf.nephoran.com/type":     "5g-core",
				"cnf.nephoran.com/function": "amf",
				"cnf.nephoran.com/vendor":   "nephoran",
				"cnf.nephoran.com/version":  "1.2.0",
				"cnf.nephoran.com/standard": "3gpp-rel16",
			},
		}

		// Deploy the AMF CNF
		deploymentStatus, err := suite.o2Manager.DeployVNF(ctx, amfDescriptor)
		suite.Require().NoError(err)
		suite.Assert().NotNil(deploymentStatus)
		suite.Assert().Equal("test-amf-cnf", deploymentStatus.Name)
		suite.Assert().Equal("PENDING", deploymentStatus.Status)
		suite.Assert().Equal("Creating", deploymentStatus.Phase)
		suite.Assert().Equal(int32(2), deploymentStatus.Replicas)

		// Verify deployment was created
		deployment := &appsv1.Deployment{}
		err = suite.k8sClient.Get(ctx, client.ObjectKey{
			Namespace: suite.testNamespace,
			Name:      "test-amf-cnf",
		}, deployment)
		suite.Require().NoError(err)

		// Validate deployment configuration
		suite.Assert().Equal("test-amf-cnf", deployment.Name)
		suite.Assert().Equal(suite.testNamespace, deployment.Namespace)
		suite.Assert().Equal(int32(2), *deployment.Spec.Replicas)
		suite.Assert().Equal("registry.nephoran.com/5g/amf:1.2.0", deployment.Spec.Template.Spec.Containers[0].Image)

		// Validate labels
		suite.Assert().Equal("test-amf-cnf", deployment.Labels["app"])
		suite.Assert().Equal("true", deployment.Labels["nephoran.com/vnf"])
		suite.Assert().Equal("o-ran", deployment.Labels["nephoran.com/type"])
		suite.Assert().Equal("5g-core", deployment.Labels["cnf.nephoran.com/type"])
		suite.Assert().Equal("amf", deployment.Labels["cnf.nephoran.com/function"])

		// Validate container configuration
		container := deployment.Spec.Template.Spec.Containers[0]
		suite.Assert().Equal("test-amf-cnf", container.Name)
		suite.Assert().Len(container.Env, 5)
		suite.Assert().Len(container.Ports, 3)
		suite.Assert().Len(container.VolumeMounts, 2)

		// Validate resource requirements
		suite.Assert().Equal("1", container.Resources.Requests.Cpu().String())
		suite.Assert().Equal("2Gi", container.Resources.Requests.Memory().String())
		suite.Assert().Equal("2", container.Resources.Limits.Cpu().String())
		suite.Assert().Equal("4Gi", container.Resources.Limits.Memory().String())

		// Validate security context
		suite.Assert().NotNil(container.SecurityContext)
		suite.Assert().True(*container.SecurityContext.RunAsNonRoot)
		suite.Assert().Equal(int64(1001), *container.SecurityContext.RunAsUser)
		suite.Assert().False(*container.SecurityContext.AllowPrivilegeEscalation)
		suite.Assert().True(*container.SecurityContext.ReadOnlyRootFilesystem)

		// Validate health checks
		suite.Assert().NotNil(container.LivenessProbe)
		suite.Assert().NotNil(container.ReadinessProbe)
		suite.Assert().NotNil(container.StartupProbe)
		suite.Assert().Equal("/health", container.LivenessProbe.HTTPGet.Path)
		suite.Assert().Equal("/ready", container.ReadinessProbe.HTTPGet.Path)
		suite.Assert().Equal("/startup", container.StartupProbe.HTTPGet.Path)

		// Validate volumes
		suite.Assert().Len(deployment.Spec.Template.Spec.Volumes, 2)
		suite.Assert().Equal("amf-config", deployment.Spec.Template.Spec.Volumes[0].Name)
		suite.Assert().Equal("amf-certs", deployment.Spec.Template.Spec.Volumes[1].Name)

		// Validate affinity rules
		suite.Assert().NotNil(deployment.Spec.Template.Spec.Affinity)
		suite.Assert().NotNil(deployment.Spec.Template.Spec.Affinity.PodAntiAffinity)

		// Verify service was created
		service := &corev1.Service{}
		err = suite.k8sClient.Get(ctx, client.ObjectKey{
			Namespace: suite.testNamespace,
			Name:      "test-amf-cnf-service",
		}, service)
		suite.Require().NoError(err)
		suite.Assert().Equal("test-amf-cnf-service", service.Name)
		suite.Assert().Len(service.Spec.Ports, 3)
		suite.Assert().Equal(int32(8080), service.Spec.Ports[0].Port)
		suite.Assert().Equal(int32(38412), service.Spec.Ports[1].Port)
		suite.Assert().Equal(int32(9090), service.Spec.Ports[2].Port)
	})
}

func (suite *CNFDeploymentTestSuite) TestSMFCNFDeployment() {
	suite.Run("Deploy 5G SMF CNF with Database Dependencies", func() {
		ctx := context.Background()

		// Define SMF CNF descriptor with database dependencies
		smfDescriptor := &o2.VNFDescriptor{
			Name:        "test-smf-cnf",
			Type:        "smf",
			Version:     "1.1.5",
			Vendor:      "Nephoran",
			Description: "5G Session Management Function with UDR integration",
			Image:       "registry.nephoran.com/5g/smf:1.1.5",
			Replicas:    3,
			Resources: &corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("1500m"),
					corev1.ResourceMemory: resource.MustParse("3Gi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("3"),
					corev1.ResourceMemory: resource.MustParse("6Gi"),
				},
			},
			Environment: []corev1.EnvVar{
				{Name: "SMF_CONFIG", Value: "/etc/smf/smf.yaml"},
				{Name: "LOG_LEVEL", Value: "DEBUG"},
				{Name: "UDR_ENDPOINT", Value: "http://udr-service:8080"},
				{Name: "PFCP_PORT", Value: "8805"},
				{Name: "DATABASE_URL", Value: "postgresql://smf:password@postgres-service:5432/smfdb"},
			},
			Ports: []corev1.ContainerPort{
				{Name: "sbi", ContainerPort: 8080, Protocol: corev1.ProtocolTCP},
				{Name: "pfcp", ContainerPort: 8805, Protocol: corev1.ProtocolUDP},
				{Name: "metrics", ContainerPort: 9090, Protocol: corev1.ProtocolTCP},
				{Name: "health", ContainerPort: 8081, Protocol: corev1.ProtocolTCP},
			},
			VolumeConfig: []o2.VolumeConfig{
				{
					Name:      "smf-config",
					MountPath: "/etc/smf",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "smf-config",
							},
						},
					},
				},
				{
					Name:      "smf-data",
					MountPath: "/var/lib/smf",
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: "smf-data-pvc",
						},
					},
				},
			},
			NetworkConfig: &o2.NetworkConfig{
				ServiceType: corev1.ServiceTypeClusterIP,
				Ports: []corev1.ServicePort{
					{Name: "sbi", Port: 8080, TargetPort: intstr.FromInt(8080)},
					{Name: "pfcp", Port: 8805, TargetPort: intstr.FromInt(8805), Protocol: corev1.ProtocolUDP},
					{Name: "metrics", Port: 9090, TargetPort: intstr.FromInt(9090)},
				},
			},
			HealthCheck: &o2.HealthCheckConfig{
				LivenessProbe: &corev1.Probe{
					ProbeHandler: corev1.ProbeHandler{
						TCPSocket: &corev1.TCPSocketAction{
							Port: intstr.FromInt(8080),
						},
					},
					InitialDelaySeconds: 60,
					PeriodSeconds:       30,
					TimeoutSeconds:      10,
					FailureThreshold:    3,
				},
				ReadinessProbe: &corev1.Probe{
					ProbeHandler: corev1.ProbeHandler{
						HTTPGet: &corev1.HTTPGetAction{
							Path: "/ready",
							Port: intstr.FromInt(8081),
						},
					},
					InitialDelaySeconds: 30,
					PeriodSeconds:       10,
					TimeoutSeconds:      5,
					FailureThreshold:    2,
				},
			},
			Tolerations: []corev1.Toleration{
				{
					Key:      "node-role.kubernetes.io/control-plane",
					Operator: corev1.TolerationOpExists,
					Effect:   corev1.TaintEffectNoSchedule,
				},
			},
			Metadata: map[string]string{
				"cnf.nephoran.com/type":        "5g-core",
				"cnf.nephoran.com/function":    "smf",
				"cnf.nephoran.com/persistence": "true",
				"cnf.nephoran.com/database":    "postgresql",
			},
		}

		// Deploy the SMF CNF
		deploymentStatus, err := suite.o2Manager.DeployVNF(ctx, smfDescriptor)
		suite.Require().NoError(err)
		suite.Assert().NotNil(deploymentStatus)
		suite.Assert().Equal("test-smf-cnf", deploymentStatus.Name)
		suite.Assert().Equal(int32(3), deploymentStatus.Replicas)

		// Verify deployment was created
		deployment := &appsv1.Deployment{}
		err = suite.k8sClient.Get(ctx, client.ObjectKey{
			Namespace: suite.testNamespace,
			Name:      "test-smf-cnf",
		}, deployment)
		suite.Require().NoError(err)

		// Validate SMF-specific configuration
		container := deployment.Spec.Template.Spec.Containers[0]

		// Validate environment variables include database connection
		envVars := make(map[string]string)
		for _, env := range container.Env {
			envVars[env.Name] = env.Value
		}
		suite.Assert().Contains(envVars, "DATABASE_URL")
		suite.Assert().Contains(envVars, "UDR_ENDPOINT")
		suite.Assert().Equal("8805", envVars["PFCP_PORT"])

		// Validate ports include PFCP UDP port
		portNames := make([]string, len(container.Ports))
		for i, port := range container.Ports {
			portNames[i] = port.Name
			if port.Name == "pfcp" {
				suite.Assert().Equal(corev1.ProtocolUDP, port.Protocol)
				suite.Assert().Equal(int32(8805), port.ContainerPort)
			}
		}
		suite.Assert().Contains(portNames, "pfcp")
		suite.Assert().Contains(portNames, "sbi")
		suite.Assert().Contains(portNames, "metrics")

		// Validate persistent volume claim mount
		volumeMountNames := make([]string, len(container.VolumeMounts))
		for i, mount := range container.VolumeMounts {
			volumeMountNames[i] = mount.Name
			if mount.Name == "smf-data" {
				suite.Assert().Equal("/var/lib/smf", mount.MountPath)
			}
		}
		suite.Assert().Contains(volumeMountNames, "smf-data")

		// Validate tolerations
		suite.Assert().Len(deployment.Spec.Template.Spec.Tolerations, 1)
		suite.Assert().Equal("node-role.kubernetes.io/control-plane", deployment.Spec.Template.Spec.Tolerations[0].Key)
	})
}

func (suite *CNFDeploymentTestSuite) TestUPFCNFDeployment() {
	suite.Run("Deploy 5G UPF CNF with High Performance Configuration", func() {
		ctx := context.Background()

		// Define UPF CNF descriptor optimized for high performance
		upfDescriptor := &o2.VNFDescriptor{
			Name:        "test-upf-cnf",
			Type:        "upf",
			Version:     "2.0.0",
			Vendor:      "Nephoran",
			Description: "5G User Plane Function with DPDK acceleration",
			Image:       "registry.nephoran.com/5g/upf:2.0.0",
			Replicas:    2,
			Resources: &corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("4"),
					corev1.ResourceMemory: resource.MustParse("8Gi"),
					"hugepages-1Gi":       resource.MustParse("4Gi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("8"),
					corev1.ResourceMemory: resource.MustParse("16Gi"),
					"hugepages-1Gi":       resource.MustParse("4Gi"),
				},
			},
			Environment: []corev1.EnvVar{
				{Name: "UPF_CONFIG", Value: "/etc/upf/upf.yaml"},
				{Name: "DPDK_MODE", Value: "true"},
				{Name: "PFCP_PORT", Value: "8805"},
				{Name: "GTP_U_PORT", Value: "2152"},
				{Name: "CPU_ISOLATION", Value: "2-7"},
			},
			Ports: []corev1.ContainerPort{
				{Name: "pfcp", ContainerPort: 8805, Protocol: corev1.ProtocolUDP},
				{Name: "gtpu", ContainerPort: 2152, Protocol: corev1.ProtocolUDP},
				{Name: "metrics", ContainerPort: 9090, Protocol: corev1.ProtocolTCP},
			},
			VolumeConfig: []o2.VolumeConfig{
				{
					Name:      "upf-config",
					MountPath: "/etc/upf",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "upf-config",
							},
						},
					},
				},
				{
					Name:      "hugepages",
					MountPath: "/dev/hugepages",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{
							Medium: "HugePages-1Gi",
						},
					},
				},
				{
					Name:      "host-dev",
					MountPath: "/dev",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: "/dev",
							Type: &[]corev1.HostPathType{corev1.HostPathDirectory}[0],
						},
					},
				},
			},
			SecurityContext: &corev1.SecurityContext{
				Privileged: &[]bool{true}[0], // Required for DPDK
				Capabilities: &corev1.Capabilities{
					Add: []corev1.Capability{"IPC_LOCK", "NET_ADMIN", "NET_RAW"},
				},
			},
			NetworkConfig: &o2.NetworkConfig{
				ServiceType: corev1.ServiceTypeClusterIP,
				Ports: []corev1.ServicePort{
					{Name: "pfcp", Port: 8805, TargetPort: intstr.FromInt(8805), Protocol: corev1.ProtocolUDP},
					{Name: "gtpu", Port: 2152, TargetPort: intstr.FromInt(2152), Protocol: corev1.ProtocolUDP},
					{Name: "metrics", Port: 9090, TargetPort: intstr.FromInt(9090)},
				},
			},
			Affinity: &corev1.Affinity{
				NodeAffinity: &corev1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
						NodeSelectorTerms: []corev1.NodeSelectorTerm{
							{
								MatchExpressions: []corev1.NodeSelectorRequirement{
									{
										Key:      "node.nephoran.com/dpdk",
										Operator: metav1.LabelSelectorOpIn,
										Values:   []string{"enabled"},
									},
									{
										Key:      "node.nephoran.com/hugepages-1gi",
										Operator: metav1.LabelSelectorOpExists,
									},
								},
							},
						},
					},
				},
			},
			Metadata: map[string]string{
				"cnf.nephoran.com/type":        "5g-core",
				"cnf.nephoran.com/function":    "upf",
				"cnf.nephoran.com/performance": "high",
				"cnf.nephoran.com/dpdk":        "true",
				"cnf.nephoran.com/hugepages":   "1Gi",
			},
		}

		// Deploy the UPF CNF
		deploymentStatus, err := suite.o2Manager.DeployVNF(ctx, upfDescriptor)
		suite.Require().NoError(err)
		suite.Assert().NotNil(deploymentStatus)
		suite.Assert().Equal("test-upf-cnf", deploymentStatus.Name)

		// Verify deployment was created
		deployment := &appsv1.Deployment{}
		err = suite.k8sClient.Get(ctx, client.ObjectKey{
			Namespace: suite.testNamespace,
			Name:      "test-upf-cnf",
		}, deployment)
		suite.Require().NoError(err)

		// Validate UPF-specific high-performance configuration
		container := deployment.Spec.Template.Spec.Containers[0]

		// Validate resource requirements include hugepages
		suite.Assert().Equal("4", container.Resources.Requests.Cpu().String())
		suite.Assert().Equal("8Gi", container.Resources.Requests.Memory().String())
		hugepagesReq := container.Resources.Requests["hugepages-1Gi"]
		suite.Assert().Equal("4Gi", hugepagesReq.String())

		// Validate security context allows privileged mode for DPDK
		suite.Assert().NotNil(container.SecurityContext)
		suite.Assert().True(*container.SecurityContext.Privileged)
		suite.Assert().Contains(container.SecurityContext.Capabilities.Add, corev1.Capability("IPC_LOCK"))
		suite.Assert().Contains(container.SecurityContext.Capabilities.Add, corev1.Capability("NET_ADMIN"))

		// Validate volume mounts for hugepages and host devices
		volumeMountPaths := make(map[string]string)
		for _, mount := range container.VolumeMounts {
			volumeMountPaths[mount.Name] = mount.MountPath
		}
		suite.Assert().Equal("/dev/hugepages", volumeMountPaths["hugepages"])
		suite.Assert().Equal("/dev", volumeMountPaths["host-dev"])

		// Validate volumes for performance optimization
		volumeTypes := make(map[string]string)
		for _, volume := range deployment.Spec.Template.Spec.Volumes {
			if volume.EmptyDir != nil {
				volumeTypes[volume.Name] = string(volume.EmptyDir.Medium)
			} else if volume.HostPath != nil {
				volumeTypes[volume.Name] = "hostPath"
			}
		}
		suite.Assert().Equal("HugePages-1Gi", volumeTypes["hugepages"])
		suite.Assert().Equal("hostPath", volumeTypes["host-dev"])

		// Validate node affinity for DPDK-enabled nodes
		nodeAffinity := deployment.Spec.Template.Spec.Affinity.NodeAffinity
		suite.Assert().NotNil(nodeAffinity)
		suite.Assert().NotNil(nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution)

		requirements := nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions
		suite.Assert().Len(requirements, 2)

		// Check for DPDK label requirement
		dpdkReq := requirements[0]
		suite.Assert().Equal("node.nephoran.com/dpdk", dpdkReq.Key)
		suite.Assert().Equal(metav1.LabelSelectorOpIn, dpdkReq.Operator)
		suite.Assert().Contains(dpdkReq.Values, "enabled")
	})
}

func (suite *CNFDeploymentTestSuite) TestCNFScaling() {
	suite.Run("Scale CNF Deployment Up and Down", func() {
		ctx := context.Background()

		// First deploy a simple CNF
		simpleDescriptor := &o2.VNFDescriptor{
			Name:     "scalable-cnf",
			Type:     "test-cnf",
			Version:  "1.0.0",
			Vendor:   "Nephoran",
			Image:    "nginx:alpine",
			Replicas: 2,
			Resources: &corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("128Mi"),
				},
			},
			Ports: []corev1.ContainerPort{
				{Name: "http", ContainerPort: 80},
			},
		}

		_, err := suite.o2Manager.DeployVNF(ctx, simpleDescriptor)
		suite.Require().NoError(err)

		// Wait for initial deployment
		time.Sleep(100 * time.Millisecond)

		// Scale up to 5 replicas
		err = suite.o2Manager.ScaleWorkload(ctx, fmt.Sprintf("%s/scalable-cnf", suite.testNamespace), 5)
		suite.Require().NoError(err)

		// Verify scaling up
		deployment := &appsv1.Deployment{}
		err = suite.k8sClient.Get(ctx, client.ObjectKey{
			Namespace: suite.testNamespace,
			Name:      "scalable-cnf",
		}, deployment)
		suite.Require().NoError(err)
		suite.Assert().Equal(int32(5), *deployment.Spec.Replicas)

		// Scale down to 1 replica
		err = suite.o2Manager.ScaleWorkload(ctx, fmt.Sprintf("%s/scalable-cnf", suite.testNamespace), 1)
		suite.Require().NoError(err)

		// Verify scaling down
		err = suite.k8sClient.Get(ctx, client.ObjectKey{
			Namespace: suite.testNamespace,
			Name:      "scalable-cnf",
		}, deployment)
		suite.Require().NoError(err)
		suite.Assert().Equal(int32(1), *deployment.Spec.Replicas)
	})
}

func (suite *CNFDeploymentTestSuite) TestCNFResourceDiscovery() {
	suite.Run("Discover CNF Resources in Cluster", func() {
		ctx := context.Background()

		// Create some test nodes for discovery
		testNodes := []*corev1.Node{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "worker-node-1",
					Labels: map[string]string{
						"node-role.kubernetes.io/worker":  "",
						"node.nephoran.com/dpdk":          "enabled",
						"node.nephoran.com/hugepages-1gi": "available",
					},
				},
				Status: corev1.NodeStatus{
					Capacity: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("8"),
						corev1.ResourceMemory: resource.MustParse("32Gi"),
						"hugepages-1Gi":       resource.MustParse("8Gi"),
					},
					Allocatable: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("7800m"),
						corev1.ResourceMemory: resource.MustParse("30Gi"),
						"hugepages-1Gi":       resource.MustParse("8Gi"),
					},
					Conditions: []corev1.NodeCondition{
						{
							Type:   corev1.NodeReady,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "worker-node-2",
					Labels: map[string]string{
						"node-role.kubernetes.io/worker": "",
						"node.nephoran.com/sriov":        "enabled",
					},
				},
				Status: corev1.NodeStatus{
					Capacity: corev1.ResourceList{
						corev1.ResourceCPU:           resource.MustParse("16"),
						corev1.ResourceMemory:        resource.MustParse("64Gi"),
						"intel.com/sriov_net_device": resource.MustParse("8"),
					},
					Allocatable: corev1.ResourceList{
						corev1.ResourceCPU:           resource.MustParse("15800m"),
						corev1.ResourceMemory:        resource.MustParse("60Gi"),
						"intel.com/sriov_net_device": resource.MustParse("8"),
					},
					Conditions: []corev1.NodeCondition{
						{
							Type:   corev1.NodeReady,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
		}

		// Create test nodes
		for _, node := range testNodes {
			err := suite.k8sClient.Create(ctx, node)
			suite.Require().NoError(err)
		}

		// Discover cluster resources
		resourceMap, err := suite.o2Manager.DiscoverResources(ctx)
		suite.Require().NoError(err)
		suite.Assert().NotNil(resourceMap)

		// Validate discovered nodes
		suite.Assert().GreaterOrEqual(len(resourceMap.Nodes), 2)
		suite.Assert().Contains(resourceMap.Nodes, "worker-node-1")
		suite.Assert().Contains(resourceMap.Nodes, "worker-node-2")

		// Validate node-1 with DPDK capabilities
		node1 := resourceMap.Nodes["worker-node-1"]
		suite.Assert().Equal("worker-node-1", node1.Name)
		suite.Assert().Contains(node1.Labels, "node.nephoran.com/dpdk")
		suite.Assert().Equal("enabled", node1.Labels["node.nephoran.com/dpdk"])
		suite.Assert().Contains(node1.Roles, "worker")

		// Validate node-2 with SR-IOV capabilities
		node2 := resourceMap.Nodes["worker-node-2"]
		suite.Assert().Equal("worker-node-2", node2.Name)
		suite.Assert().Contains(node2.Labels, "node.nephoran.com/sriov")
		suite.Assert().Equal("enabled", node2.Labels["node.nephoran.com/sriov"])

		// Validate cluster metrics
		suite.Assert().NotNil(resourceMap.Metrics)
		suite.Assert().Equal(int32(2), resourceMap.Metrics.ReadyNodes)
		suite.Assert().NotEmpty(resourceMap.Metrics.TotalCPU)
		suite.Assert().NotEmpty(resourceMap.Metrics.TotalMemory)

		// Validate namespaces discovery
		suite.Assert().Contains(resourceMap.Namespaces, suite.testNamespace)
		testNsInfo := resourceMap.Namespaces[suite.testNamespace]
		suite.Assert().Equal(suite.testNamespace, testNsInfo.Name)
		suite.Assert().Equal("Active", testNsInfo.Status)
	})
}

func (suite *CNFDeploymentTestSuite) TestCNFHealthMonitoring() {
	suite.Run("Monitor CNF Health and Status", func() {
		ctx := context.Background()

		// Deploy a CNF with health checks
		healthyDescriptor := &o2.VNFDescriptor{
			Name:     "healthy-cnf",
			Type:     "test-cnf",
			Version:  "1.0.0",
			Vendor:   "Nephoran",
			Image:    "nginx:alpine",
			Replicas: 1,
			Resources: &corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("128Mi"),
				},
			},
			Ports: []corev1.ContainerPort{
				{Name: "http", ContainerPort: 80},
			},
			HealthCheck: &o2.HealthCheckConfig{
				LivenessProbe: &corev1.Probe{
					ProbeHandler: corev1.ProbeHandler{
						TCPSocket: &corev1.TCPSocketAction{
							Port: intstr.FromInt(80),
						},
					},
					InitialDelaySeconds: 5,
					PeriodSeconds:       10,
				},
				ReadinessProbe: &corev1.Probe{
					ProbeHandler: corev1.ProbeHandler{
						TCPSocket: &corev1.TCPSocketAction{
							Port: intstr.FromInt(80),
						},
					},
					InitialDelaySeconds: 1,
					PeriodSeconds:       5,
				},
			},
		}

		_, err := suite.o2Manager.DeployVNF(ctx, healthyDescriptor)
		suite.Require().NoError(err)

		// Verify deployment has health checks configured
		deployment := &appsv1.Deployment{}
		err = suite.k8sClient.Get(ctx, client.ObjectKey{
			Namespace: suite.testNamespace,
			Name:      "healthy-cnf",
		}, deployment)
		suite.Require().NoError(err)

		container := deployment.Spec.Template.Spec.Containers[0]
		suite.Assert().NotNil(container.LivenessProbe)
		suite.Assert().NotNil(container.ReadinessProbe)
		suite.Assert().Equal(int32(5), container.LivenessProbe.InitialDelaySeconds)
		suite.Assert().Equal(int32(1), container.ReadinessProbe.InitialDelaySeconds)

		// Test retrieving VNF instance status
		instanceID := fmt.Sprintf("%s-healthy-cnf", suite.testNamespace)
		vnfInstance, err := suite.o2Adaptor.GetVNFInstance(ctx, instanceID)
		suite.Require().NoError(err)
		suite.Assert().Equal("healthy-cnf", vnfInstance.Name)
		suite.Assert().Equal("INSTANTIATED", vnfInstance.Status.State)
	})
}

func (suite *CNFDeploymentTestSuite) TearDownSuite() {
	// Clean up test resources
	if suite.o2Manager != nil {
		// Graceful shutdown if needed
	}
}

func TestCNFDeployment(t *testing.T) {
	suite.Run(t, new(CNFDeploymentTestSuite))
}
