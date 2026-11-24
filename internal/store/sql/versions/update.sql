-- Updates an existing version's mutable fields.
UPDATE versions
SET public = ?, updated_at = ?
WHERE namespace = ? AND resource = ? AND string = ?;
