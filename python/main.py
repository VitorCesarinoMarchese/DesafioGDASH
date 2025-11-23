from datetime import datetime, timezone
from flask import Flask
import requests
import os
import time
import json
import pika
import threading

app = Flask(__name__)


@app.get("/health")
def health():
    return {"status": "ok"}, 200


def get_weather(latitude, longitude):
    url = "https://api.open-meteo.com/v1/forecast"

    params = {
        "latitude": latitude,
        "longitude": longitude,
        "hourly": ",".join([
            "temperature_2m",
            "relativehumidity_2m",
            "windspeed_10m",
            "cloudcover",
            "precipitation_probability"
        ])
    }

    response = requests.get(url, params=params)
    response.raise_for_status()
    data = response.json()

    hourly = data["hourly"]

    now = datetime.now(timezone.utc)
    last_full_hour = now.replace(minute=0, second=0, microsecond=0)

    timestamps = hourly["time"]

    timestamp_objs = [
        datetime.fromisoformat(t).replace(tzinfo=timezone.utc)
        for t in timestamps
    ]

    try:
        index = timestamp_objs.index(last_full_hour)
    except ValueError:
        past_hours = [t for t in timestamp_objs if t <= last_full_hour]
        if not past_hours:
            raise Exception("Sem informacao sobre o tempo na ultima hora")
        index = timestamp_objs.index(past_hours[-1])

    result = {
        "timestamp": timestamps[index],
        "temperatura": hourly["temperature_2m"][index],
        "umidade": hourly["relativehumidity_2m"][index],
        "vento": hourly["windspeed_10m"][index],
        "ceu": hourly["cloudcover"][index],
        "probabilidade_chuva": hourly["precipitation_probability"][index]
    }

    return result


def send_to_queue(data):
    url = os.getenv("RABBITMQ_URL")
    queue_name = os.getenv("RABBITMQ_QUEUE", "weather")
    if url:
        params = pika.URLParameters(url)
        connection = pika.BlockingConnection(params)
    else:
        host = os.getenv("RABBITMQ_HOST", "rabbitmq")
        user = os.getenv("RABBITMQ_DEFAULT_USER", "guest")
        password = os.getenv("RABBITMQ_DEFAULT_PASS", "guest")
        credentials = pika.PlainCredentials(user, password)
        connection = pika.BlockingConnection(
            pika.ConnectionParameters(host=host, credentials=credentials)
        )
    channel = connection.channel()
    channel.queue_declare(queue=queue_name, durable=True)
    channel.basic_publish(
        exchange='',
        routing_key=queue_name,
        body=json.dumps(data, ensure_ascii=False),
        properties=pika.BasicProperties(
            content_type='application/json',
            delivery_mode=2
        )
    )
    connection.close()


def worker_loop():
    print("Weather worker started...")
    lat = os.getenv("OPEN_METEO_LAT")
    lon = os.getenv("OPEN_METEO_LON")

    while True:
        try:
            clima = get_weather(lat, lon)
            send_to_queue(clima)
            print("menssagem enviada para fila:", clima)
            time.sleep(30)
        except Exception as e:
            print("ERROR IN WORKER LOOP:")
            print(e)
            time.sleep(10)


if __name__ == "__main__":
    print("Starting app...")

    t = threading.Thread(target=worker_loop, daemon=True)
    t.start()

    time.sleep(1)

    app.run(host="0.0.0.0", port=8000)
