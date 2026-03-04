package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

type Resident struct {
	Name        string   `json:"name"`
	Nik         string   `json:"nik"`
	Gender      string   `json:"gender"` // ID Kelamin
	AidList     []string `json:"aid_list"`
	KerjaDetail string   `json:"kerja_detail"` // Kerja Detail
	UshDetail   string   `json:"ush_detail"`   // Ush Detail
	Age         string   `json:"age"`          // Usia
	Education   string   `json:"education"`    // ID Ijazah
	Income      string   `json:"income"`       // Income
	Pregnant    string   `json:"pregnant"`     // Hamil
	Disability  string   `json:"disability"`   // ID Difable
	Marital     string   `json:"marital"`      // Status Kawin
	Relation    string   `json:"relation"`     // Hubungan KK
	Religion    string   `json:"religion"`     // Agama
}

type Household struct {
	NoKK         string     `json:"no_kk"`
	HeadName     string     `json:"head_name"`
	Address      string     `json:"address"`
	Dusun        string     `json:"dusun"`
	Latitude     float64    `json:"latitude"`
	Longitude    float64    `json:"longitude"`
	WelfareLevel string     `json:"welfare_level"` // ID Desil
	Members      []Resident `json:"members"`
	PkhThn       string     `json:"pkh_thn"`      // Pkh Thn
	BpntThn      string     `json:"bpnt_thn"`     // Bpnt Thn
	LantaiLuas   string     `json:"lantai_luas"`  // Lantai Luas
	Keterangan   string     `json:"keterangan"`   // Keterangan
	Expenditure  string     `json:"expenditure"`  // Overall Sum
	FloorType    string     `json:"floor_type"`   // ID Lantai
	WallType     string     `json:"wall_type"`    // ID Dinding
	RoofType     string     `json:"roof_type"`    // ID Atap
	WaterSource  string     `json:"water_source"` // ID Airminum
	Sanitation   string     `json:"sanitation"`   // ID Fasbab
	IsPKH        string     `json:"is_pkh"`       // Flag from PKH Data
	IsBPNT       string     `json:"is_bpnt"`      // Flag from BPNT Data
	IsPBI        string     `json:"is_pbi"`       // Flag from PBI BPJS Data

	// Land Data
	LandCount int        `json:"land_count"` // Number of land parcels
	LandTotal float64    `json:"land_total"` // Total Land Area
	LandList  []LandInfo `json:"land_list"`  // Details of land
}

type LandInfo struct {
	AlamatDukuh  string  `json:"alamat_dukuh"`
	DesaWP       string  `json:"desa_wp"`
	LuasTanah    float64 `json:"luas_tanah"`
	LuasBangunan float64 `json:"luas_bangunan"`
	Status       string  `json:"status"` // "Dalam Desa", "Luar Desa", "Belum Terverifikasi"
}

// GeoJSON Structures
type GeoJSON struct {
	Type     string    `json:"type"`
	Features []Feature `json:"features"`
}

type Feature struct {
	Type     string   `json:"type"`
	Geometry Geometry `json:"geometry"`
}

type Geometry struct {
	Type        string          `json:"type"`
	Coordinates [][][][]float64 `json:"coordinates"` // MultiPolygon
}

func LoadBoundary(filename string) (*GeoJSON, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var geo GeoJSON
	if err := json.Unmarshal(data, &geo); err != nil {
		return nil, err
	}
	return &geo, nil
}

func IsPointInPolygon(lat, lng float64, geo *GeoJSON) bool {
	// Simple Ray Casting
	// PONDOKREJO.geojson is MultiPolygon
	for _, feature := range geo.Features {
		if feature.Geometry.Type == "MultiPolygon" {
			for _, polygon := range feature.Geometry.Coordinates {
				// Outer ring is usually the first one
				ring := polygon[0]
				if isPointInRing(lat, lng, ring) {
					return true
				}
			}
		}
	}
	return false
}

func isPointInRing(lat, lng float64, ring [][]float64) bool {
	inside := false
	j := len(ring) - 1
	for i := 0; i < len(ring); i++ {
		xi, yi := ring[i][0], ring[i][1] // GeoJSON is [lng, lat]
		xj, yj := ring[j][0], ring[j][1]

		intersect := ((yi > lat) != (yj > lat)) &&
			(lng < (xj-xi)*(lat-yi)/(yj-yi)+xi)
		if intersect {
			inside = !inside
		}
		j = i
	}
	return inside
}

func ExtractDusun(address string) string {
	// Normalize
	s := strings.ToUpper(address)

	// Remove common prefixes/suffixes for hamlet
	// PADUKUHAN, DUSUN, DSN, DK
	rePrefix := regexp.MustCompile(`\b(PADUKUHAN|DUSUN|DSN|DK)\b`)
	s = rePrefix.ReplaceAllString(s, " ")

	// Remove RT/RW and numbers
	// Regex for RT/RW followed by optional numbers/slash
	re := regexp.MustCompile(`(RT|RW)\s*[\d\/\.\-]+`)
	s = re.ReplaceAllString(s, "")

	// Remove Roman Numerals (I, II, III, IV, V) - common in hamlet sections
	reRoman := regexp.MustCompile(`\b(I|II|III|IV|V|VI|VII|VIII|IX|X)\b`)
	s = reRoman.ReplaceAllString(s, "")

	// Remove non-alphabetic chars except spaces
	reNonAlpha := regexp.MustCompile(`[^A-Z\s]`)
	s = reNonAlpha.ReplaceAllString(s, " ")

	// Trim spaces and extra whitespace
	s = strings.TrimSpace(s)
	reSpace := regexp.MustCompile(`\s+`)
	s = reSpace.ReplaceAllString(s, " ")

	// Check for "DUKU" or "DUKUH" specifically as the only content or explicit word
	if s == "DUKU" || s == "DUKUH" {
		return "Dukuh" // Standardize to "Dukuh" as per user request
	}

	// Convert to Title Case for display
	s = strings.Title(strings.ToLower(s))

	// Standardization / Correction Map
	// Dusun 1: Tempuran, Sumbersari
	// Dusun 2: Semurup, Sidarja (Sidareja)
	// Dusun 3: Jenggung (Sijenggung), Mertelu

	// Exact matches first
	switch s {
	case "Tempuran":
		return "Dusun 1 (Tempuran)"
	case "Sumbersari", "Sumber Sari":
		return "Dusun 1 (Sumbersari)"
	case "Semurup":
		return "Dusun 2 (Semurup)"
	case "Sidarja", "Sidareja":
		return "Dusun 2 (Sidarja)"
	case "Jenggung", "Sijenggung":
		return "Dusun 3 (Jenggung)"
	case "Mertelu":
		return "Dusun 3 (Mertelu)"
	default:
		// Fuzzy matching
		if strings.Contains(s, "Tempur") {
			return "Dusun 1 (Tempuran)"
		}
		if strings.Contains(s, "Sumber") {
			return "Dusun 1 (Sumbersari)"
		}
		if strings.Contains(s, "Semurup") {
			return "Dusun 2 (Semurup)"
		}
		if strings.Contains(s, "Sidar") || strings.Contains(s, "Sidare") {
			return "Dusun 2 (Sidarja)"
		}
		if strings.Contains(s, "Jenggung") || strings.Contains(s, "Sijenggung") {
			return "Dusun 3 (Jenggung)"
		}
		if strings.Contains(s, "Mertelu") {
			return "Dusun 3 (Mertelu)"
		}
	}

	return "Dusun Lainnya (" + s + ")"
}

func ParseExcel(filename string) ([]Household, error) {
	f, err := excelize.OpenFile(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("no sheets found")
	}

	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, err
	}

	// Build header index map (dynamic)
	headerRowIdx := 0
	headers := []string{}
	for i := 0; i < len(rows) && i < 10; i++ {
		row := rows[i]
		joined := strings.Join(row, ",")
		if strings.Contains(strings.ToLower(joined), "no_kk") && strings.Contains(strings.ToLower(joined), "nik") {
			headerRowIdx = i
			headers = row
			break
		}
	}
	if len(headers) == 0 && len(rows) > 2 {
		headers = rows[2]
		headerRowIdx = 2
	}

	indexOf := func(keys ...string) int {
		for idx, h := range headers {
			hh := strings.TrimSpace(strings.ToLower(h))
			for _, k := range keys {
				if hh == strings.ToLower(k) {
					return idx
				}
			}
		}
		return -1
	}

	// Try map by header names; fallback to legacy indices if not found
	ColNoKK := indexOf("no_kk", "nokk", "no kk")
	if ColNoKK < 0 {
		ColNoKK = 2
	}
	ColHeadName := indexOf("nama", "name")
	if ColHeadName < 0 {
		ColHeadName = 14
	}
	ColAddress := indexOf("alamat", "address")
	if ColAddress < 0 {
		ColAddress = 12
	}
	ColDusun := indexOf("dusun", "padukuhan", "hamlet")
	if ColDusun < 0 {
		ColDusun = 9
	}
	ColNik := indexOf("nik")
	if ColNik < 0 {
		ColNik = 231
	}
	ColName := ColHeadName
	ColCoordinate := indexOf("koordinat", "coordinate", "coord", "latlon")
	// If not found, check if "lat" and "lng" columns exist
	ColLat := indexOf("lat", "latitude")
	ColLng := indexOf("lng", "long", "longitude")

	if ColCoordinate < 0 && ColLat < 0 {
		ColCoordinate = 44
	}
	ColIDDesil := indexOf("id_desil", "desil", "welfare_level")
	// Removed default fallback to 38 (suku)
	ColGender := indexOf("sex", "jk", "gender")
	if ColGender < 0 {
		ColGender = 237
	}
	ColPregnant := indexOf("hamil", "pregnant")
	if ColPregnant < 0 {
		ColPregnant = 242
	}
	ColDisability := indexOf("cacat_id", "disability", "difabel", "difable")
	if ColDisability < 0 {
		ColDisability = 249
	}
	ColEducation := indexOf("pendidikan_kk_id", "pendidikan", "education")
	if ColEducation < 0 {
		ColEducation = 247
	}
	ColPekerjaan := indexOf("pekerjaan_id", "pekerjaan", "job")
	if ColPekerjaan < 0 {
		ColPekerjaan = 250 // Guessing or need to check
	}
	ColKawin := indexOf("status_kawin", "kawin", "marital")
	if ColKawin < 0 {
		ColKawin = 240 // Guessing
	}
	ColHubungan := indexOf("kk_level", "shdk", "hubungan")
	if ColHubungan < 0 {
		ColHubungan = 238 // Guessing
	}
	ColAgama := indexOf("agama_id", "agama", "religion")
	if ColAgama < 0 {
		ColAgama = 239 // Guessing
	}

	ColAge := indexOf("umur", "usia", "age")
	if ColAge < 0 {
		ColAge = 260
	}
	ColTanggalLahir := indexOf("tanggallahir", "tanggal_lahir", "dob")
	if ColTanggalLahir < 0 {
		ColTanggalLahir = 10 // Based on known structure
	}

	ColMemIncome := indexOf("pendapatan", "income")
	if ColMemIncome < 0 {
		ColMemIncome = 256
	}
	// Household-level extra fields (fallback to legacy)
	ColPkhThn, ColBpntThn := 110, 107
	ColLantaiLuas, ColKeterangan := 55, 19
	ColExpenditure, ColFloorType, ColWallType, ColRoofType, ColWaterSource, ColSanitation := 183, 56, 57, 58, 59, 71
	// Optional detailed columns (may not exist in new dataset)
	ColUshDetail, ColKerjaDetail := -1, -1
	// Aid flags
	ColISBpnt, ColISPkh, ColISBlt, ColISBanpem := 105, 108, 111, 117
	ColSosKur, ColSosMikro, ColSosPip, ColSosJamket := 291, 292, 293, 294

	// Load Boundary
	boundary, err := LoadBoundary("sijenggung.geojson")
	if err != nil {
		// Just log warning, don't fail, maybe just skip correction
		fmt.Println("Warning: Could not load boundary for correction:", err)
	}

	householdsMap := make(map[string]*Household)

	// Data rows start after header
	for i := headerRowIdx + 1; i < len(rows); i++ {
		row := rows[i]

		// Safety check for row length
		if len(row) <= ColNoKK {
			continue
		}

		noKK := row[ColNoKK]
		if noKK == "" {
			continue
		}

		// Parse Coordinates
		var lat, lng float64
		hasCoord := false

		// Try separate Lat/Lng columns first
		if ColLat >= 0 && ColLng >= 0 && len(row) > ColLng {
			lStr := strings.TrimSpace(row[ColLat])
			nStr := strings.TrimSpace(row[ColLng])
			if lStr != "" && nStr != "" {
				l, err1 := strconv.ParseFloat(lStr, 64)
				n, err2 := strconv.ParseFloat(nStr, 64)
				if err1 == nil && err2 == nil {
					lat = l
					lng = n
					hasCoord = true
				}
			}
		}

		if !hasCoord {
			coordStr := ""
			if ColCoordinate >= 0 && len(row) > ColCoordinate {
				coordStr = row[ColCoordinate]
			}

			if coordStr != "" {
				parts := strings.Split(coordStr, ",")
				if len(parts) == 2 {
					var err1, err2 error
					lat, err1 = strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
					lng, err2 = strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
					if err1 == nil && err2 == nil {
						hasCoord = true
					}
				}
			}
		}

		// If no coordinates, we set to 0,0.
		// User requested fallback from PKH/BPNT or checkbox for no-coords.
		// We will allow 0,0 here, and Sync functions will try to update it later.
		if !hasCoord {
			// Try to find if household already exists and has coordinates
			if h, exists := householdsMap[noKK]; exists && h.Latitude != 0 {
				lat = h.Latitude
				lng = h.Longitude
			} else {
				// Initialize with 0,0 so it's included in the list but not mapped by default
				lat = 0
				lng = 0
			}
		}

		// Get or Create Household
		hh, exists := householdsMap[noKK]
		if !exists {
			headName := ""
			if len(row) > ColHeadName {
				headName = row[ColHeadName]
			}
			address := ""
			if len(row) > ColAddress {
				address = row[ColAddress]
			}
			welfare := ""
			if len(row) > ColIDDesil && ColIDDesil >= 0 {
				welfare = row[ColIDDesil]
			}

			// Map Dusun Code to Name
			dusunVal := ""
			if len(row) > ColDusun && ColDusun >= 0 {
				dusunVal = strings.TrimSpace(row[ColDusun])
			}

			dusunName := ""
			switch dusunVal {
			case "1":
				dusunName = "Dusun 1 (Tempuran & Sumbersari)"
			case "2":
				dusunName = "Dusun 2 (Semurup & Sidarja)"
			case "3":
				dusunName = "Dusun 3 (Jenggung & Mertelu)"
			default:
				// Fallback to extraction from address
				dusunName = ExtractDusun(address)
			}

			// Get Household specific fields (Assuming these are consistent for the head or we take the first non-empty)
			pkhThn := ""
			if len(row) > ColPkhThn {
				pkhThn = row[ColPkhThn]
			}
			bpntThn := ""
			if len(row) > ColBpntThn {
				bpntThn = row[ColBpntThn]
			}
			lantaiLuas := ""
			if len(row) > ColLantaiLuas {
				lantaiLuas = row[ColLantaiLuas]
			}
			keterangan := ""
			if len(row) > ColKeterangan {
				keterangan = row[ColKeterangan]
			}

			// Helper for extra fields
			getVal := func(idx int) string {
				if len(row) > idx {
					return row[idx]
				}
				return ""
			}

			hh = &Household{
				NoKK:         noKK,
				HeadName:     headName,
				Address:      address,
				Dusun:        dusunName, // Use Extracted Name
				Latitude:     lat,
				Longitude:    lng,
				WelfareLevel: welfare,
				PkhThn:       pkhThn,
				BpntThn:      bpntThn,
				LantaiLuas:   lantaiLuas,
				Keterangan:   keterangan,
				Expenditure:  getVal(ColExpenditure),
				FloorType:    getVal(ColFloorType),
				WallType:     getVal(ColWallType),
				RoofType:     getVal(ColRoofType),
				WaterSource:  getVal(ColWaterSource),
				Sanitation:   getVal(ColSanitation),
				Members:      []Resident{},
			}
			householdsMap[noKK] = hh
		} else {
			// Update coordinates if they were missing and now found
			if hh.Latitude == 0 && hh.Longitude == 0 && hasCoord {
				hh.Latitude = lat
				hh.Longitude = lng
			}

			// Update Household fields if empty and we found data now
			if hh.PkhThn == "" && len(row) > ColPkhThn {
				hh.PkhThn = row[ColPkhThn]
			}
			if hh.BpntThn == "" && len(row) > ColBpntThn {
				hh.BpntThn = row[ColBpntThn]
			}
			if hh.LantaiLuas == "" && len(row) > ColLantaiLuas {
				hh.LantaiLuas = row[ColLantaiLuas]
			}
			if hh.Keterangan == "" && len(row) > ColKeterangan {
				hh.Keterangan = row[ColKeterangan]
			}
		}

		// Add Member
		name := ""
		if len(row) > ColName && ColName >= 0 {
			name = row[ColName]
		}
		nik := ""
		if len(row) > ColNik && ColNik >= 0 {
			nik = row[ColNik]
		}

		ushDetail := ""
		if ColUshDetail >= 0 && len(row) > ColUshDetail {
			ushDetail = row[ColUshDetail]
		}
		kerjaDetail := ""
		if ColKerjaDetail >= 0 && len(row) > ColKerjaDetail {
			kerjaDetail = row[ColKerjaDetail]
		}

		// Collect Aid Info
		var aids []string

		checkAid := func(colIdx int, aidName string) {
			if len(row) > colIdx {
				val := strings.TrimSpace(row[colIdx])
				// Assuming '1' means yes, or specific code.
				// The user data shows '1' or '2' or empty.
				// Let's assume '1' is Yes.
				if val == "1" {
					aids = append(aids, aidName)
				}
			}
		}

		checkAid(ColISBpnt, "BPNT")
		checkAid(ColISPkh, "PKH")
		checkAid(ColISBlt, "BLT")
		checkAid(ColISBanpem, "Banpres")
		checkAid(ColSosKur, "KUR")
		checkAid(ColSosMikro, "UMKM Mikro")
		checkAid(ColSosPip, "PIP")
		checkAid(ColSosJamket, "Jamkes")

		// Map selected codes to human-readable
		mapGender := func(v string) string {
			v = strings.TrimSpace(v)
			switch v {
			case "1", "L", "l":
				return "Laki-laki"
			case "2", "P", "p":
				return "Perempuan"
			default:
				return v
			}
		}
		mapYesNo := func(v string) string {
			if strings.TrimSpace(v) == "1" {
				return "Ya"
			}
			return "Tidak"
		}

		mapEducation := func(v string) string {
			v = strings.TrimSpace(v)
			switch v {
			case "1":
				return "Tidak/Belum Sekolah"
			case "2":
				return "Belum Tamat SD/Sederajat"
			case "3":
				return "Tamat SD/Sederajat"
			case "4":
				return "SLTP/Sederajat"
			case "5":
				return "SLTA/Sederajat"
			case "6":
				return "Diploma I/II"
			case "7":
				return "Akademi/Diploma III/S. Muda"
			case "8":
				return "Diploma IV/Strata I"
			case "9":
				return "Strata II"
			case "10":
				return "Strata III"
			default:
				return v
			}
		}

		mapJob := func(v string) string {
			v = strings.TrimSpace(v)
			// Common Codes based on standard
			switch v {
			case "1":
				return "Belum/Tidak Bekerja"
			case "2":
				return "Mengurus Rumah Tangga"
			case "3":
				return "Pelajar/Mahasiswa"
			case "4":
				return "Pensiunan"
			case "5":
				return "Pegawai Negeri Sipil"
			case "6":
				return "Tentara Nasional Indonesia"
			case "7":
				return "Kepolisian RI"
			case "8":
				return "Perdagangan"
			case "9":
				return "Petani/Pekebun"
			case "10":
				return "Peternak"
			case "11":
				return "Nelayan/Perikanan"
			case "12":
				return "Industri"
			case "13":
				return "Konstruksi"
			case "14":
				return "Transportasi"
			case "15":
				return "Karyawan Swasta"
			case "16":
				return "Karyawan BUMN"
			case "17":
				return "Karyawan BUMD"
			case "18":
				return "Karyawan Honorer"
			case "19":
				return "Buruh Harian Lepas"
			case "20":
				return "Buruh Tani/Perkebunan"
			case "88":
				return "Wiraswasta"
			case "89":
				return "Lainnya"
			default:
				return v
			}
		}

		mapMarital := func(v string) string {
			v = strings.TrimSpace(v)
			switch v {
			case "1":
				return "Belum Kawin"
			case "2":
				return "Kawin"
			case "3":
				return "Cerai Hidup"
			case "4":
				return "Cerai Mati"
			default:
				return v
			}
		}

		mapRelation := func(v string) string {
			v = strings.TrimSpace(v)
			switch v {
			case "1":
				return "Kepala Keluarga"
			case "2":
				return "Suami"
			case "3":
				return "Istri"
			case "4":
				return "Anak"
			case "5":
				return "Menantu"
			case "6":
				return "Cucu"
			case "7":
				return "Orangtua"
			case "8":
				return "Mertua"
			case "9":
				return "Famili Lain"
			case "10":
				return "Pembantu"
			case "11":
				return "Lainnya"
			default:
				return v
			}
		}

		mapReligion := func(v string) string {
			v = strings.TrimSpace(v)
			switch v {
			case "1":
				return "Islam"
			case "2":
				return "Kristen"
			case "3":
				return "Katholik"
			case "4":
				return "Hindu"
			case "5":
				return "Budha"
			case "6":
				return "Konghucu"
			case "7":
				return "Kepercayaan"
			default:
				return v
			}
		}

		// Use the maps
		gender := ""
		if len(row) > ColGender && ColGender >= 0 {
			gender = mapGender(row[ColGender])
		}
		preg := ""
		if len(row) > ColPregnant && ColPregnant >= 0 {
			preg = mapYesNo(row[ColPregnant])
		}
		disab := ""
		if len(row) > ColDisability && ColDisability >= 0 {
			disab = row[ColDisability]
		}
		edu := ""
		if len(row) > ColEducation && ColEducation >= 0 {
			edu = mapEducation(row[ColEducation])
		}
		age := ""
		if len(row) > ColAge && ColAge >= 0 {
			age = row[ColAge]
		}
		// Fallback: Calculate from Date of Birth
		if (age == "" || age == "0") && len(row) > ColTanggalLahir && ColTanggalLahir >= 0 {
			dobStr := strings.TrimSpace(row[ColTanggalLahir])
			// Regex for 4 digits (Year)
			reYear := regexp.MustCompile(`\d{4}`)
			yearStr := reYear.FindString(dobStr)
			if yearStr != "" {
				birthYear, err := strconv.Atoi(yearStr)
				if err == nil {
					currentYear := time.Now().Year()
					ageInt := currentYear - birthYear
					if ageInt >= 0 {
						age = strconv.Itoa(ageInt)
					}
				}
			}
		}
		memIncome := ""
		if len(row) > ColMemIncome && ColMemIncome >= 0 {
			memIncome = row[ColMemIncome]
		}

		marital := ""
		if len(row) > ColKawin && ColKawin >= 0 {
			marital = mapMarital(row[ColKawin])
		}

		relation := ""
		if len(row) > ColHubungan && ColHubungan >= 0 {
			relation = mapRelation(row[ColHubungan])
		}

		religion := ""
		if len(row) > ColAgama && ColAgama >= 0 {
			religion = mapReligion(row[ColAgama])
		}

		// Map Job to KerjaDetail if empty or numeric
		jobVal := ""
		if len(row) > ColPekerjaan && ColPekerjaan >= 0 {
			jobVal = mapJob(row[ColPekerjaan])
		}
		if kerjaDetail == "" || (len(kerjaDetail) < 3 && strings.ContainsAny(kerjaDetail, "0123456789")) {
			kerjaDetail = jobVal
		}

		member := Resident{
			Name:        name,
			Nik:         nik,
			Gender:      gender,
			AidList:     aids,
			UshDetail:   ushDetail,
			KerjaDetail: kerjaDetail,
			Age:         age,
			Education:   edu,
			Income:      memIncome,
			Pregnant:    preg,
			Disability:  disab,
			Marital:     marital,
			Relation:    relation,
			Religion:    religion,
		}

		hh.Members = append(hh.Members, member)
	}

	// Convert map to slice
	var result []Household

	// Pre-process for Coordinate Correction
	if boundary != nil {
		// Group valid households by Dusun
		dusunCentroids := make(map[string]struct {
			sumLat, sumLng float64
			count          int
		})

		for _, hh := range householdsMap {
			if hh.Latitude != 0 && hh.Longitude != 0 && hh.Dusun != "" {
				if IsPointInPolygon(hh.Latitude, hh.Longitude, boundary) {
					// Valid Point
					s := dusunCentroids[hh.Dusun]
					s.sumLat += hh.Latitude
					s.sumLng += hh.Longitude
					s.count++
					dusunCentroids[hh.Dusun] = s
				}
			}
		}

		// Apply Correction
		for _, hh := range householdsMap {
			if hh.Latitude != 0 && hh.Longitude != 0 {
				if !IsPointInPolygon(hh.Latitude, hh.Longitude, boundary) {
					// Outside!
					// Find neighbors in same Dusun
					s, ok := dusunCentroids[hh.Dusun]
					if ok && s.count > 0 {
						// Move to Centroid
						newLat := s.sumLat / float64(s.count)
						newLng := s.sumLng / float64(s.count)

						hh.Latitude = newLat
						hh.Longitude = newLng

						// Append note
						if hh.Keterangan != "" {
							hh.Keterangan += "; "
						}
						hh.Keterangan += "Koordinat perlu revisi (Digeser otomatis)"
					} else {
						// If no valid neighbors in same Dusun (unlikely if Dusun code is valid),
						// we might want to just mark it.
						if hh.Keterangan != "" {
							hh.Keterangan += "; "
						}
						hh.Keterangan += "Koordinat diluar wilayah (Perlu revisi)"
					}
				}
			}
		}
	}

	for _, hh := range householdsMap {
		// Include all households, even with 0,0 coords.
		// Filter logic in UI will handle display.
		// Sync logic will try to fill 0,0 coords.
		result = append(result, *hh)
	}

	return result, nil
}
