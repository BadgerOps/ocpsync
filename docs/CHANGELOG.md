# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

## [1.0.0] - Initial release

### Added
- Download OCP binaries from mirror.openshift.com for specified versions
- Download RHCOS images from mirror.openshift.com for specified versions
- SHA-256 checksum validation for all downloaded files
- Configurable file filtering to exclude unwanted platform variants
- Exponential backoff retry logic for failed downloads
- YAML-based configuration (`config.yaml`)

### Fixed
- Check HTTP status codes in downloadFile (404s were silently saving error pages)
- Change break to continue in downloadFileList (malformed lines no longer abort entire loop)
- Stream files through sha256 hasher instead of loading entirely into memory
- Fix exponential backoff calculation (was starting at 8s instead of 1s)
- Return errors from generateFileList instead of swallowing them
- Skip to next version in downloadHandler when sha256sum.txt download fails
- Replace panic() with logrus.Fatal() for config errors

### Changed
- Upgrade Go from 1.22 to 1.24 (fixes crypto/tls and crypto/x509 vulnerabilities)
- Upgrade gopkg.in/yaml from v2 to v3
- Upgrade golang.org/x/sys to v0.28.0
- Use filepath.Join() instead of string concatenation for paths
- Rename module to github.com/BadgerOps/ocpsync
- Bump default config versions to OCP 4.17, 4.18, 4.19
- Rewrite tests to use httptest.NewServer and t.TempDir()
- Add CI/CD pipeline with changelog-driven releases, cross-platform builds, and checksums

[Unreleased]: https://github.com/BadgerOps/ocpsync/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/BadgerOps/ocpsync/releases/tag/v1.0.0
