from dotenv import load_dotenv
from datetime import datetime, timezone
import requests
import os
import time
import json
import pika

load_dotenv("../.env")


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
            raise Exception(
                "Sem informacao sobre o tempo na ultima hora na api")
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
        host = os.getenv("RABBITMQ_HOST", "localhost")
        user = os.getenv("RABBITMQ_USER", "guest")
        password = os.getenv("RABBITMQ_PASS", "guest")
        credentials = pika.PlainCredentials(user, password)
        connection = pika.BlockingConnection(
            pika.ConnectionParameters(host=host, credentials=credentials))
    channel = connection.channel()
    channel.queue_declare(queue=queue_name, durable=True)
    channel.basic_publish(
        exchange='',
        routing_key=queue_name,
        body=json.dumps(data, ensure_ascii=False),
        properties=pika.BasicProperties(
            content_type='application/json', delivery_mode=2)
    )
    connection.close()


if __name__ == "__main__":
    while True:
        clima = get_weather(os.getenv("OPEN_METEO_LAT"),
                            os.getenv("OPEN_METEO_LON"))
        print("===== PREVISÃO DO TEMPO (ENVIADA PARA FILA) =====")
        print(f"Timestamp: {clima['timestamp']}")
        print(f"Temperatura: {clima['temperatura']}°C")
        print(f"Umidade: {clima['umidade']}%")
        print(f"Velocidade do vento: {clima['vento']} km/h")
        print(f"Cobertura do céu: {clima['ceu']}%")
        print(f"Probabilidade de chuva: {clima['probabilidade_chuva']}%")
        time.sleep(3600)
