/*

Copyright 2025.



Licensed under the Apache License, Version 2.0 (the "License");

you may not use this file except in compliance with the License.

You may obtain a copy of the License at



    http://www.apache.org/licenses/LICENSE-2.0



Unless required by applicable law or agreed to in writing, software

distributed under the License is distributed on an "AS IS" BASIS,

WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.

See the License for the specific language governing permissions and

limitations under the License.

*/

// FIXME: Adding package comment per revive linter.

// Package templates provides Helm chart templates for CNF deployment generation.

package templates

import (
	"fmt"

	nephoranv1 "github.com/nephio-project/nephoran-intent-operator/api/v1"
)

// HelmTemplateManager manages Helm chart templates for CNF deployments.

type HelmTemplateManager struct {
	Templates map[nephoranv1.CNFFunction]*HelmTemplate
}

// HelmTemplate defines a Helm chart template for a CNF function.

type HelmTemplate struct {
	Function nephoranv1.CNFFunction

	ChartName string

	ChartVersion string

	Repository string

	Values map[string]interface{}

	Dependencies []HelmDependency

	Interfaces []InterfaceTemplate

	ConfigMaps []ConfigMapTemplate

	Secrets []SecretTemplate

	Services []ServiceTemplate

	CustomValues map[string]interface{}
}

// HelmDependency defines a Helm chart dependency.

type HelmDependency struct {
	Name string `yaml:"name"`

	Version string `yaml:"version"`

	Repository string `yaml:"repository"`

	Condition string `yaml:"condition,omitempty"`
}

// InterfaceTemplate defines network interface templates.

type InterfaceTemplate struct {
	Name string

	Type string

	Port int32

	Protocol string

	ServiceType string

	Annotations map[string]string
}

// ConfigMapTemplate defines ConfigMap templates.

type ConfigMapTemplate struct {
	Name string

	Data map[string]string
}

// SecretTemplate defines Secret templates.

type SecretTemplate struct {
	Name string

	Type string

	Data map[string][]byte
}

// ServiceTemplate defines Service templates.

type ServiceTemplate struct {
	Name string

	Type string

	Ports []ServicePortTemplate

	Selector map[string]string

	Annotations map[string]string
}

// ServicePortTemplate defines service port templates.

type ServicePortTemplate struct {
	Name string

	Port int32

	TargetPort int32

	Protocol string
}

// NewHelmTemplateManager creates a new Helm template manager.

func NewHelmTemplateManager() *HelmTemplateManager {

	manager := &HelmTemplateManager{

		Templates: make(map[nephoranv1.CNFFunction]*HelmTemplate),
	}

	// Initialize 5G Core templates.

	manager.init5GCoreTemplates()

	// Initialize O-RAN templates.

	manager.initORANTemplates()

	// Initialize edge templates.

	manager.initEdgeTemplates()

	return manager

}

// GetTemplate retrieves a Helm template for a CNF function.

func (m *HelmTemplateManager) GetTemplate(function nephoranv1.CNFFunction) (*HelmTemplate, error) {

	template, exists := m.Templates[function]

	if !exists {

		return nil, fmt.Errorf("no Helm template found for CNF function: %s", function)

	}

	return template, nil

}

// GenerateValues generates Helm values for a CNF deployment.

func (m *HelmTemplateManager) GenerateValues(cnf *nephoranv1.CNFDeployment) (map[string]interface{}, error) {

	template, err := m.GetTemplate(cnf.Spec.Function)

	if err != nil {

		return nil, err

	}

	values := make(map[string]interface{})

	// Start with template default values.

	for k, v := range template.Values {

		values[k] = v

	}

	// Override with CNF-specific values.

	values["replicaCount"] = cnf.Spec.Replicas

	// Resource requirements.

	values["resources"] = map[string]interface{}{

		"requests": map[string]interface{}{

			"cpu": cnf.Spec.Resources.CPU.String(),

			"memory": cnf.Spec.Resources.Memory.String(),
		},

		"limits": map[string]interface{}{

			"cpu": cnf.Spec.Resources.CPU.String(),

			"memory": cnf.Spec.Resources.Memory.String(),
		},
	}

	// Override limits if specified.

	if cnf.Spec.Resources.MaxCPU != nil {

		limits := values["resources"].(map[string]interface{})["limits"].(map[string]interface{})

		limits["cpu"] = cnf.Spec.Resources.MaxCPU.String()

	}

	if cnf.Spec.Resources.MaxMemory != nil {

		limits := values["resources"].(map[string]interface{})["limits"].(map[string]interface{})

		limits["memory"] = cnf.Spec.Resources.MaxMemory.String()

	}

	// Storage configuration.

	if cnf.Spec.Resources.Storage != nil {

		values["persistence"] = map[string]interface{}{

			"enabled": true,

			"size": cnf.Spec.Resources.Storage.String(),
		}

	}

	// DPDK configuration.

	if cnf.Spec.Resources.DPDK != nil && cnf.Spec.Resources.DPDK.Enabled {

		values["dpdk"] = map[string]interface{}{

			"enabled": true,

			"cores": cnf.Spec.Resources.DPDK.Cores,

			"memory": cnf.Spec.Resources.DPDK.Memory,

			"driver": cnf.Spec.Resources.DPDK.Driver,
		}

	}

	// Service mesh configuration.

	if cnf.Spec.ServiceMesh != nil && cnf.Spec.ServiceMesh.Enabled {

		values["serviceMesh"] = map[string]interface{}{

			"enabled": true,

			"type": cnf.Spec.ServiceMesh.Type,
		}

		if cnf.Spec.ServiceMesh.MTLS != nil {

			values["serviceMesh"].(map[string]interface{})["mtls"] = map[string]interface{}{

				"enabled": cnf.Spec.ServiceMesh.MTLS.Enabled,

				"mode": cnf.Spec.ServiceMesh.MTLS.Mode,
			}

		}

	}

	// Monitoring configuration.

	if cnf.Spec.Monitoring != nil && cnf.Spec.Monitoring.Enabled {

		values["monitoring"] = map[string]interface{}{

			"enabled": true,
		}

		if cnf.Spec.Monitoring.Prometheus != nil {

			values["monitoring"].(map[string]interface{})["prometheus"] = map[string]interface{}{

				"enabled": cnf.Spec.Monitoring.Prometheus.Enabled,

				"port": cnf.Spec.Monitoring.Prometheus.Port,

				"path": cnf.Spec.Monitoring.Prometheus.Path,

				"interval": cnf.Spec.Monitoring.Prometheus.Interval,
			}

		}

	}

	// Auto-scaling configuration.

	if cnf.Spec.AutoScaling != nil && cnf.Spec.AutoScaling.Enabled {

		values["autoscaling"] = map[string]interface{}{

			"enabled": true,

			"minReplicas": cnf.Spec.AutoScaling.MinReplicas,

			"maxReplicas": cnf.Spec.AutoScaling.MaxReplicas,
		}

		if cnf.Spec.AutoScaling.CPUUtilization != nil {

			values["autoscaling"].(map[string]interface{})["targetCPUUtilizationPercentage"] = *cnf.Spec.AutoScaling.CPUUtilization

		}

		if cnf.Spec.AutoScaling.MemoryUtilization != nil {

			values["autoscaling"].(map[string]interface{})["targetMemoryUtilizationPercentage"] = *cnf.Spec.AutoScaling.MemoryUtilization

		}

	}

	// Security configuration.

	if len(cnf.Spec.SecurityPolicies) > 0 {

		values["security"] = map[string]interface{}{

			"policies": cnf.Spec.SecurityPolicies,
		}

	}

	return values, nil

}

// init5GCoreTemplates initializes Helm templates for 5G Core functions.

func (m *HelmTemplateManager) init5GCoreTemplates() {

	// AMF Template.

	m.Templates[nephoranv1.CNFFunctionAMF] = &HelmTemplate{

		Function: nephoranv1.CNFFunctionAMF,

		ChartName: "amf",

		ChartVersion: "1.0.0",

		Repository: "https://charts.5g-core.io",

		Values: map[string]interface{}{

			"image": map[string]interface{}{

				"repository": "5gc/amf",

				"tag": "latest",

				"pullPolicy": "IfNotPresent",
			},

			"service": map[string]interface{}{

				"type": "ClusterIP",

				"ports": map[string]interface{}{

					"sbi": 8080,

					"sctp": 38412,
				},
			},

			"config": map[string]interface{}{

				"plmnId": map[string]interface{}{

					"mcc": "001",

					"mnc": "01",
				},

				"amfId": "0x000001",

				"guami": map[string]interface{}{

					"plmnId": map[string]interface{}{

						"mcc": "001",

						"mnc": "01",
					},

					"amfRegionId": "0x01",

					"amfSetId": "0x001",

					"amfPointer": "0x01",
				},

				"tai": map[string]interface{}{

					"plmnId": map[string]interface{}{

						"mcc": "001",

						"mnc": "01",
					},

					"tac": "0x000001",
				},
			},

			"security": map[string]interface{}{

				"tls": map[string]interface{}{

					"enabled": true,
				},
			},
		},

		Interfaces: []InterfaceTemplate{

			{

				Name: "n1",

				Type: "NAS",

				Port: 38412,

				Protocol: "SCTP",

				ServiceType: "ClusterIP",
			},

			{

				Name: "n2",

				Type: "NGAP",

				Port: 38412,

				Protocol: "SCTP",

				ServiceType: "ClusterIP",
			},

			{

				Name: "sbi",

				Type: "HTTP",

				Port: 8080,

				Protocol: "TCP",

				ServiceType: "ClusterIP",
			},
		},

		ConfigMaps: []ConfigMapTemplate{

			{

				Name: "amf-config",

				Data: map[string]string{

					"amfcfg.yaml": generateAMFConfig(),
				},
			},
		},
	}

	// SMF Template.

	m.Templates[nephoranv1.CNFFunctionSMF] = &HelmTemplate{

		Function: nephoranv1.CNFFunctionSMF,

		ChartName: "smf",

		ChartVersion: "1.0.0",

		Repository: "https://charts.5g-core.io",

		Values: map[string]interface{}{

			"image": map[string]interface{}{

				"repository": "5gc/smf",

				"tag": "latest",

				"pullPolicy": "IfNotPresent",
			},

			"service": map[string]interface{}{

				"type": "ClusterIP",

				"ports": map[string]interface{}{

					"sbi": 8080,

					"pfcp": 8805,
				},
			},

			"config": map[string]interface{}{

				"plmnId": map[string]interface{}{

					"mcc": "001",

					"mnc": "01",
				},

				"nfInstanceId": "12345678-1234-1234-1234-123456789012",

				"dnn": "internet",

				"pfcp": map[string]interface{}{

					"addr": "0.0.0.0",

					"port": 8805,
				},
			},
		},

		Interfaces: []InterfaceTemplate{

			{

				Name: "sbi",

				Type: "HTTP",

				Port: 8080,

				Protocol: "TCP",

				ServiceType: "ClusterIP",
			},

			{

				Name: "n4",

				Type: "PFCP",

				Port: 8805,

				Protocol: "UDP",

				ServiceType: "ClusterIP",
			},
		},

		ConfigMaps: []ConfigMapTemplate{

			{

				Name: "smf-config",

				Data: map[string]string{

					"smfcfg.yaml": generateSMFConfig(),
				},
			},
		},
	}

	// UPF Template.

	m.Templates[nephoranv1.CNFFunctionUPF] = &HelmTemplate{

		Function: nephoranv1.CNFFunctionUPF,

		ChartName: "upf",

		ChartVersion: "1.0.0",

		Repository: "https://charts.5g-core.io",

		Values: map[string]interface{}{

			"image": map[string]interface{}{

				"repository": "5gc/upf",

				"tag": "latest",

				"pullPolicy": "IfNotPresent",
			},

			"service": map[string]interface{}{

				"type": "ClusterIP",

				"ports": map[string]interface{}{

					"pfcp": 8805,

					"gtpu": 2152,
				},
			},

			"config": map[string]interface{}{

				"dnn": "internet",

				"pfcp": map[string]interface{}{

					"addr": "0.0.0.0",

					"port": 8805,
				},

				"gtpu": map[string]interface{}{

					"addr": "0.0.0.0",

					"port": 2152,
				},
			},

			"dpdk": map[string]interface{}{

				"enabled": true,

				"cores": 4,

				"memory": 2048,

				"driver": "vfio-pci",
			},

			"resources": map[string]interface{}{

				"requests": map[string]interface{}{

					"memory": "4Gi",

					"cpu": "2",
				},

				"limits": map[string]interface{}{

					"memory": "8Gi",

					"cpu": "4",
				},
			},
		},

		Interfaces: []InterfaceTemplate{

			{

				Name: "n3",

				Type: "GTP-U",

				Port: 2152,

				Protocol: "UDP",

				ServiceType: "ClusterIP",
			},

			{

				Name: "n4",

				Type: "PFCP",

				Port: 8805,

				Protocol: "UDP",

				ServiceType: "ClusterIP",
			},

			{

				Name: "n6",

				Type: "Data",

				Port: 0,

				Protocol: "IP",

				ServiceType: "ClusterIP",
			},
		},

		ConfigMaps: []ConfigMapTemplate{

			{

				Name: "upf-config",

				Data: map[string]string{

					"upfcfg.yaml": generateUPFConfig(),
				},
			},
		},
	}

	// NRF Template.

	m.Templates[nephoranv1.CNFFunctionNRF] = &HelmTemplate{

		Function: nephoranv1.CNFFunctionNRF,

		ChartName: "nrf",

		ChartVersion: "1.0.0",

		Repository: "https://charts.5g-core.io",

		Values: map[string]interface{}{

			"image": map[string]interface{}{

				"repository": "5gc/nrf",

				"tag": "latest",

				"pullPolicy": "IfNotPresent",
			},

			"service": map[string]interface{}{

				"type": "ClusterIP",

				"ports": map[string]interface{}{

					"sbi": 8080,
				},
			},

			"config": map[string]interface{}{

				"nfInstanceId": "12345678-1234-1234-1234-123456789013",

				"database": map[string]interface{}{

					"type": "mongodb",

					"url": "mongodb://nrf-mongodb:27017",
				},
			},
		},

		Interfaces: []InterfaceTemplate{

			{

				Name: "sbi",

				Type: "HTTP",

				Port: 8080,

				Protocol: "TCP",

				ServiceType: "ClusterIP",
			},
		},

		ConfigMaps: []ConfigMapTemplate{

			{

				Name: "nrf-config",

				Data: map[string]string{

					"nrfcfg.yaml": generateNRFConfig(),
				},
			},
		},
	}

}

// initORANTemplates initializes Helm templates for O-RAN functions.

func (m *HelmTemplateManager) initORANTemplates() {

	// Near-RT RIC Template.

	m.Templates[nephoranv1.CNFFunctionNearRTRIC] = &HelmTemplate{

		Function: nephoranv1.CNFFunctionNearRTRIC,

		ChartName: "near-rt-ric",

		ChartVersion: "1.0.0",

		Repository: "https://charts.o-ran.io",

		Values: map[string]interface{}{

			"image": map[string]interface{}{

				"repository": "oran/near-rt-ric",

				"tag": "latest",

				"pullPolicy": "IfNotPresent",
			},

			"service": map[string]interface{}{

				"type": "ClusterIP",

				"ports": map[string]interface{}{

					"a1": 10000,

					"e2": 36421,
				},
			},

			"config": map[string]interface{}{

				"ricId": "12345",

				"plmnId": map[string]interface{}{

					"mcc": "001",

					"mnc": "01",
				},

				"a1": map[string]interface{}{

					"addr": "0.0.0.0",

					"port": 10000,
				},

				"e2": map[string]interface{}{

					"addr": "0.0.0.0",

					"port": 36421,
				},
			},
		},

		Interfaces: []InterfaceTemplate{

			{

				Name: "a1",

				Type: "REST",

				Port: 10000,

				Protocol: "TCP",

				ServiceType: "ClusterIP",
			},

			{

				Name: "e2",

				Type: "SCTP",

				Port: 36421,

				Protocol: "SCTP",

				ServiceType: "ClusterIP",
			},
		},

		ConfigMaps: []ConfigMapTemplate{

			{

				Name: "near-rt-ric-config",

				Data: map[string]string{

					"ric.yaml": generateNearRTRICConfig(),
				},
			},
		},
	}

	// O-DU Template.

	m.Templates[nephoranv1.CNFFunctionODU] = &HelmTemplate{

		Function: nephoranv1.CNFFunctionODU,

		ChartName: "o-du",

		ChartVersion: "1.0.0",

		Repository: "https://charts.o-ran.io",

		Values: map[string]interface{}{

			"image": map[string]interface{}{

				"repository": "oran/o-du",

				"tag": "latest",

				"pullPolicy": "IfNotPresent",
			},

			"service": map[string]interface{}{

				"type": "ClusterIP",

				"ports": map[string]interface{}{

					"f1c": 38472,

					"f1u": 2152,

					"e2": 36421,
				},
			},

			"config": map[string]interface{}{

				"duId": "1",

				"cellId": "1",

				"f1": map[string]interface{}{

					"addr": "0.0.0.0",

					"port": 38472,
				},

				"e2": map[string]interface{}{

					"addr": "0.0.0.0",

					"port": 36421,
				},
			},
		},

		Interfaces: []InterfaceTemplate{

			{

				Name: "f1c",

				Type: "F1AP",

				Port: 38472,

				Protocol: "SCTP",

				ServiceType: "ClusterIP",
			},

			{

				Name: "f1u",

				Type: "GTP-U",

				Port: 2152,

				Protocol: "UDP",

				ServiceType: "ClusterIP",
			},

			{

				Name: "e2",

				Type: "SCTP",

				Port: 36421,

				Protocol: "SCTP",

				ServiceType: "ClusterIP",
			},
		},

		ConfigMaps: []ConfigMapTemplate{

			{

				Name: "o-du-config",

				Data: map[string]string{

					"odu.yaml": generateODUConfig(),
				},
			},
		},
	}

	// O-CU-CP Template.

	m.Templates[nephoranv1.CNFFunctionOCUCP] = &HelmTemplate{

		Function: nephoranv1.CNFFunctionOCUCP,

		ChartName: "o-cu-cp",

		ChartVersion: "1.0.0",

		Repository: "https://charts.o-ran.io",

		Values: map[string]interface{}{

			"image": map[string]interface{}{

				"repository": "oran/o-cu-cp",

				"tag": "latest",

				"pullPolicy": "IfNotPresent",
			},

			"service": map[string]interface{}{

				"type": "ClusterIP",

				"ports": map[string]interface{}{

					"f1c": 38472,

					"ng": 38412,

					"xn": 36422,

					"e1": 36460,
				},
			},

			"config": map[string]interface{}{

				"cuId": "1",

				"cellId": "1",

				"f1": map[string]interface{}{

					"addr": "0.0.0.0",

					"port": 38472,
				},

				"ng": map[string]interface{}{

					"addr": "0.0.0.0",

					"port": 38412,
				},
			},
		},

		Interfaces: []InterfaceTemplate{

			{

				Name: "f1c",

				Type: "F1AP",

				Port: 38472,

				Protocol: "SCTP",

				ServiceType: "ClusterIP",
			},

			{

				Name: "ng",

				Type: "NGAP",

				Port: 38412,

				Protocol: "SCTP",

				ServiceType: "ClusterIP",
			},
		},
	}

}

// initEdgeTemplates initializes Helm templates for edge functions.

func (m *HelmTemplateManager) initEdgeTemplates() {

	// UE Simulator Template.

	m.Templates[nephoranv1.CNFFunctionUESimulator] = &HelmTemplate{

		Function: nephoranv1.CNFFunctionUESimulator,

		ChartName: "ue-simulator",

		ChartVersion: "1.0.0",

		Repository: "https://charts.edge.io",

		Values: map[string]interface{}{

			"image": map[string]interface{}{

				"repository": "edge/ue-simulator",

				"tag": "latest",

				"pullPolicy": "IfNotPresent",
			},

			"config": map[string]interface{}{

				"ues": map[string]interface{}{

					"count": 100,

					"imsiStart": "001010000000001",
				},

				"amf": map[string]interface{}{

					"addr": "amf-service",

					"port": 38412,
				},

				"gnb": map[string]interface{}{

					"addr": "gnb-service",

					"port": 38412,
				},
			},
		},
	}

}

// Configuration generators.

func generateAMFConfig() string {

	return `

info:

  version: 1.0.0

  description: AMF initial local configuration



configuration:

  amfName: AMF

  ngapIpList:

    - 0.0.0.0

  sbi:

    scheme: http

    registerIPv4: 0.0.0.0

    bindingIPv4: 0.0.0.0

    port: 8080

    tls:

      pem: cert/amf.pem

      key: cert/amf.key

  serviceNameList:

    - namf-comm

    - namf-evts

    - namf-mt

    - namf-loc

    - namf-oam

  servedGuamiList:

    - plmnId:

        mcc: 001

        mnc: 01

      amfId: cafe00

  supportTaiList:

    - plmnId:

        mcc: 001

        mnc: 01

      tac: 1

  plmnSupportList:

    - plmnId:

        mcc: 001

        mnc: 01

      snssaiList:

        - sst: 1

          sd: 010203

        - sst: 1

          sd: 112233

  supportDnnList:

    - internet

  nrfUri: http://nrf-service:8080

  security:

    integrityOrder:

      - NIA2

    cipheringOrder:

      - NEA0

  networkName:

    full: Nephoran Network

    short: Nephoran

  t3502Value: 720

  t3512Value: 3600

  non3gppDeregTimerValue: 3240

  t3513:

    enable: true

    expireTime: 6s

    maxRetryTimes: 4

  t3522:

    enable: true

    expireTime: 6s

    maxRetryTimes: 4

  t3550:

    enable: true

    expireTime: 6s

    maxRetryTimes: 4

  t3560:

    enable: true

    expireTime: 6s

    maxRetryTimes: 4

  t3565:

    enable: true

    expireTime: 6s

    maxRetryTimes: 4



logger:

  enable: true

  level: info

  reportCaller: false

`

}

func generateSMFConfig() string {

	return `

info:

  version: 1.0.0

  description: SMF initial local configuration



configuration:

  smfName: SMF

  sbi:

    scheme: http

    registerIPv4: 0.0.0.0

    bindingIPv4: 0.0.0.0

    port: 8080

    tls:

      pem: cert/smf.pem

      key: cert/smf.key

  serviceNameList:

    - nsmf-pdusession

    - nsmf-event-exposure

    - nsmf-oam

  snssaiInfos:

    - sNssai:

        sst: 1

        sd: 010203

      dnnInfos:

        - dnn: internet

          dns:

            ipv4: 8.8.8.8

            ipv6: 2001:4860:4860::8888

    - sNssai:

        sst: 1

        sd: 112233

      dnnInfos:

        - dnn: internet

          dns:

            ipv4: 8.8.8.8

            ipv6: 2001:4860:4860::8888

  plmnList:

    - mcc: "001"

      mnc: "01"

  pfcp:

    addr: 0.0.0.0

    port: 8805

  userplaneInformation:

    upNodes:

      gNB1:

        type: AN

        anIP: 192.168.179.131

      UPF:

        type: UPF

        nodeID: 0.0.0.0

    links:

      - A: gNB1

        B: UPF

  nrfUri: http://nrf-service:8080

  ulcl: false



logger:

  enable: true

  level: info

  reportCaller: false

`

}

func generateUPFConfig() string {

	return `

info:

  version: 1.0.0

  description: UPF initial local configuration



configuration:

  pfcp:

    addr: 0.0.0.0

    port: 8805

  gtpu:

    forwarder: gtp5g

    ifList:

      - addr: 0.0.0.0

        type: N3

        name: upf-n3

      - addr: 0.0.0.0

        type: N9

        name: upf-n9

  dnnList:

    - dnn: internet

      cidr: 10.60.0.0/16

      natifname: upf-n6



logger:

  enable: true

  level: info

  reportCaller: false

`

}

func generateNRFConfig() string {

	return `

info:

  version: 1.0.0

  description: NRF initial local configuration



configuration:

  MongoDBName: nephoran

  MongoDBUrl: mongodb://nrf-mongodb:27017

  sbi:

    scheme: http

    registerIPv4: 0.0.0.0

    bindingIPv4: 0.0.0.0

    port: 8080

    tls:

      pem: cert/nrf.pem

      key: cert/nrf.key

  DefaultPlmnId:

    mcc: 001

    mnc: 01

  serviceNameList:

    - nnrf-nfm

    - nnrf-disc



logger:

  enable: true

  level: info

  reportCaller: false

`

}

func generateNearRTRICConfig() string {

	return `

info:

  version: 1.0.0

  description: Near-RT RIC initial local configuration



configuration:

  ricId: 12345

  plmnId:

    mcc: 001

    mnc: 01

  a1:

    addr: 0.0.0.0

    port: 10000

  e2:

    addr: 0.0.0.0

    port: 36421

  xApps:

    enabled: true

    registry: https://xapp-registry.o-ran.io

  database:

    type: redis

    addr: near-rt-ric-redis:6379

  messaging:

    type: rmr

    port: 4560



logger:

  enable: true

  level: info

  reportCaller: false

`

}

func generateODUConfig() string {

	return `

info:

  version: 1.0.0

  description: O-DU initial local configuration



configuration:

  duId: 1

  cellId: 1

  f1:

    addr: 0.0.0.0

    port: 38472

  e2:

    addr: 0.0.0.0

    port: 36421

    ricAddr: near-rt-ric-service

    ricPort: 36421

  phyLayer:

    enabled: true

    type: oru

  fronthaul:

    enabled: true

    type: cpri

    interfaces:

      - name: fh0

        vlan: 100



logger:

  enable: true

  level: info

  reportCaller: false

`

}

// GetHelmChart returns the Helm chart reference for a CNF function.

func (m *HelmTemplateManager) GetHelmChart(function nephoranv1.CNFFunction) (*nephoranv1.HelmConfig, error) {

	template, err := m.GetTemplate(function)

	if err != nil {

		return nil, err

	}

	return &nephoranv1.HelmConfig{

		Repository: template.Repository,

		ChartName: template.ChartName,

		ChartVersion: template.ChartVersion,
	}, nil

}

// ValidateTemplate validates a Helm template configuration.

func (m *HelmTemplateManager) ValidateTemplate(template *HelmTemplate) error {

	if template == nil {

		return fmt.Errorf("template is nil")

	}

	if template.Function == "" {

		return fmt.Errorf("template function is empty")

	}

	if template.ChartName == "" {

		return fmt.Errorf("template chart name is empty")

	}

	if template.Repository == "" {

		return fmt.Errorf("template repository is empty")

	}

	return nil

}

// GetSupportedFunctions returns a list of supported CNF functions.

func (m *HelmTemplateManager) GetSupportedFunctions() []nephoranv1.CNFFunction {

	functions := make([]nephoranv1.CNFFunction, 0, len(m.Templates))

	for function := range m.Templates {

		functions = append(functions, function)

	}

	return functions

}

// GetFunctionInfo returns information about a CNF function.

func (m *HelmTemplateManager) GetFunctionInfo(function nephoranv1.CNFFunction) (map[string]interface{}, error) {

	template, err := m.GetTemplate(function)

	if err != nil {

		return nil, err

	}

	info := map[string]interface{}{

		"function": template.Function,

		"chartName": template.ChartName,

		"chartVersion": template.ChartVersion,

		"repository": template.Repository,

		"interfaces": len(template.Interfaces),

		"configMaps": len(template.ConfigMaps),

		"secrets": len(template.Secrets),

		"services": len(template.Services),
	}

	return info, nil

}
