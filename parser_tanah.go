package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// NormalizeDukuh normalizes the hamlet name based on rules
func NormalizeDukuh(dukuh, desa string) (string, string) {
	dukuh = strings.TrimSpace(strings.ToUpper(dukuh))
	desa = strings.TrimSpace(strings.ToUpper(desa))

	// Check if Village is Sijenggung
	if desa != "SIJENGGUNG" {
		return dukuh, "Penduduk Luar Desa"
	}

	// Normalization Rules for Sijenggung
	if strings.Contains(dukuh, "SIJENGGUNG") {
		return "Dukuh Sijenggung", "Dalam Desa"
	}
	if strings.Contains(dukuh, "SIDAREJA") || strings.Contains(dukuh, "SIDARJA") {
		return "Dukuh Sidareja", "Dalam Desa"
	}
	if strings.Contains(dukuh, "TEMPURAN") {
		return "Dukuh Tempuran", "Dalam Desa"
	}
	if strings.Contains(dukuh, "SUMBERSARI") {
		return "Dukuh Sumbersari", "Dalam Desa"
	}
	if strings.Contains(dukuh, "SEMURUP") {
		return "Dukuh Semurup", "Dalam Desa"
	}
	if strings.Contains(dukuh, "MERTELU") {
		return "Dukuh Mertelu", "Dalam Desa"
	}
	if strings.Contains(dukuh, "JENGGUNG") {
		return "Dukuh Jenggung", "Dalam Desa"
	}
    // Specific cases from sample
    if strings.Contains(dukuh, "KRUNGKUNGAN") || strings.Contains(dukuh, "KRUNGKRUNGAN") {
        // Assuming this maps to something or is just a name? 
        // User didn't specify mapping for Krungkungan, keep as is but marked Verified?
        // Or "Belum Terverifikasi". Let's default to "Belum Terverifikasi" if not in the main list
        return dukuh, "Belum Terverifikasi"
    }

	return dukuh, "Belum Terverifikasi"
}

func SyncTanahData(households []Household, tanahPath string) ([]Household, error) {
	f, err := excelize.OpenFile(tanahPath)
	if err != nil {
		return households, err
	}
	defer f.Close()

	rows, err := f.GetRows(f.GetSheetList()[0])
	if err != nil {
		return households, err
	}

	fmt.Println("Syncing Land Data (Tanah)...")
	
	// Headers usually at row 3 (index 2) based on sample
	// Columns:
	// NAMA PEMILIK: Index 9 (J)
	// ALAMAT DUKUH: Index 12 (M)
	// DESA WP: Index 13 (N)
	// LUAS TANAH: Index 15 (P)
	// LUAS BANGUNAN: Index 16 (Q)
	
	// Scan for correct header row if needed, but hardcoding based on `head` output
	// Row 2 (0-based) has "NAMA PEMILIK" etc.
	
	colName := 9
	colDukuh := 12
	colDesa := 13
	colLuasTanah := 15
	colLuasBangunan := 16

	matchedCount := 0
	
	for i := 3; i < len(rows); i++ {
		row := rows[i]
		if len(row) <= colLuasBangunan { continue }
		
		name := CleanName(row[colName])
		if name == "" { continue }

		dukuhRaw := row[colDukuh]
		desaRaw := row[colDesa]
		luasTanah, _ := strconv.ParseFloat(row[colLuasTanah], 64)
		luasBangunan, _ := strconv.ParseFloat(row[colLuasBangunan], 64)

		normDukuh, status := NormalizeDukuh(dukuhRaw, desaRaw)

		// Create Land Info object
		landInfo := LandInfo{
			AlamatDukuh:  normDukuh,
			DesaWP:       desaRaw,
			LuasTanah:    luasTanah,
			LuasBangunan: luasBangunan,
			Status:       status,
		}

		// Find Household Match
		matched := false
		for j := range households {
			hh := &households[j]
			
			// Match Head
			if CleanName(hh.HeadName) == name {
				matched = true
			} else {
				// Match Members
				for _, mem := range hh.Members {
					if CleanName(mem.Name) == name {
						matched = true
						break
					}
				}
			}

			if matched {
				matchedCount++
				hh.LandCount++
				hh.LandTotal += luasTanah
				hh.LandList = append(hh.LandList, landInfo)
				break // Found the owner, move to next land record
			}
		}
	}
	
	fmt.Printf("Land Data Matched: %d records assigned to residents\n", matchedCount)
	return households, nil
}
