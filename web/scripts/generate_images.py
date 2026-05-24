import os
import json
import random
import argparse
import calendar
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
JOBS = [
    "INGINER",
    "PROGRAMATOR",
    "MEDIC",
    "PROFESOR UNIVERSITAR",
    "CONTABIL",
    "SOFER",
    "MANAGER",
    "STUDENT",
    "ASISTENT",
    "OPERATOR"
]

# Demo-friendly defaults so helper scripts can stay minimal.
DEMO_DEFAULTS = {
    "ocr_success_rate": 0.88,
    "latency_min_ms": 120,
    "latency_max_ms": 1800,
    "next_month_expiry_rate": 0.35,
    "overdue_rate": 0.20,
    "professor_rate": 0.30,
    "professor_fit_rate": 0.95,
    "general_fit_rate": 0.76,
    "overdue_fit_multiplier": 0.45,
    "periodic_rate": 0.62,
    "months_back": 8,
    "trend_mode": "increasing",
    "trend_strength": 1.2,
    "seed": 42,
}

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


def weighted_choice(choices_with_weights):
    values, weights = zip(*choices_with_weights)
    return random.choices(values, weights=weights, k=1)[0]


def choose_job(professor_rate):
    professor_rate = max(0.0, min(1.0, professor_rate))
    non_prof_jobs = [j for j in JOBS if "PROFESOR" not in j]

    if random.random() < professor_rate:
        return "PROFESOR UNIVERSITAR"
    return random.choice(non_prof_jobs)


def choose_control_type(periodic_rate):
    periodic_rate = max(0.0, min(1.0, periodic_rate))
    if random.random() < periodic_rate:
        return "Periodic"

    other_controls = ["Angajare", "Adaptare", "Reluare", "Supraveghere", "Alte"]
    return random.choice(other_controls)


def choose_aviz_for_job(job, professor_fit_rate, general_fit_rate):
    fit_rate = professor_fit_rate if "PROFESOR" in job else general_fit_rate
    fit_rate = max(0.0, min(1.0, fit_rate))

    if random.random() < fit_rate:
        return weighted_choice([
            ("APT", 88),
            ("APT Conditionat", 12),
        ])

    return weighted_choice([
        ("Inapt Temporar", 70),
        ("Inapt", 30),
    ])


def pick_month_start(months_back, trend_mode, trend_strength):
    months_back = max(1, months_back)
    trend_strength = max(0.0, trend_strength)

    now = datetime.now()
    months = []
    for i in range(months_back):
        year = now.year
        month = now.month - i
        while month <= 0:
            month += 12
            year -= 1
        months.append(datetime(year, month, 1))
    months = list(reversed(months))

    if trend_mode == "increasing":
        weights = [1.0 + trend_strength * (idx + 1) for idx in range(len(months))]
    elif trend_mode == "decreasing":
        weights = [1.0 + trend_strength * (len(months) - idx) for idx in range(len(months))]
    elif trend_mode == "seasonal":
        center = (len(months) - 1) / 2.0
        weights = [1.0 + trend_strength * (1.0 - min(1.0, abs(idx - center) / max(1.0, center))) for idx in range(len(months))]
    else:
        weights = [1.0 for _ in months]

    return random.choices(months, weights=weights, k=1)[0]


def generate_timestamp(months_back, trend_mode, trend_strength):
    month_start = pick_month_start(months_back, trend_mode, trend_strength)
    days_in_month = calendar.monthrange(month_start.year, month_start.month)[1]
    day = random.randint(1, days_in_month)
    hour = random.randint(8, 17)
    minute = random.randint(0, 59)
    return datetime(month_start.year, month_start.month, day, hour, minute, 0)


def build_expiry_date(base_time, next_month_expiry_rate, overdue_rate):
    """Generate expiry date relative to record month for realistic monthly compliance trends."""
    base_month_start = base_time.replace(day=1, hour=9, minute=0, second=0, microsecond=0)
    next_month_start = (base_month_start + timedelta(days=32)).replace(day=1)
    month_after_next_start = (next_month_start + timedelta(days=32)).replace(day=1)
    base_month_end = next_month_start - timedelta(seconds=1)

    roll = random.random()
    if roll < next_month_expiry_rate:
        days_in_next_month = (month_after_next_start - next_month_start).days
        day = random.randint(1, days_in_next_month)
        return next_month_start.replace(day=day, hour=9, minute=0, second=0, microsecond=0), "next_month"

    if roll < (next_month_expiry_rate + overdue_rate):
        # Deliberately keep expiry inside the same month so bucket logic marks it as overdue.
        expiry_day = random.randint(1, max(1, base_month_end.day))
        return base_month_start.replace(day=expiry_day, hour=9, minute=0, second=0, microsecond=0), "overdue"

    return base_time + timedelta(days=random.randint(60, 360)), "future"


def generate_random_photo(
    ocr_success_rate=0.9,
    latency_min_ms=120,
    latency_max_ms=1800,
    next_month_expiry_rate=0.7,
    overdue_rate=0.15,
    professor_rate=0.25,
    professor_fit_rate=0.93,
    general_fit_rate=0.78,
    overdue_fit_multiplier=0.45,
    periodic_rate=0.60,
    months_back=6,
    trend_mode="increasing",
    trend_strength=0.9,
):
    timestamp = generate_timestamp(months_back, trend_mode, trend_strength)
    nume = random.choice(SURNAMES)
    prenume = random.choice(NAMES)
    
    # 1. Handle Selection of Control Type
    selected_control = choose_control_type(periodic_rate)
    
    # 2. Build expiry first so aviz distribution can depend on compliance status.
    time_base = timestamp.replace(hour=9, minute=0, second=0, microsecond=0)
    time_expiry, expiry_bucket = build_expiry_date(time_base, next_month_expiry_rate, overdue_rate)

    # 3. Handle Weighted Selection of Conclusion Aviz
    selected_job = choose_job(professor_rate)
    adjusted_professor_fit = professor_fit_rate
    adjusted_general_fit = general_fit_rate
    if expiry_bucket == "overdue":
        adjusted_professor_fit *= overdue_fit_multiplier
        adjusted_general_fit *= overdue_fit_multiplier
    selected_aviz = choose_aviz_for_job(selected_job, adjusted_professor_fit, adjusted_general_fit)
    
    # 4. Formulate dates into ISO 8601 (RFC3339) strings for Go's time.Time
    
    go_time_format = "%Y-%m-%dT%H:%M:%SZ"
    
    # 5. Generate Phone Extensions
    tel_clinic = f"+40 21 {random.randint(400, 409)} {random.randint(10, 99)} {random.randint(10, 99)}"
    tel_company = f"07{random.randint(22, 76)}{random.randint(100, 999)}{random.randint(100, 999)}"

    ocr_success = random.random() < ocr_success_rate
    processing_latency_ms = random.randint(latency_min_ms, latency_max_ms)

    meta = {
        "timestamp": timestamp,
        "image_type": "jpeg",
        "device_id": f"device-{random.randint(1, 5)}",
        "ocr_text": f"Fake OCR for {nume} {prenume}" if ocr_success else "OCR failed",
        "ocr_success": ocr_success,
        "processing_latency_ms": processing_latency_ms,
    }

    # 6. Populate Complete Structured Model Map Matching every Go Struct Parameter
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
        "profesie_functie": wrap_field(selected_job),
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
                    INSERT INTO photos (id, timestamp, image_type, processing_latency_ms, ocr_success, device_id, user_email, text, medical_data)
                    VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s);
                """
                
                # Execute with values. psycopg automatically converts dict to JSONB string
                cur.execute(query, (
                    id,
                    meta["timestamp"],
                    meta["image_type"],
                    meta["processing_latency_ms"],
                    meta["ocr_success"],
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
    parser.add_argument("--ocr-success-rate", type=float, default=DEMO_DEFAULTS["ocr_success_rate"], help="Probability [0.0-1.0] that OCR succeeds.")
    parser.add_argument("--latency-min-ms", type=int, default=DEMO_DEFAULTS["latency_min_ms"], help="Minimum processing latency in ms.")
    parser.add_argument("--latency-max-ms", type=int, default=DEMO_DEFAULTS["latency_max_ms"], help="Maximum processing latency in ms.")
    parser.add_argument("--next-month-expiry-rate", type=float, default=DEMO_DEFAULTS["next_month_expiry_rate"], help="Probability [0.0-1.0] that data_urm_examinari is in next calendar month.")
    parser.add_argument("--overdue-rate", type=float, default=DEMO_DEFAULTS["overdue_rate"], help="Probability [0.0-1.0] that data_urm_examinari is already overdue.")
    parser.add_argument("--professor-rate", type=float, default=DEMO_DEFAULTS["professor_rate"], help="Probability [0.0-1.0] that profesie_functie is PROFESSOR.")
    parser.add_argument("--professor-fit-rate", type=float, default=DEMO_DEFAULTS["professor_fit_rate"], help="Probability [0.0-1.0] that professor is FIT (APT/APT Conditionat).")
    parser.add_argument("--general-fit-rate", type=float, default=DEMO_DEFAULTS["general_fit_rate"], help="Probability [0.0-1.0] that non-professor is FIT (APT/APT Conditionat).")
    parser.add_argument("--overdue-fit-multiplier", type=float, default=DEMO_DEFAULTS["overdue_fit_multiplier"], help="Multiplier [0.0-1.0] applied to fit rates for overdue records.")
    parser.add_argument("--periodic-rate", type=float, default=DEMO_DEFAULTS["periodic_rate"], help="Probability [0.0-1.0] that control type is Periodic.")
    parser.add_argument("--months-back", type=int, default=DEMO_DEFAULTS["months_back"], help="How many months back to distribute generated records.")
    parser.add_argument("--trend-mode", choices=["flat", "increasing", "decreasing", "seasonal"], default=DEMO_DEFAULTS["trend_mode"], help="Monthly volume pattern for generated records.")
    parser.add_argument("--trend-strength", type=float, default=DEMO_DEFAULTS["trend_strength"], help="Trend intensity for monthly distribution pattern.")
    parser.add_argument("--seed", type=int, default=DEMO_DEFAULTS["seed"], help="Optional random seed for reproducible datasets.")
    parser.add_argument(
        "--obs_portrait",
        action="store_true",
        help="Render generated JPEGs directly at 1080x1920 for OBS virtual camera input."
    )
    args = parser.parse_args()

    # Keep bounds sane and deterministic for test data generation.
    args.ocr_success_rate = max(0.0, min(1.0, args.ocr_success_rate))
    args.next_month_expiry_rate = max(0.0, min(1.0, args.next_month_expiry_rate))
    args.overdue_rate = max(0.0, min(1.0, args.overdue_rate))
    args.professor_rate = max(0.0, min(1.0, args.professor_rate))
    args.professor_fit_rate = max(0.0, min(1.0, args.professor_fit_rate))
    args.general_fit_rate = max(0.0, min(1.0, args.general_fit_rate))
    args.overdue_fit_multiplier = max(0.0, min(1.0, args.overdue_fit_multiplier))
    args.periodic_rate = max(0.0, min(1.0, args.periodic_rate))
    args.months_back = max(1, args.months_back)
    args.trend_strength = max(0.0, args.trend_strength)
    if args.seed is not None:
        random.seed(args.seed)
        np.random.seed(args.seed)
    if args.latency_min_ms < 0:
        args.latency_min_ms = 0
    if args.latency_max_ms < args.latency_min_ms:
        args.latency_max_ms = args.latency_min_ms

    os.makedirs(PHOTO_DIR, exist_ok=True)
    os.makedirs(VALIDATION_DIR, exist_ok=True)
    
    env = Environment(loader=FileSystemLoader("templates"))
    layouts = ["layout_1.html", "layout_2.html", "layout_3.html"]
    fonts = ["Arial", "Courier New", "Times New Roman", "Georgia", "Verdana"]

    print(f"🚀 Mapping all db fields to Romanian forms. Compiling {args.count} elements into '{OUT_DIR}'...")

    stats = {
        "aviz": {"APT": 0, "APT Conditionat": 0, "Inapt Temporar": 0, "Inapt": 0},
        "professors": {"total": 0, "fit": 0},
        "expiry": {"next_month": 0, "overdue": 0, "future": 0},
        "months": {},
    }

    with sync_playwright() as p:
        browser = p.chromium.launch()
        
        for idx in range(1, args.count + 1):
            meta, record_data = generate_random_photo(
                ocr_success_rate=args.ocr_success_rate,
                latency_min_ms=args.latency_min_ms,
                latency_max_ms=args.latency_max_ms,
                next_month_expiry_rate=args.next_month_expiry_rate,
                overdue_rate=args.overdue_rate,
                professor_rate=args.professor_rate,
                professor_fit_rate=args.professor_fit_rate,
                general_fit_rate=args.general_fit_rate,
                overdue_fit_multiplier=args.overdue_fit_multiplier,
                periodic_rate=args.periodic_rate,
                months_back=args.months_back,
                trend_mode=args.trend_mode,
                trend_strength=args.trend_strength,
            )
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

            # Lightweight generation summary for quick stats sanity checks
            aviz_value = record_data["aviz_medical"]["value"]
            if aviz_value in stats["aviz"]:
                stats["aviz"][aviz_value] += 1

            job_value = record_data["profesie_functie"]["value"]
            if "PROFESOR" in job_value:
                stats["professors"]["total"] += 1
                if aviz_value.startswith("APT"):
                    stats["professors"]["fit"] += 1

            expiry_dt = datetime.strptime(record_data["data_urm_examinari"]["value"], "%Y-%m-%dT%H:%M:%SZ")
            now = datetime.now()
            next_month_start = (now.replace(day=1) + timedelta(days=32)).replace(day=1)
            next_next_month_start = (next_month_start + timedelta(days=32)).replace(day=1)
            if next_month_start <= expiry_dt < next_next_month_start:
                stats["expiry"]["next_month"] += 1
            elif expiry_dt < now:
                stats["expiry"]["overdue"] += 1
            else:
                stats["expiry"]["future"] += 1

            month_key = meta["timestamp"].strftime("%Y-%m")
            stats["months"][month_key] = stats["months"].get(month_key, 0) + 1

        browser.close()
        
    print(f"✨ Generation finalized. Every single struct field is visually represented in Romanian!")
    print("\n=== Generation Summary (useful for Statistics page validation) ===")
    print(f"Aviz counts: {stats['aviz']}")
    print(f"Professor fit: {stats['professors']['fit']} / {stats['professors']['total']}")
    print(f"Expiry buckets: {stats['expiry']}")
    print("Monthly volume:")
    for month in sorted(stats["months"].keys()):
        print(f"  {month}: {stats['months'][month]}")

if __name__ == "__main__":
    main()
