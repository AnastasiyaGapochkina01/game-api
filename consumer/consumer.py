import os
import redis
import json
from flask import Flask, jsonify

app = Flask(__name__)

# Подключение к Redis с параметрами из окружения
redis_host = os.getenv("REDIS_HOST", "redis")
redis_port = int(os.getenv("REDIS_PORT", 6379))
r = redis.Redis(host=redis_host, port=redis_port, decode_responses=True)

@app.route("/characters", methods=["GET"])
def get_characters():
    keys = r.keys("char:*")
    chars = []
    for key in keys:
        data = r.get(key)
        if data:
            try:
                chars.append(json.loads(data))
            except json.JSONDecodeError:
                pass
    return jsonify(chars)

@app.route("/health", methods=["GET"])
def health():
    return jsonify({"status": "ok"})

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8081)

