# E2Manager API Migration Guide

## SendControlMessage Method Signature Change

### Breaking Change Notice
The `SendControlMessage` method signature has been updated to explicitly require a `nodeID` parameter for better control and clarity.

### Old Signature (Deprecated)
```go
func (m *E2Manager) SendControlMessage(ctx context.Context, controlReq *RICControlRequest) (*RICControlAcknowledge, error)
```

### New Signature
```go
func (m *E2Manager) SendControlMessage(ctx context.Context, nodeID string, controlReq *RICControlRequest) (*RICControlAcknowledge, error)
```

### Migration Path

#### Option 1: Use the Legacy Method (Temporary)
For immediate compatibility, use the `SendControlMessageLegacy` method which maintains the old signature:

```go
// Old code
ack, err := e2Manager.SendControlMessage(ctx, controlReq)

// Temporary fix - use legacy method
ack, err := e2Manager.SendControlMessageLegacy(ctx, controlReq)
```

**Note**: This method is deprecated and will be removed in a future version. It logs a deprecation warning on each use.

#### Option 2: Update to New Signature (Recommended)
Update your code to explicitly provide the target node ID:

```go
// Old code
ack, err := e2Manager.SendControlMessage(ctx, controlReq)

// New code - explicit node ID
nodeID := "your-target-node-id"
ack, err := e2Manager.SendControlMessage(ctx, nodeID, controlReq)
```

### Finding the Node ID

If you were relying on automatic node ID extraction from the control header, you can:

1. **Extract it manually before calling**:
```go
nodeID := e2Manager.extractNodeIDFromControlHeader(controlReq.RICControlHeader)
if nodeID == "" {
    // Handle missing node ID
}
ack, err := e2Manager.SendControlMessage(ctx, nodeID, controlReq)
```

2. **Get available nodes**:
```go
nodes := e2Manager.GetConnectedNodes()
if len(nodes) > 0 {
    nodeID := nodes[0] // Or implement selection logic
    ack, err := e2Manager.SendControlMessage(ctx, nodeID, controlReq)
}
```

### Benefits of the Change

1. **Explicit Control**: Callers now have explicit control over which node receives the control message
2. **Better Error Handling**: Invalid node IDs are caught immediately
3. **Improved Debugging**: Clear visibility of target nodes in logs
4. **Performance**: No need to parse headers when the node ID is already known

### Timeline

- **Current Release**: Both methods available, legacy method logs deprecation warnings
- **Next Minor Release**: Legacy method marked as deprecated in documentation
- **Next Major Release**: Legacy method will be removed

### Example Migration

Before:
```go
func sendControlToNode(ctx context.Context, e2mgr *E2Manager, req *RICControlRequest) error {
    // Old approach - node ID hidden in control header
    ack, err := e2mgr.SendControlMessage(ctx, req)
    if err != nil {
        return fmt.Errorf("control message failed: %w", err)
    }
    // Process acknowledgment
    return nil
}
```

After:
```go
func sendControlToNode(ctx context.Context, e2mgr *E2Manager, nodeID string, req *RICControlRequest) error {
    // New approach - explicit node ID
    ack, err := e2mgr.SendControlMessage(ctx, nodeID, req)
    if err != nil {
        return fmt.Errorf("control message to node %s failed: %w", nodeID, err)
    }
    // Process acknowledgment
    return nil
}
```

### Questions or Issues?

If you encounter any issues during migration, please:
1. Check the [API documentation](../../../docs/API_REFERENCE.md)
2. Review the [examples](../examples/)
3. Open an issue on the project repository