# source{d} Engine Usage Examples

_You can find a few more examples in the [quickstart guide](../README.md#5-start-executing-queries)._

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

</p>
</details>
