# SCITT MongoDB to Memgraph Data Tool

This tool extracts SCITT statements from MongoDB and builds a property graph in Memgraph.

## Graph Structure

![Graph Structure](graph-structure.png)

### Nodes

- **Statement**: Each statement from MongoDB with properties:
  - `entry_id`: Unique statement identifier
  - `leaf_hash`: Hash of the statement leaf
  - `cty`: Content type
  - `payload_hash_alg`: Hash algorithm used
  - `payload_hash`: Hash of the payload
  - `registered_at`: Registration timestamp

- **Issuer**: Each unique issuer (from `iss` field)
  - `uri`: Issuer URI

- **Subject**: Each unique subject (from `sub` field)
  - `uri`: Subject URI

- **Artifact**: Each unique artifact location (from `payload_location` field)
  - `location`: Artifact URL

### Relationships

- `(Statement)-[:IS_ISSUED_BY]->(Issuer)`: Links statements to their issuers
- `(Statement)-[:IS_STATEMENT_ABOUT]->(Subject)`: Links statements to their subjects
- `(Statement)-[:HAS_ARTIFACT_AT_LOCATION]->(Artifact)`: Links statements to artifact locations

## Prerequisites

- Python 3.13+
- [uv](https://docs.astral.sh/uv/) package manager
- MongoDB with `scitt_demo` database
- Memgraph instance running

## Installation

Dependencies are already defined in `pyproject.toml`. Install them with:

```bash
uv sync
```

## Configuration

Configure the tool using environment variables:

```bash
# MongoDB settings (defaults shown)
export MONGODB_URI="mongodb://localhost:27017/"
export MONGODB_DATABASE="scitt_demo"
export MONGODB_COLLECTION="statements"

# Memgraph settings (defaults shown)
export MEMGRAPH_HOST="127.0.0.1"
export MEMGRAPH_PORT="7687"
```

## Usage

Run the script:

```bash
uv run main.py
```

This will:
1. Connect to MongoDB and Memgraph
2. Clear existing data from Memgraph (comment out `clear_memgraph()` in `main()` to preserve data)
3. Create indexes for better query performance
4. Import all statements from MongoDB
5. Create nodes and relationships in Memgraph
6. Display statistics about the created graph

## Example Queries

After importing, you can query the graph in Memgraph. Here are some useful Cypher queries:

### Quick Data Exploration (START HERE!)

This comprehensive query gives you an overview of the entire graph structure, showing all node types and their relationships:

```cypher
MATCH (s:Statement)
OPTIONAL MATCH (s)-[:IS_ISSUED_BY]->(i:Issuer)
OPTIONAL MATCH (s)-[:IS_STATEMENT_ABOUT]->(sub:Subject)
OPTIONAL MATCH (s)-[:HAS_ARTIFACT_AT_LOCATION]->(a:Artifact)
RETURN s.entry_id, s.cty, s.registered_at,
       i.uri as issuer,
       sub.uri as subject,
       a.location as artifact_location
ORDER BY s.entry_id
LIMIT 20;
```

This will show you:
- 20 statements with their entry IDs, content types, and registration times
- The issuer for each statement
- The subject of each statement
- The artifact location for each statement

### Explore Graph Visually

To see the graph structure visually in Memgraph Lab:

```cypher
MATCH path = (s:Statement)-[r]->(n)
RETURN path
LIMIT 50;
```

This shows the first 50 relationships in a visual graph format, making it easy to understand the data structure.

### Find all statements issued by a specific issuer

```cypher
MATCH (s:Statement)-[:IS_ISSUED_BY]->(i:Issuer {uri: "urn:supply-chain:apex-semiconductor-corp"})
RETURN s;
```

### Find all artifacts referenced by statements

```cypher
MATCH (s:Statement)-[:HAS_ARTIFACT_AT_LOCATION]->(a:Artifact)
RETURN s.entry_id, a.location;
```

### Find the relationship between statements and subjects

```cypher
MATCH (s:Statement)-[:IS_STATEMENT_ABOUT]->(sub:Subject)
RETURN s.entry_id, sub.uri
ORDER BY s.entry_id;
```

### Find all statements for a specific subject

```cypher
MATCH (s:Statement)-[:IS_STATEMENT_ABOUT]->(sub:Subject {uri: "urn:supply-chain:apex-semiconductor-corp"})
RETURN s;
```

### Get complete graph for a statement

```cypher
MATCH (s:Statement {entry_id: 52})
OPTIONAL MATCH (s)-[:IS_ISSUED_BY]->(i:Issuer)
OPTIONAL MATCH (s)-[:IS_STATEMENT_ABOUT]->(sub:Subject)
OPTIONAL MATCH (s)-[:HAS_ARTIFACT_AT_LOCATION]->(a:Artifact)
RETURN s, i, sub, a;
```

### Count statements per issuer

```cypher
MATCH (s:Statement)-[:IS_ISSUED_BY]->(i:Issuer)
RETURN i.uri, count(s) as statement_count
ORDER BY statement_count DESC;
```

## Development

The script is structured with the following functions:

- `connect_mongodb()`: Connects to MongoDB
- `connect_memgraph()`: Connects to Memgraph
- `clear_memgraph()`: Clears all data from Memgraph
- `create_indexes()`: Creates indexes for performance
- `create_or_merge_node()`: Creates or updates a node
- `create_relationship()`: Creates a relationship between nodes
- `import_statements()`: Main import logic
- `print_statistics()`: Displays graph statistics

## Notes

- The script uses `MERGE` operations to avoid creating duplicate nodes
- Indexes are created on key properties for query performance
- The script clears Memgraph data by default - comment out the `clear_memgraph()` call to preserve existing data
- Progress is printed every 10 statements during import
