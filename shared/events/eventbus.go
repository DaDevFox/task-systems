// PublishDynamic dynamically publishes an event with flexible payload handling
func (eb *EventBus) PublishDynamic(ctx context.Context, eventType string, payload map[string]interface{}) error {
    dynamicPayload, err := json.Marshal(payload)
    if err != nil {
        return fmt.Errorf("failed to encode dynamic payload: %w", err)
    }

    event := &pb.Event{
        Id:            uuid.New().String(),
        Type:          pb.EventType(pb.EventType_value[eventType]),
        SourceService: eb.serviceName,
        Timestamp:     timestamppb.Now(),
        Payload: &anypb.Any{
            Value: dynamicPayload,
        },
    }

    return eb.Publish(ctx, pb.EventType(pb.EventType_value[eventType]), event)
}