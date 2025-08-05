# XPath Validation Documentation

## validateXPathCondition Function Documentation

This document provides comprehensive documentation for the `validateXPathCondition` function that would be implemented in `pkg/oran/o1/yang_models.go`.

### Function Signature
```go
func (sv *StandardYANGValidator) validateXPathCondition(condition string, nodeSchema interface{}) error
```

### Purpose
Validates XPath condition syntax against YANG schema definitions to ensure conditions are syntactically correct and semantically valid within the context of the given schema node.

### Supported XPath Condition Patterns

#### 1. Basic Comparison Operators
```xpath
// Equality
current() = 'value'
../sibling-node = 42

// Inequality
current() != 'value'
../sibling-node != 0

// Numeric comparisons
count(../node-list) > 5
string-length(current()) <= 255
../numeric-leaf >= 0
../numeric-leaf < 100
```

#### 2. Logical Operators
```xpath
// AND conditions
current() = 'active' and ../enabled = 'true'

// OR conditions
current() = 'primary' or current() = 'secondary'

// NOT conditions
not(../deprecated = 'true')

// Complex combinations
(current() = 'custom' and ../type != 'default') or ../override = 'true'
```

#### 3. String Functions
```xpath
// String matching
starts-with(current(), 'prefix-')
contains(../description, 'keyword')
string-length(current()) > 0

// String manipulation
substring(current(), 1, 3) = 'ABC'
concat(../prefix, '-', current()) = 'full-name'
```

#### 4. Node Set Functions
```xpath
// Counting nodes
count(../interface[enabled='true']) > 0

// Existence checks
../backup-config
not(../deprecated-feature)

// Position checks
position() = 1
last()
```

#### 5. Path Expressions
```xpath
// Relative paths
../sibling-node
../../parent-sibling/child
./child-node

// Absolute paths (from root)
/config/system/hostname

// Descendant paths
.//any-descendant
../container//specific-leaf
```

#### 6. YANG-Specific Patterns
```xpath
// Leafref validation
current() = /interfaces/interface[name=current()/../interface-ref]/type

// Must statement patterns
../start-time < ../end-time

// When statement patterns
../protocol = 'bgp'

// Unique constraint patterns
count(../item[id=current()/id]) = 1
```

### Validation Rules

#### 1. Syntax Validation
- **Well-formed XPath**: Expression must be syntactically valid XPath 1.0
- **Balanced parentheses**: All opening parentheses must have matching closing ones
- **Valid operators**: Only supported XPath operators are allowed
- **Function names**: Function calls must use valid XPath function names

#### 2. Schema Context Validation
- **Path validity**: Referenced paths must exist in the YANG schema
- **Type compatibility**: Comparisons must be between compatible types
- **Node existence**: Referenced nodes must be defined in the schema
- **Cardinality**: Path expressions must respect schema cardinality

#### 3. YANG Compliance
- **Namespace awareness**: Prefixed names must resolve to valid namespaces
- **Feature dependencies**: Conditions on feature-dependent nodes are validated
- **Type restrictions**: Values must comply with YANG type restrictions
- **Range/length constraints**: Numeric and string values must respect constraints

### Error Handling

The function returns specific error types for different validation failures:

```go
// Syntax errors
ErrInvalidXPathSyntax: "invalid XPath syntax: %s"

// Schema errors
ErrNodeNotFound: "node '%s' not found in schema"
ErrTypeMismatch: "type mismatch: cannot compare %s with %s"

// YANG-specific errors
ErrInvalidLeafref: "leafref path '%s' does not resolve to a valid node"
ErrConstraintViolation: "value violates constraint: %s"
```

### Examples

#### Valid Conditions
```go
// Simple comparison
"current() = 'enabled'"

// Complex business rule
"../protocol = 'ospf' and count(../area[area-id='0.0.0.0']) = 1"

// Leafref validation
"current() = /routing/ospf/area[area-id=current()/../area-ref]/type"
```

#### Invalid Conditions
```go
// Syntax error
"current() == 'value'"  // Error: Invalid operator '=='

// Schema error
"../nonexistent-node = 'value'"  // Error: Node not found

// Type error
"../numeric-leaf = 'string'"  // Error: Type mismatch
```

### Best Practices

1. **Use current() for self-reference**: Always use `current()` instead of `.` for clarity
2. **Prefer relative paths**: Use relative paths when possible for better reusability
3. **Validate early**: Validate conditions during schema compilation, not runtime
4. **Clear error messages**: Provide context about which part of the condition failed
5. **Performance considerations**: Avoid complex expressions in frequently evaluated conditions

### Implementation Notes

- The validator should cache parsed expressions for performance
- Support for XPath 2.0 features should be clearly documented if added
- Custom YANG functions should be registered with the XPath evaluator
- Schema context must be properly maintained during recursive validation

This documentation should be updated as new patterns are supported or validation rules change.