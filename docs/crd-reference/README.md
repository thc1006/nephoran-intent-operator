# Nephoran Intent Operator - CRD Reference Documentation

## Overview

This directory contains comprehensive reference documentation for all Custom Resource Definitions (CRDs) in the Nephoran Intent Operator. The documentation is built using MkDocs with the Material theme and provides interactive, searchable reference material for operators and developers.

## Documentation Structure

```
docs/crd-reference/
├── ../mkdocs.yml              # MkDocs configuration (in docs/ root)
├── requirements.txt           # Python dependencies
├── build-docs.sh             # Build script
├── docs/                     # Documentation source
│   ├── index.md             # Home page
│   ├── crds/                # CRD documentation
│   │   ├── networkintent/   # NetworkIntent CRD
│   │   ├── e2nodeset/       # E2NodeSet CRD
│   │   ├── managedelement/  # ManagedElement CRD
│   │   └── disaster-recovery/ # DR CRDs
│   ├── api/                 # API reference
│   ├── examples/            # Usage examples
│   └── troubleshooting/     # Troubleshooting guides
├── generated/               # Auto-generated CRD specs
└── site/                   # Built documentation (git-ignored)
```

## Available CRDs

### Core Resources

1. **NetworkIntent** - Transform natural language intents into network operations
   - Natural language processing with LLM
   - Multi-component deployment support
   - Resource management and constraints
   - Network slicing capabilities

2. **E2NodeSet** - Manage O-RAN E2 nodes
   - E2 interface management (v1.0-v3.0)
   - RIC connectivity and health monitoring
   - Simulation capabilities for testing
   - Horizontal scaling support

3. **ManagedElement** - Configure O-RAN managed elements
   - O1, A1, and E2 interface configuration
   - FCAPS management support
   - Credential management
   - Multi-vendor compatibility

### Disaster Recovery Resources

4. **DisasterRecoveryPlan** - Comprehensive DR strategies
   - RTO/RPO targets
   - Automated failover triggers
   - Testing and validation
   - Multi-region support

5. **BackupPolicy** - Automated backup management
   - Scheduled backups with cron syntax
   - Retention policies
   - Encryption and compression
   - Multiple storage providers

6. **FailoverPolicy** - Regional failover configuration
   - Health check monitoring
   - DNS failover support
   - Traffic management
   - Data synchronization

## Building Documentation

### Prerequisites

- Python 3.8+
- pip (Python package manager)
- Git
- Make (optional)

### Quick Start

1. **Install dependencies:**
   ```bash
   cd docs/crd-reference
   python3 -m venv venv
   source venv/bin/activate  # On Windows: venv\Scripts\activate
   pip install -r requirements.txt
   ```

2. **Build documentation:**
   ```bash
   mkdocs build
   ```

3. **Serve locally:**
   ```bash
   mkdocs serve
   # Documentation available at http://localhost:8000
   ```

### Using Build Script

```bash
# Build documentation
./build-docs.sh

# Build and serve locally
./build-docs.sh --serve

# Deploy to GitHub Pages
./build-docs.sh --deploy
```

## Development Workflow

### Adding New Documentation

1. Create markdown files in appropriate directories
2. Update `docs/mkdocs.yml` navigation if needed
3. Follow existing formatting patterns
4. Include examples and troubleshooting

### Documentation Standards

- Use clear, concise language
- Include code examples for all features
- Provide both basic and advanced examples
- Add diagrams where helpful (Mermaid supported)
- Include field validation rules
- Document default values
- Add cross-references to related topics

### Testing Documentation

1. **Check for broken links:**
   ```bash
   mkdocs build
   linkchecker site/index.html
   ```

2. **Validate YAML examples:**
   ```bash
   find docs -name '*.md' -exec grep -l '```yaml' {} \; | \
     xargs -I {} sh -c 'python scripts/validate-yaml.py {}'
   ```

3. **Preview changes:**
   ```bash
   mkdocs serve --dirtyreload
   ```

## Deployment

### GitHub Pages

The documentation is automatically deployed to GitHub Pages when changes are pushed to the main branch.

**Manual deployment:**
```bash
mkdocs gh-deploy --force
```

**GitHub Actions workflow:**
- Triggered on push to main branch
- Builds and validates documentation
- Deploys to GitHub Pages
- Available at: https://nephoran.github.io/intent-operator/

### Versioning

We use `mike` for documentation versioning:

```bash
# Deploy new version
mike deploy v1.0.0 latest --push

# List versions
mike list

# Set default version
mike set-default latest
```

## Features

### Search
- Full-text search across all documentation
- Search suggestions and highlighting
- Keyboard shortcuts (/ to search)

### Navigation
- Tabbed navigation for major sections
- Breadcrumbs for context
- Table of contents for each page
- Previous/Next page navigation

### Code Examples
- Syntax highlighting for YAML, Go, Bash
- Copy button for code blocks
- Line numbers for reference
- Tabbed examples for alternatives

### Responsive Design
- Mobile-friendly layout
- Print-optimized CSS
- Dark/Light theme toggle
- Accessible navigation

## Contributing

### Documentation Contributions

1. Fork the repository
2. Create a feature branch
3. Make documentation changes
4. Test locally with `mkdocs serve`
5. Submit pull request

### Style Guide

- **Headings**: Use sentence case
- **Code blocks**: Include language identifier
- **Links**: Use relative paths for internal links
- **Images**: Store in `docs/assets/images/`
- **Examples**: Provide context and explanation

### Review Process

1. Technical accuracy review
2. Grammar and style check
3. Example validation
4. Link verification
5. Build verification

## Troubleshooting

### Common Issues

**Build fails with missing dependencies:**
```bash
pip install --upgrade pip
pip install -r requirements.txt --force-reinstall
```

**Permission denied on build script:**
```bash
chmod +x build-docs.sh
```

**MkDocs serve fails to start:**
```bash
# Check if port 8000 is in use
lsof -i :8000
# Use different port
mkdocs serve -a localhost:8001
```

**GitHub Pages deployment fails:**
```bash
# Ensure gh-pages branch exists
git checkout --orphan gh-pages
git rm -rf .
echo "Documentation" > index.html
git add index.html
git commit -m "Initial gh-pages"
git push origin gh-pages
git checkout main
```

## Resources

### Documentation
- [MkDocs Documentation](https://www.mkdocs.org/)
- [Material for MkDocs](https://squidfunk.github.io/mkdocs-material/)
- [Mermaid Diagrams](https://mermaid-js.github.io/mermaid/)

### Tools
- [YAML Validator](https://www.yamllint.com/)
- [Markdown Linter](https://github.com/DavidAnson/markdownlint)
- [Link Checker](https://github.com/linkchecker/linkchecker)

### Support
- GitHub Issues: [Report documentation issues](https://github.com/nephoran/intent-operator/issues)
- Discussions: [Ask questions](https://github.com/nephoran/intent-operator/discussions)
- Slack: [Join #documentation channel](https://nephoran.slack.com)

## License

The documentation is licensed under the [Apache License 2.0](../../LICENSE), same as the Nephoran Intent Operator project.

## Maintainers

- Documentation Lead: [documentation@nephoran.io](mailto:documentation@nephoran.io)
- Technical Writers: See [CONTRIBUTING.md](../../CONTRIBUTING.md)

---

**Last Updated:** January 2025  
**Version:** 1.0.0  
**Status:** Production Ready
