---
name: "documentation"
description: "Guidelines for writing and maintaining project documentation. Invoke when creating or updating documentation files."
---

# Documentation Guidelines

## Purpose
Ensure clear, consistent, and useful documentation.

## When to Invoke
- Creating new documentation files
- Updating existing documentation
- Writing README files
- Creating API documentation

## Documentation Structure

```
docs/
├── README.md              # Documentation index
├── guides/
│   ├── DEVELOPER_GUIDE.md # Development guide
│   ├── DEPLOYMENT_GUIDE.md # Deployment procedures
│   ├── API_DOCUMENTATION.md # API reference
│   └── OPERATIONS_GUIDE.md # Operations manual
└── architecture/
    └── ARCHITECTURE.md    # System architecture
```

## README Template

```markdown
# Project Name

Brief description of the project.

## Quick Start

### Prerequisites
- List requirements

### Installation
```bash
# Installation commands
```

### Usage
```bash
# Usage examples
```

## Documentation

- [Developer Guide](docs/guides/DEVELOPER_GUIDE.md)
- [API Documentation](docs/guides/API_DOCUMENTATION.md)

## Contributing

Brief contribution guidelines.

## License

License information.
```

## API Documentation Format

```markdown
## Endpoint Name

**Method**: `GET/POST/PUT/DELETE`
**Path**: `/api/v1/resource`

### Description
What this endpoint does.

### Parameters

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `id` | string | Yes | Resource ID |

### Request
```json
{
  "field": "value"
}
```

### Response
```json
{
  "data": {}
}
```

### Errors

| Code | Description |
|------|-------------|
| 400 | Invalid request |
| 404 | Not found |
```

## Constraints
- Keep documentation up-to-date with code
- Use clear, simple language
- Include code examples where helpful
- DO NOT document obvious things
- DO NOT let documentation become stale
