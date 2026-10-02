package aws

import (
	"context"
	"errors"
	"fmt"

	"github.com/angstromsports/seven-test-tui/internal/models"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type DynamoDBClient struct {
	client    *dynamodb.Client
	prefix    string
	tableName string
}

func NewDynamoDBClient(ctx context.Context, region, prefix, tableName string) (*DynamoDBClient, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &DynamoDBClient{
		client:    dynamodb.NewFromConfig(cfg),
		prefix:    prefix,
		tableName: tableName,
	}, nil
}

// AssertWritable prevents commands from writing outside an explicitly enabled prefix.
func AssertWritable(prefix string) error {
	if !models.IsWriteAllowed(prefix) {
		return fmt.Errorf("writes are not enabled for prefix %q", prefix)
	}
	return nil
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

func (d *DynamoDBClient) UpdateFixture(ctx context.Context, fixture models.Fixture) error {
	if err := AssertWritable(d.prefix); err != nil {
		return err
	}

	expr, err := buildFixtureUpdateExpression(fixture)
	if err != nil {
		return err
	}

	_, err = d.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:                 aws.String(d.tableName),
		Key:                       fixtureKey(fixture),
		UpdateExpression:          expr.Update(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})
	if err != nil {
		return fmt.Errorf("failed to update fixture: %w", err)
	}

	return nil
}

func (d *DynamoDBClient) UpdateGameWeek(ctx context.Context, gameWeek models.GameWeek) error {
	if err := AssertWritable(d.prefix); err != nil {
		return err
	}

	expr, err := buildGameWeekUpdateExpression(gameWeek)
	if err != nil {
		return err
	}

	_, err = d.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(d.tableName),
		Key: map[string]types.AttributeValue{
			"gameWeekId": &types.AttributeValueMemberS{Value: gameWeek.GameWeekID},
		},
		UpdateExpression:          expr.Update(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})
	if err != nil {
		return fmt.Errorf("failed to update gameweek: %w", err)
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

func fixtureKey(fixture models.Fixture) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{
		"gameWeekId": &types.AttributeValueMemberS{Value: fixture.GameWeekID},
		"fixtureId":  &types.AttributeValueMemberS{Value: fixture.FixtureID},
	}
}

type updateExpression struct {
	builder expression.UpdateBuilder
	count   int
}

func (u *updateExpression) add(name string, value any) {
	if u.count == 0 {
		u.builder = expression.Set(expression.Name(name), expression.Value(value))
	} else {
		u.builder = u.builder.Set(expression.Name(name), expression.Value(value))
	}
	u.count++
}

func (u *updateExpression) remove(name string) {
	if u.count == 0 {
		u.builder = expression.Remove(expression.Name(name))
	} else {
		u.builder = u.builder.Remove(expression.Name(name))
	}
	u.count++
}

func (u *updateExpression) build(emptyErr error) (expression.Expression, error) {
	if u.count == 0 {
		return expression.Expression{}, emptyErr
	}
	expr, err := expression.NewBuilder().WithUpdate(u.builder).Build()
	if err != nil {
		return expression.Expression{}, fmt.Errorf("failed to build update expression: %w", err)
	}
	return expr, nil
}

func buildFixtureUpdateExpression(fixture models.Fixture) (expression.Expression, error) {
	var update updateExpression

	if fixture.StartDate != "" {
		update.add("startDate", fixture.StartDate)
	}
	if fixture.HomeScore != nil {
		update.add("homeScore", fixture.HomeScore)
	} else if fixture.RemoveAttributes["homeScore"] {
		update.remove("homeScore")
	}
	if fixture.AwayScore != nil {
		update.add("awayScore", fixture.AwayScore)
	} else if fixture.RemoveAttributes["awayScore"] {
		update.remove("awayScore")
	}
	if fixture.Period != "" {
		update.add("period", fixture.Period)
	}
	if fixture.Goals != nil {
		update.add("goals", fixture.Goals)
	} else if fixture.RemoveAttributes["goals"] {
		update.remove("goals")
	}
	// Zero is the intentional PRE_MATCH reset value and must reach DynamoDB.
	update.add("clockTimeMin", fixture.ClockTimeMin)
	if fixture.FixtureStatus != "" {
		update.add("fixtureStatus", fixture.FixtureStatus)
	}
	if fixture.Metadata != nil {
		update.add("metadata", fixture.Metadata)
	}
	if fixture.HomeTeamID != "" {
		update.add("homeTeamId", fixture.HomeTeamID)
	}
	if fixture.AwayTeamID != "" {
		update.add("awayTeamId", fixture.AwayTeamID)
	}

	return update.build(errors.New("fixture has no populated fields to update"))
}

func buildGameWeekUpdateExpression(gameWeek models.GameWeek) (expression.Expression, error) {
	var set updateExpression

	if gameWeek.CompetitionCalendarID != "" {
		set.add("competitionCalendarId", gameWeek.CompetitionCalendarID)
	}
	if gameWeek.Label != "" {
		set.add("label", gameWeek.Label)
	}
	if gameWeek.CustomerStartDate != "" {
		set.add("customerStartDate", gameWeek.CustomerStartDate)
	}
	if gameWeek.CustomerEndDate != "" {
		set.add("customerEndDate", gameWeek.CustomerEndDate)
	}
	if gameWeek.FixturesStartDate != "" {
		set.add("fixturesStartDate", gameWeek.FixturesStartDate)
	}
	if gameWeek.FixturesEndDate != "" {
		set.add("fixturesEndDate", gameWeek.FixturesEndDate)
	}
	if gameWeek.Locked != nil {
		set.add("locked", gameWeek.Locked)
	}

	return set.build(errors.New("gameweek has no populated fields to update"))
}
