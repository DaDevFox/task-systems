using System;
using System.Threading.Tasks;
using Grpc.Net.Client;
using Inventory.V1;

namespace ConnectionTest
{
    class Program
    {
        static async Task Main(string[] args)
        {
            try
            {
                Console.WriteLine("Testing gRPC connection to inventory service...");

                // Test the address formatting logic
                string address = "localhost:50052";
                string formattedAddress = address;
                if (!address.StartsWith("http://") && !address.StartsWith("https://"))
                {
                    formattedAddress = $"http://{address}";
                }

                Console.WriteLine($"Connecting to: {formattedAddress}");

                // Create gRPC channel
                using var channel = GrpcChannel.ForAddress(formattedAddress);
                var client = new InventoryService.InventoryServiceClient(channel);

                // Test the connection with a simple ping call
                var response = await client.GetInventoryStatusAsync(new GetInventoryStatusRequest());

                Console.WriteLine("✅ Connection successful!");
                Console.WriteLine($"Received response with {response.Items.Count} items");

                foreach (var item in response.Items)
                {
                    Console.WriteLine($"  - {item.Name}: {item.CurrentLevel}/{item.MaxCapacity} {item.UnitId}");
                }
            }
            catch (Exception ex)
            {
                Console.WriteLine($"❌ Connection failed: {ex.Message}");
                if (ex.InnerException != null)
                {
                    Console.WriteLine($"   Inner exception: {ex.InnerException.Message}");
                }
            }
        }
    }
}
