#! /bin/bash

#  to be run from scripts

python -m venv venv-images/
source venv-images/bin/activate
pip install -r requirements.txt
playwright install chromium


python generate_images.py --out_dir ../medical-images/generated-obs-native --count 10 --obs_portrait
