#!/usr/bin/env python3
"""
SCITT MongoDB to Memgraph Graph Builder

This script extracts statements from MongoDB's scitt_demo database and builds
a graph in Memgraph with the following structure:

Nodes:
- Statement: Each statement from MongoDB
- Issuer: Each unique iss value
- Subject: Each unique sub value
- Artifact: Each unique payload_location

Relationships:
- (Statement)-[:IS_ISSUED_BY]->(Issuer)
- (Statement)-[:IS_STATEMENT_ABOUT]->(Subject)
- (Statement)-[:HAS_ARTIFACT_AT_LOCATION]->(Artifact)
"""

import os
from typing import Dict, Any
from pymongo import MongoClient
from gqlalchemy import Memgraph
from dotenv import load_dotenv

# Load environment variables from .env file
load_dotenv()

# Configuration from environment variables with defaults
MONGODB_URI = os.getenv("MONGODB_URI", "mongodb://localhost:27017/")
MONGODB_DATABASE = os.getenv("MONGODB_DATABASE", "scitt_demo")
MONGODB_COLLECTION = os.getenv("MONGODB_COLLECTION", "statements")

MEMGRAPH_HOST = os.getenv("MEMGRAPH_HOST", "127.0.0.1")
MEMGRAPH_PORT = int(os.getenv("MEMGRAPH_PORT", "7687"))


def connect_mongodb():
    """Connect to MongoDB and return the collection."""
    print(f"Connecting to MongoDB at {MONGODB_URI}")
    client = MongoClient(MONGODB_URI)
    db = client[MONGODB_DATABASE]
    collection = db[MONGODB_COLLECTION]
    return collection


def connect_memgraph():
    """Connect to Memgraph."""
    print(f"Connecting to Memgraph at {MEMGRAPH_HOST}:{MEMGRAPH_PORT}")
    return Memgraph(host=MEMGRAPH_HOST, port=MEMGRAPH_PORT)


def clear_memgraph(memgraph: Memgraph):
    """Clear all data from Memgraph."""
    print("Clearing existing data from Memgraph...")
    memgraph.execute("MATCH (n) DETACH DELETE n;")
    print("Memgraph cleared.")


def create_indexes(memgraph: Memgraph):
    """Create indexes in Memgraph for better performance."""
    print("Creating indexes...")
    try:
        memgraph.execute("CREATE INDEX ON :Statement(entry_id);")
        memgraph.execute("CREATE INDEX ON :Issuer(uri);")
        memgraph.execute("CREATE INDEX ON :Subject(uri);")
        memgraph.execute("CREATE INDEX ON :Artifact(location);")
        print("Indexes created.")
    except Exception as e:
        print(f"Note: Some indexes may already exist: {e}")


def create_or_merge_node(memgraph: Memgraph, label: str, key: str, value: str, properties: Dict[str, Any] = None):
    """Create or merge a node in Memgraph."""
    props = properties or {}
    props[key] = value

    # Build properties string for Cypher
    props_str = ", ".join([f"{k}: ${k}" for k in props.keys()])

    query = f"""
    MERGE (n:{label} {{{key}: ${key}}})
    SET n = {{{props_str}}}
    RETURN n
    """

    memgraph.execute(query, props)


def create_relationship(memgraph: Memgraph, from_label: str, from_key: str, from_value: Any,
                       rel_type: str, to_label: str, to_key: str, to_value: Any):
    """Create a relationship between two nodes."""
    query = f"""
    MATCH (a:{from_label} {{{from_key}: $from_value}})
    MATCH (b:{to_label} {{{to_key}: $to_value}})
    MERGE (a)-[r:{rel_type}]->(b)
    RETURN r
    """

    memgraph.execute(query, {"from_value": from_value, "to_value": to_value})


def import_statements(collection, memgraph: Memgraph):
    """Import statements from MongoDB to Memgraph."""
    statements = list(collection.find())
    total = len(statements)

    print(f"\nFound {total} statements to import.")

    for idx, statement in enumerate(statements, 1):
        entry_id = statement.get("entry_id")
        iss = statement.get("iss")
        sub = statement.get("sub")
        payload_location = statement.get("payload_location")

        if idx % 10 == 0 or idx == 1 or idx == total:
            print(f"Processing statement {idx}/{total} (entry_id: {entry_id})...")

        # Create Statement node
        statement_props = {
            "entry_id": entry_id,
            "leaf_hash": statement.get("leaf_hash"),
            "cty": statement.get("cty"),
            "payload_hash_alg": statement.get("payload_hash_alg"),
            "payload_hash": statement.get("payload_hash"),
            "registered_at": str(statement.get("registered_at")) if statement.get("registered_at") else None,
        }
        create_or_merge_node(memgraph, "Statement", "entry_id", entry_id, statement_props)

        # Create Issuer node and relationship
        if iss:
            create_or_merge_node(memgraph, "Issuer", "uri", iss)
            create_relationship(memgraph, "Statement", "entry_id", entry_id,
                              "IS_ISSUED_BY", "Issuer", "uri", iss)

        # Create Subject node and relationship
        if sub:
            create_or_merge_node(memgraph, "Subject", "uri", sub)
            create_relationship(memgraph, "Statement", "entry_id", entry_id,
                              "IS_STATEMENT_ABOUT", "Subject", "uri", sub)

        # Create Artifact node and relationship
        if payload_location:
            create_or_merge_node(memgraph, "Artifact", "location", payload_location)
            create_relationship(memgraph, "Statement", "entry_id", entry_id,
                              "HAS_ARTIFACT_AT_LOCATION", "Artifact", "location", payload_location)

    print(f"\nSuccessfully imported {total} statements!")


def print_statistics(memgraph: Memgraph):
    """Print statistics about the created graph."""
    print("\n" + "="*60)
    print("Graph Statistics")
    print("="*60)

    # Count nodes by label
    for label in ["Statement", "Issuer", "Subject", "Artifact"]:
        result = list(memgraph.execute_and_fetch(f"MATCH (n:{label}) RETURN count(n) as count"))
        count = result[0]['count'] if result else 0
        print(f"{label} nodes: {count}")

    # Count relationships
    for rel_type in ["IS_ISSUED_BY", "IS_STATEMENT_ABOUT", "HAS_ARTIFACT_AT_LOCATION"]:
        result = list(memgraph.execute_and_fetch(f"MATCH ()-[r:{rel_type}]->() RETURN count(r) as count"))
        count = result[0]['count'] if result else 0
        print(f"{rel_type} relationships: {count}")

    print("="*60 + "\n")


def main():
    """Main function to orchestrate the data transfer."""
    print("="*60)
    print("SCITT MongoDB to Memgraph Graph Builder")
    print("="*60 + "\n")

    try:
        # Connect to databases
        mongo_collection = connect_mongodb()
        memgraph = connect_memgraph()

        # Clear existing data (optional - comment out if you want to preserve existing data)
        clear_memgraph(memgraph)

        # Create indexes for performance
        create_indexes(memgraph)

        # Import statements
        import_statements(mongo_collection, memgraph)

        # Print statistics
        print_statistics(memgraph)

        print("Data transfer complete!")

    except Exception as e:
        print(f"Error: {e}")
        raise


if __name__ == "__main__":
    main()
