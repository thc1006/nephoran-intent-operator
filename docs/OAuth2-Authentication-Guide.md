# OAuth2 Authentication Integration Guide

## Overview

The Nephoran Intent Operator now supports enterprise-grade OAuth2 authentication with support for multiple identity providers including Azure AD, Okta, Keycloak, and Google OAuth2. This guide provides comprehensive setup and configuration instructions.

## Supported Identity Providers

### 1. Azure Active Directory (Azure AD)
- **Use Case**: Enterprise environments using Microsoft 365 or Azure
- **Features**: Group-based role mapping, conditional access support
- **Required Information**:
  - Tenant ID
  - Client ID (Application ID)
  - Client Secret

### 2. Okta
- **Use Case**: Organizations using Okta for identity management
- **Features**: Group claims, custom scopes, MFA support
- **Required Information**:
  - Okta Domain (e.g., `company.okta.com`)
  - Client ID
  - Client Secret

### 3. Keycloak
- **Use Case**: Self-hosted identity management or on-premises deployments
- **Features**: Custom realms, role-based access control
- **Required Information**:
  - Keycloak Base URL
  - Realm Name
  - Client ID
  - Client Secret

### 4. Google OAuth2
- **Use Case**: Organizations using Google Workspace
- **Features**: Google Groups integration, OAuth2 compliance
- **Required Information**:
  - Google Client ID
  - Google Client Secret

## Configuration Steps

### Step 1: Enable Authentication

Edit the OAuth2 configuration in `deployments/kustomize/base/llm-processor/oauth2-config.yaml`:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: oauth2-environment
  namespace: nephoran-system
data:
  AUTH_ENABLED: "true"        # Enable OAuth2 authentication
  REQUIRE_AUTH: "true"        # Require authentication for protected endpoints
  RBAC_ENABLED: "true"        # Enable role-based access control
```

### Step 2: Configure Identity Provider

#### For Azure AD:

1. **Register Application in Azure AD**:
   ```bash
   # Using Azure CLI
   az ad app create --display-name "Nephoran Intent Operator" \
     --reply-urls "https://your-domain.com/auth/callback/azure-ad"
   ```

2. **Update Configuration**:
   ```yaml
   data:
     AZURE_ENABLED: "true"
     AZURE_CLIENT_ID: "your-application-id"
     AZURE_TENANT_ID: "your-tenant-id"
     AZURE_SCOPES: "openid,profile,email,User.Read"
   ```

3. **Set Client Secret**:
   ```yaml
   apiVersion: v1
   kind: Secret
   metadata:
     name: oauth2-secrets
   stringData:
     azure-client-secret: "your-azure-client-secret"
   ```

#### For Okta:

1. **Create Application in Okta Admin Console**:
   - Application Type: Web Application
   - Grant Types: Authorization Code
   - Redirect URI: `https://your-domain.com/auth/callback/okta`

2. **Update Configuration**:
   ```yaml
   data:
     OKTA_ENABLED: "true"
     OKTA_CLIENT_ID: "your-okta-client-id"
     OKTA_DOMAIN: "your-company.okta.com"
     OKTA_SCOPES: "openid,profile,email,groups"
   ```

#### For Keycloak:

1. **Create Client in Keycloak Admin Console**:
   - Client Type: OpenID Connect
   - Access Type: Confidential
   - Valid Redirect URIs: `https://your-domain.com/auth/callback/keycloak`

2. **Update Configuration**:
   ```yaml
   data:
     KEYCLOAK_ENABLED: "true"
     KEYCLOAK_CLIENT_ID: "nephoran-intent-operator"
     KEYCLOAK_BASE_URL: "https://keycloak.company.com"
     KEYCLOAK_REALM: "telecom"
   ```

### Step 3: Configure Role-Based Access Control (RBAC)

The system supports multiple role levels:

- **Admin Roles**: `admin`, `system-admin`, `nephoran-admin`
- **Operator Roles**: `operator`, `network-operator`, `telecom-operator`
- **Read-Only Roles**: `viewer`, `readonly`, `guest`

#### Role Mapping Configuration:

```yaml
rbac:
  role_mapping:
    # Map provider roles to internal roles
    "Global Administrator": ["admin"]
    "Network Administrator": ["operator"]
    "Telecom Operator": ["network-operator"]
    "Read Only": ["viewer"]
  
  group_mapping:
    # Map provider groups to internal roles
    "Nephoran-Admins": ["admin"]
    "Telecom-Operators": ["operator"]
    "Network-Engineers": ["network-operator"]
    "Viewers": ["viewer"]
```

#### Permission Matrix:

| Role | Intent Create | Intent Read | Intent Update | Intent Delete | E2 Nodes | System Admin |
|------|--------------|-------------|---------------|---------------|----------|--------------|
| admin | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| operator | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ |
| network-operator | ✅ | ✅ | ✅ | ❌ | ✅ | ❌ |
| viewer | ❌ | ✅ | ❌ | ❌ | ❌ | ❌ |

### Step 4: Deploy with Authentication

1. **Apply OAuth2 Configuration**:
   ```bash
   kubectl apply -f deployments/kustomize/base/llm-processor/oauth2-config.yaml
   ```

2. **Update Secrets**:
   ```bash
   # Generate secure JWT secret key
   JWT_SECRET=$(openssl rand -base64 32)
   kubectl create secret generic oauth2-secrets \
     --from-literal=jwt-secret-key="$JWT_SECRET" \
     --from-literal=azure-client-secret="your-azure-secret" \
     -n nephoran-system
   ```

3. **Deploy Services**:
   ```bash
   kubectl apply -k deployments/kustomize/base/llm-processor/
   ```

## API Usage Examples

### 1. OAuth2 Login Flow

#### Initiate Login:
```bash
curl "https://llm-processor.nephoran.com/auth/login/azure-ad"
```

This redirects to the identity provider for authentication.

#### Handle Callback:
After successful authentication, the user is redirected to:
```
https://llm-processor.nephoran.com/auth/callback/azure-ad?code=...&state=...
```

The callback returns a JWT token:
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "Bearer",
  "expires_in": 86400,
  "user_info": {
    "subject": "user@company.com",
    "email": "user@company.com",
    "name": "John Doe",
    "groups": ["Telecom-Operators"],
    "roles": ["operator"],
    "provider": "azure-ad"
  }
}
```

### 2. Making Authenticated Requests

#### Process Intent (Requires Operator Role):
```bash
curl -X POST https://llm-processor.nephoran.com/process \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -H "Content-Type: application/json" \
  -d '{
    "intent": "Deploy AMF with 3 replicas in production",
    "metadata": {
      "priority": "high",
      "environment": "production"
    }
  }'
```

#### Get System Status (Requires Admin Role):
```bash
curl -X GET https://llm-processor.nephoran.com/admin/status \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

### 3. Token Refresh

```bash
curl -X POST https://llm-processor.nephoran.com/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "your-refresh-token",
    "provider": "azure-ad"
  }'
```

### 4. User Information

```bash
curl -X GET https://llm-processor.nephoran.com/auth/userinfo \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

## Security Best Practices

### 1. JWT Secret Key Management
- **Minimum Length**: 256 bits (32 characters)
- **Storage**: Kubernetes Secret with restricted RBAC
- **Rotation**: Regular rotation (recommended: every 90 days)

```bash
# Generate secure secret key
openssl rand -base64 32

# Update secret
kubectl patch secret oauth2-secrets -n nephoran-system \
  --type merge -p='{"data":{"jwt-secret-key":"'$(echo -n "new-secret-key" | base64)'"}}'
```

### 2. Client Secret Protection
- Store OAuth2 client secrets in Kubernetes Secrets
- Use separate secrets for different environments
- Implement secret rotation policies

### 3. Network Security
- Enable TLS for all authentication endpoints
- Use network policies to restrict access
- Implement rate limiting on authentication endpoints

### 4. Access Control
- Follow principle of least privilege
- Regular access reviews and role audits
- Monitor authentication events and failures

## Troubleshooting

### Common Issues

#### 1. Authentication Failed
```
Error: Authentication failed
```

**Solutions**:
- Verify OAuth2 provider configuration
- Check client ID and secret
- Validate redirect URLs
- Review provider-specific settings

#### 2. Invalid JWT Token
```
Error: Invalid token: token is malformed
```

**Solutions**:
- Check JWT secret key configuration
- Verify token hasn't expired
- Validate token format and signing method

#### 3. Access Denied
```
Error: Required roles: [operator]
```

**Solutions**:
- Verify user role mapping
- Check group membership in identity provider
- Review RBAC configuration

#### 4. Provider Not Found
```
Error: Unknown provider
```

**Solutions**:
- Verify provider is enabled in configuration
- Check provider name spelling
- Ensure provider configuration is complete

### Debug Commands

#### Check OAuth2 Configuration:
```bash
kubectl get configmap oauth2-config -n nephoran-system -o yaml
kubectl get configmap oauth2-environment -n nephoran-system -o yaml
```

#### Verify Secrets:
```bash
kubectl get secret oauth2-secrets -n nephoran-system -o yaml
```

#### Check Service Logs:
```bash
kubectl logs -l app=llm-processor -n nephoran-system --tail=100
```

#### Test Authentication Endpoints:
```bash
# Health check (should work without authentication)
curl https://llm-processor.nephoran.com/healthz

# Protected endpoint (should require authentication)
curl https://llm-processor.nephoran.com/process
```

## Environment Variables Reference

### Core Authentication Settings
- `AUTH_ENABLED`: Enable/disable OAuth2 authentication
- `REQUIRE_AUTH`: Require authentication for protected endpoints
- `JWT_SECRET_KEY`: Secret key for JWT token signing
- `RBAC_ENABLED`: Enable role-based access control

### Provider-Specific Settings

#### Azure AD:
- `AZURE_ENABLED`: Enable Azure AD provider
- `AZURE_CLIENT_ID`: Azure AD application ID
- `AZURE_CLIENT_SECRET`: Azure AD client secret
- `AZURE_TENANT_ID`: Azure AD tenant ID
- `AZURE_SCOPES`: OAuth2 scopes (comma-separated)

#### Okta:
- `OKTA_ENABLED`: Enable Okta provider
- `OKTA_CLIENT_ID`: Okta client ID
- `OKTA_CLIENT_SECRET`: Okta client secret
- `OKTA_DOMAIN`: Okta domain (e.g., company.okta.com)
- `OKTA_SCOPES`: OAuth2 scopes (comma-separated)

#### Keycloak:
- `KEYCLOAK_ENABLED`: Enable Keycloak provider
- `KEYCLOAK_CLIENT_ID`: Keycloak client ID
- `KEYCLOAK_CLIENT_SECRET`: Keycloak client secret
- `KEYCLOAK_BASE_URL`: Keycloak server URL
- `KEYCLOAK_REALM`: Keycloak realm name

#### Google:
- `GOOGLE_ENABLED`: Enable Google OAuth2 provider
- `GOOGLE_CLIENT_ID`: Google client ID
- `GOOGLE_CLIENT_SECRET`: Google client secret
- `GOOGLE_SCOPES`: OAuth2 scopes (comma-separated)

## Production Deployment Checklist

- [ ] OAuth2 provider configured and tested
- [ ] Client secrets stored securely in Kubernetes Secrets
- [ ] JWT secret key generated with sufficient entropy
- [ ] Role mappings configured correctly
- [ ] TLS certificates configured for HTTPS endpoints
- [ ] Network policies restricting access to authentication services
- [ ] Monitoring and alerting configured for authentication failures
- [ ] Backup and recovery procedures documented
- [ ] User access reviews scheduled
- [ ] Security audit completed

## Integration with Existing Services

### Kubernetes RBAC Integration
The OAuth2 system integrates with Kubernetes RBAC for additional access control:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: nephoran-operators
  namespace: nephoran-system
subjects:
- kind: User
  name: system:serviceaccount:nephoran-system:llm-processor
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: Role
  name: nephoran-operator
  apiGroup: rbac.authorization.k8s.io
```

### Monitoring Integration
Authentication events are automatically logged and can be monitored:

```bash
# View authentication logs
kubectl logs -l app=llm-processor -n nephoran-system | grep "auth"

# Monitor authentication metrics via Prometheus
curl http://prometheus:9090/api/v1/query?query=auth_requests_total
```

This OAuth2 integration provides enterprise-grade authentication and authorization for the Nephoran Intent Operator, ensuring secure access control while maintaining operational efficiency.