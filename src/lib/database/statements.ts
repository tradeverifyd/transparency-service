/**
 * Statement metadata queries
 *
 * Provides query operations for the statements table including:
 * - Insertion of new statements
 * - Finding statements by issuer, subject, content type, type
 * - Finding statements by date range
 * - Combined queries with multiple filters
 * - Getting statements by entry ID or hash
 */

import { Database } from "bun:sqlite";

export interface Statement {
  entry_id?: number;
  statement_hash: string;
  iss: string;
  sub: string | null;
  cty: string | null;
  typ: string | null;
  payload_hash_alg: number;
  payload_hash: string;
  preimage_content_type: string | null;
  payload_location: string | null;
  registered_at?: string;
  tree_size_at_registration: number;
  entry_tile_key: string;
  entry_tile_offset: number;
}

export interface StatementQueryFilters {
  iss?: string;
  sub?: string;
  cty?: string;
  typ?: string;
  registered_after?: string;
  registered_before?: string;
}

/**
 * Insert a new statement into the database
 */
export function insertStatement(db: Database, statement: Statement): number {
  const stmt = db.prepare(`
    INSERT INTO statements (
      statement_hash, iss, sub, cty, typ,
      payload_hash_alg, payload_hash,
      preimage_content_type, payload_location,
      tree_size_at_registration, entry_tile_key, entry_tile_offset
    ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
  `);

  const result = stmt.run(
    statement.statement_hash,
    statement.iss,
    statement.sub,
    statement.cty,
    statement.typ,
    statement.payload_hash_alg,
    statement.payload_hash,
    statement.preimage_content_type,
    statement.payload_location,
    statement.tree_size_at_registration,
    statement.entry_tile_key,
    statement.entry_tile_offset
  );

  return Number(result.lastInsertRowid);
}

/**
 * Find statements by issuer URL
 */
export function findStatementsByIssuer(db: Database, iss: string): Statement[] {
  const stmt = db.prepare(`
    SELECT * FROM statements WHERE iss = ? ORDER BY registered_at DESC
  `);

  return stmt.all(iss) as Statement[];
}

/**
 * Find statements by subject
 */
export function findStatementsBySubject(db: Database, sub: string): Statement[] {
  const stmt = db.prepare(`
    SELECT * FROM statements WHERE sub = ? ORDER BY registered_at DESC
  `);

  return stmt.all(sub) as Statement[];
}

/**
 * Find statements by content type
 */
export function findStatementsByContentType(db: Database, cty: string): Statement[] {
  const stmt = db.prepare(`
    SELECT * FROM statements WHERE cty = ? ORDER BY registered_at DESC
  `);

  return stmt.all(cty) as Statement[];
}

/**
 * Find statements by type
 */
export function findStatementsByType(db: Database, typ: string): Statement[] {
  const stmt = db.prepare(`
    SELECT * FROM statements WHERE typ = ? ORDER BY registered_at DESC
  `);

  return stmt.all(typ) as Statement[];
}

/**
 * Find statements within a date range
 */
export function findStatementsByDateRange(
  db: Database,
  startDate: string,
  endDate: string
): Statement[] {
  const stmt = db.prepare(`
    SELECT * FROM statements
    WHERE registered_at BETWEEN ? AND ?
    ORDER BY registered_at DESC
  `);

  return stmt.all(startDate, endDate) as Statement[];
}

/**
 * Find statements using combined filters
 */
export function findStatementsBy(
  db: Database,
  filters: StatementQueryFilters
): Statement[] {
  const conditions: string[] = [];
  const params: any[] = [];

  if (filters.iss) {
    conditions.push("iss = ?");
    params.push(filters.iss);
  }

  if (filters.sub) {
    conditions.push("sub = ?");
    params.push(filters.sub);
  }

  if (filters.cty) {
    conditions.push("cty = ?");
    params.push(filters.cty);
  }

  if (filters.typ) {
    conditions.push("typ = ?");
    params.push(filters.typ);
  }

  if (filters.registered_after) {
    conditions.push("registered_at >= ?");
    params.push(filters.registered_after);
  }

  if (filters.registered_before) {
    conditions.push("registered_at <= ?");
    params.push(filters.registered_before);
  }

  if (conditions.length === 0) {
    // No filters, return all statements
    const stmt = db.prepare("SELECT * FROM statements ORDER BY registered_at DESC");
    return stmt.all() as Statement[];
  }

  const whereClause = conditions.join(" AND ");
  const query = `SELECT * FROM statements WHERE ${whereClause} ORDER BY registered_at DESC`;
  const stmt = db.prepare(query);

  return stmt.all(...params) as Statement[];
}

/**
 * Get statement by entry ID
 */
export function getStatementByEntryId(db: Database, entryId: number): Statement | null {
  const stmt = db.prepare("SELECT * FROM statements WHERE entry_id = ?");
  const result = stmt.get(entryId);

  return result ? (result as Statement) : null;
}

/**
 * Get statement by hash
 */
export function getStatementByHash(db: Database, hash: string): Statement | null {
  const stmt = db.prepare("SELECT * FROM statements WHERE statement_hash = ?");
  const result = stmt.get(hash);

  return result ? (result as Statement) : null;
}
