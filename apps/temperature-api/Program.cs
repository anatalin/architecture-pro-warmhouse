var builder = WebApplication.CreateBuilder(args);
var app = builder.Build();

var rng = new Random();

string LocationFromSensorId(string id) => id switch
{
    "1" => "Living Room",
    "2" => "Bedroom",
    "3" => "Kitchen",
    _   => "Unknown"
};

string SensorIdFromLocation(string location) => location switch
{
    "Living Room" => "1",
    "Bedroom"     => "2",
    "Kitchen"     => "3",
    _             => "0"
};

object BuildResponse(string location, string sensorId)
{
    var value = Math.Round(15.0 + rng.NextDouble() * 15.0, 1);
    return new
    {
        value,
        unit        = "C",
        timestamp   = DateTime.UtcNow,
        location,
        status      = "active",
        sensor_id   = sensorId,
        sensor_type = "temperature",
        description = $"Temperature sensor in {location}"
    };
}

// GET /temperature?location=<location>
app.MapGet("/temperature", (string? location, string? sensor_id) =>
{
    var loc = location ?? "";
    var sid = sensor_id ?? "";

    if (string.IsNullOrEmpty(loc))
        loc = LocationFromSensorId(sid);

    if (string.IsNullOrEmpty(sid))
        sid = SensorIdFromLocation(loc);

    return Results.Ok(BuildResponse(loc, sid));
});

// GET /temperature/{sensorId}
app.MapGet("/temperature/{sensorId}", (string sensorId) =>
{
    var location = LocationFromSensorId(sensorId);
    return Results.Ok(BuildResponse(location, sensorId));
});

app.Run("http://0.0.0.0:8081");
