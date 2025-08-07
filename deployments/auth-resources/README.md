# Nephoran Intent Operator Authentication Integration

## Overview

This integration provides comprehensive OAuth2 multi-provider authentication for the Nephoran Intent Operator, including:

- OAuth2 authentication with GitHub, Google, Azure AD
- LDAP/Active Directory integration  
- JWT token management with rotation
- Role-based access control (RBAC)
- Session management with SSO support
- Comprehensive security features (PKCE, CSRF, rate limiting)
- Audit logging and monitoring

## Quick Start

1. Deploy authentication resources:
```bash
kubectl apply -f namespace.yaml
kubectl apply -f rbac.yaml  
kubectl apply -f secrets.yaml
kubectl apply -f configmap.yaml
kubectl apply -f deployment.yaml
kubectl apply -f service.yaml
```

2. Configure OAuth2 providers in secrets.yaml
3. Update environment variables in configmap.yaml
4. Deploy the updated main applications with authentication

## Configuration

See the configuration files in this directory for detailed setup instructions.

## Security Features

- Multi-factor authentication support
- Token blacklisting and rotation
- Rate limiting and DDoS protection
- Audit logging for compliance
- HTTPS/TLS enforcement options
- CSRF protection
- Session security with secure cookies

## Integration Points

The authentication system integrates with:
- HTTP API endpoints (/process, /admin, /stream)
- Kubernetes controllers (NetworkIntent, E2NodeSet, etc.)
- Resource-level RBAC and ownership
- Audit trail for all operations
- Metrics and monitoring

## Production Deployment

For production deployment:
1. Replace default secrets with strong keys
2. Configure proper TLS certificates  
3. Set up OAuth2 providers properly
4. Configure LDAP if using Active Directory
5. Enable audit logging
6. Set up monitoring and alerting

## Multi-tenancy

The system supports multi-tenant deployments with:
- Namespace-based isolation
- User-specific resource filtering
- Tenant-aware RBAC policies
- Per-tenant audit trails

