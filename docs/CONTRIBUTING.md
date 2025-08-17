# Contributing to PDP Server

## Welcome Contributors!

Thank you for your interest in contributing to the PDP Server project. This document provides guidelines and information for contributors.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Contributing Process](#contributing-process)
- [Coding Standards](#coding-standards)
- [Testing Guidelines](#testing-guidelines)
- [Documentation](#documentation)
- [Issue Reporting](#issue-reporting)
- [Pull Request Process](#pull-request-process)

## Code of Conduct

This project adheres to a code of conduct adapted from the [Contributor Covenant](https://www.contributor-covenant.org/). By participating, you are expected to uphold this code.

### Our Standards

- **Be Respectful**: Treat everyone with respect and kindness
- **Be Inclusive**: Welcome newcomers and diverse perspectives
- **Be Collaborative**: Work together constructively
- **Be Professional**: Maintain professional communication

## Getting Started

### Prerequisites

- **Go 1.21+**: [Installation Guide](https://golang.org/doc/install)
- **Git**: For version control
- **Make**: For build automation (optional)
- **Docker**: For containerized development (optional)

### First Time Setup

1. **Fork the Repository**
   ```bash
   # Fork on GitHub, then clone your fork
   git clone https://github.com/your-username/pdp-server.git
   cd pdp-server
   ```

2. **Add Upstream Remote**
   ```bash
   git remote add upstream https://github.com/original-org/pdp-server.git
   ```

3. **Install Dependencies**
   ```bash
   go mod download
   ```

4. **Build and Test**
   ```bash
   go build -o pdp-server cmd/server/main.go
   go test ./...
   ```

## Development Setup

### Environment Configuration

1. **Copy Configuration**
   ```bash
   cp config.yaml.example config.yaml
   ```

2. **Set Up Development Environment**
   ```bash
   # Create development directories
   mkdir -p dev-data/{blobs,db}
   
   # Set development config
   export PDP_DATA_DIR=./dev-data
   export PDP_LOG_LEVEL=debug
   ```

3. **Run Development Server**
   ```bash
   go run cmd/server/main.go
   ```

### Development Tools

#### Recommended IDE Extensions

**VS Code:**
- Go extension by Google
- Go Test Explorer
- GitLens
- REST Client

**GoLand/IntelliJ:**
- Go plugin
- Database Navigator
- HTTP Client

#### Useful Commands

```bash
# Run with live reload (install air first: go install github.com/cosmtrek/air@latest)
air

# Format code
go fmt ./...

# Lint code (install golangci-lint first)
golangci-lint run

# Generate mocks (install mockgen first)
go generate ./...

# Run specific tests
go test ./pkg/api -v

# Run tests with coverage
go test -cover ./...
```

## Contributing Process

### 1. Choose an Issue

- Look for issues labeled `good first issue` for beginners
- Check `help wanted` labels for areas needing contribution
- Discuss complex changes in issues before starting

### 2. Create a Branch

```bash
# Sync with upstream
git fetch upstream
git checkout main
git merge upstream/main

# Create feature branch
git checkout -b feature/your-feature-name
```

### 3. Make Changes

- Follow coding standards (see below)
- Write tests for new functionality
- Update documentation as needed
- Commit changes with clear messages

### 4. Test Your Changes

```bash
# Run all tests
go test ./...

# Run integration tests
./test_complete.sh

# Test specific functionality
go test ./pkg/api -run TestUploadPiece
```

### 5. Submit Pull Request

- Push your branch to your fork
- Create a pull request with clear description
- Link related issues
- Wait for review and address feedback

## Coding Standards

### Go Style Guide

We follow the standard Go style guide with some project-specific conventions:

#### Code Formatting

```bash
# Use gofmt for formatting
go fmt ./...

# Use goimports for import organization
goimports -w .
```

#### Naming Conventions

```go
// Good: Clear, descriptive names
func CreateProofSet(ctx context.Context, data ProofSetData) (*ProofSet, error)

type PieceService struct {
    blobStore  blobstore.Blobstore
    piriClient service.PDPService
}

// Bad: Unclear abbreviations
func CreatePS(ctx context.Context, d PSD) (*PS, error)
```

#### Error Handling

```go
// Good: Wrap errors with context
func (s *PieceService) UploadPiece(ctx context.Context, data []byte) error {
    if err := s.validateData(data); err != nil {
        return fmt.Errorf("data validation failed: %w", err)
    }
    
    if err := s.blobStore.Put(ctx, key, bytes.NewReader(data)); err != nil {
        return fmt.Errorf("failed to store piece data: %w", err)
    }
    
    return nil
}

// Bad: Swallowing errors
func (s *PieceService) UploadPiece(ctx context.Context, data []byte) error {
    s.validateData(data) // Error ignored
    s.blobStore.Put(ctx, key, bytes.NewReader(data)) // Error ignored
    return nil
}
```

#### Interface Design

```go
// Good: Small, focused interfaces
type Blobstore interface {
    Put(ctx context.Context, key string, data io.Reader) error
    Get(ctx context.Context, key string) (io.ReadCloser, error)
    Delete(ctx context.Context, key string) error
}

// Bad: Large, monolithic interfaces
type Storage interface {
    PutBlob(ctx context.Context, key string, data io.Reader) error
    GetBlob(ctx context.Context, key string) (io.ReadCloser, error)
    DeleteBlob(ctx context.Context, key string) error
    CreateDatabase() error
    MigrateDatabase() error
    QueryDatabase(query string) ([]map[string]interface{}, error)
    // ... many more methods
}
```

### Package Organization

```
pkg/
├── api/           # HTTP API handlers and routing
├── blobstore/     # File storage abstractions
├── config/        # Configuration management
├── models/        # Database models and types
├── piece/         # Piece management business logic
├── proofset/      # Proof set management
├── service/       # External service adapters
└── watcher/       # Background services
```

### Documentation Standards

#### Code Comments

```go
// PieceService manages the lifecycle of data pieces for Filecoin storage.
// It handles piece preparation, upload, and CommP calculation with
// power-of-2 padding for Filecoin compatibility.
type PieceService struct {
    blobStore    blobstore.Blobstore
    piriService  service.PDPService
    db          *gorm.DB
}

// UploadPiece stores piece data with proper Filecoin formatting.
// The data is padded to the next power of 2 and CommP is calculated
// for blockchain registration.
//
// Parameters:
//   - ctx: Request context for cancellation
//   - fileContent: Raw piece data to be stored
//
// Returns:
//   - PieceInfo: Metadata about the stored piece
//   - error: Any error that occurred during processing
func (s *PieceService) UploadPiece(ctx context.Context, fileContent []byte) (*PieceInfo, error) {
    // Implementation...
}
```

## Testing Guidelines

### Test Structure

```go
func TestPieceService_UploadPiece(t *testing.T) {
    tests := []struct {
        name        string
        input       []byte
        wantSize    int64
        wantErr     bool
        errContains string
    }{
        {
            name:     "valid small file",
            input:    []byte("hello world"),
            wantSize: 16, // Next power of 2 after 11
            wantErr:  false,
        },
        {
            name:        "empty file",
            input:       []byte{},
            wantErr:     true,
            errContains: "empty file",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### Test Categories

#### Unit Tests
- Test individual functions and methods
- Use mocks for external dependencies
- Fast execution (< 1 second per test)

```go
func TestPadToPowerOfTwo(t *testing.T) {
    testCases := []struct {
        input    []byte
        expected int
    }{
        {[]byte("hello"), 8},
        {[]byte("hello world"), 16},
        {make([]byte, 1024), 1024},
    }

    for _, tc := range testCases {
        result := padToPowerOfTwo(tc.input)
        assert.Equal(t, tc.expected, len(result))
    }
}
```

#### Integration Tests
- Test component interactions
- Use real dependencies where possible
- Test realistic scenarios

```go
func TestPieceUploadIntegration(t *testing.T) {
    // Set up real blobstore and database
    tempDir := t.TempDir()
    blobStore, err := blobstore.NewFileBlobstore(tempDir)
    require.NoError(t, err)

    db := setupTestDB(t)
    service := piece.NewPieceService(blobStore, mockPiriService, db)

    // Test the complete flow
    testData := []byte("integration test data")
    pieceInfo, err := service.UploadPiece(context.Background(), testData)
    
    require.NoError(t, err)
    assert.NotEmpty(t, pieceInfo.PieceCID)
    assert.Equal(t, int64(32), pieceInfo.Size) // Next power of 2
}
```

#### End-to-End Tests
- Test complete workflows via HTTP API
- Use the test script: `./test_complete.sh`

### Mocking Guidelines

Use interfaces for testability:

```go
//go:generate mockgen -source=blobstore.go -destination=mocks/mock_blobstore.go

type Blobstore interface {
    Put(ctx context.Context, key string, data io.Reader) error
    Get(ctx context.Context, key string) (io.ReadCloser, error)
    Delete(ctx context.Context, key string) error
}
```

## Documentation

### README Updates

When adding features, update the main README.md:
- Feature descriptions
- API endpoint changes
- Configuration options
- Usage examples

### API Documentation

Update `docs/API.md` for any API changes:
- New endpoints
- Parameter changes
- Response format changes
- Error codes

### Architecture Documentation

Update `docs/ARCHITECTURE.md` for structural changes:
- New components
- Data flow changes
- Integration patterns
- Technology stack updates

## Issue Reporting

### Bug Reports

Use the bug report template:

```markdown
**Describe the bug**
A clear description of what the bug is.

**To Reproduce**
Steps to reproduce the behavior:
1. Go to '...'
2. Click on '....'
3. See error

**Expected behavior**
What you expected to happen.

**Environment:**
- OS: [e.g. Ubuntu 20.04]
- Go version: [e.g. 1.21]
- PDP Server version: [e.g. v1.0.0]

**Additional context**
Any other context about the problem.
```

### Feature Requests

Use the feature request template:

```markdown
**Is your feature request related to a problem?**
A clear description of what the problem is.

**Describe the solution you'd like**
A clear description of what you want to happen.

**Describe alternatives you've considered**
Alternative solutions or features you've considered.

**Additional context**
Any other context about the feature request.
```

## Pull Request Process

### Before Submitting

- [ ] Code follows project style guidelines
- [ ] Self-review of the code completed
- [ ] Tests added for new functionality
- [ ] All tests pass locally
- [ ] Documentation updated as needed
- [ ] Commit messages are clear and descriptive

### PR Description Template

```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix (non-breaking change that fixes an issue)
- [ ] New feature (non-breaking change that adds functionality)
- [ ] Breaking change (fix or feature that causes existing functionality to not work as expected)
- [ ] Documentation update

## Testing
- [ ] Unit tests pass
- [ ] Integration tests pass
- [ ] Manual testing completed

## Checklist
- [ ] Code follows style guidelines
- [ ] Self-review completed
- [ ] Documentation updated
- [ ] Tests added/updated
```

### Review Process

1. **Automated Checks**: CI/CD pipeline runs tests and linting
2. **Code Review**: Maintainers review code for quality and correctness
3. **Testing**: Additional testing by reviewers if needed
4. **Approval**: At least one maintainer approval required
5. **Merge**: Squash and merge or rebase merge based on complexity

### After Merge

- Delete your feature branch
- Update your local main branch
- Close related issues if applicable

## Development Workflow

### Git Workflow

We use a simplified Git Flow:

```bash
# Main branch: stable, production-ready code
main

# Feature branches: new features and bug fixes
feature/add-authentication
feature/improve-error-handling
bugfix/fix-memory-leak

# Release branches: prepare for releases (if needed)
release/v1.1.0
```

### Commit Messages

Follow conventional commits:

```bash
# Format: type(scope): description

feat(api): add authentication middleware
fix(piece): resolve memory leak in upload handler
docs(readme): update installation instructions
test(api): add integration tests for proof sets
refactor(storage): extract blobstore interface
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation only changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code change that neither fixes a bug nor adds a feature
- `test`: Adding missing tests
- `chore`: Changes to build process or auxiliary tools

### Release Process

1. **Version Bumping**: Follow semantic versioning (MAJOR.MINOR.PATCH)
2. **Changelog**: Update CHANGELOG.md with new features and fixes
3. **Tagging**: Create git tags for releases
4. **Documentation**: Update version-specific documentation

## Getting Help

### Communication Channels

- **GitHub Issues**: Bug reports and feature requests
- **GitHub Discussions**: General questions and discussions
- **Email**: [maintainer-email] for private matters

### Resources

- [Go Documentation](https://golang.org/doc/)
- [Echo Framework Guide](https://echo.labstack.com/guide/)
- [GORM Documentation](https://gorm.io/docs/)
- [Piri Documentation](https://github.com/storacha/piri)

## Recognition

Contributors will be recognized in:
- CONTRIBUTORS.md file
- Release notes for significant contributions
- GitHub contributor statistics

Thank you for contributing to PDP Server! 🎉
