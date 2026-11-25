-- Deletes a version and its archive metadata.
DELETE FROM versions
WHERE namespace = ? AND resource = ? AND string = ?;
