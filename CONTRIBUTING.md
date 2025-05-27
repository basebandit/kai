# Contributing to Kai

Thank you for your interest in contributing to Kai! We welcome contributions of all kinds, including bug fixes, new Kubernetes features, and documentation improvements.

## Quick Start

### Prerequisites
- Go 1.23 or later
- Git

### Setup
1. Fork the repository
2. Clone your fork:
   ```bash
   git clone https://github.com/YOUR_USERNAME/kai.git
   cd kai
   ```
3. Install dependencies:
   ```bash
   go mod tidy
   ```
4. Verify setup:
   ```bash
   go test -v ./...
   ```

## Contributing Process

### 1. Create an Issue First
**For new features or significant changes**, please create an issue to discuss:
- What you want to build
- How you plan to implement it
- Why it's needed

For small bug fixes, you can skip directly to step 2.

### 2. Development Workflow
1. Create a branch:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. Make your changes following our [guidelines](#development-guidelines)

3. Test your changes:
   ```bash
   # Run tests
   go test -v ./...
   
   # Run linting (if available)
   golangci-lint run
   ```

4. Commit with clear messages:
   ```bash
   git commit -m "Add support for ConfigMaps (#123)"
   ```

5. Push and create a Pull Request

### 3. Pull Request Requirements
- [ ] Tests pass locally
- [ ] New functionality includes tests
- [ ] Clear description of changes
- [ ] References related issue(s)

## Development Guidelines

### Code Style
- Follow standard Go conventions (`go fmt`)
- Use meaningful names for variables and functions
- Add comments for exported functions
- Handle errors appropriately

### Testing
- Write unit tests for new functionality
- Mock external dependencies (Kubernetes API)
- Test both success and error cases
- Keep tests deterministic

### What We're Looking For
- **Kubernetes Resources**: Support for new resource types (ConfigMaps, Secrets, Ingress, etc.)
- **Bug Fixes**: Issues with existing functionality
- **Documentation**: README updates, code comments, examples
- **Code Quality**: Performance improvements, refactoring

## Project Structure
```
kai/
├── cluster/          # Kubernetes resource management
├── tools/           # MCP tool implementations  
├── cmd/kai/         # Main application
├── testmocks/       # Test utilities
└── *.go            # Core interfaces and types
```

## Getting Help

- **Questions**: Open an issue with the "question" label
- **Bugs**: Check existing issues first, then create a new one
- **Features**: Create an issue to discuss before implementing

## Code of Conduct

Be respectful and constructive. We want Kai to be welcoming to contributors of all experience levels.

---

**Ready to contribute?** Start by browsing our [open issues](https://github.com/basebandit/kai/issues) or creating a new one!