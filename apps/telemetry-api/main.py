import asyncio
import json
import os
from contextlib import asynccontextmanager
from datetime import datetime
from typing import List

import aio_pika
import asyncpg
from fastapi import FastAPI, Query
from pydantic import BaseModel

RABBITMQ_URL = os.getenv("RABBITMQ_URL", "amqp://guest:guest@localhost/")
DATABASE_URL = os.getenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/smarthome")
QUEUE_NAME = "sensor_telemetry"

db_pool: asyncpg.Pool | None = None
mq_connection: aio_pika.RobustConnection | None = None


@asynccontextmanager
async def lifespan(app: FastAPI):
    global db_pool, mq_connection

    db_pool = await asyncpg.create_pool(DATABASE_URL)
    async with db_pool.acquire() as conn:
        await conn.execute("""
            CREATE TABLE IF NOT EXISTS telemetry (
                id          SERIAL PRIMARY KEY,
                sensor_id   INTEGER NOT NULL,
                value       FLOAT   NOT NULL,
                status      VARCHAR(20) NOT NULL,
                recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
            );
            CREATE INDEX IF NOT EXISTS idx_telemetry_sensor_id   ON telemetry(sensor_id);
            CREATE INDEX IF NOT EXISTS idx_telemetry_recorded_at ON telemetry(recorded_at);
        """)

    mq_connection = await aio_pika.connect_robust(RABBITMQ_URL)
    consume_task = asyncio.create_task(consume_messages(mq_connection))

    yield

    consume_task.cancel()
    await mq_connection.close()
    await db_pool.close()


app = FastAPI(lifespan=lifespan)


async def consume_messages(connection: aio_pika.RobustConnection) -> None:
    channel = await connection.channel()
    await channel.set_qos(prefetch_count=10)
    queue = await channel.declare_queue(QUEUE_NAME, durable=True)

    async with queue.iterator() as q:
        async for message in q:
            async with message.process():
                try:
                    data = json.loads(message.body)
                    async with db_pool.acquire() as conn:
                        await conn.execute(
                            "INSERT INTO telemetry (sensor_id, value, status) VALUES ($1, $2, $3)",
                            int(data["sensor_id"]),
                            float(data["value"]),
                            str(data["status"]),
                        )
                except Exception as e:
                    print(f"Error processing telemetry message: {e}")


class TelemetryRecord(BaseModel):
    id: int
    sensor_id: int
    value: float
    status: str
    recorded_at: datetime


@app.get("/api/v1/telemetry/{sensor_id}", response_model=List[TelemetryRecord])
async def get_telemetry(
    sensor_id: int,
    from_time: datetime = Query(alias="from"),
    to_time: datetime = Query(alias="to"),
):
    async with db_pool.acquire() as conn:
        rows = await conn.fetch(
            """
            SELECT id, sensor_id, value, status, recorded_at
            FROM telemetry
            WHERE sensor_id = $1 AND recorded_at >= $2 AND recorded_at <= $3
            ORDER BY recorded_at
            """,
            sensor_id, from_time, to_time,
        )
    return [TelemetryRecord(**dict(r)) for r in rows]
