package functional_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	inventorypb "github.com/DaDevFox/task-systems/inventory-core/backend/pkg/proto/inventory/v1"
	inventorysupport "github.com/DaDevFox/task-systems/inventory-core/backend/testsupport"
	eventspb "github.com/DaDevFox/task-systems/shared/pkg/proto/events/v1"
	usercoresupport "github.com/DaDevFox/task-systems/user-core/backend/testsupport"
)

func TestInventorySystemEndToEnd(t *testing.T) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	userServer := usercoresupport.StartUserCoreTestServer(t, ctx)

	inventoryServer, err := inventorysupport.StartInventoryCoreTestServer(t, ctx, userServer.Address)
	if err != nil {
		t.Fatalf("failed to start inventory server: %v", err)
	}

	eventsCh := make(chan *eventspb.Event, 1)
	handler := func(handlerCtx context.Context, event *eventspb.Event) error {
		select {
		case eventsCh <- event:
		default:
		}
		return nil
	}

	inventoryServer.EventBus.Subscribe(eventspb.EventType_INVENTORY_LEVEL_CHANGED, handler)

	authContext := metadata.NewOutgoingContext(ctx, metadata.Pairs("authorization", fmt.Sprintf("Bearer %s", userServer.AccessToken)))

	unitResp, err := inventoryServer.Client.AddUnit(authContext, &inventorypb.AddUnitRequest{
		Name:                 "Integration Kilogram",
		Symbol:               "ikg",
		Description:          "Integration test unit",
		BaseConversionFactor: 1,
		Category:             "test",
	})
	switch {
	case err != nil:
		t.Fatalf("add unit failed: %v", err)
	case unitResp == nil || unitResp.Unit == nil:
		t.Fatal("add unit returned nil response")
	}

	itemResp, err := inventoryServer.Client.AddInventoryItem(authContext, &inventorypb.AddInventoryItemRequest{
		Name:              "Integration Item",
		Description:       "Item created via end-to-end test",
		InitialLevel:      100,
		MaxCapacity:       200,
		LowStockThreshold: 20,
		UnitId:            unitResp.Unit.Id,
	})
	switch {
	case err != nil:
		t.Fatalf("add inventory item failed: %v", err)
	case itemResp == nil || itemResp.Item == nil:
		t.Fatal("add inventory item returned nil response")
	}

	updateResp, err := inventoryServer.Client.UpdateInventoryLevel(authContext, &inventorypb.UpdateInventoryLevelRequest{
		ItemId:   itemResp.Item.Id,
		NewLevel: 80,
		Reason:   "integration adjustment",
	})
	switch {
	case err != nil:
		t.Fatalf("update inventory level failed: %v", err)
	case updateResp == nil || updateResp.Item == nil:
		t.Fatal("update inventory level returned nil response")
	case updateResp.Item.CurrentLevel != 80:
		t.Fatalf("expected current level 80, got %.2f", updateResp.Item.CurrentLevel)
	}

	select {
	case event := <-eventsCh:
		if event == nil {
			t.Fatal("received nil event")
		}

		if event.Type != eventspb.EventType_INVENTORY_LEVEL_CHANGED {
			t.Fatalf("expected INVENTORY_LEVEL_CHANGED event, got %s", event.Type.String())
		}
	case <-time.After(2 * time.Second):
		t.Fatal("expected inventory level change event")
	}

	_, err = inventoryServer.Client.GetInventoryItem(context.Background(), &inventorypb.GetInventoryItemRequest{ItemId: itemResp.Item.Id})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected unauthenticated error without token, got %v", err)
	}
}

func TestInventoryServerFailsWithInvalidUserCoreAddress(t *testing.T) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	t.Cleanup(cancel)

	handle, err := inventorysupport.StartInventoryCoreTestServer(t, ctx, "127.0.0.1:0")
	if err == nil {
		handle.Shutdown()
		t.Fatal("expected error when user-core address is unreachable")
	}
}
