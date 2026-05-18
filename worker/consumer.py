import redis
import json
import os
from dotenv import load_dotenv
load_dotenv()

host = os.getenv("REDIS_HOST", "localhost")
port = int(os.getenv("REDIS_PORT", "6379"))

GROUP_NAME = 'ml-workers'
STREAM_KEY = 'vigil:events'

r = redis.Redis(
    host=host,
    port=port,
    decode_responses=True
)

# Create consumer group if it doesn't exist
try:
    r.xgroup_create(STREAM_KEY, GROUP_NAME, id='$', mkstream=True)
except redis.exceptions.ResponseError as e:
    if "BUSYGROUP" not in str(e):
        raise

def process(fields):
    print(f"processing: {fields}")

print("Worker started, waiting for events...")
while True:
    # Block for up to 5 seconds waiting for events
    events = r.xreadgroup(
        groupname=GROUP_NAME,
        consumername='worker-1',
        streams={STREAM_KEY: '>'},
        count=10,
        block=5000
    )

    if not events:
        continue

    for stream_name, messages in events:
        for msg_id, fields in messages:
            process(fields)
            r.xack(STREAM_KEY, GROUP_NAME, msg_id)
