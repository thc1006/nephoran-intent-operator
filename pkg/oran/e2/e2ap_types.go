package e2

// E2AP Additional Type Definitions following O-RAN.WG3.E2AP-v03.01.

// This file contains only truly unique types that are not duplicated anywhere else.

// E2NodeType represents the type of E2 node (unique to this file).

type E2NodeType int

const (

	// E2NodeTypegNB holds e2nodetypegnb value.

	E2NodeTypegNB E2NodeType = iota

	// E2NodeTypeeNB holds e2nodetypeenb value.

	E2NodeTypeeNB

	// E2NodeTypeNgENB holds e2nodetypengenb value.

	E2NodeTypeNgENB

	// E2NodeTypeEnGNB holds e2nodetypeengnb value.

	E2NodeTypeEnGNB
)

// InterfaceDirection represents the direction of an interface.

type InterfaceDirection int

const (

	// InterfaceDirectionIncoming holds interfacedirectionincoming value.

	InterfaceDirectionIncoming InterfaceDirection = iota

	// InterfaceDirectionOutgoing holds interfacedirectionoutgoing value.

	InterfaceDirectionOutgoing
)

// TLSConfig represents TLS configuration.

type TLSConfig struct {
	Enabled bool `json:"enabled"`

	CertFile string `json:"cert_file,omitempty"`

	KeyFile string `json:"key_file,omitempty"`

	CAFile string `json:"ca_file,omitempty"`

	InsecureSkipVerify bool `json:"insecure_skip_verify"`
}

// Local types needed by ASN1 codec that were removed from original duplicates.

// These have different structures from the canonical E2AP message types.

// RANFunctionIDCauseItem needed by ASN1 codec.

type RANFunctionIDCauseItem struct {
	RANFunctionID int `json:"ran_function_id"`

	Cause E2Cause `json:"cause"`
}

// E2Cause needed by ASN1 codec (local version with different structure).

type E2Cause struct {
	CauseType E2CauseType `json:"cause_type"`

	CauseValue int `json:"cause_value"`
}

// E2CauseType for ASN1 codec.

type E2CauseType int

const (

	// E2CauseTypeRIC holds e2causetyperic value.

	E2CauseTypeRIC E2CauseType = iota

	// E2CauseTypeRICService holds e2causetypericservice value.

	E2CauseTypeRICService

	// E2CauseTypeE2Node holds e2causetypee2node value.

	E2CauseTypeE2Node

	// E2CauseTypeTransport holds e2causetypetransport value.

	E2CauseTypeTransport

	// E2CauseTypeProtocol holds e2causetypeprotocol value.

	E2CauseTypeProtocol

	// E2CauseTypeMisc holds e2causetypemisc value.

	E2CauseTypeMisc
)

// RICActionAdmittedItem needed by ASN1 codec.

type RICActionAdmittedItem struct {
	ActionID int `json:"action_id"`
}
