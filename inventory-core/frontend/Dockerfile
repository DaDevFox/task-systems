# SDK image for building the application
FROM mcr.microsoft.com/dotnet/sdk:8.0-alpine AS build

WORKDIR /src

# Copy solution and project files
COPY ["frontend/InventoryClient.sln", "frontend/"]
COPY ["frontend/src/InventoryClient/InventoryClient.csproj", "frontend/src/InventoryClient/"]
COPY ["frontend/tests/InventoryClient.Tests/InventoryClient.Tests.csproj", "frontend/tests/InventoryClient.Tests/"]
COPY ["frontend/ConnectionTest/ConnectionTest.csproj", "frontend/ConnectionTest/"]

# Restore dependencies
RUN dotnet restore "frontend/InventoryClient.sln"

# Copy the rest of the source code
COPY . .

# Build the application
WORKDIR "/src/frontend/src/InventoryClient"
RUN dotnet build "InventoryClient.csproj" -c Release -o /app/build

# Publish the application
FROM build AS publish
RUN dotnet publish "InventoryClient.csproj" -c Release -o /app/publish /p:UseAppHost=false

# Final runtime image
FROM mcr.microsoft.com/dotnet/runtime:8.0-alpine AS final

WORKDIR /app
COPY --from=publish /app/publish .

# Create a non-root user and set permissions
RUN addgroup -S appgroup && adduser -S appuser -G appgroup && \
    chown -R appuser:appgroup /app
USER appuser

ENTRYPOINT ["dotnet", "InventoryClient.dll"]
