import json
import os

INPUT_PATH = r"D:\xampp\htdocs\petasjg\clasnet-peta-desa-master\peta_desa.geojson"
OUTPUT_PATH = r"D:\xampp\htdocs\petasjg\Veda-Dashboard-Pondokrejo\sijenggung.geojson"

def extract_sijenggung():
    print(f"Reading {INPUT_PATH}...")
    try:
        with open(INPUT_PATH, 'r', encoding='utf-8') as f:
            data = json.load(f)
    except Exception as e:
        print(f"Error reading input file: {e}")
        return

    features = data.get('features', [])
    found_feature = None

    print(f"Total features: {len(features)}")

    for feature in features:
        props = feature.get('properties', {})
        # Check various possible property names for village name
        name = props.get('Nama_Desa_', '') or props.get('NAMOBJ', '') or props.get('DESA', '') or props.get('VILLAGE', '')
        
        if "Sijenggung" in str(name):
            print(f"Found Sijenggung! Properties: {props}")
            found_feature = feature
            break
    
    if found_feature:
        # Create a new FeatureCollection with just this feature
        new_geojson = {
            "type": "FeatureCollection",
            "features": [found_feature]
        }
        
        try:
            with open(OUTPUT_PATH, 'w', encoding='utf-8') as f:
                json.dump(new_geojson, f, indent=2)
            print(f"Successfully saved Sijenggung boundary to {OUTPUT_PATH}")
        except Exception as e:
            print(f"Error writing output file: {e}")
    else:
        print("Sijenggung feature not found in the input GeoJSON.")

if __name__ == "__main__":
    extract_sijenggung()
