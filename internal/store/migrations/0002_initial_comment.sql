ALTER TABLE comments ADD COLUMN is_initial INTEGER NOT NULL DEFAULT 0;

UPDATE comments
SET is_initial = 1
WHERE id IN (
  SELECT MIN(id) FROM comments GROUP BY annotation_id
);
