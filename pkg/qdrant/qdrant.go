package qdrant

import (
	"context"
	"crypto/md5"
	"fmt"
	"regexp"
	"time"

	pb "github.com/qdrant/go-client/qdrant"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ... (rest of the code remains the same)

// reUUID is a pre-compiled regex for UUID validation.
var reUUID = regexp.MustCompile("^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}$")

// Ping checks if Qdrant is reachable.
func (c *qdrantImpl) Ping(ctx context.Context) error {
	_, err := c.collectionsClient.List(ctx, &pb.ListCollectionsRequest{})
	if err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}
	return nil
}

// GetPointsClient returns the underlying points client for advanced operations.
func (c *qdrantImpl) GetPointsClient() pb.PointsClient {
	return c.pointsClient
}

// GetCollectionsClient returns the underlying collections client for advanced operations.
func (c *qdrantImpl) GetCollectionsClient() pb.CollectionsClient {
	return c.collectionsClient
}

// GetDefaultTimeout returns the default timeout for operations.
func (c *qdrantImpl) GetDefaultTimeout() time.Duration {
	return c.defaultTimeout
}

// CreateCollection creates a new collection in Qdrant.
func (c *qdrantImpl) CreateCollection(ctx context.Context, name string, vectorSize uint64, distance pb.Distance) error {
	if name == "" {
		return ErrEmptyCollection
	}
	if vectorSize == 0 {
		return ErrInvalidVectorSize
	}
	_, err := c.collectionsClient.Create(ctx, &pb.CreateCollection{
		CollectionName: name,
		VectorsConfig: &pb.VectorsConfig{
			Config: &pb.VectorsConfig_Params{
				Params: &pb.VectorParams{
					Size:     vectorSize,
					Distance: distance,
				},
			},
		},
	})
	if err != nil {
		return WrapError(err, "failed to create collection")
	}
	return nil
}

// DeleteCollection deletes a collection from Qdrant.
func (c *qdrantImpl) DeleteCollection(ctx context.Context, name string) error {
	if name == "" {
		return ErrEmptyCollection
	}
	_, err := c.collectionsClient.Delete(ctx, &pb.DeleteCollection{CollectionName: name})
	if err != nil {
		return WrapError(err, "failed to delete collection")
	}
	return nil
}

// CollectionExists checks if a collection exists.
func (c *qdrantImpl) CollectionExists(ctx context.Context, name string) (bool, error) {
	if name == "" {
		return false, ErrEmptyCollection
	}
	resp, err := c.collectionsClient.Get(ctx, &pb.GetCollectionInfoRequest{CollectionName: name})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return false, nil
		}
		return false, fmt.Errorf("check collection: %w", err)
	}
	return resp != nil, nil
}

// GetCollectionInfo retrieves information about a collection.
func (c *qdrantImpl) GetCollectionInfo(ctx context.Context, name string) (*CollectionInfo, error) {
	if name == "" {
		return nil, ErrEmptyCollection
	}
	resp, err := c.collectionsClient.Get(ctx, &pb.GetCollectionInfoRequest{CollectionName: name})
	if err != nil {
		return nil, WrapError(err, "failed to get collection info")
	}
	if resp.Result == nil {
		return nil, ErrCollectionNotFound
	}
	info := &CollectionInfo{
		Name:        name,
		Status:      resp.Result.Status.String(),
		PointsCount: resp.Result.GetPointsCount(),
	}
	if resp.Result.Config != nil && resp.Result.Config.Params != nil {
		if vectorConfig := resp.Result.Config.Params.VectorsConfig; vectorConfig != nil {
			if params := vectorConfig.GetParams(); params != nil {
				info.VectorSize = params.Size
				info.Distance = params.Distance.String()
			}
		}
	}
	return info, nil
}

// ListCollections lists all collections.
func (c *qdrantImpl) ListCollections(ctx context.Context) ([]string, error) {
	resp, err := c.collectionsClient.List(ctx, &pb.ListCollectionsRequest{})
	if err != nil {
		return nil, WrapError(err, "failed to list collections")
	}
	collections := make([]string, 0, len(resp.Collections))
	for _, col := range resp.Collections {
		collections = append(collections, col.Name)
	}
	return collections, nil
}

// UpsertPoint inserts or updates a point in a collection.
func (c *qdrantImpl) UpsertPoint(ctx context.Context, collectionName string, point Point) error {
	if collectionName == "" {
		return ErrEmptyCollection
	}
	if point.ID == "" {
		return ErrInvalidPointID
	}
	if len(point.Vector) == 0 {
		return ErrInvalidVector
	}
	payloadMap, err := pb.TryValueMap(point.Payload)
	if err != nil {
		return WrapError(err, "failed to convert payload")
	}
	// Determine if ID is UUID or string
	var pointId *pb.PointId
	if isValidUUID(point.ID) {
		pointId = &pb.PointId{PointIdOptions: &pb.PointId_Uuid{Uuid: point.ID}}
	} else {
		pointId = &pb.PointId{PointIdOptions: &pb.PointId_Num{Num: generateHashNumber(point.ID)}}
	}

	qdrantPoint := &pb.PointStruct{
		Id:      pointId,
		Vectors: &pb.Vectors{VectorsOptions: &pb.Vectors_Vector{Vector: &pb.Vector{Data: point.Vector}}},
		Payload: payloadMap,
	}
	_, err = c.pointsClient.Upsert(ctx, &pb.UpsertPoints{
		CollectionName: collectionName,
		Points:         []*pb.PointStruct{qdrantPoint},
	})
	if err != nil {
		return WrapError(err, "failed to upsert point")
	}
	return nil
}

// UpsertPoints inserts or updates multiple points in a collection.
func (c *qdrantImpl) UpsertPoints(ctx context.Context, collectionName string, points []Point) error {
	if collectionName == "" {
		return ErrEmptyCollection
	}
	if len(points) == 0 {
		return nil
	}
	qdrantPoints := make([]*pb.PointStruct, 0, len(points))
	for _, point := range points {
		if point.ID == "" {
			return ErrInvalidPointID
		}
		if len(point.Vector) == 0 {
			return ErrInvalidVector
		}
		payloadMap, err := pb.TryValueMap(point.Payload)
		if err != nil {
			return WrapError(err, "failed to convert payload")
		}
		// Determine if ID is UUID or string
		var pointId *pb.PointId
		if isValidUUID(point.ID) {
			pointId = &pb.PointId{PointIdOptions: &pb.PointId_Uuid{Uuid: point.ID}}
		} else {
			pointId = &pb.PointId{PointIdOptions: &pb.PointId_Num{Num: generateHashNumber(point.ID)}}
		}

		qdrantPoints = append(qdrantPoints, &pb.PointStruct{
			Id:      pointId,
			Vectors: &pb.Vectors{VectorsOptions: &pb.Vectors_Vector{Vector: &pb.Vector{Data: point.Vector}}},
			Payload: payloadMap,
		})
	}
	_, err := c.pointsClient.Upsert(ctx, &pb.UpsertPoints{
		CollectionName: collectionName,
		Points:         qdrantPoints,
	})
	if err != nil {
		return WrapError(err, "failed to upsert points")
	}
	return nil
}

// DeletePoint deletes a point from a collection.
func (c *qdrantImpl) DeletePoint(ctx context.Context, collectionName string, pointID string) error {
	if collectionName == "" {
		return ErrEmptyCollection
	}
	if pointID == "" {
		return ErrInvalidPointID
	}
	_, err := c.pointsClient.Delete(ctx, &pb.DeletePoints{
		CollectionName: collectionName,
		Points: &pb.PointsSelector{
			PointsSelectorOneOf: &pb.PointsSelector_Points{
				Points: &pb.PointsIdsList{
					Ids: []*pb.PointId{{PointIdOptions: &pb.PointId_Uuid{Uuid: pointID}}},
				},
			},
		},
	})
	if err != nil {
		return WrapError(err, "failed to delete point")
	}
	return nil
}

// GetPoint retrieves a point by ID.
func (c *qdrantImpl) GetPoint(ctx context.Context, collectionName string, pointID string) (*Point, error) {
	if collectionName == "" {
		return nil, ErrEmptyCollection
	}
	if pointID == "" {
		return nil, ErrInvalidPointID
	}
	resp, err := c.pointsClient.Get(ctx, &pb.GetPoints{
		CollectionName: collectionName,
		Ids:            []*pb.PointId{{PointIdOptions: &pb.PointId_Uuid{Uuid: pointID}}},
		WithPayload:    &pb.WithPayloadSelector{SelectorOptions: &pb.WithPayloadSelector_Enable{Enable: true}},
		WithVectors:    &pb.WithVectorsSelector{SelectorOptions: &pb.WithVectorsSelector_Enable{Enable: true}},
	})
	if err != nil {
		return nil, WrapError(err, "failed to get point")
	}
	if len(resp.Result) == 0 {
		return nil, ErrPointNotFound
	}
	result := resp.Result[0]
	var vector []float32
	if vectors := result.Vectors; vectors != nil {
		if v := vectors.GetVector(); v != nil {
			vector = v.Data
		}
	}
	payload := make(map[string]interface{})
	for key, value := range result.Payload {
		payload[key] = valueToInterface(value)
	}
	return &Point{ID: pointID, Vector: vector, Payload: payload}, nil
}

// Search performs a vector similarity search.
func (c *qdrantImpl) Search(ctx context.Context, collectionName string, vector []float32, limit uint64) ([]SearchResult, error) {
	if collectionName == "" {
		return nil, ErrEmptyCollection
	}
	if len(vector) == 0 {
		return nil, ErrInvalidVector
	}
	if limit == 0 {
		limit = DefaultSearchLimit
	}
	resp, err := c.pointsClient.Search(ctx, &pb.SearchPoints{
		CollectionName: collectionName,
		Vector:         vector,
		Limit:          limit,
		WithPayload:    &pb.WithPayloadSelector{SelectorOptions: &pb.WithPayloadSelector_Enable{Enable: true}},
	})
	if err != nil {
		return nil, WrapError(err, "failed to search")
	}
	return c.searchResultsFromHits(resp.Result), nil
}

// SearchWithFilter performs a vector similarity search with payload filter and optional score threshold.
// If scoreThreshold > 0, Qdrant will only return results with score >= threshold (server-side filtering).
func (c *qdrantImpl) SearchWithFilter(ctx context.Context, collectionName string, vector []float32, limit uint64, filter *pb.Filter, scoreThreshold float32) ([]SearchResult, error) {
	if collectionName == "" {
		return nil, ErrEmptyCollection
	}
	if len(vector) == 0 {
		return nil, ErrInvalidVector
	}
	if limit == 0 {
		limit = DefaultSearchLimit
	}
	req := &pb.SearchPoints{
		CollectionName: collectionName,
		Vector:         vector,
		Limit:          limit,
		Filter:         filter,
		WithPayload:    &pb.WithPayloadSelector{SelectorOptions: &pb.WithPayloadSelector_Enable{Enable: true}},
	}
	if scoreThreshold > 0 {
		req.ScoreThreshold = &scoreThreshold
	}
	resp, err := c.pointsClient.Search(ctx, req)
	if err != nil {
		return nil, WrapError(err, "failed to search with filter")
	}
	return c.searchResultsFromHits(resp.Result), nil
}

// SearchBatch performs multiple vector searches in a single request.
func (c *qdrantImpl) SearchBatch(ctx context.Context, collectionName string, vectors [][]float32, limit uint64) ([][]SearchResult, error) {
	if collectionName == "" {
		return nil, ErrEmptyCollection
	}
	if len(vectors) == 0 {
		return nil, ErrInvalidVector
	}
	if limit == 0 {
		limit = DefaultSearchLimit
	}
	searches := make([]*pb.SearchPoints, 0, len(vectors))
	for _, vector := range vectors {
		if len(vector) == 0 {
			return nil, ErrInvalidVector
		}
		searches = append(searches, &pb.SearchPoints{
			CollectionName: collectionName,
			Vector:         vector,
			Limit:          limit,
			WithPayload:    &pb.WithPayloadSelector{SelectorOptions: &pb.WithPayloadSelector_Enable{Enable: true}},
		})
	}
	resp, err := c.pointsClient.SearchBatch(ctx, &pb.SearchBatchPoints{
		CollectionName: collectionName,
		SearchPoints:   searches,
	})
	if err != nil {
		return nil, WrapError(err, "failed to batch search")
	}
	allResults := make([][]SearchResult, 0, len(resp.Result))
	for _, batchResult := range resp.Result {
		allResults = append(allResults, c.searchResultsFromHits(batchResult.Result))
	}
	return allResults, nil
}

// CountPoints returns the number of points in a collection.
func (c *qdrantImpl) CountPoints(ctx context.Context, collectionName string) (uint64, error) {
	if collectionName == "" {
		return 0, ErrEmptyCollection
	}
	resp, err := c.pointsClient.Count(ctx, &pb.CountPoints{CollectionName: collectionName})
	if err != nil {
		return 0, WrapError(err, "failed to count points")
	}
	if resp.Result == nil {
		return 0, nil
	}
	return resp.Result.Count, nil
}

// ScrollPoints scrolls points with an optional filter (offset is the next-page cursor from a previous response).
func (c *qdrantImpl) ScrollPoints(ctx context.Context, collectionName string, filter *pb.Filter, limit uint32, withPayload bool, offset *pb.PointId) ([]Point, *pb.PointId, error) {
	if collectionName == "" {
		return nil, nil, ErrEmptyCollection
	}
	if limit == 0 {
		limit = 100
	}
	wp := &pb.WithPayloadSelector{SelectorOptions: &pb.WithPayloadSelector_Enable{Enable: withPayload}}
	resp, err := c.pointsClient.Scroll(ctx, &pb.ScrollPoints{
		CollectionName: collectionName,
		Filter:         filter,
		Limit:          &limit,
		WithPayload:    wp,
		Offset:         offset,
	})
	if err != nil {
		return nil, nil, WrapError(err, "failed to scroll points")
	}
	out := make([]Point, 0, len(resp.Result))
	for _, rp := range resp.Result {
		out = append(out, retrievedPointToPoint(rp))
	}
	return out, resp.NextPageOffset, nil
}

func retrievedPointToPoint(rp *pb.RetrievedPoint) Point {
	if rp == nil {
		return Point{}
	}
	id := PointIDString(rp.Id)
	payload := make(map[string]interface{})
	for k, v := range rp.Payload {
		payload[k] = valueToInterface(v)
	}
	var vector []float32
	if rp.Vectors != nil {
		if v := rp.Vectors.GetVector(); v != nil {
			vector = v.Data
		}
	}
	return Point{ID: id, Vector: vector, Payload: payload}
}

// searchResultsFromHits maps Qdrant hit results to SearchResult slice.
func (c *qdrantImpl) searchResultsFromHits(hits []*pb.ScoredPoint) []SearchResult {
	results := make([]SearchResult, 0, len(hits))
	for _, hit := range hits {
		payload := make(map[string]interface{})
		for key, value := range hit.Payload {
			payload[key] = valueToInterface(value)
		}
		id := PointIDString(hit.Id)
		results = append(results, SearchResult{ID: id, Score: hit.Score, Payload: payload})
	}
	return results
}

// isValidUUID checks if the string is a valid UUID using the pre-compiled reUUID regex.
func isValidUUID(uuid string) bool {
	return reUUID.MatchString(uuid)
}

// generateHashNumber generates a numeric hash from string ID
func generateHashNumber(id string) uint64 {
	hash := md5.Sum([]byte(id))
	// Use first 8 bytes to create a uint64
	return uint64(hash[0])<<56 | uint64(hash[1])<<48 | uint64(hash[2])<<40 | uint64(hash[3])<<32 |
		uint64(hash[4])<<24 | uint64(hash[5])<<16 | uint64(hash[6])<<8 | uint64(hash[7])
}

// Close closes the gRPC connection
func (c *qdrantImpl) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
