WITH updated_table AS (
    UPDATE "MY_TABLE" AS t1
    SET website = t2.domain
    FROM (
        SELECT id, REGEXP_REPLACE(website, '(https?:\/\/)?(www\.)?([^\/|\"]*).*?', '\3') AS domain
        FROM "MY_TABLE"
    ) AS t2
    WHERE t1.id = t2.id
    RETURNING *
)
SELECT t1.name AS spot_name,
       t2.domain,
       t2.domain_count
FROM updated_table t1
JOIN (
    SELECT domain, COUNT(*) AS domain_count
    FROM updated_table
    GROUP BY domain
    HAVING COUNT(*) > 1
) t2 ON t1.website = t2.domain;
