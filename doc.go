// Package ftp provides a thin, testable FTP client built on github.com/jlaffaye/ftp.
//
// Design constraints (for use in application frameworks):
//   - No viper, no global singletons, no log.Fatal inside the library.
//   - Credentials are supplied by the caller; there is no anonymous default user.
//   - Use NewClient + Close from main or a dedicated worker; pass context for cancel/timeouts.
//
// Capabilities:
//   - Client composes Uploader, Downloader, and Lister (inject narrow interfaces in tests).
//   - Streaming transfers with configurable buffer size; optional REST/SIZE resume.
//   - ListEach and WalkEach for large directories without holding giant []string slices.
//
// Limitations:
//   - Plain FTP only (port 21); FTPS/TLS is not implemented.
//   - A single Client supports one data connection at a time; do not run concurrent transfers.
package ftp
