-- Check if exclude_from_overview column exists
SELECT COUNT(*) as count
FROM pragma_table_info('portfolio')
WHERE name='exclude_from_overview';

-- Add exclude_from_overview column if it doesn't exist
ALTER TABLE portfolio ADD COLUMN exclude_from_overview BOOLEAN DEFAULT FALSE;

-- Set default value for existing rows
UPDATE portfolio
SET exclude_from_overview = FALSE
WHERE exclude_from_overview IS NULL;

-- If the old is_hidden column exists, migrate its values and drop it
UPDATE portfolio
SET exclude_from_overview = is_hidden
WHERE EXISTS (
    SELECT 1 FROM pragma_table_info('portfolio') WHERE name='is_hidden'
);

DROP TABLE IF EXISTS old_portfolio;
CREATE TABLE old_portfolio AS SELECT * FROM portfolio;
CREATE TABLE portfolio AS SELECT id, name, description, is_archived, exclude_from_overview FROM old_portfolio;
DROP TABLE old_portfolio;
