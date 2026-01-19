# Implementation Plan: Google OCR Integration

## Overview

This implementation plan breaks down the Google Cloud Vision OCR integration into discrete coding tasks. The implementation will add Google Cloud Vision dependencies, integrate the existing Google OCR client into the factory initialization, add environment variable support for credentials, and create comprehensive tests.

## Tasks

- [x] 1. Add Google Cloud Vision dependencies to go.mod
  - Run `go get cloud.google.com/go/vision/v2@latest`
  - Run `go get google.golang.org/api@latest`
  - Run `go mod tidy` to clean up dependencies
  - Verify the project builds without import errors
  - _Requirements: 1.1, 1.2, 1.3, 1.4_

- [x] 2. Fix import statements in google_captcha.go
  - Update import from `cloud.google.com/go/vision/apiv1` to `cloud.google.com/go/vision/v2/apiv1`
  - Update import from `cloud.google.com/go/vision/v2/apiv1/visionpb` to `cloud.google.com/go/vision/v2/apiv1/visionpb`
  - Verify the file compiles without errors
  - _Requirements: 1.1_

- [x] 3. Add environment variable support for Google credentials
  - [x] 3.1 Update config.go overrideFromEnv function
    - Add logic to read `GOOGLE_CREDENTIALS_JSON` environment variable
    - Override or add Google provider configuration when env var is set
    - Follow the same pattern as existing Ali/Tencent env var handling
    - _Requirements: 3.2_

  - [ ]* 3.2 Write unit tests for environment variable override
    - Test GOOGLE_CREDENTIALS_JSON overrides YAML config
    - Test empty env var doesn't override YAML
    - Test env var adds Google provider when not in YAML
    - _Requirements: 3.2_

- [x] 4. Integrate Google client into factory initialization
  - [x] 4.1 Update initClients function in internal/job/getCodeJob.go
    - Load configuration using existing config system
    - Iterate through config.Captcha.Providers
    - For each Google provider, call NewGoogleCaptchaClient with credentials_json
    - Append successfully created clients to the clients slice
    - Log errors for failed initializations but continue with other providers
    - _Requirements: 2.1, 2.2, 2.3, 2.4, 7.1_

  - [ ]* 4.2 Write unit tests for factory integration
    - Test initialization with valid Google provider config
    - Test initialization with invalid credentials (should log error, not crash)
    - Test initialization with multiple providers including Google
    - Test that Google clients are added to the pool
    - _Requirements: 2.1, 2.2, 2.3, 2.4_

- [x] 5. Checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

- [ ]* 6. Write property-based tests for Google OCR client
  - [ ]* 6.1 Create google_captcha_property_test.go file
    - Set up gopter property testing framework
    - Configure minimum 100 iterations per property test
    - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.5_

  - [ ]* 6.2 Write property test for valid configuration
    - **Property 1: Valid Configuration Creates Client**
    - Generate random valid credential JSON structures
    - Verify client creation succeeds for all valid inputs
    - Tag: `// Feature: google-ocr-integration, Property 1: Valid Configuration Creates Client`
    - **Validates: Requirements 2.1, 3.3**

  - [ ]* 6.3 Write property test for invalid credentials
    - **Property 2: Invalid Credentials Produce Errors**
    - Generate random invalid/malformed JSON strings
    - Verify client creation fails with error for all invalid inputs
    - Tag: `// Feature: google-ocr-integration, Property 2: Invalid Credentials Produce Errors`
    - **Validates: Requirements 2.2, 3.4, 6.1**

  - [ ]* 6.4 Write property test for base64 decoding
    - **Property 4: Base64 Decoding Handles Valid Inputs**
    - Generate random valid base64 strings (with and without data URL prefix)
    - Verify decoding succeeds without error for all valid inputs
    - Tag: `// Feature: google-ocr-integration, Property 4: Base64 Decoding Handles Valid Inputs`
    - **Validates: Requirements 4.1**

  - [ ]* 6.5 Write property test for invalid base64
    - **Property 5: Invalid Base64 Returns Decoding Error**
    - Generate random invalid base64 strings
    - Verify decoding error is returned for all invalid inputs
    - Tag: `// Feature: google-ocr-integration, Property 5: Invalid Base64 Returns Decoding Error`
    - **Validates: Requirements 6.3**

  - [ ]* 6.6 Write property test for client pool composition
    - **Property 6: Client Pool Includes All Valid Providers**
    - Generate random provider configurations with multiple providers
    - Verify all valid providers (including Google) appear in initialized pool
    - Tag: `// Feature: google-ocr-integration, Property 6: Client Pool Includes All Valid Providers`
    - **Validates: Requirements 2.3, 2.4, 7.1**

  - [ ]* 6.7 Write property test for load balancer rotation
    - **Property 7: Load Balancer Rotates Through Clients**
    - Generate random client pools of various sizes
    - Make N requests where N equals pool size
    - Verify each client is selected exactly once
    - Tag: `// Feature: google-ocr-integration, Property 7: Load Balancer Rotates Through Clients`
    - **Validates: Requirements 7.2, 7.3**

  - [ ]* 6.8 Write property test for interface compliance
    - **Property 8: Interface Compliance**
    - Generate random GoogleCaptchaClient instances
    - Verify RemoteClient interface assignment works
    - Verify both interface methods are callable
    - Tag: `// Feature: google-ocr-integration, Property 8: Interface Compliance`
    - **Validates: Requirements 7.4**

- [ ]* 7. Write unit tests for Google OCR client
  - [ ]* 7.1 Write test for client creation with valid credentials
    - Test NewGoogleCaptchaClient with mock valid credentials
    - Verify client is created successfully
    - _Requirements: 2.1_

  - [ ]* 7.2 Write test for client creation with empty credentials
    - Test NewGoogleCaptchaClient with empty string (uses default credentials)
    - Handle case where default credentials may not be available
    - _Requirements: 2.1_

  - [ ]* 7.3 Write test for DoWithBase64Img with data URL prefix
    - Test with base64 string containing "data:image/png;base64," prefix
    - Verify prefix is stripped and image is processed
    - _Requirements: 4.1_

  - [ ]* 7.4 Write test for DoWithBase64Img without prefix
    - Test with plain base64 string
    - Verify image is processed correctly
    - _Requirements: 4.1_

  - [ ]* 7.5 Write test for DoWithReader
    - Test with io.Reader containing image bytes
    - Verify image is processed correctly
    - _Requirements: 4.1_

  - [ ]* 7.6 Write test for empty image response
    - Test with image containing no text
    - Verify empty CaptchaResponse is returned
    - _Requirements: 4.3_

  - [ ]* 7.7 Write test for Close method
    - Test that Close() can be called without error
    - Verify resource cleanup
    - _Requirements: 5.1, 5.4_

  - [ ]* 7.8 Write test for interface implementation
    - Verify GoogleCaptchaClient implements RemoteClient interface
    - Test both DoWithBase64Img and DoWithReader through interface
    - _Requirements: 7.4_

- [x] 8. Update configuration validation
  - [x] 8.1 Verify Google provider validation in config.go
    - Ensure validation checks credentials_json is non-empty for Google providers
    - Ensure validation returns descriptive error for missing credentials
    - _Requirements: 3.3_

  - [ ]* 8.2 Write tests for Google provider validation
    - Test validation passes with valid Google provider config
    - Test validation fails with empty credentials_json
    - Test validation fails with invalid provider type
    - _Requirements: 3.3, 3.4_

- [x] 9. Update documentation
  - [x] 9.1 Update README.md with Google OCR setup instructions
    - Add section on Google Cloud Vision API setup
    - Document credential configuration options
    - Add example configuration snippets
    - _Requirements: 3.1, 3.2_

  - [x] 9.2 Update config.example.yaml comments
    - Ensure Google provider example is clear and complete
    - Add notes about credential options
    - _Requirements: 3.1_

- [x] 10. Final checkpoint - Ensure all tests pass
  - Run all unit tests
  - Run all property-based tests
  - Verify the application builds successfully
  - Test with actual Google Cloud credentials (if available)
  - Ensure all tests pass, ask the user if questions arise.

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP
- The existing google_captcha.go implementation is already functional, it just needs dependencies and integration
- Configuration structure already supports Google providers, minimal changes needed
- Load balancer already supports multiple providers through the RemoteClient interface
- Property tests validate universal correctness properties across many inputs
- Unit tests validate specific examples and edge cases
- Integration with the factory is the critical path for making Google OCR functional
