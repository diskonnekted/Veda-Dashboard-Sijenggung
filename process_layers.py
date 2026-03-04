import shapefile
import json
import os

# Configuration
SIJENGGUNG_GEOJSON = "sijenggung.geojson"
INPUT_DIR = r"..\KAB. BANJARNEGARA"
OUTPUT_DIR = "layers"

LAYERS_TO_PROCESS = {
    "PEMUKIMAN_AR_25K.shp": "pemukiman.geojson",
    "SUNGAI_LN_25K.shp": "sungai.geojson",
    "JALAN_LN_25K.shp": "jalan.geojson",
    "PENDIDIKAN_PT_25K.shp": "pendidikan.geojson",
    "KESEHATAN_PT_25K.shp": "kesehatan.geojson",
    "SARANAIBADAH_PT_25K.shp": "ibadah.geojson",
    "PEMERINTAHAN_PT_25K.shp": "pemerintahan.geojson",
    "AGRISAWAH_AR_25K.shp": "sawah.geojson",
    "AGRIKEBUN_AR_25K.shp": "kebun.geojson",
    "TOPONIMI_PT_25K.shp": "toponimi.geojson",
    "KONTUR_LN_25K.shp": "kontur.geojson",
}

def get_bbox(geojson_path):
    with open(geojson_path, 'r') as f:
        data = json.load(f)
    
    min_x, min_y = float('inf'), float('inf')
    max_x, max_y = float('-inf'), float('-inf')

    def update_bounds(coords):
        nonlocal min_x, min_y, max_x, max_y
        for c in coords:
            if isinstance(c[0], list):
                update_bounds(c)
            else:
                x, y = c[0], c[1]
                min_x = min(min_x, x)
                min_y = min(min_y, y)
                max_x = max(max_x, x)
                max_y = max(max_y, y)

    for feature in data['features']:
        geom = feature['geometry']
        if geom['type'] == 'Polygon':
            update_bounds(geom['coordinates'])
        elif geom['type'] == 'MultiPolygon':
            update_bounds(geom['coordinates'])
            
    return [min_x, min_y, max_x, max_y]

def intersect(bbox1, bbox2):
    # bbox = [minX, minY, maxX, maxY]
    return not (bbox1[2] < bbox2[0] or bbox1[0] > bbox2[2] or bbox1[3] < bbox2[1] or bbox1[1] > bbox2[3])

def process_shapefile(shp_name, output_name, clip_bbox):
    shp_path = os.path.join(INPUT_DIR, shp_name)
    if not os.path.exists(shp_path):
        print(f"Skipping {shp_name}: File not found")
        return

    print(f"Processing {shp_name}...")
    sf = shapefile.Reader(shp_path)
    fields = [field[0] for field in sf.fields[1:]]
    
    features = []
    
    for shape, record in zip(sf.shapes(), sf.records()):
        # Check intersection
        if hasattr(shape, 'bbox'):
            shape_bbox = shape.bbox
        else:
            # Calculate bbox from points if missing (e.g. PointZ)
            xs = [p[0] for p in shape.points]
            ys = [p[1] for p in shape.points]
            if not xs or not ys:
                continue
            shape_bbox = [min(xs), min(ys), max(xs), max(ys)]

        if intersect(shape_bbox, clip_bbox):
            # Convert record to dict
            record_dict = {}
            for i, field in enumerate(fields):
                val = record[i]
                if isinstance(val, bytes):
                    val = val.decode('utf-8', errors='replace')
                record_dict[field] = val
            
            feature = {
                "type": "Feature",
                "properties": record_dict,
                "geometry": shape.__geo_interface__
            }
            features.append(feature)
            
    if features:
        output_path = os.path.join(OUTPUT_DIR, output_name)
        with open(output_path, 'w') as f:
            json.dump({"type": "FeatureCollection", "features": features}, f)
        print(f"Saved {len(features)} features to {output_name}")
    else:
        print(f"No features found for {shp_name} within boundary")

def main():
    if not os.path.exists(OUTPUT_DIR):
        os.makedirs(OUTPUT_DIR)
        
    print("Calculating Sijenggung BBox...")
    clip_bbox = get_bbox(SIJENGGUNG_GEOJSON)
    print(f"BBox: {clip_bbox}")
    
    # Add a small buffer to the bbox to catch features on the edge
    buffer = 0.005 # approx 500m
    clip_bbox[0] -= buffer
    clip_bbox[1] -= buffer
    clip_bbox[2] += buffer
    clip_bbox[3] += buffer
    print(f"Buffered BBox: {clip_bbox}")

    for shp, out in LAYERS_TO_PROCESS.items():
        process_shapefile(shp, out, clip_bbox)

if __name__ == "__main__":
    main()
