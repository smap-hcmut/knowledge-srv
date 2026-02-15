package qdrant

import (
	"context"

	pb "github.com/qdrant/go-client/qdrant"
	"google.golang.org/protobuf/types/known/structpb"
)

// UpsertPoint inserts or updates a point in a collection
func (c *Client) UpsertPoint(ctx context.Context, collectionName string, point Point) error {
	if collectionName == "" {
		return ErrEmptyCollection
	}
	if point.ID == "" {
		return ErrInvalidPointID
	}
	if len(point.Vector) == 0 {
		return ErrInvalidVector
	}

	// Convert payload to structpb
	payload, err := structpb.NewStruct(point.Payload)
	if err != nil {
		return WrapError(err, "failed to convert payload")
	}

	// Create point struct
	qdrantPoint := &pb.PointStruct{
		Id: &pb.PointId{
			PointIdOptions: &pb.PointId_Uuid{
				Uuid: point.ID,
			},
		},
		Vectors: &pb.Vectors{
			VectorsOptions: &pb.Vectors_Vector{
				Vector: &pb.Vector{
					Data: point.Vector,
				},
			},
		},
		Payload: payload.Fields,
	}

	// Upsert point
	_, err = c.pointsClient.Upsert(ctx, &pb.UpsertPoints{
		CollectionName: collectionName,
		Points:         []*pb.PointStruct{qdrantPoint},
	})

	if err != nil {
		return WrapError(err, "failed to upsert point")
	}

	return nil
}

// UpsertPoints inserts or updates multiple points in a collection
func (c *Client) UpsertPoints(ctx context.Context, collectionName string, points []Point) error {
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

		payload, err := structpb.NewStruct(point.Payload)
		if err != nil {
			return WrapError(err, "failed to convert payload")
		}

		qdrantPoint := &pb.PointStruct{
			Id: &pb.PointId{
				PointIdOptions: &pb.PointId_Uuid{
					Uuid: point.ID,
				},
			},
			Vectors: &pb.Vectors{
				VectorsOptions: &pb.Vectors_Vector{
					Vector: &pb.Vector{
						Data: point.Vector,
					},
				},
			},
			Payload: payload.Fields,
		}

		qdrantPoints = append(qdrantPoints, qdrantPoint)
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

// DeletePoint deletes a point from a collection
func (c *Client) DeletePoint(ctx context.Context, collectionName string, pointID string) error {
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
					Ids: []*pb.PointId{
						{
							PointIdOptions: &pb.PointId_Uuid{
								Uuid: pointID,
							},
						},
					},
				},
			},
		},
	})

	if err != nil {
		return WrapError(err, "failed to delete point")
	}

	return nil
}

// GetPoint retrieves a point by ID
func (c *Client) GetPoint(ctx context.Context, collectionName string, pointID string) (*Point, error) {
	if collectionName == "" {
		return nil, ErrEmptyCollection
	}
	if pointID == "" {
		return nil, ErrInvalidPointID
	}

	resp, err := c.pointsClient.Get(ctx, &pb.GetPoints{
		CollectionName: collectionName,
		Ids: []*pb.PointId{
			{
				PointIdOptions: &pb.PointId_Uuid{
					Uuid: pointID,
				},
			},
		},
		WithPayload: &pb.WithPayloadSelector{
			SelectorOptions: &pb.WithPayloadSelector_Enable{
				Enable: true,
			},
		},
		WithVectors: &pb.WithVectorsSelector{
			SelectorOptions: &pb.WithVectorsSelector_Enable{
				Enable: true,
			},
		},
	})

	if err != nil {
		return nil, WrapError(err, "failed to get point")
	}

	if len(resp.Result) == 0 {
		return nil, ErrPointNotFound
	}

	result := resp.Result[0]

	// Extract vector
	var vector []float32
	if vectors := result.Vectors; vectors != nil {
		if v := vectors.GetVector(); v != nil {
			vector = v.Data
		}
	}

	// Extract payload
	payload := make(map[string]interface{})
	for key, value := range result.Payload {
		payload[key] = value.AsInterface()
	}

	return &Point{
		ID:      pointID,
		Vector:  vector,
		Payload: payload,
	}, nil
}
