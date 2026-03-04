import shapefile
import json

shp_path = r"D:\xampp\htdocs\petasjg\KAB. BANJARNEGARA\ADMINISTRASIDESA_AR_25K.shp"
output_path = r"D:\xampp\htdocs\petasjg\Veda-Dashboard-Pondokrejo\sijenggung.geojson"

print(f"Reading {shp_path}...")
sf = shapefile.Reader(shp_path)
fields = [field[0] for field in sf.fields[1:]]

features = []
center_lat = 0
center_lon = 0

for shape, record in zip(sf.shapes(), sf.records()):
    # Convert record to dict, handling bytes if necessary
    record_dict = {}
    for i, field in enumerate(fields):
        val = record[i]
        if isinstance(val, bytes):
            val = val.decode('utf-8', errors='replace')
        record_dict[field] = val
        
    if "Sijenggung" in str(record_dict.get("NAMOBJ", "")):
        print("Found Sijenggung!")
        
        # Convert geometry
        geom = shape.__geo_interface__
        
        # Calculate Bounding Box Center
        bbox = shape.bbox # [minX, minY, maxX, maxY]
        center_lon = (bbox[0] + bbox[2]) / 2
        center_lat = (bbox[1] + bbox[3]) / 2
        
        print(f"Center: {center_lat}, {center_lon}")
        
        feature = {
            "type": "Feature",
            "properties": record_dict,
            "geometry": geom
        }
        features.append(feature)

if not features:
    print("Sijenggung not found!")
else:
    geojson = {
        "type": "FeatureCollection",
        "features": features
    }

    with open(output_path, "w") as f:
        json.dump(geojson, f)
        
    print("Saved to", output_path)
