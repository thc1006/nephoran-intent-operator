# Nephoran Intent Operator Documentation

This directory contains the complete documentation for the Nephoran Intent Operator, built using MkDocs Material theme and deployed via GitHub Pages.

## Quick Links

- 📚 **[Live Documentation](https://nephoran.github.io/nephoran-intent-operator/)**
- 🚀 **[Getting Started](getting-started/quickstart.md)**
- 🏗️ **[Architecture](architecture/system-architecture.md)**
- 📖 **[API Reference](api/index.md)**

## Documentation Structure

```
docs/
├── index.md                    # Homepage
├── getting-started/           # Installation and quick start guides
├── architecture/              # System architecture and design
├── user-guide/               # User documentation and guides
├── api/                      # API reference documentation
├── operations/               # Production deployment and operations
├── developer/                # Development and contribution guides
├── oran/                     # O-RAN specific documentation
├── tutorials/                # Step-by-step tutorials
├── reference/                # Configuration reference and CLI docs
├── community/                # Contributing, changelog, license
├── assets/                   # Stylesheets, images, and JavaScript
│   ├── stylesheets/
│   └── javascripts/
└── snippets/                 # Reusable content snippets
```

## Building Documentation Locally

### Prerequisites

- Python 3.8+
- pip package manager

### Setup

```bash
# Install documentation dependencies
pip install -r requirements-docs.txt

# Or use poetry if available
poetry install --only docs
```

### Development Server

```bash
# Start live-reloading development server
mkdocs serve

# Custom host/port
mkdocs serve --dev-addr=127.0.0.1:8080

# Strict mode (treat warnings as errors)
mkdocs serve --strict
```

The development server will be available at `http://127.0.0.1:8000` with automatic reloading on file changes.

### Building Static Site

```bash
# Build static site
mkdocs build

# Clean build (remove previous build artifacts)
mkdocs build --clean

# Build with verbose output
mkdocs build --verbose

# Strict build (fail on warnings)
mkdocs build --strict
```

Built documentation will be in the `site/` directory.

## Documentation Features

### Content Features

- **Responsive Design**: Mobile-optimized Material Design theme
- **Search**: Full-text search with highlighting
- **Dark/Light Mode**: Automatic theme switching
- **Code Highlighting**: Syntax highlighting for 100+ languages
- **Diagrams**: Mermaid diagram support
- **Tabs**: Tabbed content sections
- **Admonitions**: Callout boxes for tips, warnings, etc.
- **Tables**: Sortable tables with search
- **Math**: LaTeX math rendering

### Development Features

- **Live Reload**: Automatic browser refresh on changes
- **Link Checking**: Automated link validation
- **Git Integration**: Automatic page edit links and revision dates
- **API Documentation**: Auto-generated from OpenAPI specs
- **Code Snippets**: Reusable content includes

### SEO and Accessibility

- **Structured Data**: Schema.org markup
- **Open Graph**: Social media preview cards  
- **Accessibility**: WCAG 2.1 AA compliance
- **Performance**: Optimized assets and lazy loading
- **Analytics**: Google Analytics integration

## Content Guidelines

### Writing Style

- Use clear, concise language
- Write for your audience (operators, developers, etc.)
- Include practical examples and code samples
- Use consistent terminology (see glossary)
- Follow the [Microsoft Writing Style Guide](https://docs.microsoft.com/en-us/style-guide/welcome/)

### Markdown Conventions

```markdown
# Use sentence case for headings
## Configuration options

# Use fenced code blocks with language
```yaml
apiVersion: v1
kind: ConfigMap
```

# Use admonitions for important information
!!! tip "Pro tip"
    This is a helpful tip for users.

# Use tables for structured data
| Parameter | Type | Description |
|-----------|------|-------------|
| name      | string | Resource name |
```

### Documentation Types

- **Tutorials**: Learning-oriented, step-by-step guides
- **How-to Guides**: Problem-oriented, practical solutions  
- **Reference**: Information-oriented, comprehensive details
- **Explanation**: Understanding-oriented, theory and context

## Automated Processes

### GitHub Pages Deployment

Documentation is automatically built and deployed on:
- Pushes to `main` branch (docs changes)
- Pull requests (preview builds)
- Manual workflow dispatch

### Link Checking

Automated link checking runs:
- On pull requests (immediate feedback)
- Daily scheduled runs (catch external link rot)
- Manual workflow dispatch

Broken links are reported as GitHub issues with automatic updates.

### CRD Documentation

Custom Resource Definition documentation is automatically generated from:
- CRD YAML files in `deployments/crds/`
- OpenAPI specifications in `docs/api/`
- Live cluster resources (when available)

## Contributing to Documentation

### Quick Edits

For small changes:
1. Click the edit button (pencil icon) on any page
2. Make changes in GitHub's web editor
3. Submit a pull request

### Major Changes

For substantial changes:
1. Fork the repository
2. Create a feature branch
3. Make changes locally and test with `mkdocs serve`
4. Submit a pull request

### Documentation Pull Requests

All documentation pull requests:
- Run automated link checking
- Include preview deployments
- Require review from docs team
- Must pass CI/CD checks

## Troubleshooting

### Common Issues

**Build failures:**
```bash
# Clear pip cache
pip cache purge

# Reinstall dependencies
pip install --force-reinstall -r requirements-docs.txt

# Check for conflicting packages
pip check
```

**Slow builds:**
```bash
# Use cached builds
mkdocs build --use-directory-urls

# Disable plugins temporarily
mkdocs build --config-file mkdocs-minimal.yml
```

**Link check failures:**
- Check for typos in file paths and URLs
- Ensure external links are accessible
- Update redirect configuration if needed

### Getting Help

- 📖 [MkDocs Documentation](https://www.mkdocs.org/)
- 🎨 [Material Theme Docs](https://squidfunk.github.io/mkdocs-material/)
- 💬 [GitHub Discussions](https://github.com/nephoran/nephoran-intent-operator/discussions)
- 🐛 [Report Documentation Issues](https://github.com/nephoran/nephoran-intent-operator/issues/new?template=documentation.md)

## License

Documentation is licensed under [Creative Commons Attribution 4.0 International](https://creativecommons.org/licenses/by/4.0/).

Code examples and configurations are licensed under [Apache License 2.0](https://github.com/nephoran/nephoran-intent-operator/blob/main/LICENSE).