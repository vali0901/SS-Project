import os
import json
import random
import argparse
from datetime import datetime, timedelta
import cv2
import numpy as np
from jinja2 import Environment, FileSystemLoader
from playwright.sync_api import sync_playwright
import uuid
import psycopg

# Database Connection String
DB_CONN = "postgresql://admin:supersecret@localhost:5432/mqtt_streaming"
OUT_DIR = "../uploads/"
PHOTO_DIR = OUT_DIR + 'photos/'
VALIDATION_DIR = OUT_DIR + 'validation/'

# --- Localization Seed Tables ---
NAMES = ["Ion", "Maria", "Andrei", "Elena", "Radu", "Ana", "George", "Ioana", "Mihai", "Cristina", "Alexandru", "Gabriela", "Florin", "Daniela", "Vlad"]
SURNAMES = ["Popescu", "Ionescu", "Dumitru", "Stoica", "Radu", "Gheorghe", "Matei", "Florea", "Costea", "Marinescu", "Dinu", "Toma", "Stanciu", "Neagu", "Preda"]
JOBS = ["Inginer", "Programator", "Medic", "Profesor", "Contabil", "Șofer", "Manager", "Student", "Asistent", "Operator"]

STR_STREETS = ["Aleea Trandafirilor", "Strada Primăverii", "Bulevardul Unirii", "Calea Victoriei", "Strada Mihai Eminescu", "Splaiul Independenței", "Bulevardul Ion Mihalache"]
STR_CLINICS = ["Clinica MedLife București", "Spitalul Regina Maria", "Sanador Victoriei", "Centrul Medical Arcadia", "S.C. ANIMA SPECIALITY MEDICAL SERVICES S.R.L."]
STR_COMPANIES = ["Logistica Română SRL", "Tehnologii Digitale SA", "UNIVERSITATEA NAȚIONALĂ DE ȘTIINȚĂ ȘI TEHNOLOGIE POLITEHNICA BUCUREȘTI", "Distribuție Muntenia SRL", "Compania Globală SRL"]
STR_DEPARTMENTS = ["FACULTATEA DE AUTOMATICA SI CALCULATOARE", "Departamentul Administrativ", "Secția Logistică și Transport", "Direcția Resurse Umane"]

OBS_CANVAS_WIDTH = 1080
OBS_CANVAS_HEIGHT = 1920

def wrap_field(value):
    """Wraps values directly into your generic ExtractedField application database layout structures."""
    return {
        "value": value,
        "confidence": round(random.uniform(0.95, 1.0), 2),
        "is_edited": False,
        "is_validated": random.choice([True, False])
    }

def generate_random_photo():
    timestamp = datetime.now() - timedelta(days=random.randint(0, 45))
    nume = random.choice(SURNAMES)
    prenume = random.choice(NAMES)
    
    # 1. Handle Selection of Control Type
    control_types = ["Angajare", "Periodic", "Adaptare", "Reluare", "Supraveghere", "Alte"]
    selected_control = random.choice(control_types)
    
    # 2. Handle Weighted Selection of Conclusion Aviz
    aviz_types = ["APT", "APT Conditionat", "Inapt Temporar", "Inapt"]
    selected_aviz = random.choices(aviz_types, weights=[72, 14, 9, 5], k=1)[0]
    
    # 3. Formulate Dates Into ISO 8601 (RFC3339) Strings for Go's time.Time
    time_base = timestamp.replace(hour=9, minute=0, second=0, microsecond=0)
    time_expiry = time_base + timedelta(days=365)
    
    go_time_format = "%Y-%m-%dT%H:%M:%SZ"
    
    # 4. Generate Phone Extensions
    tel_clinic = f"+40 21 {random.randint(400, 409)} {random.randint(10, 99)} {random.randint(10, 99)}"
    tel_company = f"07{random.randint(22, 76)}{random.randint(100, 999)}{random.randint(100, 999)}"

    meta = {
        "timestamp": timestamp,
        "image_type": "jpeg",
        "device_id": f"device-{random.randint(1, 5)}",
        "ocr_text": f"Fake OCR for {nume} {prenume}"
    }

    # 5. Populate Complete Structured Model Map Matching every Go Struct Parameter
    model_map = {
        # Header - Unitatea Medicala
        "unitate_medicala": wrap_field(random.choice(STR_CLINICS)),
        "adresa_unitate_medicala": wrap_field(f"{random.choice(STR_STREETS)}, nr. {random.randint(1, 45)}, Sector {random.choice([1,2,3,4,5,6])}"),
        "telefon_unitate_medicala": wrap_field(tel_clinic),

        # Header - Tip Fisa
        "numar_fisa": wrap_field(str(random.randint(100000, 999999))),

        # Sectiune Angajator / Institutie
        "societate_unitate": wrap_field(random.choice(STR_COMPANIES)),
        "adresa_angajator": wrap_field(f"{random.choice(STR_STREETS)}, nr. {random.randint(10, 320)}, București"),
        "telefon_angajator": wrap_field(tel_company),

        # Date Personale Angajat
        "nume": wrap_field(nume),
        "prenume": wrap_field(prenume),
        "cnp": wrap_field(f"{random.choice([1, 2, 5, 6])}{random.randint(50, 99):02d}{random.randint(1, 12):02d}{random.randint(1, 28):02d}{random.randint(100000, 999999)}"),

        # Date Profesionale
        "profesie_functie": wrap_field(random.choice(JOBS).upper()),
        "loc_de_munca": wrap_field(random.choice(STR_DEPARTMENTS) + ", București"),

        # Date Medicale - Tip Control Boolean Array Block
        "tip_control": wrap_field(f"Control {selected_control}"),
        "control_angajare": wrap_field(selected_control == "Angajare"),
        "control_periodic": wrap_field(selected_control == "Periodic"),
        "control_adaptare": wrap_field(selected_control == "Adaptare"),
        "control_reluare": wrap_field(selected_control == "Reluare"),
        "control_supraveghere": wrap_field(selected_control == "Supraveghere"),
        "control_alte": wrap_field(selected_control == "Alte"),

        # Date Medicale - Conclusion Block
        "aviz_medical": wrap_field(selected_aviz.upper()),
        "aviz_apt": wrap_field(selected_aviz == "APT"),
        "aviz_apt_conditionat": wrap_field(selected_aviz == "APT Conditionat"),
        "aviz_inapt_temporar": wrap_field(selected_aviz == "Inapt Temporar"),
        "aviz_inapt": wrap_field(selected_aviz == "Inapt"),

        # Core Metadata Bottom Section
        "recomandari": wrap_field("Nicio recomandare restrictivă. Apt pentru lucrul curent." if selected_aviz == "APT" else "Necesită reevaluare medicală de control în termen de 30 de zile cu scrisoare medicală."),
        "data": wrap_field(time_base.strftime(go_time_format)),
        "data_urm_examinari": wrap_field(time_expiry.strftime(go_time_format)),
        
        # UI Rendering helpers used to pass localized text masks cleanly into layout structures
        "display_data": time_base.strftime("%d/%m/%Y"),
        "display_data_urm": time_expiry.strftime("%d/%m/%Y")
    }
    return meta, model_map

def apply_image_degradation(image_path):
    """Processes images with pure OpenCV execution loops to mimic scanner imperfections."""
    img = cv2.imread(image_path)
    if img is None:
        return

    # 1. Scanner Alignment Shift (Rotation between -1.1 and 1.1 degrees)
    angle = random.uniform(-1.1, 1.1)
    h, w = img.shape[:2]
    M = cv2.getRotationMatrix2D((w // 2, h // 2), angle, 1.0)
    img = cv2.warpAffine(img, M, (w, h), borderMode=cv2.BORDER_REPLICATE)

    # # 2. Scanning Lens Defocus/Blur (45% occurrence rate)
    # if random.random() < 0.45:
    #     blur_mode = random.choice(["gaussian", "motion"])
    #     if blur_mode == "gaussian":
    #         img = cv2.GaussianBlur(img, (3, 3), 0)
    #     elif blur_mode == "motion":
    #         size = 5
    #         kernel_motion = np.zeros((size, size))
    #         for i in range(size):
    #             kernel_motion[i, random.randint(0, size - 1)] = 1
    #         kernel_motion = kernel_motion / size
    #         img = cv2.filter2D(img, -1, kernel_motion)

    # 3. Dynamic Artifact Compression Noise
    jpeg_quality = random.randint(76, 89)
    encode_param = [int(cv2.IMWRITE_JPEG_QUALITY), jpeg_quality]
    success, encimg = cv2.imencode('.jpg', img, encode_param)
    if success:
        img = cv2.imdecode(encimg, cv2.IMREAD_COLOR)

    cv2.imwrite(image_path, img)

def insert_db(id, meta, medical_data):
    try:
        # Establish connection to PostgreSQL
        with psycopg.connect(DB_CONN) as conn:
            with conn.cursor() as cur:
                
                # Check current count
                cur.execute("SELECT COUNT(*) FROM photos;")
                current_count = cur.fetchone()[0]
                print(f"Current document count: {current_count}")
            
                    
                # Prepare SQL Insert Statement
                query = """
                    INSERT INTO photos (id, timestamp, image_type, device_id, user_email, text, medical_data)
                    VALUES (%s, %s, %s, %s, %s, %s, %s);
                """
                
                # Execute with values. psycopg automatically converts dict to JSONB string
                cur.execute(query, (
                    id,
                    meta["timestamp"],
                    meta["image_type"],
                    meta["device_id"],
                    "test@mail.com",
                    meta["ocr_text"],
                    json.dumps(medical_data)
                ))
                
                # Commit explicitly if your connection configuration needs it
                conn.commit()
                
                # Check new count
                cur.execute("SELECT COUNT(*) FROM photos;")
                new_count = cur.fetchone()[0]
                print(f"New document count: {new_count}")
                
    except Exception as e:
        print(f"An error occurred: {e}")
        print("Ensure PostgreSQL is running and credentials match.")

def main():
    parser = argparse.ArgumentParser(description="Generate comprehensive dataset matching Go structs completely.")
    parser.add_argument("--insert-db", action="store_true", help="also insert into db")
    parser.add_argument("--count", type=int, default=5, help="Number of data iterations.")
    parser.add_argument(
        "--obs_portrait",
        action="store_true",
        help="Render generated JPEGs directly at 1080x1920 for OBS virtual camera input."
    )
    args = parser.parse_args()

    os.makedirs(PHOTO_DIR, exist_ok=True)
    os.makedirs(VALIDATION_DIR, exist_ok=True)
    
    env = Environment(loader=FileSystemLoader("templates"))
    layouts = ["layout_1.html", "layout_2.html", "layout_3.html"]
    fonts = ["Arial", "Courier New", "Times New Roman", "Georgia", "Verdana"]

    print(f"🚀 Mapping all db fields to Romanian forms. Compiling {args.count} elements into '{OUT_DIR}'...")

    with sync_playwright() as p:
        browser = p.chromium.launch()
        
        for idx in range(1, args.count + 1):
            meta, record_data = generate_random_photo()
            selected_layout = random.choice(layouts)
            selected_font = random.choice(fonts)
            
            # Render out the targeted HTML template
            template = env.get_template(selected_layout)
            html_content = template.render(d=record_data, font=selected_font)
            
            if args.obs_portrait:
                v_width, v_height = OBS_CANVAS_WIDTH, OBS_CANVAS_HEIGHT
            else:
                # Match landscape viewport rules specifically for the dual-page Fișă de Aptitudine layout
                v_width, v_height = (850, 1150)
            
            page = browser.new_page(viewport={"width": v_width, "height": v_height})
            page.set_content(html_content)
            page.wait_for_timeout(150)  # Safe execution render block buffer
            
            id = str(uuid.uuid4())
            img_path = os.path.join(PHOTO_DIR, f"{id}.jpg")
            json_path = os.path.join(VALIDATION_DIR, f"{id}.json")
            
            page.screenshot(path=img_path, type="jpeg", quality=100)
            page.close()
            
            # Run image degradation
            # apply_image_degradation(img_path)
            
            # Remove localized string formatters used solely for UI visualization
            record_data.pop("display_data", None)
            record_data.pop("display_data_urm", None)
            
            # Save strict Go JSON data models
            with open(json_path, "w", encoding="utf-8") as file_out:
                json.dump(record_data, file_out, indent=4, ensure_ascii=False)

            if args.insert_db:
                insert_db(id, meta, record_data)

        browser.close()
        
    print(f"✨ Generation finalized. Every single struct field is visually represented in Romanian!")

if __name__ == "__main__":
    main()
