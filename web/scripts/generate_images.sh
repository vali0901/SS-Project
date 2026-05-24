#!/bin/bash

# to be run from scripts

python3 -m venv venv-images/
source venv-images/bin/activate
pip install -r requirements.txt
playwright install chromium

python3 generate_images.py --count 10  --insert-db
