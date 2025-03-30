/*
   Copyright Farcloser.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package options

import (
	"io"

	"go.farcloser.world/containers/specs"
)

// ImageList specifies options for `image list`.
type ImageList struct {
	Stdout io.Writer
	// GOptions is the global options
	GOptions *Global
	// Quiet only show numeric IDs
	Quiet bool
	// NoTrunc don't truncate output
	NoTrunc bool
	// Format the output using the given Go template, e.g, '{{json .}}', 'wide'
	Format string
	// Filter output based on conditions provided, for the --filter argument
	Filters []string
	// NameAndRefFilter filters images by name and reference
	NameAndRefFilter []string
	// Digests show digests (compatible with Docker, unlike ID)
	Digests bool
	// Names show image names
	Names bool
	// All (unimplemented yet, always true)
	All bool
}

// ImageConvert specifies options for `image convert`.
type ImageConvert struct {
	SourceRef      string
	DestinationRef string

	// #region generic flags
	// Uncompress convert tar.gz layers to uncompressed tar layers
	Uncompress bool
	// Oci convert Docker media types to OCI media types
	Oci bool
	// #endregion

	// #region platform flags
	// Platforms convert content for a specific platform
	Platforms []string
	// AllPlatforms convert content for all platforms
	AllPlatforms bool
	// #endregion

	// Format the output using the given Go template, e.g, 'json'
	Format string

	// #region zstd flags
	// Zstd convert legacy tar(.gz) layers to zstd. Should be used in conjunction with '--oci'
	Zstd bool
	// ZstdCompressionLevel zstd compression level
	ZstdCompressionLevel int
	// #endregion

	// #region zstd:chunked flags
	// ZstdChunked convert legacy tar(.gz) layers to zstd:chunked for lazy pulling. Should be used in conjunction with
	// '--oci'
	ZstdChunked bool
	// ZstdChunkedCompressionLevel zstd compression level
	ZstdChunkedCompressionLevel int
	// ZstdChunkedChunkSize zstd chunk size
	ZstdChunkedChunkSize int
	// ZstdChunkedRecordIn read 'ctr-remote optimize --record-out=<FILE>' record file (EXPERIMENTAL)
	ZstdChunkedRecordIn string
	// #endregion
}

// ImageCrypt specifies options for `image encrypt` and `image decrypt`.
type ImageCrypt struct {
	SourceRef      string
	DestinationRef string
	// Platforms convert content for a specific platform
	Platforms []string
	// AllPlatforms convert content for all platforms
	AllPlatforms bool
	// GpgHomeDir the GPG homedir to use; by default gpg uses ~/.gnupg
	GpgHomeDir string
	// GpgVersion the GPG version ("v1" or "v2"), default will make an educated guess
	GpgVersion string
	// Keys a secret key's filename and an optional password separated by colon;
	Keys []string
	// DecRecipients recipient of the image; used only for PKCS7 and must be a x509 certificate
	DecRecipients []string
	// Recipients of the image is the person who can decrypt it in the form specified above (i.e. jwe:/path/to/pubkey)
	Recipients []string
}

// ImageInspect specifies options for `image inspect`.
type ImageInspect struct {
	Stdout   io.Writer
	GOptions *Global
	// Mode Inspect mode, "dockercompat" for Docker-compatible output, "native" for containerd-native output
	Mode string
	// Format the output using the given Go template, e.g, 'json'
	Format string
	// Platform inspect content for a specific platform
	Platform string
}

// ImagePush specifies options for `(image) push`.
type ImagePush struct {
	Stdout      io.Writer
	GOptions    *Global
	SignOptions ImageSign
	SociOptions Soci
	// Platforms convert content for a specific platform
	Platforms []string
	// AllPlatforms convert content for all platforms
	AllPlatforms bool

	// Suppress verbose output
	Quiet bool
	// AllowNondistributableArtifacts allow pushing non-distributable artifacts
	AllowNondistributableArtifacts bool
}

// RemoteSnapshotterFlags are used for pulling with remote snapshotters
// e.g. SOCI
type RemoteSnapshotterFlags struct {
	SociIndexDigest string
}

// ImagePull specifies options for `(image) pull`.
type ImagePull struct {
	Stdout io.Writer
	Stderr io.Writer
	// ProgressOutputToStdout directs progress output to stdout instead of stderr
	ProgressOutputToStdout bool

	GOptions      *Global
	VerifyOptions ImageVerify
	// Unpack the image for the current single platform.
	// If nil, it will unpack automatically if only 1 platform is specified.
	Unpack *bool
	// Content for specific platforms. Empty if `--all-platforms` is true
	OCISpecPlatform []specs.Platform
	// Pull mode
	Mode string
	// Suppress verbose output
	Quiet bool
	// Flags to pass into remote snapshotters
	RFlags RemoteSnapshotterFlags
}

// ImageTag specifies options for `(image) tag`.
type ImageTag struct {
	// GOptions is the global options
	GOptions *Global
	// Source is the image to be referenced.
	Source string
	// Target is the image to be created.
	Target string
}

// ImageRemove specifies options for `rmi` and `image rm`.
type ImageRemove struct {
	Stdout io.Writer
	// GOptions is the global options
	GOptions *Global
	// Force removal of the image
	Force bool
	// Async asynchronous mode or not
	Async bool
}

// ImagePrune specifies options for `image prune` and `image rm`.
type ImagePrune struct {
	Stdout io.Writer
	// GOptions is the global options.
	GOptions *Global
	// All Remove all unused images, not just dangling ones.
	All bool
	// Filters output based on conditions provided for the --filter argument
	Filters []string
	// Force will not prompt for confirmation.
	Force bool
}

// ImageSave specifies options for `(image) save`.
type ImageSave struct {
	Stdout   io.Writer
	GOptions *Global
	// Export content for all platforms
	AllPlatforms bool
	// Export content for a specific platform
	Platform []string
}

// ImageSign contains options for signing an image. It contains options from
// all providers. The `provider` field determines which provider is used.
type ImageSign struct {
	// Provider used to sign the image (none|cosign|notation)
	Provider string
	// CosignKey Path to the private key file, KMS URI or Kubernetes Secret for --sign=cosign
	CosignKey string
	// NotationKeyName Signing key name for a key previously added to notation's key list for --sign=notation
	NotationKeyName string
}

// ImageVerify contains options for verifying an image. It contains options from
// all providers. The `provider` field determines which provider is used.
type ImageVerify struct {
	// Provider used to verify the image (none|cosign|notation)
	Provider string
	// CosignKey Path to the public key file, KMS URI or Kubernetes Secret for --verify=cosign
	CosignKey string
	// CosignCertificateIdentity The identity expected in a valid Fulcio certificate for --verify=cosign. Valid values
	// include email address, DNS names, IP addresses, and URIs. Either --cosign-certificate-identity or
	// --cosign-certificate-identity-regexp must be set for keyless flows
	CosignCertificateIdentity string
	// CosignCertificateIdentityRegexp A regular expression alternative to --cosign-certificate-identity for
	// --verify=cosign. Accepts the Go regular expression syntax described at https://golang.org/s/re2syntax. Either
	// --cosign-certificate-identity or --cosign-certificate-identity-regexp must be set for keyless flows
	CosignCertificateIdentityRegexp string
	// CosignCertificateOidcIssuer The OIDC issuer expected in a valid Fulcio certificate for --verify=cosign, e.g.
	// https://token.actions.githubusercontent.com or https://oauth2.sigstore.dev/auth. Either
	// --cosign-certificate-oidc-issuer or --cosign-certificate-oidc-issuer-regexp must be set for keyless flows
	CosignCertificateOidcIssuer string
	// CosignCertificateOidcIssuerRegexp A regular expression alternative to --certificate-oidc-issuer for
	// --verify=cosign. Accepts the Go regular expression syntax described at https://golang.org/s/re2syntax. Either
	// --cosign-certificate-oidc-issuer or --cosign-certificate-oidc-issuer-regexp must be set for keyless flows
	CosignCertificateOidcIssuerRegexp string
}

// Soci contains options for SOCI.
type Soci struct {
	// Span size that soci index uses to segment layer data. Default is 4 MiB.
	SpanSize int64
	// Minimum layer size to build zTOC for. Smaller layers won't have zTOC and not lazy pulled. Default is 10 MiB.
	MinLayerSize int64
}

// ImageLoad specifies options for `(image) load`.
type ImageLoad struct {
	Stdout   io.Writer
	Stdin    io.Reader
	GOptions *Global
	// Input read from tar archive file, instead of STDIN
	Input string
	// Platform import content for a specific platform
	Platform []string
	// AllPlatforms import content for all platforms
	AllPlatforms bool
	// Quiet suppresses the load output.
	Quiet bool
}
