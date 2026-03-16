package aws

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
)

type EventBridgeClient struct {
	client   *eventbridge.Client
	eventBus string
}

func NewEventBridgeClient(ctx context.Context, region, eventBus string) (*EventBridgeClient, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &EventBridgeClient{
		client:   eventbridge.NewFromConfig(cfg),
		eventBus: eventBus,
	}, nil
}

// BuildEventSource returns the event source string for a given prefix.
func BuildEventSource(prefix string) string {
	if prefix == "" || prefix == "dev" {
		return "int-dev.gameWeekManagement"
	}
	return prefix + ".gameWeekManagement"
}

func (c *EventBridgeClient) CloseGameWeek(ctx context.Context, prefix, gameWeekID string) error {
	source := BuildEventSource(prefix)

	detail, _ := json.Marshal(map[string]interface{}{
		"payload": map[string]string{
			"gameWeekId": gameWeekID,
		},
	})

	_, err := c.client.PutEvents(ctx, &eventbridge.PutEventsInput{
		Entries: []types.PutEventsRequestEntry{
			{
				EventBusName: aws.String(c.eventBus),
				Source:       aws.String(source),
				DetailType:   aws.String("EndOfGameWeekType"),
				Detail:       aws.String(string(detail)),
			},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to send EndOfGameWeek event: %w", err)
	}

	return nil
}
