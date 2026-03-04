import shapefile

sf = shapefile.Reader(r"D:\xampp\htdocs\petasjg\KAB. BANJARNEGARA\ADMINISTRASIDESA_AR_25K.shp")
fields = [field[0] for field in sf.fields[1:]]
print("Fields:", fields)

# Iterate and find Sijenggung
for record in sf.records():
    record_dict = dict(zip(fields, record))
    if "SIJENGGUNG" in str(record_dict.get("NAMOBJ", "")).upper():
        print("Found Sijenggung:", record_dict)
        break
