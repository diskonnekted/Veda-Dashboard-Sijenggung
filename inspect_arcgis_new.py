import requests
import json
import sys

def inspect_arcgis_item(item_id):
    # Try to get data directly (works for Web Maps and some Apps)
    url = f"https://www.arcgis.com/sharing/rest/content/items/{item_id}/data?f=json"
    print(f"Fetching data from: {url}")
    
    try:
        response = requests.get(url)
        response.raise_for_status()
        data = response.json()
        
        # Save for inspection
        with open('arcgis_webmap_new.json', 'w') as f:
            json.dump(data, f, indent=2)
            
        print("Data saved to arcgis_webmap_new.json")
        
        if 'operationalLayers' in data:
            print(f"Found {len(data['operationalLayers'])} operational layers.")
            for layer in data['operationalLayers']:
                print(f"- {layer.get('title', 'Untitled')} ({layer.get('layerType')})")
                print(f"  URL: {layer.get('url')}")
        
    except Exception as e:
        print(f"Error: {e}")
        return None

if __name__ == "__main__":
    # Extracted from URL: https://www.arcgis.com/apps/mapviewer/index.html?webmap=5b04e42d5b184427ae42410c29f3171e
    webmap_id = "5b04e42d5b184427ae42410c29f3171e"
    print(f"Inspecting WebMap ID: {webmap_id}")
    
    inspect_arcgis_item(webmap_id)
