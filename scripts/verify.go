package main

import (
	"context"
	"fmt"
	"log"

	pb "github.com/qdrant/go-client/qdrant"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	conn, err := grpc.Dial("172.16.19.20:6334", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to Qdrant: %v", err)
	}
	defer conn.Close()

	client := pb.NewCollectionsClient(conn)
	pointsClient := pb.NewPointsClient(conn)
	ctx := context.Background()

	projectCollection := "proj_44444444-4444-4444-4444-444444444444"
	macroCollection := "macro_insights"

	fmt.Println("=== Checking Collections ===")
	
	// Check project collection
	if _, err := client.Get(ctx, &pb.GetCollectionInfoRequest{CollectionName: projectCollection}); err != nil {
		fmt.Printf("❌ Project collection NOT found: %v\n", err)
	} else {
		fmt.Printf("✅ Project collection exists: %s\n", projectCollection)
	}

	// Check macro insights collection
	if _, err := client.Get(ctx, &pb.GetCollectionInfoRequest{CollectionName: macroCollection}); err != nil {
		fmt.Printf("❌ Macro insights collection NOT found: %v\n", err)
	} else {
		fmt.Printf("✅ Macro insights collection exists: %s\n", macroCollection)
	}

	fmt.Println("\n=== Counting Points ===")
	
	// Count project collection points
	if resp, err := pointsClient.Count(ctx, &pb.CountPoints{CollectionName: projectCollection}); err != nil {
		fmt.Printf("❌ Error counting project points: %v\n", err)
	} else {
		fmt.Printf("Points in %s: %d (expected: 2)\n", projectCollection, resp.Result.Count)
	}

	// Count macro insights collection points
	if resp, err := pointsClient.Count(ctx, &pb.CountPoints{CollectionName: macroCollection}); err != nil {
		fmt.Printf("❌ Error counting macro points: %v\n", err)
	} else {
		fmt.Printf("Points in %s: %d (expected: 8)\n", macroCollection, resp.Result.Count)
	}

	fmt.Println("\n=== Verifying rag_document_type ===")
	
	// Filter for insight_card
	insightFilter := &pb.Filter{
		Must: []*pb.Condition{
			{
				ConditionOneOf: &pb.Condition_Field{
					Field: &pb.FieldCondition{
						Key: "rag_document_type",
						Match: &pb.Match{
							MatchValue: &pb.Match_Keyword{Keyword: "insight_card"},
						},
					},
				},
			},
		},
	}

	if resp, err := pointsClient.Count(ctx, &pb.CountPoints{CollectionName: macroCollection, Filter: insightFilter}); err != nil {
		fmt.Printf("❌ Error counting insight_card: %v\n", err)
	} else {
		fmt.Printf("rag_document_type=insight_card: %d (expected: 7)\n", resp.Result.Count)
	}

	// Filter for report_digest
	digestFilter := &pb.Filter{
		Must: []*pb.Condition{
			{
				ConditionOneOf: &pb.Condition_Field{
					Field: &pb.FieldCondition{
						Key: "rag_document_type",
						Match: &pb.Match{
							MatchValue: &pb.Match_Keyword{Keyword: "report_digest"},
						},
					},
				},
			},
		},
	}

	if resp, err := pointsClient.Count(ctx, &pb.CountPoints{CollectionName: macroCollection, Filter: digestFilter}); err != nil {
		fmt.Printf("❌ Error counting report_digest: %v\n", err)
	} else {
		fmt.Printf("rag_document_type=report_digest: %d (expected: 1)\n", resp.Result.Count)
	}

	fmt.Println("\n✅ Verification completed!")
}
