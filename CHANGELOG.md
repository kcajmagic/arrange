# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## [Unreleased]
- added a viper option for aggregating decode hooks
- added a viper option for decoding encoding.TextUnmarshaler implementations

## [v0.1.7]
- added sonar integration
- added code climate badges
- refactored arrangehttp reflection logic around options for simplicity and consistency
- expose an optional fx.Printer for arrange informational output
- expose a testing fx.Printer to redirect output to testing.T and testing.B
- introduce arrange.Unmarshaler rather than having everything depend directly on viper
- separated Use vs Inject in builders to make API usage clearer

## [v0.1.6]
- struct field traversal
- fx.In detection
- support for injecting all options into servers and clients
- support for immutable and precomputed HTTP headers
- moved TLS code out of arrangehttp, as that package was getting too large
- added test utilities for generate TLS certificates for unit tests
- separated listener creation into its own interface to make customization easier

## [v0.1.5]
- arrangehttp decorators (e.g. middleware) may now be injected

## [v0.1.4]
- http.Roundtripper decoration to support things like metrics and logging

## [v0.1.3]
- Added http.Client support to arrangehttp
- Simpler TLS configuration
- Some better examples in godoc

## [v0.1.2]
- Upgrade mapstructure to v1.3.2
- Added convenient viper.DecoderConfigOption implementations
- Removed UnmarshalExact, as it is superfluous
- Added a simple way to unmarshal multiple keys at once
- Streamlined unmarshal/provide API
- Added arrangehttp, with support for producing http.Server objects from external configuration

## v0.1.0
- Initial creation

[Unreleased]: https://github.com/xmidt-org/arrange/compare/v0.1.7..HEAD
[v0.1.7]: https://github.com/xmidt-org/arrange/compare/v0.1.6...v0.1.7
[v0.1.6]: https://github.com/xmidt-org/arrange/compare/v0.1.5...v0.1.6
[v0.1.5]: https://github.com/xmidt-org/arrange/compare/v0.1.4...v0.1.5
[v0.1.4]: https://github.com/xmidt-org/arrange/compare/v0.1.3...v0.1.4
[v0.1.3]: https://github.com/xmidt-org/arrange/compare/v0.1.2...v0.1.3
[v0.1.2]: https://github.com/xmidt-org/arrange/compare/v0.1.0...v0.1.2
