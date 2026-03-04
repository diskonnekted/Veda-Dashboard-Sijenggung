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
        with open('arcgis_data.json', 'w') as f:
            json.dump(data, f, indent=2)
            
        print("Data saved to arcgis_data.json")
        
        # Check if it points to a webmap
        if 'source' in data:
             print(f"Source: {data['source']}")
        
        if 'values' in data and 'webmap' in data['values']:
             webmap_id = data['values']['webmap']
             print(f"Found WebMap ID in App Config: {webmap_id}")
             return webmap_id
             
        if 'operationalLayers' in data:
            print("Found operationalLayers directly in item data (This is likely a WebMap).")
            return item_id # The item itself is the webmap or has the layers
            
        return None
        
    except Exception as e:
        print(f"Error: {e}")
        return None

def fetch_webmap_details(webmap_id):
    url = f"https://www.arcgis.com/sharing/rest/content/items/{webmap_id}/data?f=json"
    print(f"Fetching WebMap details from: {url}")
    
    try:
        response = requests.get(url)
        response.raise_for_status()
        data = response.json()
        
        with open('arcgis_webmap.json', 'w') as f:
            json.dump(data, f, indent=2)
            
        print("WebMap data saved to arcgis_webmap.json")
        
        if 'operationalLayers' in data:
            print(f"Found {len(data['operationalLayers'])} operational layers.")
            for layer in data['operationalLayers']:
                print(f"- {layer.get('title', 'Untitled')} ({layer.get('layerType')})")
                print(f"  URL: {layer.get('url')}")
        
    except Exception as e:
        print(f"Error fetching WebMap: {e}")

if __name__ == "__main__":
    app_id = "45d1a697c2284169aa40898ba0b922d2"
    print(f"Inspecting App ID: {app_id}")
    
    # First check the App ID
    result_id = inspect_arcgis_item(app_id)
    
    # If the App ID pointed to a Web Map (common in Instant Apps), fetch that Web Map
    if result_id and result_id != app_id:
        fetch_webmap_details(result_id)
    elif result_id == app_id:
        print("The App ID provided seems to be the WebMap itself or contains the layers directly.")
    else:
        print("Could not find WebMap ID from App data. The App ID might be a Web Mapping Application that references a Web Map in a different way.")

