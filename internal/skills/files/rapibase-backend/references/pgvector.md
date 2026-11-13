# pgvector — embeddings and semantic search

The `vector` extension comes enabled in the instance. The table and its index go in a file
(`rapibase sql --file schema.sql`, MCP: `execute_sql`):

```sql
CREATE TABLE docs (id bigserial PRIMARY KEY, content text, embedding vector(1536));
CREATE INDEX ON docs USING hnsw (embedding vector_cosine_ops);
-- insert: pass the embedding as '[0.12,-0.03,...]'::vector
```

The search is a single query with the vector as a parameter, in its own call (a script of several
statements takes no params and returns no rows):

```bash
# top 5 most similar to the vector $1 (MCP: execute_sql with params)
rapibase sql 'SELECT id, content, 1 - (embedding <=> $1::vector) AS score FROM docs ORDER BY embedding <=> $1::vector LIMIT 5' --params '["[0.12,-0.03,...]"]'
```

Generating the embeddings is up to the caller (or to a function that calls the embeddings API of
OpenAI/Voyage/etc. with `fetch` and an `FN_` secret: rapibase-functions skill). Expose the search as an RPC
function ([rpc.md](rpc.md)) or as an HTTP function.
