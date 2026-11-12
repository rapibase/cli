# pgvector — embeddings and semantic search

The `vector` extension comes enabled in the instance. Full pattern (run it with
`rapibase sql --file schema.sql`, MCP: `execute_sql`):

```sql
CREATE TABLE docs (id bigserial PRIMARY KEY, content text, embedding vector(1536));
CREATE INDEX ON docs USING hnsw (embedding vector_cosine_ops);
-- insert: pass the embedding as '[0.12,-0.03,...]'::vector
-- top 5 most similar to the vector $1:
SELECT id, content, 1 - (embedding <=> $1::vector) AS score
FROM docs ORDER BY embedding <=> $1::vector LIMIT 5;
```

Generating the embeddings is up to the caller (or to a function that calls the embeddings API of
OpenAI/Voyage/etc. with `fetch` and an `FN_` secret: rapibase-functions skill). Expose the search as an RPC
function ([rpc.md](rpc.md)) or as an HTTP function.
