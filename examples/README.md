# source{d} Engine Usage Examples

_You can find a few more examples in the quick start guide._

**Extract all functions as UAST nodes for Java files from HEAD**:

```sql
SELECT
    files.repository_id,
    files.file_path,
    UAST(files.blob_content, LANGUAGE(files.file_path, files.blob_content), '//FunctionGroup') as functions
FROM files
NATURAL JOIN commit_files
NATURAL JOIN commits
NATURAL JOIN refs
WHERE
    refs.ref_name= 'HEAD'
    AND LANGUAGE(files.file_path,files.blob_content) = 'Java'
LIMIT 10;
```

**Find all files where 'trim' method is called**:

```sql
SELECT * FROM (
  SELECT
      files.repository_id,
      files.file_path,
      UAST(files.blob_content, LANGUAGE(files.file_path, files.blob_content), '//*[@roleCallee]/Identifier[@Name="trim"]') as functionCall
  FROM files
  NATURAL JOIN commit_files
  NATURAL JOIN commits
  NATURAL JOIN refs
  WHERE
      refs.ref_name = 'HEAD'
) t WHERE ARRAY_LENGTH(functionCall) > 0
```

**Last commit messages in HEAD for every repository**

```sql
SELECT c.commit_message
FROM refs r
NATURAL JOIN commits c
WHERE r.ref_name = 'HEAD'
```

**Top 10 repositories by contributor count (all branches)**

```sql
SELECT repository_id,contributor_count FROM (
  SELECT
    repository_id,
    COUNT(DISTINCT commit_author_email) AS contributor_count
  FROM commits
  GROUP BY repository_id
) AS q
ORDER BY contributor_count DESC LIMIT 10
```

**Get all LICENSE blobs using pilosa index**

```sql
SELECT blob_content FROM files WHERE file_path = 'LICENSE'
```

**10 top repos by file count in HEAD**

```sql
SELECT repository_id, num_files FROM (
  SELECT COUNT(f.*) num_files, f.repository_id
  FROM ref_commits r
  NATURAL JOIN commit_files cf
  NATURAL JOIN files f
  WHERE r.ref_name = 'HEAD' GROUP BY f.repository_id
) AS t
ORDER BY num_files DESC LIMIT 10
```

**Top committers per repository**

```sql
SELECT * FROM (
  SELECT
    commit_author_email as author,
    repository_id as id,
    count(*) as num_commits
    FROM commits
    GROUP BY commit_author_email, repository_id
) AS t
ORDER BY num_commits DESC
```

**Top committers in all repositories**

```sql
SELECT * FROM (
  SELECT
    commit_author_email as author,
    count(*) as num_commits
    FROM commits
    GROUP BY commit_author_email
) AS t
ORDER BY num_commits DESC
```
