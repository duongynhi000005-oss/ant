-- Users of at least a given age.
SELECT * FROM users WHERE age >= @min ORDER BY name
