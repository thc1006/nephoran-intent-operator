# Contributing to Nephoran Intent Operator

## Welcome Contributors! 🚀

We're excited that you're interested in contributing to the Nephoran Intent Operator project! This is an open-source initiative exploring how Large Language Models can transform telecommunications network operations through intent-driven automation.

Whether you're a telecommunications expert, a Kubernetes developer, an AI/ML practitioner, or someone passionate about network automation, there are many ways you can contribute to this innovative project.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [How to Contribute](#how-to-contribute)
- [Development Guidelines](#development-guidelines)
- [Contribution Types](#contribution-types)
- [Pull Request Process](#pull-request-process)
- [Community Guidelines](#community-guidelines)
- [Recognition and Rewards](#recognition-and-rewards)

## Code of Conduct

By participating in this project, you agree to abide by our [Code of Conduct](CODE_OF_CONDUCT.md). We are committed to providing a welcoming and inspiring community for all. Please take a moment to read through our community standards.

## Getting Started

### Prerequisites

Before contributing, ensure you have:

- **Go 1.23+** installed
- **Docker** and **Kubernetes** access (kind, minikube, or cloud)
- **Git** configured with your GitHub account
- Basic understanding of Kubernetes operators
- (Optional) Telecommunications or O-RAN knowledge

### Development Environment Setup

1. **Fork and Clone**
   ```bash
   git clone https://github.com/[your-username]/nephoran-intent-operator
   cd nephoran-intent-operator
   ```

2. **Install Dependencies**
   ```bash
   make deps
   go mod tidy
   ```

3. **Set up Development Environment**
   ```bash
   # Set required environment variables
   export OPENAI_API_KEY="sk-your-development-key"
   export LOG_LEVEL="debug"
   
   # Create local development cluster
   kind create cluster --name nephoran-dev
   ```

4. **Run Tests**
   ```bash
   make test
   go test ./pkg/...
   ```

5. **Build and Deploy Locally**
   ```bash
   make build
   make docker-build
   make deploy-dev
   ```

### First Contribution Ideas

Great first contributions for new contributors:

- 📖 **Documentation**: Fix typos, improve clarity, add examples
- 🐛 **Bug Reports**: File detailed bug reports with reproduction steps
- ✨ **Examples**: Add new NetworkIntent examples for different use cases
- 🧪 **Tests**: Improve test coverage for existing components
- 🌐 **Knowledge Base**: Contribute telecommunications domain knowledge
- 🏗️ **Tooling**: Improve development scripts and automation

## How to Contribute

### 1. Choose Your Contribution Type

We welcome various types of contributions:

**Code Contributions:**
- Bug fixes and improvements
- New features and enhancements
- Performance optimizations
- Test coverage improvements

**Documentation Contributions:**
- API documentation improvements
- Tutorial and guide creation
- Architecture documentation
- Translation to other languages

**Knowledge Base Contributions:**
- Telecommunications standards documentation
- Best practices and patterns
- Troubleshooting guides
- Integration examples

**Community Contributions:**
- Blog posts and articles
- Conference presentations
- Community event organization
- Mentoring new contributors

### 2. Find an Issue to Work On

Check our issue tracker for opportunities:

- 🏷️ **good first issue**: Perfect for newcomers
- 🏷️ **help wanted**: Issues where we need community help
- 🏷️ **documentation**: Documentation improvements needed
- 🏷️ **enhancement**: Feature requests and improvements
- 🏷️ **bug**: Confirmed bugs that need fixing

### 3. Discuss Before Large Changes

For significant changes:
1. Open an issue to discuss your proposal
2. Get feedback from maintainers
3. Create a design document if needed
4. Start implementation after approval

## Development Guidelines

### Code Quality Standards

**Go Code Standards:**
- Follow [Effective Go](https://golang.org/doc/effective_go.html) guidelines
- Use `gofmt` and `golint` for code formatting
- Write comprehensive unit tests (aim for 80%+ coverage)
- Include integration tests for complex features
- Use meaningful variable and function names
- Add clear comments for complex logic

**Example Code Style:**
```go
// NetworkIntentProcessor handles the processing of network intents
// through LLM integration and validation workflows.
type NetworkIntentProcessor struct {
    llmClient    LLMClient
    ragService   RAGService
    validator    IntentValidator
    logger       logr.Logger
}

// ProcessIntent translates a natural language intent into deployment specifications.
// It returns the processed result and any errors encountered during processing.
func (p *NetworkIntentProcessor) ProcessIntent(ctx context.Context, intent *NetworkIntent) (*ProcessingResult, error) {
    // Validate input parameters
    if err := p.validator.ValidateIntent(intent); err != nil {
        return nil, fmt.Errorf("intent validation failed: %w", err)
    }
    
    // Process through LLM pipeline
    result, err := p.processWithLLM(ctx, intent)
    if err != nil {
        p.logger.Error(err, "LLM processing failed", "intent", intent.Name)
        return nil, err
    }
    
    return result, nil
}
```

**YAML/Kubernetes Standards:**
- Use consistent indentation (2 spaces)
- Include comprehensive metadata and labels
- Add descriptive comments for complex configurations
- Follow Kubernetes resource naming conventions
- Include resource limits and requests

### Testing Requirements

**Unit Tests:**
```go
func TestNetworkIntentProcessor_ProcessIntent(t *testing.T) {
    tests := []struct {
        name           string
        intent         *NetworkIntent
        expectedResult *ProcessingResult
        expectedError  string
    }{
        {
            name: "valid intent processing",
            intent: &NetworkIntent{
                ObjectMeta: metav1.ObjectMeta{Name: "test-intent"},
                Spec: NetworkIntentSpec{
                    Intent: "Deploy AMF for testing",
                },
            },
            expectedResult: &ProcessingResult{Status: "success"},
            expectedError:  "",
        },
        {
            name: "invalid intent rejection",
            intent: &NetworkIntent{
                ObjectMeta: metav1.ObjectMeta{Name: "invalid-intent"},
                Spec: NetworkIntentSpec{
                    Intent: "", // Empty intent should fail
                },
            },
            expectedResult: nil,
            expectedError:  "intent validation failed",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            processor := NewTestNetworkIntentProcessor()
            result, err := processor.ProcessIntent(context.Background(), tt.intent)
            
            if tt.expectedError != "" {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), tt.expectedError)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tt.expectedResult.Status, result.Status)
            }
        })
    }
}
```

**Integration Tests:**
- Test complete workflows end-to-end
- Use testcontainers for external dependencies
- Include failure scenario testing
- Test with realistic data volumes

### Documentation Standards

**Code Documentation:**
- Include package-level documentation
- Document all exported functions and types
- Use GoDoc format consistently
- Include usage examples where helpful

**API Documentation:**
- Maintain OpenAPI specifications
- Include request/response examples
- Document error codes and responses
- Provide SDK usage examples

**User Documentation:**
- Write for your target audience
- Include step-by-step instructions
- Provide working examples
- Keep documentation current with code changes

## Contribution Types

### 🐛 Bug Fixes

**Process:**
1. Search existing issues to avoid duplicates
2. Create detailed bug report if none exists
3. Reproduce the bug locally
4. Write failing tests that demonstrate the bug
5. Implement fix with minimal changes
6. Ensure tests pass and no regressions
7. Update documentation if needed

**Bug Report Template:**
```markdown
**Bug Description:** Clear description of the issue

**Steps to Reproduce:**
1. Step one
2. Step two
3. Step three

**Expected Behavior:** What should happen

**Actual Behavior:** What actually happens

**Environment:**
- OS: [e.g., macOS 12.0]
- Go Version: [e.g., 1.23.0]
- Kubernetes Version: [e.g., 1.28.0]
- Nephoran Version: [e.g., v0.1.0]

**Additional Context:** Logs, screenshots, etc.
```

### ✨ New Features

**Feature Development Process:**
1. Open feature request issue for discussion
2. Get maintainer approval before implementation
3. Create design document for complex features
4. Implement with comprehensive tests
5. Update documentation and examples
6. Submit pull request with detailed description

**Feature Request Template:**
```markdown
**Feature Summary:** Brief description of the feature

**Problem Statement:** What problem does this solve?

**Proposed Solution:** How would you solve this problem?

**Alternatives Considered:** What other approaches did you consider?

**Additional Context:** Mockups, examples, related issues
```

### 📖 Documentation

**Documentation Contributions:**
- Fix typos and grammatical errors
- Improve clarity and organization
- Add missing information
- Create new tutorials and guides
- Translate to other languages
- Update outdated information

**Documentation Guidelines:**
- Use clear, concise language
- Include practical examples
- Test all code examples
- Follow existing style and structure
- Consider different audience levels

### 🧪 Testing

**Testing Contributions:**
- Increase unit test coverage
- Add integration tests
- Create performance benchmarks
- Improve test reliability
- Add chaos engineering tests
- Create load testing scenarios

### 🌐 Knowledge Base

**Knowledge Contributions:**
- Telecommunications standards documentation
- Network function specifications
- Best practice guides
- Troubleshooting procedures
- Integration patterns
- Performance optimization tips

**Knowledge Base Guidelines:**
- Use authoritative sources
- Include proper citations
- Keep content current
- Organize logically
- Make content searchable

### 🎯 Performance

**Performance Contributions:**
- Identify performance bottlenecks
- Optimize algorithms and data structures
- Reduce memory usage
- Improve concurrent processing
- Enhance caching strategies
- Create performance benchmarks

## Pull Request Process

### 1. Before Creating a Pull Request

**Checklist:**
- [ ] Issue exists for the change (create one if needed)
- [ ] Changes are discussed with maintainers
- [ ] All tests pass locally
- [ ] Code follows project style guidelines
- [ ] Documentation is updated
- [ ] Commit messages are clear and descriptive

### 2. Creating the Pull Request

**Branch Naming:**
```bash
# Feature branches
feature/add-network-slicing-support

# Bug fix branches  
bugfix/fix-intent-validation-error

# Documentation branches
docs/update-getting-started-guide

# Hotfix branches
hotfix/critical-security-patch
```

**Commit Message Format:**
```
type(scope): short description

Longer description if needed, explaining what changed and why.

Closes #123
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Maintenance tasks

**Pull Request Template:**
```markdown
## Summary
Brief description of changes

## Type of Change
- [ ] Bug fix (non-breaking change that fixes an issue)
- [ ] New feature (non-breaking change that adds functionality) 
- [ ] Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] Documentation update

## Changes Made
- Change 1
- Change 2
- Change 3

## Testing
- [ ] Unit tests added/updated
- [ ] Integration tests added/updated
- [ ] Manual testing completed
- [ ] Performance impact assessed

## Documentation
- [ ] Code comments updated
- [ ] README updated (if needed)
- [ ] API documentation updated (if needed)
- [ ] User documentation updated (if needed)

## Checklist
- [ ] My code follows the project's style guidelines
- [ ] I have performed a self-review of my code
- [ ] I have commented my code, particularly in hard-to-understand areas
- [ ] I have made corresponding changes to the documentation
- [ ] My changes generate no new warnings
- [ ] I have added tests that prove my fix is effective or that my feature works
- [ ] New and existing unit tests pass locally with my changes
```

### 3. Review Process

**Maintainer Review:**
- Code quality and style
- Architecture and design
- Test coverage and quality
- Documentation completeness
- Performance implications
- Security considerations

**Community Review:**
- Functional correctness
- Usability and user experience
- Edge case handling
- Integration compatibility

**Addressing Feedback:**
- Respond to comments constructively
- Make requested changes promptly
- Ask questions if feedback is unclear
- Re-request review after changes

### 4. Merge Requirements

Before merging, ensure:
- [ ] All CI checks pass
- [ ] At least one maintainer approval
- [ ] All review comments addressed
- [ ] Documentation updated
- [ ] No merge conflicts
- [ ] Backwards compatibility maintained (or breaking changes documented)

## Community Guidelines

### Communication Channels

**GitHub Issues and Discussions:**
- Use for technical discussions
- Search before creating new issues
- Use appropriate labels and templates
- Be respectful and constructive

**Slack Community:**
- Join at [nephoran-community.slack.com]
- Real-time collaboration and support
- Share ideas and get quick feedback
- Network with other contributors

**Mailing Lists:**
- Development discussions: dev@nephoran-project.org
- User support: users@nephoran-project.org
- Security issues: security@nephoran-project.org

### Best Practices

**Communication:**
- Be respectful and inclusive
- Ask questions when unsure
- Provide context in discussions
- Give credit where due
- Help others learn and grow

**Collaboration:**
- Review others' pull requests
- Share knowledge and expertise
- Participate in design discussions
- Mentor new contributors
- Celebrate community achievements

**Quality:**
- Test your changes thoroughly
- Write clear documentation
- Follow established patterns
- Consider backwards compatibility
- Think about maintenance burden

### Mentorship Program

**For New Contributors:**
- Assigned experienced mentor
- Regular check-ins and guidance
- Help with first contributions
- Access to exclusive learning resources
- Recognition for achievement milestones

**Becoming a Mentor:**
- Demonstrate expertise and helpfulness
- Volunteer through community channels
- Complete mentor training program
- Commit to regular engagement
- Help grow the contributor community

## Recognition and Rewards

### Contributor Recognition

**Monthly Recognition:**
- Feature contributions in newsletter
- Social media highlights
- Contributor spotlights on website
- Recognition at community events

**Annual Recognition:**
- Contributor awards ceremony
- Annual report acknowledgments
- Conference speaking opportunities
- Special contributor badges/swag

### Contribution Levels

**New Contributor:**
- First successful pull request
- Community welcome package
- Access to contributor channels
- Invitation to contributor events

**Regular Contributor:**
- 5+ merged contributions
- Enhanced repository permissions
- Early access to new features
- Input on project roadmap

**Core Contributor:**
- Significant ongoing contributions
- Code review privileges
- Maintainer consideration
- Leadership opportunities

**Maintainer:**
- Demonstrated expertise and commitment
- Repository administration access
- Release management participation
- Strategic decision making input

### Swag and Rewards

**Contributor Swag:**
- Stickers and patches
- T-shirts and hoodies
- Branded accessories
- Limited edition items

**Conference Opportunities:**
- Speaking opportunities
- Conference attendance sponsorship
- Booth representation
- Community meetup organization

## Getting Help

### Where to Get Support

**Technical Questions:**
- GitHub Discussions for detailed questions
- Slack #development channel for quick questions
- Stack Overflow with 'nephoran' tag
- Office hours (monthly video calls)

**Process Questions:**
- GitHub Issues for process clarification
- Slack #contributors channel
- Direct message to maintainers
- Community forum discussions

**Mentorship:**
- Request mentor assignment
- Join newcomer study groups
- Attend contributor onboarding sessions
- Participate in pair programming sessions

### Resources for Learning

**Documentation:**
- [Developer Guide](docs/DEVELOPER_GUIDE.md)
- [Architecture Overview](docs/architecture.md)
- [API Reference](docs/API_REFERENCE.md)
- [Testing Guide](docs/testing/README.md)

**External Resources:**
- Kubernetes operator development
- Go programming best practices
- Telecommunications fundamentals
- AI/ML for infrastructure

**Training Materials:**
- Video tutorial series
- Interactive workshops
- Code review sessions
- Design thinking workshops

## Thank You! 🎉

Thank you for your interest in contributing to the Nephoran Intent Operator project! Every contribution, no matter how small, helps advance the future of intelligent network operations.

Remember:
- Start small and learn as you go
- Ask questions when you need help
- Celebrate your contributions and those of others
- Help make the telecommunications industry more accessible
- Have fun building the future of network automation!

**Happy Contributing!** 🚀

---

## Quick Reference

**Repository:** https://github.com/nephoran/intent-operator
**Documentation:** https://docs.nephoran-project.org  
**Community Slack:** https://nephoran-community.slack.com
**Issue Tracker:** https://github.com/nephoran/intent-operator/issues
**Discussions:** https://github.com/nephoran/intent-operator/discussions

For questions about contributing, reach out to:
- Project Maintainers: maintainers@nephoran-project.org
- Community Manager: community@nephoran-project.org
- Security Issues: security@nephoran-project.org