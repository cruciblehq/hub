// Package store implements the [registry.Registry] interface using SQL databases.
//
// Provides persistent storage of namespaces, resources, versions, channels, and
// archives. Supports any SQL database with a compatible driver (SQLite,
// PostgreSQL, MySQL, etc.). Uses SQL for ACID transactions and referential
// integrity. Thread-safe for concurrent access.
package store

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"

	"github.com/cruciblehq/spec/reference"
	"github.com/cruciblehq/spec/registry"
)

const (
	errMsgCreateNamespace       = "unable to create namespace due to internal error"
	errMsgRetrieveNamespace     = "unable to retrieve namespace information"
	errMsgSaveNamespaceChanges  = "unable to save namespace changes"
	errMsgDeleteNamespace       = "unable to delete namespace - it may contain resources"
	errMsgRetrieveNamespaceList = "unable to retrieve namespace list"
	errMsgNamespaceNotFound     = "namespace not found"
	errMsgNamespaceExists       = "namespace already exists"

	errMsgCreateResource       = "unable to create resource due to internal error"
	errMsgRetrieveResource     = "unable to retrieve resource information"
	errMsgSaveResourceChanges  = "unable to save resource changes"
	errMsgDeleteResource       = "unable to delete resource - it may contain versions or channels"
	errMsgRetrieveResourceList = "unable to retrieve resource list for namespace"
	errMsgResourceNotFound     = "resource not found"
	errMsgResourceExists       = "resource already exists"

	errMsgCreateVersion        = "unable to create version due to internal error"
	errMsgRetrieveVersion      = "unable to retrieve version information"
	errMsgSaveVersionChanges   = "unable to save version changes"
	errMsgDeleteVersion        = "unable to delete version - it may be referenced by channels"
	errMsgRetrieveVersionList  = "unable to retrieve version list for resource"
	errMsgVersionNotFound      = "version not found"
	errMsgVersionExists        = "version already exists"
	errMsgVersionPublic        = "version is public and cannot change"
	errMsgVersionSuperseded    = "public version %q has the same major and minor and is higher"
	errMsgPublicPrerelease     = "a prerelease cannot be made public"
	errMsgPublicWithoutArchive = "a version without an archive cannot be made public"

	errMsgStoreArchive      = "unable to store archive metadata"
	errMsgUpdateArchive     = "unable to update archive metadata"
	errMsgAccessArchiveFile = "unable to access archive file - file may be missing or inaccessible"
	errMsgArchiveNotFound   = "archive not found"

	errMsgCreateChannel       = "unable to create channel - ensure the target version exists"
	errMsgUpdateChannel       = "unable to update channel - ensure the target version exists"
	errMsgRetrieveChannel     = "unable to retrieve channel information"
	errMsgDeleteChannel       = "unable to delete channel"
	errMsgRetrieveChannelList = "unable to retrieve channel list for resource"
	errMsgChannelNotFound     = "channel not found"
	errMsgChannelExists       = "channel already exists"
)

// Implements the [registry.Registry] interface using SQL databases.
//
// Provides persistent storage of namespaces, resources, versions, channels,
// and archives. Supports any SQL database with a compatible driver (SQLite,
// PostgreSQL, MySQL, etc.). Uses SQL for ACID transactions and referential
// integrity. Thread-safe for concurrent access. The store does not own
// the database connection. The caller is responsible for connection lifecycle
// management, including calling Close() on the *sql.DB.
type SQLStore struct {
	db          *sql.DB      // Database connection
	logger      *slog.Logger // Logger for store operations
	archiveRoot string       // Root directory for archive storage
	mu          sync.RWMutex // Protects archive storage operations
}

// Creates a new SQL database-backed registry store.
//
// The caller is responsible for:
//   - Opening the database connection with appropriate driver and settings
//   - Enabling driver-specific features (e.g., PRAGMA foreign_keys for SQLite)
//   - Managing connection lifecycle (calling Close() when done)
//   - Providing the archiveRoot directory where archive files will be stored
//
// The store will create the necessary schema if it doesn't exist.
func New(ctx context.Context, db *sql.DB, archiveRoot string, logger *slog.Logger) (*SQLStore, error) {
	logger = defaultLogger(logger)

	if _, err := db.ExecContext(ctx, sqlSchema); err != nil {
		logger.Error("failed to create schema", "error", err)
		return nil, &registry.Error{
			Code:    registry.ErrorCodeInternalError,
			Message: "failed to create schema",
		}
	}

	s := &SQLStore{
		db:          db,
		logger:      logger,
		archiveRoot: archiveRoot,
	}

	if err := s.cleanupOrphanArchiveFiles(ctx); err != nil {
		logger.Warn("failed to clean orphan archive files", "error", err)
	}

	return s, nil
}

// Creates a new namespace in the registry.
func (s *SQLStore) CreateNamespace(ctx context.Context, info registry.NamespaceInfo) (ns *registry.Namespace, err error) {
	if err := registry.ValidateNamespace(info.Name); err != nil {
		return nil, &registry.Error{Code: registry.ErrorCodeBadRequest, Message: err.Error()}
	}

	if ns, err = s.insertNamespace(ctx, info); err != nil {
		if _, err := s.getNamespace(ctx, info.Name); err == nil {
			return nil, &registry.Error{
				Code:    registry.ErrorCodeNamespaceExists,
				Message: errMsgNamespaceExists,
			}
		}

		return nil, s.logAndReturnError(registry.ErrorCodeInternalError, errMsgCreateNamespace, err, "namespace", info.Name)
	}

	return ns, nil
}

// Retrieves a namespace with resource summaries.
func (s *SQLStore) ReadNamespace(ctx context.Context, namespace string) (*registry.Namespace, error) {
	if err := registry.ValidateNamespace(namespace); err != nil {
		return nil, &registry.Error{Code: registry.ErrorCodeBadRequest, Message: err.Error()}
	}

	ns, err := s.getNamespace(ctx, namespace)
	if err == sql.ErrNoRows {
		return nil, &registry.Error{Code: registry.ErrorCodeNotFound, Message: errMsgNamespaceNotFound}
	}
	if err != nil {
		return nil, s.logAndReturnError(registry.ErrorCodeInternalError, errMsgRetrieveNamespace, err, "namespace", namespace)
	}

	resources, err := s.listResources(ctx, namespace)
	if err != nil {
		return nil, s.logAndReturnError(registry.ErrorCodeInternalError, errMsgRetrieveResourceList, err, "namespace", namespace)
	}
	ns.Resources = resources

	return ns, nil
}

// Updates a namespace's mutable metadata.
func (s *SQLStore) UpdateNamespace(ctx context.Context, namespace string, info registry.NamespaceInfo) (*registry.Namespace, error) {
	if err := registry.ValidateNamespace(namespace); err != nil {
		return nil, &registry.Error{Code: registry.ErrorCodeBadRequest, Message: err.Error()}
	}

	ns, err := s.updateNamespace(ctx, namespace, info)
	if err == sql.ErrNoRows {
		return nil, &registry.Error{Code: registry.ErrorCodeNotFound, Message: errMsgNamespaceNotFound}
	}
	if err != nil {
		return nil, s.logAndReturnError(registry.ErrorCodeInternalError, errMsgSaveNamespaceChanges, err, "namespace", namespace)
	}

	resources, err := s.listResources(ctx, namespace)
	if err != nil {
		return nil, s.logAndReturnError(registry.ErrorCodeInternalError, errMsgRetrieveResourceList, err, "namespace", namespace)
	}
	ns.Resources = resources

	return ns, nil
}

// Permanently deletes a namespace.
func (s *SQLStore) DeleteNamespace(ctx context.Context, namespace string) error {
	if err := registry.ValidateNamespace(namespace); err != nil {
		return &registry.Error{Code: registry.ErrorCodeBadRequest, Message: err.Error()}
	}

	if err := s.deleteNamespace(ctx, namespace); err != nil {
		return s.logAndReturnError(registry.ErrorCodeInternalError, errMsgDeleteNamespace, err, "namespace", namespace)
	}

	return nil
}

// Returns all namespaces in the registry.
func (s *SQLStore) ListNamespaces(ctx context.Context) (*registry.NamespaceList, error) {
	namespaces, err := s.listNamespaces(ctx)
	if err != nil {
		return nil, s.logAndReturnError(registry.ErrorCodeInternalError, errMsgRetrieveNamespaceList, err)
	}
	return &registry.NamespaceList{Namespaces: namespaces}, nil
}

// Creates a new resource in a namespace.
func (s *SQLStore) CreateResource(ctx context.Context, namespace string, info registry.ResourceInfo) (*registry.Resource, error) {
	if err := registry.ValidateIdentifier(namespace, info.Name); err != nil {
		return nil, &registry.Error{Code: registry.ErrorCodeBadRequest, Message: err.Error()}
	}

	res, err := s.insertResource(ctx, namespace, info)
	if err != nil {
		if _, err := s.getResource(ctx, namespace, info.Name); err == nil {
			return nil, &registry.Error{Code: registry.ErrorCodeResourceExists, Message: errMsgResourceExists}
		}

		if _, err := s.getNamespace(ctx, namespace); err == sql.ErrNoRows {
			return nil, &registry.Error{Code: registry.ErrorCodeNotFound, Message: errMsgNamespaceNotFound}
		}

		return nil, s.logAndReturnError(registry.ErrorCodeInternalError, errMsgCreateResource, err, "namespace", namespace, "resource", info.Name)
	}

	return res, nil
}

// Retrieves a resource with version and channel summaries.
func (s *SQLStore) ReadResource(ctx context.Context, namespace, resource string) (*registry.Resource, error) {
	if err := registry.ValidateIdentifier(namespace, resource); err != nil {
		return nil, &registry.Error{Code: registry.ErrorCodeBadRequest, Message: err.Error()}
	}

	res, err := s.getResource(ctx, namespace, resource)
	if err == sql.ErrNoRows {
		return nil, &registry.Error{Code: registry.ErrorCodeNotFound, Message: errMsgResourceNotFound}
	}
	if err != nil {
		return nil, s.logAndReturnError(registry.ErrorCodeInternalError, errMsgRetrieveResource, err, "namespace", namespace)
	}

	versions, err := s.listVersions(ctx, namespace, resource)
	if err != nil {
		return nil, s.logAndReturnError(registry.ErrorCodeInternalError, errMsgRetrieveVersionList, err, "namespace", namespace, "resource", resource)
	}
	res.Versions = versions

	channels, err := s.listChannels(ctx, namespace, resource)
	if err != nil {
		return nil, s.logAndReturnError(registry.ErrorCodeInternalError, errMsgRetrieveChannelList, err, "namespace", namespace, "resource", resource)
	}
	res.Channels = channels

	return res, nil
}

// Updates a resource's mutable metadata.
func (s *SQLStore) UpdateResource(ctx context.Context, namespace, resource string, info registry.ResourceInfo) (*registry.Resource, error) {
	if err := registry.ValidateIdentifier(namespace, resource); err != nil {
		return nil, &registry.Error{Code: registry.ErrorCodeBadRequest, Message: err.Error()}
	}

	res, err := s.updateResource(ctx, namespace, resource, info)
	if err == sql.ErrNoRows {
		return nil, &registry.Error{Code: registry.ErrorCodeNotFound, Message: errMsgResourceNotFound}
	}
	if err != nil {
		return nil, s.logAndReturnError(registry.ErrorCodeInternalError, errMsgSaveResourceChanges, err, "namespace", namespace, "resource", resource)
	}

	versions, err := s.listVersions(ctx, namespace, resource)
	if err != nil {
		return nil, s.logAndReturnError(registry.ErrorCodeInternalError, errMsgRetrieveVersionList, err, "namespace", namespace, "resource", resource)
	}
	res.Versions = versions

	channels, err := s.listChannels(ctx, namespace, resource)
	if err != nil {
		return nil, s.logAndReturnError(registry.ErrorCodeInternalError, errMsgRetrieveChannelList, err, "namespace", namespace, "resource", resource)
	}
	res.Channels = channels

	return res, nil
}

// Permanently deletes a resource.
func (s *SQLStore) DeleteResource(ctx context.Context, namespace, resource string) error {
	if err := registry.ValidateIdentifier(namespace, resource); err != nil {
		return &registry.Error{Code: registry.ErrorCodeBadRequest, Message: err.Error()}
	}

	if err := s.deleteResource(ctx, namespace, resource); err != nil {
		return s.logAndReturnError(registry.ErrorCodeInternalError, errMsgDeleteResource, err, "namespace", namespace, "resource", resource)
	}

	return nil
}

// Returns all resources in a namespace.
func (s *SQLStore) ListResources(ctx context.Context, namespace string) (*registry.ResourceList, error) {
	if err := registry.ValidateNamespace(namespace); err != nil {
		return nil, &registry.Error{Code: registry.ErrorCodeBadRequest, Message: err.Error()}
	}

	resources, err := s.listResources(ctx, namespace)
	if err != nil {
		return nil, s.logAndReturnError(registry.ErrorCodeInternalError, errMsgRetrieveResourceList, err, "namespace", namespace)
	}

	return &registry.ResourceList{Resources: resources}, nil
}

// Creates a new private version for a resource.
//
// Fails with [registry.ErrorCodeVersionSuperseded] if a public version with
// the same major and minor is higher.
func (s *SQLStore) CreateVersion(ctx context.Context, namespace, resource string, info registry.VersionInfo) (*registry.Version, error) {
	if err := registry.ValidateReference(namespace, resource, info.String); err != nil {
		return nil, &registry.Error{Code: registry.ErrorCodeBadRequest, Message: err.Error()}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.checkReleaseLine(ctx, namespace, resource, info.String); err != nil {
		return nil, err
	}

	v, err := s.insertVersion(ctx, namespace, resource, info)
	if err == nil {
		return v, nil
	}

	if _, checkErr := s.getVersion(ctx, namespace, resource, info.String); checkErr == nil {
		return nil, &registry.Error{Code: registry.ErrorCodeVersionExists, Message: errMsgVersionExists}
	}

	if _, checkErr := s.getResource(ctx, namespace, resource); checkErr == sql.ErrNoRows {
		return nil, &registry.Error{Code: registry.ErrorCodeNotFound, Message: errMsgResourceNotFound}
	}

	return nil, s.logAndReturnError(registry.ErrorCodeInternalError, errMsgCreateVersion, err, "namespace", namespace, "resource", resource, "version", info.String)
}

// Retrieves a version with its archive details.
func (s *SQLStore) ReadVersion(ctx context.Context, namespace, resource, version string) (*registry.Version, error) {
	if err := registry.ValidateReference(namespace, resource, version); err != nil {
		return nil, &registry.Error{Code: registry.ErrorCodeBadRequest, Message: err.Error()}
	}

	v, err := s.getVersion(ctx, namespace, resource, version)
	if err == sql.ErrNoRows {
		return nil, &registry.Error{Code: registry.ErrorCodeNotFound, Message: errMsgVersionNotFound}
	}
	if err != nil {
		return nil, s.logAndReturnError(registry.ErrorCodeInternalError, errMsgRetrieveVersion, err, "namespace", namespace, "resource", resource, "version", version)
	}
	return v, nil
}

// Updates a private version.
//
// A public version fails with [registry.ErrorCodeVersionPublic]. Setting
// info.Public makes the version public, after [SQLStore.checkCanBePublic].
func (s *SQLStore) UpdateVersion(ctx context.Context, namespace, resource, version string, info registry.VersionInfo) (*registry.Version, error) {
	if err := registry.ValidateReference(namespace, resource, version); err != nil {
		return nil, &registry.Error{Code: registry.ErrorCodeBadRequest, Message: err.Error()}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.getVersion(ctx, namespace, resource, version)
	if err != nil {
		return nil, s.versionError(err, errMsgRetrieveVersion, namespace, resource, version)
	}
	if current.Public {
		return nil, &registry.Error{Code: registry.ErrorCodeVersionPublic, Message: errMsgVersionPublic}
	}
	if info.Public {
		if err := s.checkCanBePublic(ctx, current); err != nil {
			return nil, err
		}
	}

	v, err := s.updateVersion(ctx, namespace, resource, version, info.Public)
	if err == sql.ErrNoRows {
		return nil, &registry.Error{Code: registry.ErrorCodeNotFound, Message: errMsgVersionNotFound}
	}
	if err != nil {
		return nil, s.logAndReturnError(registry.ErrorCodeInternalError, errMsgSaveVersionChanges, err, "namespace", namespace, "resource", resource, "version", version)
	}
	return v, nil
}

// Checks that a private version can be made public.
//
// The version needs an archive, must not be a prerelease, and must pass
// [SQLStore.checkReleaseLine].
func (s *SQLStore) checkCanBePublic(ctx context.Context, v *registry.Version) error {
	if v.Archive == nil {
		return &registry.Error{Code: registry.ErrorCodeBadRequest, Message: errMsgPublicWithoutArchive}
	}
	parsed, err := reference.ParseVersion(v.String)
	if err != nil {
		return &registry.Error{Code: registry.ErrorCodeBadRequest, Message: err.Error()}
	}
	if parsed.IsPrerelease() {
		return &registry.Error{Code: registry.ErrorCodeBadRequest, Message: errMsgPublicPrerelease}
	}
	return s.checkReleaseLine(ctx, v.Namespace, v.Resource, v.String)
}

// Checks that version does not go back on its release line.
//
// Fails with [registry.ErrorCodeVersionSuperseded] if a public version with the
// same major and minor is higher. Private versions do not count.
func (s *SQLStore) checkReleaseLine(ctx context.Context, namespace, resource, version string) error {
	candidate, err := reference.ParseVersion(version)
	if err != nil {
		return &registry.Error{Code: registry.ErrorCodeBadRequest, Message: err.Error()}
	}

	versions, err := s.listVersions(ctx, namespace, resource)
	if err != nil {
		return s.logAndReturnError(registry.ErrorCodeInternalError, errMsgRetrieveVersionList, err, "namespace", namespace, "resource", resource)
	}

	for _, other := range versions {
		if other.Public && isHigherOnLine(other.String, candidate) {
			return &registry.Error{Code: registry.ErrorCodeVersionSuperseded, Message: fmt.Sprintf(errMsgVersionSuperseded, other.String)}
		}
	}
	return nil
}

// Whether version has the same major and minor as candidate and is higher.
func isHigherOnLine(version string, candidate *reference.Version) bool {
	v, err := reference.ParseVersion(version)
	if err != nil || v.Major != candidate.Major || v.Minor != candidate.Minor {
		return false
	}
	c, ok := v.Compare(candidate)
	return ok && c > 0
}

// Permanently deletes a version.
func (s *SQLStore) DeleteVersion(ctx context.Context, namespace, resource, version string) error {
	if err := registry.ValidateReference(namespace, resource, version); err != nil {
		return &registry.Error{Code: registry.ErrorCodeBadRequest, Message: err.Error()}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	v, err := s.getVersion(ctx, namespace, resource, version)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return s.logAndReturnError(registry.ErrorCodeInternalError, errMsgRetrieveVersion, err, "namespace", namespace, "resource", resource, "version", version)
	}

	if err := s.deleteVersion(ctx, namespace, resource, version); err != nil {
		return s.logAndReturnError(registry.ErrorCodeInternalError, errMsgDeleteVersion, err, "namespace", namespace, "resource", resource, "version", version)
	}

	if v.Archive != nil {
		if err := s.removeArchiveFileAndParents(*v.Archive); err != nil {
			s.logger.Warn("failed to remove archive file after version deletion", "namespace", namespace, "resource", resource, "version", version, "path", *v.Archive, "error", err)
		}
	}
	return nil
}

// Returns all versions for a resource.
func (s *SQLStore) ListVersions(ctx context.Context, namespace, resource string) (*registry.VersionList, error) {
	if err := registry.ValidateIdentifier(namespace, resource); err != nil {
		return nil, &registry.Error{Code: registry.ErrorCodeBadRequest, Message: err.Error()}
	}

	versions, err := s.listVersions(ctx, namespace, resource)
	if err != nil {
		return nil, s.logAndReturnError(registry.ErrorCodeInternalError, errMsgRetrieveVersionList, err, "namespace", namespace, "resource", resource)
	}
	return &registry.VersionList{Versions: versions}, nil
}

// Uploads an archive for a version.
//
// A private version's archive is replaced. A public version fails with
// [registry.ErrorCodeVersionPublic].
func (s *SQLStore) UploadArchive(ctx context.Context, namespace, resource, version string, archiveReader io.Reader) (*registry.Version, error) {
	if err := registry.ValidateReference(namespace, resource, version); err != nil {
		return nil, &registry.Error{Code: registry.ErrorCodeBadRequest, Message: err.Error()}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	previous, err := s.getVersion(ctx, namespace, resource, version)
	if err != nil {
		return nil, s.versionError(err, errMsgRetrieveVersion, namespace, resource, version)
	}
	if previous.Public {
		return nil, &registry.Error{Code: registry.ErrorCodeVersionPublic, Message: errMsgVersionPublic}
	}

	digest, archivePath, size, err := s.storeArchiveFile(namespace, resource, version, archiveReader)
	if err != nil {
		return nil, s.logAndReturnError(registry.ErrorCodeInternalError, errMsgStoreArchive, err, "namespace", namespace, "resource", resource, "version", version)
	}

	if err := s.uploadArchive(ctx, namespace, resource, version, digest, archivePath, size); err != nil {
		os.Remove(archivePath)
		return nil, s.versionError(err, errMsgUpdateArchive, namespace, resource, version)
	}

	if previous.Archive != nil && *previous.Archive != archivePath {
		s.removeSupersededArchive(*previous.Archive, namespace, resource, version)
	}

	v, err := s.getVersion(ctx, namespace, resource, version)
	if err != nil {
		return nil, s.logAndReturnError(registry.ErrorCodeInternalError, errMsgRetrieveVersion, err, "namespace", namespace, "resource", resource, "version", version)
	}

	return v, nil
}

// Maps a failed query for a version to a registry error.
//
// A missing row is a not found error. Anything else is an internal error with
// message msg.
func (s *SQLStore) versionError(err error, msg, namespace, resource, version string) error {
	if err == sql.ErrNoRows {
		return s.logAndReturnError(registry.ErrorCodeNotFound, errMsgVersionNotFound, err, "namespace", namespace, "resource", resource, "version", version)
	}
	return s.logAndReturnError(registry.ErrorCodeInternalError, msg, err, "namespace", namespace, "resource", resource, "version", version)
}

// Removes the archive file a new upload replaced.
//
// A failure is only logged, because the new archive is already in place.
func (s *SQLStore) removeSupersededArchive(path, namespace, resource, version string) {
	if err := s.removeArchiveFileAndParents(path); err != nil {
		s.logger.Warn("failed to remove superseded archive file", "namespace", namespace, "resource", resource, "version", version, "path", path, "error", err)
	}
}

// Returns a reader for a version's archive.
func (s *SQLStore) DownloadArchive(ctx context.Context, namespace, resource, version string) (io.ReadCloser, error) {
	if err := registry.ValidateReference(namespace, resource, version); err != nil {
		return nil, &registry.Error{Code: registry.ErrorCodeBadRequest, Message: err.Error()}
	}

	v, err := s.getVersion(ctx, namespace, resource, version)
	if err == sql.ErrNoRows {
		return nil, &registry.Error{Code: registry.ErrorCodeNotFound, Message: errMsgVersionNotFound}
	}
	if err != nil {
		return nil, s.logAndReturnError(registry.ErrorCodeInternalError, errMsgRetrieveVersion, err, "namespace", namespace, "resource", resource, "version", version)
	}

	if v.Archive == nil {
		return nil, &registry.Error{Code: registry.ErrorCodeNotFound, Message: errMsgArchiveNotFound}
	}

	file, err := os.Open(*v.Archive)
	if err != nil {
		return nil, s.logAndReturnError(registry.ErrorCodeInternalError, errMsgAccessArchiveFile, err, "namespace", namespace, "resource", resource, "version", version)
	}

	return file, nil
}

// Creates a new channel.
func (s *SQLStore) CreateChannel(ctx context.Context, namespace, resource string, info registry.ChannelInfo) (*registry.Channel, error) {
	if err := registry.ValidateChannelInfo(namespace, resource, info); err != nil {
		return nil, &registry.Error{Code: registry.ErrorCodeBadRequest, Message: err.Error()}
	}

	if err := s.insertChannel(ctx, namespace, resource, info); err != nil {
		if _, checkErr := s.getChannel(ctx, namespace, resource, info.Name); checkErr == nil {
			return nil, &registry.Error{Code: registry.ErrorCodeChannelExists, Message: errMsgChannelExists}
		}

		if _, checkErr := s.getResource(ctx, namespace, resource); checkErr == sql.ErrNoRows {
			return nil, &registry.Error{Code: registry.ErrorCodeNotFound, Message: errMsgResourceNotFound}
		}

		return nil, s.logAndReturnError(registry.ErrorCodeInternalError, errMsgCreateChannel, err, "namespace", namespace, "resource", resource, "channel", info.Name)
	}

	c, err := s.getChannel(ctx, namespace, resource, info.Name)
	if err != nil {
		return nil, s.logAndReturnError(registry.ErrorCodeInternalError, errMsgRetrieveChannel, err, "namespace", namespace, "resource", resource, "channel", info.Name)
	}
	return c, nil
}

// Updates a channel's mutable metadata.
func (s *SQLStore) UpdateChannel(ctx context.Context, namespace, resource, channel string, info registry.ChannelInfo) (*registry.Channel, error) {
	if err := registry.ValidateChannelInfo(namespace, resource, info); err != nil {
		return nil, &registry.Error{Code: registry.ErrorCodeBadRequest, Message: err.Error()}
	}

	c, err := s.updateChannel(ctx, namespace, resource, info)
	if err == sql.ErrNoRows {
		return nil, &registry.Error{Code: registry.ErrorCodeNotFound, Message: errMsgChannelNotFound}
	}
	if err != nil {
		return nil, s.logAndReturnError(registry.ErrorCodeInternalError, errMsgUpdateChannel, err, "namespace", namespace, "resource", resource, "channel", info.Name)
	}
	return c, nil
}

// Retrieves a channel with its version details.
func (s *SQLStore) ReadChannel(ctx context.Context, namespace, resource, channel string) (*registry.Channel, error) {
	if err := registry.ValidateChannelReference(namespace, resource, channel); err != nil {
		return nil, &registry.Error{Code: registry.ErrorCodeBadRequest, Message: err.Error()}
	}

	c, err := s.getChannel(ctx, namespace, resource, channel)
	if err == sql.ErrNoRows {
		return nil, &registry.Error{Code: registry.ErrorCodeNotFound, Message: errMsgChannelNotFound}
	}
	if err != nil {
		return nil, s.logAndReturnError(registry.ErrorCodeInternalError, errMsgRetrieveChannel, err, "namespace", namespace, "resource", resource, "channel", channel)
	}
	return c, nil
}

// Permanently deletes a channel.
func (s *SQLStore) DeleteChannel(ctx context.Context, namespace, resource, channel string) error {
	if err := registry.ValidateChannelReference(namespace, resource, channel); err != nil {
		return &registry.Error{Code: registry.ErrorCodeBadRequest, Message: err.Error()}
	}

	if err := s.deleteChannel(ctx, namespace, resource, channel); err != nil {
		return s.logAndReturnError(registry.ErrorCodeInternalError, errMsgDeleteChannel, err, "namespace", namespace, "resource", resource, "channel", channel)
	}
	return nil
}

// Returns all channels for a resource.
func (s *SQLStore) ListChannels(ctx context.Context, namespace, resource string) (*registry.ChannelList, error) {
	if err := registry.ValidateIdentifier(namespace, resource); err != nil {
		return nil, &registry.Error{Code: registry.ErrorCodeBadRequest, Message: err.Error()}
	}

	channels, err := s.listChannels(ctx, namespace, resource)
	if err != nil {
		return nil, s.logAndReturnError(registry.ErrorCodeInternalError, errMsgRetrieveChannelList, err, "namespace", namespace, "resource", resource)
	}
	return &registry.ChannelList{Channels: channels}, nil
}
