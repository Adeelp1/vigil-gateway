import math


METHOD_SCORES = {
    'GET':     0.0,
    'POST':    0.5,
    'PUT':     0.3,
    'DELETE':  0.8,
    'PATCH':   0.3,
    'OPTIONS': 0.1,
}

PATH_SCORES = {
    ('/admin', '/root', '/config', '/env'): 1.0,
    ('/login', '/auth', '/token'): 0.8,
    ('/user', '/account'): 0.4
}

# sample attack patterns
ATTACK_PATTERNS = [
    # Brute force login
    [1.0, 0.8, 1.0, 0.001],  # POST, auth path, 401, very fast

    # Admin scanning
    [0.5, 1.0, 0.8, 0.001],  # any method, admin path, 4xx, fast

    # Credential stuffing
    [1.0, 0.7, 1.0, 0.002],  # POST, login path, 401, fast

    # Path traversal
    [0.3, 0.9, 0.8, 0.001],  # GET, suspicious path, 4xx, fast
]

def cosine_similarity(vector: list, pattern: list) -> float:
    dot = sum(v * p for v, p in zip(vector, pattern))
    mag_a = math.sqrt(sum(v ** 2 for v in vector))
    mag_b = math.sqrt(sum(p ** 2 for p in pattern))
    if mag_a == 0 or mag_b == 0:
        return 0.0
    
    return dot / (mag_a * mag_b)

def built_vector(event: dict) -> list:
    method_score = METHOD_SCORES.get(event['method'], 0.5)

    path = event['path'].lower()
    path_score = 0.1
    for prefixes, score in PATH_SCORES.items():
        if any(prefix in path for prefix in prefixes):
            path_score = score
            break
    
    status = int(event['status'])
    if status in (400, 401, 403):
        status_score = 1.0
    elif status >= 500:
        status_score = 0.5
    elif status >= 400:
        status_score = 0.7
    else:
        status_score = 0.0
    
    duration = int(event['duration_ms'])
    if duration < 5:
        speed_score = 1.0
    elif duration <= 50:
        speed_score = 0.5
    else:
        speed_score = 0.0
    
    return [method_score, path_score, status_score, speed_score]