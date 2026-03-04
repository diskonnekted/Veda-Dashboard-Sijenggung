import json
import math
import os

def web_mercator_to_latlon(x, y):
    if x is None or y is None:
        return 0, 0
    lon = (x / 20037508.34) * 180
    lat_rad = (y / 20037508.34) * 180 * math.pi / 180
    lat = 180 / math.pi * (2 * math.atan(math.exp(lat_rad)) - math.pi / 2)
    return lon, lat

def convert_esri_to_geojson(features, geometry_type):
    geojson_features = []
    
    for ef in features:
        props = ef.get('attributes', {})
        geom = ef.get('geometry', {})
        
        geojson_geom = None
        
        if geometry_type == 'esriGeometryPoint':
            x = geom.get('x')
            y = geom.get('y')
            if x and y:
                lon, lat = web_mercator_to_latlon(x, y)
                geojson_geom = {
                    "type": "Point",
                    "coordinates": [lon, lat]
                }
        elif geometry_type == 'esriGeometryPolyline':
            paths = []
            for path in geom.get('paths', []):
                new_path = []
                for pt in path:
                    lon, lat = web_mercator_to_latlon(pt[0], pt[1])
                    new_path.append([lon, lat])
                paths.append(new_path)
            
            if paths:
                if len(paths) == 1:
                    geojson_geom = {
                        "type": "LineString",
                        "coordinates": paths[0]
                    }
                else:
                    geojson_geom = {
                        "type": "MultiLineString",
                        "coordinates": paths
                    }
        elif geometry_type == 'esriGeometryPolygon':
            rings = []
            for ring in geom.get('rings', []):
                new_ring = []
                for pt in ring:
                    lon, lat = web_mercator_to_latlon(pt[0], pt[1])
                    new_ring.append([lon, lat])
                rings.append(new_ring)
            
            if rings:
                geojson_geom = {
                    "type": "Polygon",
                    "coordinates": rings
                }

        if geojson_geom:
            geojson_features.append({
                "type": "Feature",
                "properties": props,
                "geometry": geojson_geom
            })
            
    return {
        "type": "FeatureCollection",
        "features": geojson_features
    }

def main():
    if not os.path.exists('arcgis_webmap.json'):
        print("arcgis_webmap.json not found!")
        return

    with open('arcgis_webmap.json', 'r') as f:
        data = json.load(f)
        
    layers_to_extract = {
        "Titik Lokasi TPS_VALID": "tps_locations.geojson",
        "Titik Lojasi BS_VALID": "waste_banks.geojson",
        "Rute Terbaru_VALID": "waste_routes.geojson",
        "Batas Admin Kecamatan": "district_boundary.geojson"
    }
    
    for op_layer in data.get('operationalLayers', []):
        title = op_layer.get('title')
        if title in layers_to_extract:
            print(f"Processing layer: {title}")
            
            # Check for featureCollection
            fc = op_layer.get('featureCollection')
            if fc and 'layers' in fc:
                for sublayer in fc['layers']:
                    layer_def = sublayer.get('layerDefinition', {})
                    geom_type = layer_def.get('geometryType')
                    features = sublayer.get('featureSet', {}).get('features', [])
                    
                    print(f"  - Found {len(features)} features. Type: {geom_type}")
                    
                    geojson = convert_esri_to_geojson(features, geom_type)
                    
                    filename = layers_to_extract[title]
                    with open(filename, 'w') as out:
                        json.dump(geojson, out, indent=2)
                    print(f"  - Saved to {filename}")

if __name__ == "__main__":
    main()
