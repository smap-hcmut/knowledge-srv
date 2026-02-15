package qdrant

import (
	"context"

	pb "github.com/qdrant/go-client/qdrant"
)

// CreateCollection creates a new collection in Qdrant
func (c *Client) CreateCollection(ctx context.Context, name string, vectorSize uint64, distance pb.Distance) error {
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

// DeleteCollection deletes a collection from Qdrant
func (c *Client) DeleteCollection(ctx context.Context, name string) error {
	if name == "" {
		return ErrEmptyCollection
	}

	_, err := c.collectionsClient.Delete(ctx, &pb.DeleteCollection{
		CollectionName: name,
	})

	if err != nil {
		return WrapError(err, "failed to delete collection")
	}

	return nil
}

// CollectionExists checks if a collection exists
func (c *Client) CollectionExists(ctx context.Context, name string) (bool, error) {
	if name == "" {
		return false, ErrEmptyCollection
	}

	resp, err := c.collectionsClient.Get(ctx, &pb.GetCollectionInfoRequest{
		CollectionName: name,
	})

	if err != nil {
		return false, nil
	}

	return resp != nil, nil
}

// GetCollectionInfo retrieves information about a collection
func (c *Client) GetCollectionInfo(ctx context.Context, name string) (*CollectionInfo, error) {
	if name == "" {
		return nil, ErrEmptyCollection
	}

	resp, err := c.collectionsClient.Get(ctx, &pb.GetCollectionInfoRequest{
		CollectionName: name,
	})

	if err != nil {
		return nil, WrapError(err, "failed to get collection info")
	}

	if resp.Result == nil {
		return nil, ErrCollectionNotFound
	}

	info := &CollectionInfo{
		Name:        name,
		Status:      resp.Result.Status.String(),
		PointsCount: resp.Result.PointsCount,
	}

	// Extract vector config if available
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

// ListCollections lists all collections
func (c *Client) ListCollections(ctx context.Context) ([]string, error) {
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
