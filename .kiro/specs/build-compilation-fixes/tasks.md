# Implementation Plan: Build Compilation Fixes

## Overview

Fix 27 typecheck errors in test files by updating mock implementations, correcting method signatures, and fixing type mismatches. This will enable the project to compile successfully.

## Tasks

- [ ] 1. Update MockCachePlugin to implement core.CachePlugin interface
  - Add Initialize method to MockCachePlugin
  - Update Get method signature to accept context.Context
  - Update Set method signature to accept context.Context and correct parameters
  - Add Delete method if missing
  - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5_

- [x] 2. Fix MockDatabasePlugin event storage type
  - Change events field from []*core.BlockchainEvent to map[string]*core.BlockchainEvent
  - Update event initialization in test setup
  - Verify all test files using MockDatabasePlugin are updated
  - _Requirements: 2.1, 2.2, 2.3, 2.4_

- [ ] 3. Fix QueryEvents method calls in tests
  - Update all QueryEvents calls to use context.Context parameter
  - Remove pagination parameters from QueryEvents calls
  - Handle results as interface{} and type assert as needed
  - _Requirements: 3.1, 3.2, 3.3, 3.4_

- [ ] 4. Fix cache method calls in tests
  - Update cache.Get calls to include context.Context parameter
  - Update cache.Set calls to use correct signature (ctx, key, []byte, ttl)
  - Verify all cache operations use context
  - _Requirements: 4.1, 4.2, 4.3, 4.4_

- [ ] 5. Fix missing imports and undefined methods
  - Verify fixtures package import path is correct
  - Remove SetPoolMetadata calls or add method to UniswapIndexer
  - Fix any other undefined method references
  - _Requirements: 5.1, 5.2, 5.3, 6.1, 6.2, 6.3_

- [ ] 6. Verify clean compilation
  - Run golangci-lint on test files
  - Verify 0 typecheck errors
  - Run tests to ensure they execute
  - _Requirements: 7.1, 7.2, 7.3_

</content>
</invoke>