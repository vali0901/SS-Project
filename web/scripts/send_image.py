# TODO: Implement mTLS security - See docs/SECURITY_IMPLEMENTATION.md
import time
import os
import socket
import json
import paho.mqtt.client as mqtt
import sys
import io
import ssl
import requests
from PIL import Image, ImageDraw
import argparse
import threading

# Configuration
BROKER = "127.0.0.1"
PORT = 8883
API_URL = "http://127.0.0.1:8080"

# ===== CHANGE THIS FOR EACH DEVICE =====
DEVICE_ID = "python-sender-2"  # Unique ID for this device
DEVICE_NAME = "Python Test Device"  # Human-readable name
# ========================================

# Default Credentials
DEFAULT_EMAIL = "test@mail.com"
DEFAULT_PASSWORD = "test"

# Topics
REGISTER_TOPIC = f"register/{DEVICE_ID}"
PHOTO_TOPIC = f"ssproject/images/{DEVICE_ID}"

# Global variable for token and state
TOKEN = None
args = None

# Get absolute paths
SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
PROJECT_ROOT = os.path.dirname(SCRIPT_DIR)

SECRETS_DIR = os.path.join(PROJECT_ROOT, "secrets")
CA_CRT = os.path.join(SECRETS_DIR, "ca.crt")
CLIENT_CRT = os.path.join(SECRETS_DIR, "web.crt")
CLIENT_KEY = os.path.join(SECRETS_DIR, "web.key")

def get_token_from_api(email, password):
    """Log in to the API and get a JWT token"""
    try:
        print(f"Logging in to {API_URL} as {email}...")
        response = requests.post(
            f"{API_URL}/login",
            json={"email": email, "password": password},
            timeout=10
        )
        if response.status_code == 200:
            data = response.json()
            token = data.get("token")
            print("Login successful, token obtained.")
            return token
        else:
            print(f"Login failed ({response.status_code}): {response.text}")
            return None
    except Exception as e:
        print(f"Error during login: {e}")
        return None

def get_local_ip():
    """Get the local IP address of this machine"""
    try:
        s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        s.connect(("8.8.8.8", 80))
        ip = s.getsockname()[0]
        s.close()
        return ip
    except:
        return "unknown"

def create_test_image():
    """Create a test image with timestamp and device ID"""
    img = Image.new('RGB', (300, 150), color='white')
    d = ImageDraw.Draw(img)
    d.text((10, 20), f"Device: {DEVICE_ID}", fill='black')
    d.text((10, 50), f"Time: {time.strftime('%H:%M:%S')}", fill='black')
    d.text((10, 80), "HELLO ADMIN", fill='blue')
    
    img_byte_arr = io.BytesIO()
    img.save(img_byte_arr, format='JPEG')
    return img_byte_arr.getvalue()

def load_image_from_file(path):
    """Load image from file and convert to bytes"""
    try:
        with Image.open(path) as img:
            # Convert to RGB if necessary (e.g. for PNGs with transparency)
            if img.mode in ('RGBA', 'P'):
                img = img.convert('RGB')
            
            img_byte_arr = io.BytesIO()
            img.save(img_byte_arr, format='JPEG')
            return img_byte_arr.getvalue()
    except Exception as e:
        print(f"Error loading image {path}: {e}")
        sys.exit(1)

def on_connect(client, userdata, flags, rc):
    if rc == 0:
        print("Connected to MQTT Broker!")
        
        # Step 1: Register the device
        local_ip = get_local_ip()
        registration_data = {
            "name": DEVICE_NAME,
            "ip": local_ip,
            "port": str(PORT),
            "token": TOKEN
        }
        print(f"Registering device: {REGISTER_TOPIC}")
        client.publish(REGISTER_TOPIC, json.dumps(registration_data))
        
        # Give a small delay for registration to process on server
        threading.Timer(0.5, lambda: send_photo(client)).start()
    else:
        print(f"Failed to connect, return code {rc}")
        sys.exit(1)

def send_photo(client):
    # Step 2: Send the photo
    print(f"Publishing image to: {PHOTO_TOPIC}")
    
    image_data = None
    if args and args.image:
        file_path = args.image
        print(f"Sending file: {file_path}")
        image_data = load_image_from_file(file_path)
    else:
        print("Sending generated test image (no file argument provided)")
        image_data = create_test_image()

    client.publish(PHOTO_TOPIC, image_data)

def on_publish(client, userdata, mid):
    # Registration is mid=1, Photo is mid=2
    if mid >= 2:
        print("\n✅ Photo sent successfully!")
        client.disconnect()

def on_disconnect(client, userdata, rc):
    print("Disconnected from MQTT Broker")
    sys.exit(0)

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Send image to SS-Project MQTT Broker (HTTP Auth)')
    parser.add_argument('image', nargs='?', help='Path to image file (optional)')
    parser.add_argument('--token', help='JWT Authentication Token (overrides login)')
    parser.add_argument('--email', help=f'Email for login (default: {DEFAULT_EMAIL})')
    parser.add_argument('--password', help=f'Password for login (default: {DEFAULT_PASSWORD})')
    parser.add_argument('--id', default=DEVICE_ID, help=f'Device ID (default: {DEVICE_ID})')
    parser.add_argument('--name', default=DEVICE_NAME, help=f'Device Name (default: {DEVICE_NAME})')
    
    args = parser.parse_args()
    
    DEVICE_ID = args.id
    DEVICE_NAME = args.name
    TOKEN = args.token or os.environ.get('MQTT_TOKEN')
    
    # Update topics
    REGISTER_TOPIC = f"register/{DEVICE_ID}"
    PHOTO_TOPIC = f"ssproject/images/{DEVICE_ID}"

    if not TOKEN:
        email = args.email or DEFAULT_EMAIL
        password = args.password or DEFAULT_PASSWORD
        TOKEN = get_token_from_api(email, password)
        
    if not TOKEN:
        print("ERROR: Authentication failed. No token available.")
        sys.exit(1)

    # Create MQTT client
    client = mqtt.Client(client_id=DEVICE_ID)
    client.on_connect = on_connect
    client.on_publish = on_publish
    client.on_disconnect = on_disconnect

    client.tls_set(ca_certs=CA_CRT, certfile=CLIENT_CRT, keyfile=CLIENT_KEY, tls_version=ssl.PROTOCOL_TLSv1_2)

    print(f"Device ID: {DEVICE_ID}")
    print(f"Connecting to {BROKER}:{PORT}...")
    try:
        client.connect(BROKER, PORT, 60)
        client.loop_forever()
    except Exception as e:
        print(f"Connection failed: {e}")
        sys.exit(1)
