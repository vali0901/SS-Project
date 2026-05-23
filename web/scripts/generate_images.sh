#! /bin/bash

#  to be run from scripts

python -m venv venv-images/
pip install -r requirements.txt
playwright install chromium


python generate_images.py --out_dir ../medical-images/generated --count 10