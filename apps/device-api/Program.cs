using System.Text.Json;
using Dapper;
using Npgsql;

DefaultTypeMap.MatchNamesWithUnderscores = true;

var builder = WebApplication.CreateBuilder(args);
builder.Services.ConfigureHttpJsonOptions(opts =>
    opts.SerializerOptions.PropertyNamingPolicy = JsonNamingPolicy.SnakeCaseLower);

var app = builder.Build();

var connString = ParseConnectionString(
    Environment.GetEnvironmentVariable("DATABASE_URL")
    ?? "Host=localhost;Port=5432;Username=postgres;Password=postgres;Database=smarthome");

NpgsqlConnection Conn() => new(connString);

// GET /api/v1/sensors
app.MapGet("/api/v1/sensors", async () =>
{
    await using var conn = Conn();
    var sensors = await conn.QueryAsync<Sensor>(
        "SELECT id, name, type, location, value, unit, status, last_updated, created_at FROM sensors ORDER BY id");
    return Results.Ok(sensors);
});

// GET /api/v1/sensors/{id}
app.MapGet("/api/v1/sensors/{id:int}", async (int id) =>
{
    await using var conn = Conn();
    var sensor = await conn.QueryFirstOrDefaultAsync<Sensor>(
        "SELECT id, name, type, location, value, unit, status, last_updated, created_at FROM sensors WHERE id = @id",
        new { id });
    return sensor is null
        ? Results.NotFound(new { error = "Sensor not found" })
        : Results.Ok(sensor);
});

// POST /api/v1/sensors
app.MapPost("/api/v1/sensors", async (SensorCreate body) =>
{
    await using var conn = Conn();
    var sensor = await conn.QueryFirstAsync<Sensor>(@"
        INSERT INTO sensors (name, type, location, unit, status, last_updated, created_at)
        VALUES (@Name, @Type, @Location, @Unit, 'inactive', @Now, @Now)
        RETURNING id, name, type, location, value, unit, status, last_updated, created_at",
        new { body.Name, body.Type, body.Location, Unit = body.Unit ?? "", Now = DateTime.UtcNow });
    return Results.Created($"/api/v1/sensors/{sensor.Id}", sensor);
});

// PUT /api/v1/sensors/{id}
app.MapPut("/api/v1/sensors/{id:int}", async (int id, SensorUpdate body) =>
{
    await using var conn = Conn();

    var sets = new List<string> { "last_updated = @Now" };
    var p = new DynamicParameters();
    p.Add("Id", id);
    p.Add("Now", DateTime.UtcNow);

    if (!string.IsNullOrEmpty(body.Name))     { sets.Add("name = @Name");         p.Add("Name", body.Name); }
    if (!string.IsNullOrEmpty(body.Type))     { sets.Add("type = @Type");         p.Add("Type", body.Type); }
    if (!string.IsNullOrEmpty(body.Location)) { sets.Add("location = @Location"); p.Add("Location", body.Location); }
    if (body.Value.HasValue)                  { sets.Add("value = @Value");        p.Add("Value", body.Value); }
    if (!string.IsNullOrEmpty(body.Unit))     { sets.Add("unit = @Unit");         p.Add("Unit", body.Unit); }
    if (!string.IsNullOrEmpty(body.Status))   { sets.Add("status = @Status");     p.Add("Status", body.Status); }

    var sql = $"UPDATE sensors SET {string.Join(", ", sets)} WHERE id = @Id " +
              "RETURNING id, name, type, location, value, unit, status, last_updated, created_at";

    var sensor = await conn.QueryFirstOrDefaultAsync<Sensor>(sql, p);
    return sensor is null
        ? Results.NotFound(new { error = "Sensor not found" })
        : Results.Ok(sensor);
});

// DELETE /api/v1/sensors/{id}
app.MapDelete("/api/v1/sensors/{id:int}", async (int id) =>
{
    await using var conn = Conn();
    var affected = await conn.ExecuteAsync("DELETE FROM sensors WHERE id = @id", new { id });
    return affected == 0
        ? Results.NotFound(new { error = "Sensor not found" })
        : Results.Ok(new { message = "Sensor deleted successfully" });
});

// PATCH /api/v1/sensors/{id}/value
app.MapMethods("/api/v1/sensors/{id:int}/value", ["PATCH"], async (int id, ValueUpdate body) =>
{
    await using var conn = Conn();
    var affected = await conn.ExecuteAsync(
        "UPDATE sensors SET value = @Value, status = @Status, last_updated = @Now WHERE id = @Id",
        new { body.Value, body.Status, Now = DateTime.UtcNow, Id = id });
    return affected == 0
        ? Results.NotFound(new { error = "Sensor not found" })
        : Results.Ok(new { message = "Sensor value updated successfully" });
});

app.Run("http://0.0.0.0:8080");

// ---------- helpers ----------

static string ParseConnectionString(string raw)
{
    if (!raw.StartsWith("postgres://") && !raw.StartsWith("postgresql://"))
        return raw;
    var uri = new Uri(raw);
    var parts = uri.UserInfo.Split(':');
    return $"Host={uri.Host};Port={uri.Port};Username={parts[0]};Password={parts[1]};Database={uri.AbsolutePath.TrimStart('/')}";
}

// ---------- models ----------

public class Sensor
{
    public int Id { get; set; }
    public string Name { get; set; } = "";
    public string Type { get; set; } = "";
    public string Location { get; set; } = "";
    public double Value { get; set; }
    public string Unit { get; set; } = "";
    public string Status { get; set; } = "";
    public DateTime LastUpdated { get; set; }
    public DateTime CreatedAt { get; set; }
}

public class SensorCreate
{
    public string Name { get; set; } = "";
    public string Type { get; set; } = "";
    public string Location { get; set; } = "";
    public string? Unit { get; set; }
}

public class SensorUpdate
{
    public string? Name { get; set; }
    public string? Type { get; set; }
    public string? Location { get; set; }
    public double? Value { get; set; }
    public string? Unit { get; set; }
    public string? Status { get; set; }
}

public class ValueUpdate
{
    public double Value { get; set; }
    public string Status { get; set; } = "";
}
