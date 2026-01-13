package service

const (
	testServiceName     = "test-service"
	testItemID          = "test-item-1"
	testItemName        = "Test Item"
	testItemDescription = "Original description"
	expectedNoError     = "Expected no error, got %v"
	expectedResponse    = "Expected response, got nil"
	expectedItemChanged = "Expected item to be changed"
	expectedNilResponse = "Expected nil response on error"
	expectedGRPCError   = "Expected gRPC status error"
	expectedInvalidArg  = "Expected InvalidArgument code, got %v"
	updatedName         = "Updated Name"
	databaseError       = "database error"
)
