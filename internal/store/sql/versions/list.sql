-- Lists all versions for a resource.
--
-- Returns version metadata including archive information if uploaded.
SELECT 
    string,
    public,
    created_at,
    updated_at,
    digest,
    size
FROM versions
WHERE namespace = ? AND resource = ?
ORDER BY string DESC;
