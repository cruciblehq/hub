package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/cruciblehq/spec/registry"
)

// Executes an INSERT statement for a new namespace.
//
// Returns the created namespace with initialized timestamps on success, or the
// raw database error on failure without any translation or logging.
func (s *SQLStore) insertNamespace(ctx context.Context, info registry.NamespaceInfo) (*registry.Namespace, error) {
	now := time.Now().Unix()

	_, err := s.db.ExecContext(ctx, sqlNamespacesInsert,
		info.Name,
		info.Description,
		now, // created_at
		now, // updated_at
	)

	if err != nil {
		return nil, err
	}

	return &registry.Namespace{
		Name:        info.Name,
		Description: info.Description,
		Resources:   []registry.ResourceSummary{},
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// Queries a namespace by name from the database.
//
// Returns sql.ErrNoRows if the namespace does not exist. Returns the namespace
// without resource summaries on success.
func (s *SQLStore) getNamespace(ctx context.Context, name string) (*registry.Namespace, error) {

	var ns registry.Namespace

	if err := s.db.QueryRowContext(ctx, sqlNamespacesGet, name).Scan(
		&ns.Name,
		&ns.Description,
		&ns.CreatedAt,
		&ns.UpdatedAt,
	); err != nil {
		return nil, err
	}

	return &ns, nil
}

// Executes an UPDATE statement for a namespace's mutable fields.
//
// Returns the updated namespace on success, sql.ErrNoRows if the namespace does
// not exist, or the raw database error on failure without any translation or logging.
func (s *SQLStore) updateNamespace(ctx context.Context, namespace string, info registry.NamespaceInfo) (*registry.Namespace, error) {
	now := time.Now().Unix()

	result, err := s.db.ExecContext(ctx, sqlNamespacesUpdate, info.Description, now, namespace)
	if err != nil {
		return nil, err
	}

	if rows, _ := result.RowsAffected(); rows == 0 {
		return nil, sql.ErrNoRows
	}

	// Return updated namespace - need to query for created_at. We could use a
	// RETURNING clause, but the cost would be to lose compatibility with some
	// databases, like MySQL.
	return s.getNamespace(ctx, namespace)
}

// Executes a DELETE statement for a namespace.
//
// Returns the raw database error on failure without any translation or logging.
// Foreign key constraints prevent deletion if the namespace contains resources.
func (s *SQLStore) deleteNamespace(ctx context.Context, namespace string) error {
	_, err := s.db.ExecContext(ctx, sqlNamespacesDelete, namespace)
	return err
}

// Executes an INSERT statement for a new resource.
//
// Returns the created resource with initialized timestamps on success, or the
// raw database error on failure without any translation or logging.
func (s *SQLStore) insertResource(ctx context.Context, namespace string, info registry.ResourceInfo) (*registry.Resource, error) {
	now := time.Now().Unix()

	_, err := s.db.ExecContext(ctx, sqlResourcesInsert, namespace, info.Name, info.Type, info.Description, now, now)
	if err != nil {
		return nil, err
	}

	return &registry.Resource{
		Namespace:   namespace,
		Name:        info.Name,
		Type:        info.Type,
		Description: info.Description,
		Versions:    []registry.VersionSummary{},
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// Queries a resource by namespace and name from the database.
//
// Returns sql.ErrNoRows if the resource does not exist. Returns the resource
// without version summaries on success.
func (s *SQLStore) getResource(ctx context.Context, namespace, resource string) (*registry.Resource, error) {
	var res registry.Resource
	var ns string

	err := s.db.QueryRowContext(ctx, sqlResourcesGet, namespace, resource).Scan(
		&ns, &res.Name, &res.Type, &res.Description, &res.CreatedAt, &res.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	res.Namespace = namespace
	return &res, nil
}

// Executes an UPDATE statement for a resource's mutable fields.
//
// Returns the updated resource on success, sql.ErrNoRows if the resource does
// not exist, or the raw database error on failure without any translation or logging.
func (s *SQLStore) updateResource(ctx context.Context, namespace, resource string, info registry.ResourceInfo) (*registry.Resource, error) {
	now := time.Now().Unix()

	result, err := s.db.ExecContext(ctx, sqlResourcesUpdate, info.Type, info.Description, now, namespace, resource)
	if err != nil {
		return nil, err
	}

	if rows, _ := result.RowsAffected(); rows == 0 {
		return nil, sql.ErrNoRows
	}

	// Return updated resource - need to query for created_at.  We could use a
	// RETURNING clause, but the cost would be to lose compatibility with some
	// databases, like MySQL.
	return s.getResource(ctx, namespace, resource)
}

// Executes a DELETE statement for a resource.
//
// Returns the raw database error on failure without any translation or logging.
// Foreign key constraints prevent deletion if the resource contains versions.
func (s *SQLStore) deleteResource(ctx context.Context, namespace, resource string) error {
	_, err := s.db.ExecContext(ctx, sqlResourcesDelete, namespace, resource)
	return err
}

// Executes an INSERT statement for a new version.
//
// Returns the created version with initialized timestamps on success, or the
// raw database error on failure without any translation or logging. Archive
// fields (Digest, Size, Archive) are initialized to nil and must be populated
// via uploadArchive.
func (s *SQLStore) insertVersion(ctx context.Context, namespace, resource string, info registry.VersionInfo) (*registry.Version, error) {
	now := time.Now().Unix()

	_, err := s.db.ExecContext(ctx, sqlVersionsInsert, namespace, resource, info.String, now, now)
	if err != nil {
		return nil, err
	}

	return &registry.Version{
		Namespace: namespace,
		Resource:  resource,
		String:    info.String,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// Queries a version by namespace, resource, and version string from the database.
//
// Returns sql.ErrNoRows if the version does not exist. Archive fields (Digest, Size,
// Archive) are populated if an archive has been uploaded, otherwise they remain nil.
func (s *SQLStore) getVersion(ctx context.Context, namespace, resource, version string) (*registry.Version, error) {
	var v registry.Version
	var digest, path sql.NullString
	var size sql.NullInt64

	err := s.db.QueryRowContext(ctx, sqlVersionsGet, namespace, resource, version).Scan(
		&v.String, &v.Public, &digest, &size, &path, &v.CreatedAt, &v.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	v.Namespace = namespace
	v.Resource = resource

	if digest.Valid {
		v.Digest = &digest.String
	}
	if size.Valid {
		v.Size = &size.Int64
	}
	if path.Valid {
		v.Archive = &path.String
	}

	return &v, nil
}

// Executes an UPDATE statement for a version's mutable fields.
//
// Returns the updated version on success, sql.ErrNoRows if the version does not
// exist, or the raw database error on failure without any translation or logging.
func (s *SQLStore) updateVersion(ctx context.Context, namespace, resource, version string, public bool) (*registry.Version, error) {
	now := time.Now().Unix()

	result, err := s.db.ExecContext(ctx, sqlVersionsUpdate, public, now, namespace, resource, version)
	if err != nil {
		return nil, err
	}

	if rows, _ := result.RowsAffected(); rows == 0 {
		return nil, sql.ErrNoRows
	}

	// Return updated version - need to query for created_at and archive
	// details. We could use a RETURNING clause, but the cost would be to lose
	// compatibility with some databases, like MySQL.
	return s.getVersion(ctx, namespace, resource, version)
}

// Executes a DELETE statement for a version.
//
// Returns the raw database error on failure without any translation or logging.
func (s *SQLStore) deleteVersion(ctx context.Context, namespace, resource, version string) error {
	_, err := s.db.ExecContext(ctx, sqlVersionsDelete, namespace, resource, version)
	return err
}

// Queries all namespaces from the database.
//
// Returns the raw database error on failure without any translation or logging.
func (s *SQLStore) listNamespaces(ctx context.Context) ([]registry.NamespaceSummary, error) {
	rows, err := s.db.QueryContext(ctx, sqlNamespacesList)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var namespaces []registry.NamespaceSummary
	for rows.Next() {
		var ns registry.NamespaceSummary
		if err := rows.Scan(&ns.Name, &ns.Description, &ns.CreatedAt, &ns.ResourceCount); err != nil {
			return nil, err
		}
		namespaces = append(namespaces, ns)
	}
	return namespaces, rows.Err()
}

// Queries all resources in a namespace from the database.
//
// Returns the raw database error on failure without any translation or logging.
func (s *SQLStore) listResources(ctx context.Context, namespace string) ([]registry.ResourceSummary, error) {
	rows, err := s.db.QueryContext(ctx, sqlResourcesList, namespace)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var resources []registry.ResourceSummary
	for rows.Next() {
		var rs registry.ResourceSummary
		var latestVersion sql.NullString
		if err := rows.Scan(&rs.Name, &rs.Type, &rs.Description, &rs.CreatedAt, &rs.UpdatedAt, &rs.VersionCount, &latestVersion); err != nil {
			return nil, err
		}
		if latestVersion.Valid {
			rs.LatestVersion = &latestVersion.String
		}
		resources = append(resources, rs)
	}
	return resources, rows.Err()
}

// Queries all versions for a resource from the database.
//
// Returns the raw database error on failure without any translation or logging.
func (s *SQLStore) listVersions(ctx context.Context, namespace, resource string) ([]registry.VersionSummary, error) {
	rows, err := s.db.QueryContext(ctx, sqlVersionsList, namespace, resource)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []registry.VersionSummary
	for rows.Next() {
		var vs registry.VersionSummary
		var digest, size sql.NullString
		if err := rows.Scan(&vs.String, &vs.Public, &vs.CreatedAt, &vs.UpdatedAt, &digest, &size); err != nil {
			return nil, err
		}
		versions = append(versions, vs)
	}
	return versions, rows.Err()
}

// Executes an UPDATE statement to set archive metadata for a version.
//
// Returns sql.ErrNoRows if the version does not exist, or the raw database
// error on failure without any translation or logging.
func (s *SQLStore) uploadArchive(ctx context.Context, namespace, resource, version, digest, path string, size int64) error {
	now := time.Now().Unix()

	result, err := s.db.ExecContext(ctx, sqlVersionsUpload, digest, size, path, now, namespace, resource, version)
	if err != nil {
		return err
	}

	if rows, _ := result.RowsAffected(); rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// Returns all archive paths currently referenced by versions.
//
// The returned map is used for orphan archive garbage collection.
func (s *SQLStore) listArchivePaths(ctx context.Context) (map[string]struct{}, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT path FROM versions WHERE path IS NOT NULL AND path != ''")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	paths := make(map[string]struct{})
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, err
		}
		paths[path] = struct{}{}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return paths, nil
}
