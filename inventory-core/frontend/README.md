# Inventory Management Frontend

This is an Avalonia/.NET frontend for the Task Systems inventory management service. The application provides a modern, cross-platform GUI for managing inventory levels and reporting data to the predictive algorithm system.

## Features

- **Real-time Inventory Display**: View current inventory levels with visual indicators
- **Stock Status Monitoring**: Automatic low stock and empty item alerts
- **Predictive Analytics**: Display predicted consumption times and confidence scores
- **gRPC Integration**: Communicates with the backend via protobuf/gRPC
- **Reusable Components**: Shared UI components for use across multiple Task Systems services

## Architecture

### Project Structure

```
frontend/
├── src/
│   ├── InventoryClient/           # Main application
│   │   ├── Models/                # Data models and ViewModels
│   │   ├── Services/              # gRPC service clients
│   │   ├── ViewModels/            # MVVM ViewModels
│   │   └── Views/                 # UI Views and Controls
│   └── TaskSystems.Shared/        # Reusable components
│       ├── Controls/              # Custom UI controls
│       ├── Converters/            # Value converters
│       ├── Services/              # Base service interfaces
│       └── ViewModels/            # Base ViewModels
└── InventoryClient.sln
```

### Key Components

1. **InventoryItemViewModel**: Represents individual inventory items with UI-specific properties
2. **MainViewModel**: Main application logic and data management
3. **InventoryGrpcService**: Service client for communicating with the gRPC backend
4. **StatusIndicator**: Reusable control for showing item status across services
5. **ServiceClientBase**: Base class for all gRPC service clients

## Prerequisites

- .NET 8.0 SDK
- Windows, macOS, or Linux (Avalonia is cross-platform)
- gRPC backend service running (for full functionality)

## Setup

1. **Clone and Build**:
   ```bash
   cd frontend
   dotnet restore
   dotnet build
   ```

2. **Run the Application**:
   ```bash
   cd src/InventoryClient
   dotnet run
   ```

3. **Connect to Backend**:
   - Enter server address (e.g., `localhost:5000`)
   - Click "Connect" to establish gRPC connection
   - Use "Refresh" to reload inventory data

## gRPC Integration

The application is designed to work with the protobuf definitions in `../../proto/inventory.proto`. 

### Current Implementation

Currently using a mock service for development. To enable full gRPC functionality:

1. **Generate C# Protobuf Classes**:
   ```bash
   # The project is configured to auto-generate from inventory.proto
   dotnet build  # This will generate the gRPC client classes
   ```

2. **Update Service Implementation**:
   - Replace `InventoryGrpcService` mock methods with real gRPC calls
   - Use the generated client classes from the protobuf compilation

### Supported Operations

The frontend supports these inventory operations:

- `GetInventoryStatus`: Overview of all inventory items
- `ListInventoryItems`: Paginated item listing with filtering
- `UpdateInventoryLevel`: Report new inventory levels
- `PredictConsumption`: Get consumption predictions
- `AddInventoryItem`: Add new items to inventory

## UI Features

### Main Dashboard

- **Connection Status**: Visual indicator of gRPC connection state
- **Inventory Summary**: Total items, low stock count, empty items
- **Real-time Updates**: Automatic refresh when connected

### Item Management

- **DataGrid View**: Sortable, filterable inventory list
- **Status Indicators**: Color-coded status (Normal/Low/Empty)
- **Progress Bars**: Visual capacity representation
- **Prediction Data**: Days remaining and confidence scores

### Filtering and Search

- **Low Stock Filter**: Show only items needing attention
- **Text Search**: Search by name or description
- **Real-time Filtering**: Immediate results as you type

## Extensibility

The application is designed for reuse across multiple Task Systems services:

### Shared Components

- **StatusIndicator**: Use for any service status display
- **ServiceClientBase**: Extend for new gRPC services
- **ServiceViewModelBase**: Base class for service-specific ViewModels

### Adding New Services

1. Create new service client extending `ServiceClientBase`
2. Create service-specific ViewModels extending `ServiceViewModelBase`
3. Reuse shared UI components and converters
4. Add to dependency injection container

## Development

### Mock Data

The application includes mock data for development and testing:

- 4 sample inventory items with different stock levels
- Simulated gRPC connection with delays
- Realistic data for UI testing

### MVVM Pattern

Follows standard MVVM pattern with:

- **Models**: Plain data objects
- **ViewModels**: Business logic and UI state
- **Views**: XAML UI definitions
- **Commands**: User actions (Connect, Refresh, Update)

### Data Binding

Uses two-way data binding for:

- Server connection settings
- Search and filter controls
- Item selection and updates
- Real-time status updates

## Deployment

The application can be deployed as:

1. **Self-contained**: Includes .NET runtime
   ```bash
   dotnet publish -c Release --self-contained
   ```

2. **Framework-dependent**: Requires .NET runtime on target
   ```bash
   dotnet publish -c Release
   ```

3. **Single File**: All dependencies in one executable
   ```bash
   dotnet publish -c Release --self-contained -p:PublishSingleFile=true
   ```

## Contributing

When extending this frontend:

1. Follow the established MVVM pattern
2. Use the shared components where possible
3. Implement proper error handling and logging
4. Update mock data for testing new features
5. Maintain cross-platform compatibility

## Future Enhancements

Planned improvements:

- [ ] Real-time updates via gRPC streaming
- [ ] Offline mode with local caching
- [ ] Advanced filtering and sorting options
- [ ] Batch operations for multiple items
- [ ] Export functionality (CSV, Excel)
- [ ] Custom alerting and notifications
- [ ] Multi-language support
- [ ] Theme customization
