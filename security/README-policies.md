# Security Policies

## OPA Policy Evaluation

The CI pipeline evaluates Open Policy Agent (OPA) policies in `security/policies/*.rego` using the configuration file `security/configs/security-config.json`.

### Adding .rego Policies

1. Place `.rego` files in `security/policies/`
2. Policies should define `data.security.allow` rules
3. The input configuration is provided via `security-config.json`

### Expected Input Structure

```json
{
  "environment": "production",
  "security_level": "standard", 
  "policies": {
    "enabled": true,
    "enforcement": "warn"
  }
}
```

### Example Policy

```rego
package security

default allow = false

allow {
    input.policies.enabled == true
    input.security_level in ["standard", "high"]
}
```

The CI will gracefully handle missing policies or configuration files with warnings.