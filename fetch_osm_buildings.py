import json
import urllib.request
import urllib.parse
import sys
import os

def get_bbox(geojson_file):
    with open(geojson_file, 'r') as f:
        data = json.load(f)
    
    # Initialize with inverse values
    min_lat, min_lon = 90.0, 180.0
    max_lat, max_lon = -90.0, -180.0
    
    def process_coord(coord):
        nonlocal min_lat, min_lon, max_lat, max_lon
        # coord should be [lon, lat]
        lon, lat = coord[0], coord[1]
        if lat < min_lat: min_lat = lat
        if lon < min_lon: min_lon = lon
        if lat > max_lat: max_lat = lat
        if lon > max_lon: max_lon = lon

    def traverse_coords(coords):
        # Check if it's a coordinate pair [lon, lat] where lon/lat are numbers
        if len(coords) >= 2 and isinstance(coords[0], (int, float)):
            process_coord(coords)
        elif isinstance(coords, list):
            for item in coords:
                traverse_coords(item)

    for feature in data['features']:
        traverse_coords(feature['geometry']['coordinates'])
    
    return min_lat, min_lon, max_lat, max_lon

def fetch_osm_buildings(min_lat, min_lon, max_lat, max_lon):
    overpass_url = "https://overpass-api.de/api/interpreter"
    # Overpass QL query
    query = f"""
    [out:json][timeout:25];
    (
      node["building"]({min_lat},{min_lon},{max_lat},{max_lon});
      way["building"]({min_lat},{min_lon},{max_lat},{max_lon});
      relation["building"]({min_lat},{min_lon},{max_lat},{max_lon});
    );
    out center;
    """
    
    print(f"Querying Overpass API with bbox: {min_lat},{min_lon},{max_lat},{max_lon}")
    data = urllib.parse.urlencode({'data': query}).encode('utf-8')
    req = urllib.request.Request(overpass_url, data=data)
    
    try:
        with urllib.request.urlopen(req) as response:
            return json.loads(response.read().decode('utf-8'))
    except urllib.error.URLError as e:
        print(f"Error fetching data: {e}")
        sys.exit(1)

def convert_to_geojson_points(osm_data):
    features = []
    if 'elements' not in osm_data:
        return {"type": "FeatureCollection", "features": []}

    for element in osm_data['elements']:
        lat, lon = 0, 0
        if element['type'] == 'node':
            lat, lon = element['lat'], element['lon']
        elif 'center' in element:
            lat, lon = element['center']['lat'], element['center']['lon']
        elif 'lat' in element and 'lon' in element:
             lat, lon = element['lat'], element['lon']
        else:
            continue # Skip if no coordinates
        
        tags = element.get('tags', {})
        feature = {
            "type": "Feature",
            "properties": {
                "REMARK": "Gedung/Bangunan",
                "osm_id": element['id'],
                "type": tags.get('building', 'yes'),
                "name": tags.get('name', ''),
                "amenity": tags.get('amenity', ''),
                "shop": tags.get('shop', ''),
                "tourism": tags.get('tourism', ''),
                "leisure": tags.get('leisure', ''),
                "office": tags.get('office', ''),
                "craft": tags.get('craft', ''),
                "religion": tags.get('religion', ''),
                "emergency": tags.get('emergency', '')
            },
            "geometry": {
                "type": "Point",
                "coordinates": [lon, lat]
            }
        }
        features.append(feature)
    
    return {
        "type": "FeatureCollection",
        "features": features
    }

def main():
    geojson_file = 'PONDOKREJO.geojson'
    if not os.path.exists(geojson_file):
        print(f"{geojson_file} not found")
        sys.exit(1)
        
    print("Calculating bounding box...")
    min_lat, min_lon, max_lat, max_lon = get_bbox(geojson_file)
    print(f"BBox: {min_lat}, {min_lon}, {max_lat}, {max_lon}")
    
    print("Fetching OSM data...")
    osm_data = fetch_osm_buildings(min_lat, min_lon, max_lat, max_lon)
    
    count = len(osm_data.get('elements', []))
    print(f"Found {count} buildings.")
    
    if count > 0:
        print("Converting to GeoJSON...")
        geojson = convert_to_geojson_points(osm_data)
        
        output_file = 'bangunan-point-pondokrejo_osm.json'
        with open(output_file, 'w') as f:
            json.dump(geojson, f, indent=2)
        print(f"Saved to {output_file}")
    else:
        print("No buildings found in OSM for this area.")

if __name__ == "__main__":
    main()
