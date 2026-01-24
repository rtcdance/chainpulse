# Implementation Plan: golangci-lint Fixes - January 17

## Overview

Fix compilation errors in the discovery package by consolidating duplicate type declarations, resolving undefined references, and ensuring proper imports.

## Tasks

- [ ] 1. Analyze and document duplicate declarations
  - Identify all duplicate types in service_discovery_advanced.go and session_manager.go
  - Document which implementation is more complete
  - Create consolidation plan
  - _Requirements: 1.1, 1.2_

- [ ] 2. Remove duplicate ServiceLoadBalancer from session_manager.go
  - Delete the ServiceLoadBalancer type definition from session_manager.go
  - Delete the NewServiceLoadBalancer function from session_manager.go
  - Delete the ServiceEndpoint type definition from session_manager.go
  - Update any references to use the consolidated version from service_discovery_advanced.go
  - _Requirements: 1.2, 1.3, 3.1_

- [ ] 3. Remove duplicate ServiceEndpointCache from session_manager.go
  - Delete the ServiceEndpointCache type definition from session_manager.go
  - Delete the NewServiceEndpointCache function from session_manager.go
  - Update any references to use the consolidated version from service_discovery_advanced.go
  - _Requirements: 1.2, 1.3, 3.2_

- [ ] 4. Fix undefined RedisClusterManager reference
  - Analyze usage of RedisClusterManager in session_manager.go
  - Either define RedisClusterManager or remove the reference
  - Update NewSessionManagerWithCache to work without RedisClusterManager
  - _Requirements: 2.1, 2.3_

- [ ] 5. Verify config package import in service_registry.go
  - Check if config.ConsulClient exists in pkg/infrastructure/config
  - If not, create a stub or update the import
  - Verify the import path is correct
  - _Requirements: 2.2, 2.3_

- [ ] 6. Verify all type references are correct
  - Search for all references to ServiceLoadBalancer in the codebase
  - Search for all references to ServiceEndpointCache in the codebase
  - Update any references to use the consolidated versions
  - _Requirements: 1.3, 3.3_

- [ ] 7. Run golangci-lint on discovery package
  - Execute: golangci-lint run ./pkg/infrastructure/discovery/...
  - Verify no redeclaration errors
  - Verify no undefined reference errors
  - Document any remaining issues
  - _Requirements: 4.1_

- [ ] 8. Run full test suite
  - Execute: go test ./pkg/infrastructure/discovery/...
  - Verify all tests pass
  - Fix any test failures
  - _Requirements: 4.2_

- [ ] 9. Verify full golangci-lint passes
  - Execute: golangci-lint run ./...
  - Verify discovery package has zero errors
  - Document final status
  - _Requirements: 4.1, 4.3_

- [ ] 10. Document changes
  - Create summary of changes made
  - Document consolidation decisions
  - Update any affected documentation
  - _Requirements: 4.3_

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Consolidation prioritizes the more complete implementation
- All changes must maintain backward compatibility where possible
