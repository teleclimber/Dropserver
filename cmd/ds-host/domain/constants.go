package domain

// AppExtractedPackageMaxSize is the max size that a package is allowed to inflate to.
// 1Gb for now.
const AppExtractedPackageMaxSize = int64(1 << 30)

const AppManifestMaxFileSize = int64(1 << 10 * 10) // 10kb

const AppNameMaxLength = 30
const AppShortDescriptionMaxLength = 60

// ZipBackupExtractedPackageMaxSize is the max size that a backup file is allowed to inflate to.
// 1Gb for now.
const ZipBackupExtractedPackageMaxSize = int64(1 << 30)
