package o1

import (
	"context"
	"fmt"

	"github.com/openshift/go-netconf/netconf"
	"golang.org/x/crypto/ssh"

	nephoranv1alpha1 "nephoran-intent-operator/pkg/apis/nephoran/v1alpha1"
)

// O1Adaptor is responsible for handling O1 interface communications.
type O1Adaptor struct {
	// In a real implementation, you would manage a pool of sessions.
}

// NewO1Adaptor creates a new O1Adaptor.
func NewO1Adaptor() *O1Adaptor {
	return &O1Adaptor{}
}

// ApplyConfiguration connects to a ManagedElement and applies a configuration.
func (a *O1Adaptor) ApplyConfiguration(ctx context.Context, me *nephoranv1alpha1.ManagedElement, config string) error {
	if me.Spec.O1.Host == "" || me.Spec.O1.Port == 0 {
		return fmt.Errorf("O1 configuration for ManagedElement %s is incomplete", me.Name)
	}

	// In a real implementation, you would fetch user/password from a secret.
	sshConfig := &ssh.ClientConfig{
		User: "netconf-user",
		Auth: []ssh.AuthMethod{
			ssh.Password("netconf-password"),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	addr := fmt.Sprintf("%s:%d", me.Spec.O1.Host, me.Spec.O1.Port)
	session, err := netconf.DialSSH(addr, sshConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to NETCONF server for %s: %w", me.Name, err)
	}
	defer session.Close()

	// Send a basic <edit-config> operation.
	// The `config` variable would contain a valid YANG/XML payload.
	reply, err := session.EditConfig(netconf.Running, config)
	if err != nil {
		return fmt.Errorf("failed to apply configuration for %s: %w", me.Name, err)
	}

	if !reply.OK {
		return fmt.Errorf("failed to apply configuration for %s: NETCONF reply contains errors: %v", me.Name, reply.Errors)
	}

	return nil
}
