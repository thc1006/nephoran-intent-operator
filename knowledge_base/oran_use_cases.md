# O-RAN Alliance - Use Cases and Requirements

## 4.1 Traffic Steering
The O-RAN architecture enables intelligent traffic steering across different cell sites and radio access technologies. The Near-Real-Time RIC (RAN Intelligent Controller) is responsible for making traffic steering decisions based on policies (A1) and network conditions.

A common A1 policy for traffic steering is of type 'TS-2'. It requires parameters such as 'slice_dnn' and 'priority_level'.
Example A1 Policy for a UPF:
{
  "policy_type_id": "TS-2",
  "policy_data": {
    "slice_dnn": "enterprise_slice_1",
    "priority_level": 5
  }
}

## 6.3 UPF Configuration
The User Plane Function (UPF) is a critical component for routing user data. Its configuration is managed via the O1 interface. The configuration is typically provided as an XML payload.
A default UPF deployment should use the image 'registry.example.com/5g/upf:4.5.6' and 2 replicas.
A sample O1 configuration for a UPF is:
<config>
  <interface name="N3">
    <address>10.0.0.1</address>
  </interface>
  <interface name="N6">
    <address>192.168.1.1</address>
  </interface>
</config>
