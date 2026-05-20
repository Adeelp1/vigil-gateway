import redis
import os
import time

from dotenv import load_dotenv

from ml import built_vector, cosine_similarity, ATTACK_PATTERNS

load_dotenv()

host = os.getenv("REDIS_HOST", "localhost")
port = int(os.getenv("REDIS_PORT", "6379"))

GROUP_NAME = 'ml-workers'
STREAM_KEY = 'vigil:events'
SCORE_KEY = 'vigil:score'
CONSUMER_NAME = f"worker-{os.getpid()}"
RECOVERY_INTERVAL = 15

r = redis.Redis(
    host=host,
    port=port,
    decode_responses=True
)

# Create consumer group if it doesn't exist
try:
    r.xgroup_create(STREAM_KEY, GROUP_NAME, id='$', mkstream=True)
    print(f"Created consumer group '{GROUP_NAME}' on stream '{STREAM_KEY}'")
except redis.exceptions.ResponseError as e:
    if "BUSYGROUP" not in str(e):
        raise

def running_average(old_score: float, new_score: float) -> float:
    return old_score * 0.7 + new_score * 0.3

def update_score(ip: str, new_score: float) -> None:
    old_record = r.get(f"{SCORE_KEY}:{ip}")

    if old_record:
        old_score = float(old_record.split('|')[0])
        final_score = running_average(old_score, new_score)
    else:
        final_score = new_score
    
    value = f"{final_score:.6f}|{int(time.time())}"
    r.set(f"{SCORE_KEY}:{ip}", value, ex=1800) # 30 min TTL
    print(f"Updated score for {ip}: {final_score:.3f}")


def process(fields) -> float:
    print(f"processing: {fields}")
    vector = built_vector(event=fields)

    max_similarity = max(cosine_similarity(vector, pattern) for pattern in ATTACK_PATTERNS)

    threat_score = max(0.0, min(1.0, max_similarity))

    print(f"ip={fields['ip']} score={threat_score:.3f} path={fields['path']}")

    return threat_score


def recover_pending_messages():
    try:
        result = r.xautoclaim(
            STREAM_KEY,
            GROUP_NAME,
            CONSUMER_NAME,
            min_idle_time=5000,  # 5 sec
            start_id='0-0',
            count=10
        )

        _, messages, _ = result

        for msg_id, fields in messages:
            try:
                print(f"Recovered pending message: {msg_id}")

                score = process(fields)
                update_score(fields['ip'], score)

                r.xack(STREAM_KEY, GROUP_NAME, msg_id)

            except Exception as e:
                print(f"Failed recovered message {msg_id}: {e}")

    except Exception as e:
        print(f"Recovery error: {e}")


last_recovery = 0

print("Worker started, waiting for events...")
while True:
    now = time.time()

    if now - last_recovery > RECOVERY_INTERVAL:
        recover_pending_messages()
        last_recovery = now

    # Block for up to 5 seconds waiting for events
    events = r.xreadgroup(
        groupname=GROUP_NAME,
        consumername=CONSUMER_NAME,
        streams={STREAM_KEY: '>'},
        count=10,
        block=5000
    )

    if not events:
        continue

    for stream_name, messages in events:
        for msg_id, fields in messages:
            try:
                score = process(fields)
                update_score(fields['ip'], score)
                r.xack(STREAM_KEY, GROUP_NAME, msg_id)
            except Exception as e:
                print(f"failed to process event {msg_id}: {e}")
                recover_pending_messages()

