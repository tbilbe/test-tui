package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type DynamoDBClient struct {
	client    *dynamodb.Client
	tableName string
}

func NewDynamoDBClient(ctx context.Context, region, tableName string) (*DynamoDBClient, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &DynamoDBClient{
		client:    dynamodb.NewFromConfig(cfg),
		tableName: tableName,
	}, nil
}

func (d *DynamoDBClient) QueryFixturesByGameWeek(ctx context.Context, gameWeekID string) ([]map[string]types.AttributeValue, error) {
	input := &dynamodb.QueryInput{
		TableName:              aws.String(d.tableName),
		KeyConditionExpression: aws.String("gameWeekId = :gw"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":gw": &types.AttributeValueMemberS{Value: gameWeekID},
		},
	}

	result, err := d.client.Query(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to query fixtures: %w", err)
	}

	return result.Items, nil
}

func (d *DynamoDBClient) PutFixture(ctx context.Context, fixture map[string]types.AttributeValue) error {
	input := &dynamodb.PutItemInput{
		TableName: aws.String(d.tableName),
		Item:      fixture,
	}

	_, err := d.client.PutItem(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to put fixture: %w", err)
	}

	return nil
}

func (d *DynamoDBClient) UpdateFixture(ctx context.Context, item interface{}) error {
	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		return fmt.Errorf("failed to marshal fixture: %w", err)
	}

	return d.PutFixture(ctx, av)
}

func (d *DynamoDBClient) UpdateGameWeek(ctx context.Context, item interface{}) error {
	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		return fmt.Errorf("failed to marshal gameweek: %w", err)
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(d.tableName),
		Item:      av,
	}

	_, err = d.client.PutItem(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to put gameweek: %w", err)
	}

	return nil
}

func (d *DynamoDBClient) QueryPlayersByGameWeek(ctx context.Context, gameWeekID string) ([]map[string]types.AttributeValue, error) {
	input := &dynamodb.QueryInput{
		TableName:              aws.String(d.tableName),
		KeyConditionExpression: aws.String("gameWeekId = :gw"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":gw": &types.AttributeValueMemberS{Value: gameWeekID},
		},
	}

	result, err := d.client.Query(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to query players: %w", err)
	}

	return result.Items, nil
}
