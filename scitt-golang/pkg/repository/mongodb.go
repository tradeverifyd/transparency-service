package repository

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoDBRepository implements Repository using MongoDB
type MongoDBRepository struct {
	client   *mongo.Client
	db       *mongo.Database
	stmts    *mongo.Collection // statements collection
	treeSize *mongo.Collection // tree_size collection (singleton)
}

// MongoDBOptions holds configuration for MongoDB connection
type MongoDBOptions struct {
	URI      string // MongoDB connection URI
	Database string // Database name
}

// NewMongoDBRepository creates a new MongoDB repository
func NewMongoDBRepository(ctx context.Context, options MongoDBOptions) (*MongoDBRepository, error) {
	// Connect to MongoDB
	client, err := mongo.Connect(ctx, options2ClientOptions(&options.URI))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Ping to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		client.Disconnect(ctx)
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	db := client.Database(options.Database)

	repo := &MongoDBRepository{
		client:   client,
		db:       db,
		stmts:    db.Collection("statements"),
		treeSize: db.Collection("tree_size"),
	}

	// Initialize indexes and schema
	if err := repo.initializeSchema(ctx); err != nil {
		client.Disconnect(ctx)
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return repo, nil
}

// Convert URI string to ClientOptions
func options2ClientOptions(uri *string) *options.ClientOptions {
	return options.Client().ApplyURI(*uri)
}

// initializeSchema creates indexes for efficient querying
func (r *MongoDBRepository) initializeSchema(ctx context.Context) error {
	// Create unique index on leaf_hash
	_, err := r.stmts.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "leaf_hash", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return fmt.Errorf("failed to create leaf_hash index: %w", err)
	}

	// Create indexes for querying
	indexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "iss", Value: 1}}},
		{Keys: bson.D{{Key: "sub", Value: 1}}},
		{Keys: bson.D{{Key: "cty", Value: 1}}},
		{Keys: bson.D{{Key: "typ", Value: 1}}},
		{Keys: bson.D{{Key: "registered_at", Value: -1}}}, // Descending for recent-first queries
	}

	_, err = r.stmts.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return fmt.Errorf("failed to create statement indexes: %w", err)
	}

	// Initialize tree_size document if it doesn't exist
	_, err = r.treeSize.UpdateOne(
		ctx,
		bson.M{"_id": "current"},
		bson.M{"$setOnInsert": bson.M{
			"tree_size":    int64(0),
			"last_updated": time.Now(),
		}},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		return fmt.Errorf("failed to initialize tree_size: %w", err)
	}

	return nil
}

// statementDoc represents a statement document in MongoDB
type statementDoc struct {
	EntryID             int64     `bson:"entry_id"`
	LeafHash            string    `bson:"leaf_hash"`
	Iss                 string    `bson:"iss"`
	Sub                 *string   `bson:"sub,omitempty"`
	Cty                 *string   `bson:"cty,omitempty"`
	Typ                 *string   `bson:"typ,omitempty"`
	PayloadHashAlg      int       `bson:"payload_hash_alg"`
	PayloadHash         string    `bson:"payload_hash"`
	PreimageContentType *string   `bson:"preimage_content_type,omitempty"`
	PayloadLocation     *string   `bson:"payload_location,omitempty"`
	RegisteredAt        time.Time `bson:"registered_at"`
	EntryTileKey        string    `bson:"entry_tile_key"`
	EntryTileOffset     int       `bson:"entry_tile_offset"`
}

// treeSizeDoc represents the singleton tree size document
type treeSizeDoc struct {
	ID          string    `bson:"_id"` // Always "current"
	TreeSize    int64     `bson:"tree_size"`
	LastUpdated time.Time `bson:"last_updated"`
}

// InsertStatement inserts a new statement metadata
func (r *MongoDBRepository) InsertStatement(ctx context.Context, stmt *StatementMetadata) (int64, error) {
	// Use the EntryID from the provided statement metadata
	// The caller is responsible for managing entry IDs via GetCurrentTreeSize/SetCurrentTreeSize
	entryID := stmt.EntryID

	// Create document
	stmtDoc := statementDoc{
		EntryID:             entryID,
		LeafHash:            stmt.LeafHash,
		Iss:                 stmt.Iss,
		Sub:                 stmt.Sub,
		Cty:                 stmt.Cty,
		Typ:                 stmt.Typ,
		PayloadHashAlg:      stmt.PayloadHashAlg,
		PayloadHash:         stmt.PayloadHash,
		PreimageContentType: stmt.PreimageContentType,
		PayloadLocation:     stmt.PayloadLocation,
		RegisteredAt:        time.Now(),
		EntryTileKey:        stmt.EntryTileKey,
		EntryTileOffset:     stmt.EntryTileOffset,
	}

	_, err := r.stmts.InsertOne(ctx, stmtDoc)
	if err != nil {
		return 0, fmt.Errorf("failed to insert statement: %w", err)
	}

	return entryID, nil
}

// GetStatementByEntryID retrieves a statement by entry ID
func (r *MongoDBRepository) GetStatementByEntryID(ctx context.Context, entryID int64) (*StatementMetadata, error) {
	var doc statementDoc
	err := r.stmts.FindOne(ctx, bson.M{"entry_id": entryID}).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("statement with entry ID %d not found", entryID)
		}
		return nil, fmt.Errorf("failed to get statement by entry ID: %w", err)
	}

	return docToStatementMetadata(&doc), nil
}

// GetStatementByLeafHash retrieves a statement by leaf hash
func (r *MongoDBRepository) GetStatementByLeafHash(ctx context.Context, leafHash string) (*StatementMetadata, error) {
	var doc statementDoc
	err := r.stmts.FindOne(ctx, bson.M{"leaf_hash": leafHash}).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("statement with leaf hash %s not found", leafHash)
		}
		return nil, fmt.Errorf("failed to get statement by leaf hash: %w", err)
	}

	return docToStatementMetadata(&doc), nil
}

// QueryStatements queries statements with filters
func (r *MongoDBRepository) QueryStatements(ctx context.Context, query StatementQuery) ([]*StatementMetadata, error) {
	filter := bson.M{}

	if query.Iss != nil {
		filter["iss"] = *query.Iss
	}
	if query.Sub != nil {
		filter["sub"] = *query.Sub
	}
	if query.Cty != nil {
		filter["cty"] = *query.Cty
	}
	if query.Typ != nil {
		filter["typ"] = *query.Typ
	}
	if query.RegisteredAfter != nil {
		if filter["registered_at"] == nil {
			filter["registered_at"] = bson.M{}
		}
		filter["registered_at"].(bson.M)["$gte"] = *query.RegisteredAfter
	}
	if query.RegisteredBefore != nil {
		if filter["registered_at"] == nil {
			filter["registered_at"] = bson.M{}
		}
		filter["registered_at"].(bson.M)["$lte"] = *query.RegisteredBefore
	}

	opts := options.Find().SetSort(bson.D{{Key: "registered_at", Value: -1}})

	if query.Limit > 0 {
		opts.SetLimit(int64(query.Limit))
	}
	if query.Offset > 0 {
		opts.SetSkip(int64(query.Offset))
	}

	cursor, err := r.stmts.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to query statements: %w", err)
	}
	defer cursor.Close(ctx)

	var results []*StatementMetadata
	for cursor.Next(ctx) {
		var doc statementDoc
		if err := cursor.Decode(&doc); err != nil {
			return nil, fmt.Errorf("failed to decode statement: %w", err)
		}
		results = append(results, docToStatementMetadata(&doc))
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return results, nil
}

// GetCurrentTreeSize returns the current tree size
func (r *MongoDBRepository) GetCurrentTreeSize(ctx context.Context) (int64, error) {
	var doc treeSizeDoc
	err := r.treeSize.FindOne(ctx, bson.M{"_id": "current"}).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to get current tree size: %w", err)
	}

	return doc.TreeSize, nil
}

// SetCurrentTreeSize sets the current tree size
func (r *MongoDBRepository) SetCurrentTreeSize(ctx context.Context, size int64) error {
	if size < 0 {
		return fmt.Errorf("tree size cannot be negative: %d", size)
	}

	_, err := r.treeSize.UpdateOne(
		ctx,
		bson.M{"_id": "current"},
		bson.M{
			"$set": bson.M{
				"tree_size":    size,
				"last_updated": time.Now(),
			},
		},
		options.Update().SetUpsert(true),
	)

	if err != nil {
		return fmt.Errorf("failed to update tree size: %w", err)
	}

	return nil
}

// IncrementTreeSize atomically increments the tree size and returns the new value
func (r *MongoDBRepository) IncrementTreeSize(ctx context.Context) (int64, error) {
	result := r.treeSize.FindOneAndUpdate(
		ctx,
		bson.M{"_id": "current"},
		bson.M{
			"$inc": bson.M{"tree_size": 1},
			"$set": bson.M{"last_updated": time.Now()},
		},
		options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After),
	)

	var doc treeSizeDoc
	if err := result.Decode(&doc); err != nil {
		return 0, fmt.Errorf("failed to increment tree size: %w", err)
	}

	return doc.TreeSize, nil
}

// BeginTx begins a transaction
func (r *MongoDBRepository) BeginTx(ctx context.Context) (Transaction, error) {
	session, err := r.client.StartSession()
	if err != nil {
		return nil, fmt.Errorf("failed to start session: %w", err)
	}

	if err := session.StartTransaction(); err != nil {
		session.EndSession(ctx)
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}

	return &mongoTx{
		session: session,
		repo:    r,
		ctx:     ctx,
	}, nil
}

// Clear removes all data from the repository (for development/testing)
func (r *MongoDBRepository) Clear(ctx context.Context) error {
	// First, count how many documents exist
	count, err := r.stmts.CountDocuments(ctx, bson.M{})
	if err != nil {
		return fmt.Errorf("failed to count statements: %w", err)
	}
	fmt.Printf("Found %d documents in statements collection before deletion\n", count)
	fmt.Printf("Database: %s, Collection: %s\n", r.db.Name(), r.stmts.Name())

	// Delete all statements
	result, err := r.stmts.DeleteMany(ctx, bson.M{})
	if err != nil {
		return fmt.Errorf("failed to clear statements: %w", err)
	}

	// Log the number of documents deleted for debugging
	fmt.Printf("Deleted %d documents from statements collection\n", result.DeletedCount)

	// Count again to verify
	countAfter, err := r.stmts.CountDocuments(ctx, bson.M{})
	if err != nil {
		return fmt.Errorf("failed to count statements after deletion: %w", err)
	}
	fmt.Printf("Found %d documents in statements collection after deletion\n", countAfter)

	// Reset tree size to 0
	_, err = r.treeSize.UpdateOne(
		ctx,
		bson.M{"_id": "current"},
		bson.M{"$set": bson.M{
			"tree_size":    int64(0),
			"last_updated": time.Now(),
		}},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		return fmt.Errorf("failed to reset tree size: %w", err)
	}

	return nil
}

// Close closes the MongoDB connection
func (r *MongoDBRepository) Close() error {
	if r.client != nil {
		return r.client.Disconnect(context.Background())
	}
	return nil
}

// mongoTx implements Transaction for MongoDB
type mongoTx struct {
	session mongo.Session
	repo    *MongoDBRepository
	ctx     context.Context
}

// Commit commits the transaction
func (t *mongoTx) Commit() error {
	err := t.session.CommitTransaction(t.ctx)
	t.session.EndSession(t.ctx)
	return err
}

// Rollback rolls back the transaction
func (t *mongoTx) Rollback() error {
	err := t.session.AbortTransaction(t.ctx)
	t.session.EndSession(t.ctx)
	return err
}

// Transaction methods delegate to the repository
// In a real implementation, these would use the session context

func (t *mongoTx) InsertStatement(ctx context.Context, stmt *StatementMetadata) (int64, error) {
	return t.repo.InsertStatement(mongo.NewSessionContext(ctx, t.session), stmt)
}

func (t *mongoTx) GetStatementByEntryID(ctx context.Context, entryID int64) (*StatementMetadata, error) {
	return t.repo.GetStatementByEntryID(mongo.NewSessionContext(ctx, t.session), entryID)
}

func (t *mongoTx) GetStatementByLeafHash(ctx context.Context, leafHash string) (*StatementMetadata, error) {
	return t.repo.GetStatementByLeafHash(mongo.NewSessionContext(ctx, t.session), leafHash)
}

func (t *mongoTx) QueryStatements(ctx context.Context, query StatementQuery) ([]*StatementMetadata, error) {
	return t.repo.QueryStatements(mongo.NewSessionContext(ctx, t.session), query)
}

func (t *mongoTx) GetCurrentTreeSize(ctx context.Context) (int64, error) {
	return t.repo.GetCurrentTreeSize(mongo.NewSessionContext(ctx, t.session))
}

func (t *mongoTx) SetCurrentTreeSize(ctx context.Context, size int64) error {
	return t.repo.SetCurrentTreeSize(mongo.NewSessionContext(ctx, t.session), size)
}

func (t *mongoTx) IncrementTreeSize(ctx context.Context) (int64, error) {
	return t.repo.IncrementTreeSize(mongo.NewSessionContext(ctx, t.session))
}

func (t *mongoTx) BeginTx(ctx context.Context) (Transaction, error) {
	return nil, fmt.Errorf("nested transactions not supported")
}

func (t *mongoTx) Clear(ctx context.Context) error {
	return t.repo.Clear(mongo.NewSessionContext(ctx, t.session))
}

func (t *mongoTx) Close() error {
	return fmt.Errorf("cannot close transaction directly, use Commit or Rollback")
}

// Helper function to convert MongoDB document to StatementMetadata
func docToStatementMetadata(doc *statementDoc) *StatementMetadata {
	return &StatementMetadata{
		EntryID:                doc.EntryID,
		LeafHash:               doc.LeafHash,
		Iss:                    doc.Iss,
		Sub:                    doc.Sub,
		Cty:                    doc.Cty,
		Typ:                 doc.Typ,
		PayloadHashAlg:      doc.PayloadHashAlg,
		PayloadHash:         doc.PayloadHash,
		PreimageContentType: doc.PreimageContentType,
		PayloadLocation:     doc.PayloadLocation,
		RegisteredAt:        doc.RegisteredAt,
		EntryTileKey:        doc.EntryTileKey,
		EntryTileOffset:     doc.EntryTileOffset,
	}
}
