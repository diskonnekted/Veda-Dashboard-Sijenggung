package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

func CleanName(name string) string {
	name = strings.ToUpper(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, ".", " ")
	name = strings.ReplaceAll(name, ",", " ")
	
	prefixes := []string{"BAPAK ", "IBU ", "BPK ", "SDR ", "SDRI ", "H ", "HJ ", "K ", "KH ", "ALM ", "ALMH "}
	for _, p := range prefixes {
		if strings.HasPrefix(name, p) {
			name = strings.TrimPrefix(name, p)
		}
	}
	
	// Handle Bin/Binti
	if idx := strings.Index(name, " BIN "); idx != -1 {
		name = name[:idx]
	}
	if idx := strings.Index(name, " BINTI "); idx != -1 {
		name = name[:idx]
	}
	
	// Remove extra spaces
	fields := strings.Fields(name)
	return strings.Join(fields, " ")
}

func SyncPKHData(households []Household, pkhPath string) ([]Household, error) {
	f, err := excelize.OpenFile(pkhPath)
	if err == nil {
		defer f.Close()
		rows, err := f.GetRows(f.GetSheetList()[0])
		if err == nil {
			fmt.Println("Syncing PKH Data...")
			matchedCount := 0
			for i := 1; i < len(rows); i++ {
				row := rows[i]
				if len(row) < 2 {
					continue
				}

				name := CleanName(row[1])
				var lat, lng float64
				hasCoord := false

				if len(row) > 5 {
					parts := strings.Split(row[5], ",")
					if len(parts) == 2 {
						l, e1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
						ln, e2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
						if e1 == nil && e2 == nil {
							lat = l
							lng = ln
							hasCoord = true
						}
					}
				}

				for j := range households {
					hh := &households[j]
					matched := false
					if CleanName(hh.HeadName) == name {
						matched = true
					}
					if !matched {
						for _, mem := range hh.Members {
							if CleanName(mem.Name) == name {
								matched = true
								break
							}
						}
					}

					if matched {
						matchedCount++
						hh.IsPKH = "1"
						if hasCoord {
							// Only use PKH coordinates if the main data has no coordinates (0,0)
							if hh.Latitude == 0 && hh.Longitude == 0 {
								hh.Latitude = lat
								hh.Longitude = lng
							}
						}
						break
					}
				}
			}
			fmt.Printf("PKH Data Matched: %d records\n", matchedCount)
		}
	} else {
		fmt.Printf("Warning: PKH file not found/readable: %v\n", err)
	}

	return households, nil
}

func SyncBPNTData(households []Household, bpntPath string) ([]Household, error) {
	f, err := excelize.OpenFile(bpntPath)
	if err != nil {
		return households, err
	}
	defer f.Close()

	rows, err := f.GetRows(f.GetSheetList()[0])
	if err != nil {
		return households, err
	}

	fmt.Println("Syncing BPNT Data...")
	matchedCount := 0
	
	// Headers: NO, NAMA, ALAMAT, DESIL, KETERANGAN
	// Indices: 0, 1, 2, 3, 4
	
	for i := 1; i < len(rows); i++ {
		row := rows[i]
		if len(row) < 2 { continue }
		
		name := CleanName(row[1])
		desil := ""
		if len(row) > 3 {
			desil = strings.TrimSpace(row[3])
		}

		matched := false
		// Try Exact Match first
		for j := range households {
			hh := &households[j]
			if CleanName(hh.HeadName) == name {
				matched = true
				matchedCount++
				hh.IsBPNT = "1"
				if desil != "" { hh.WelfareLevel = desil }
				break
			}
		}
		
		if !matched {
			// Try fuzzy match (contains)
			for j := range households {
				hh := &households[j]
				hName := CleanName(hh.HeadName)
				if strings.Contains(hName, name) || strings.Contains(name, hName) {
					matched = true
					matchedCount++
					hh.IsBPNT = "1"
					if desil != "" { hh.WelfareLevel = desil }
					break
				}
			}
		}
	}
	fmt.Printf("BPNT Data Matched: %d records\n", matchedCount)
	
	return households, nil
}

func SyncPBIData(households []Household, pbiPath string) ([]Household, error) {
	f, err := excelize.OpenFile(pbiPath)
	if err != nil {
		// Log error but continue
		fmt.Printf("Warning: Could not open PBI data: %v\n", err)
		return households, nil
	}
	defer f.Close()

	rows, err := f.GetRows(f.GetSheetList()[0])
	if err != nil {
		return households, err
	}

	fmt.Println("Syncing PBI BPJS Data...")
	matchedCount := 0
	
	// Headers: No, Nama Kepala Keluarga, Alamat, Desil Kesejahteraan, Status Keaktifan
	// Indices: 0, 1, 2, 3, 4
	
	for i := 1; i < len(rows); i++ {
		row := rows[i]
		if len(row) < 2 { continue }
		
		name := CleanName(row[1])
		desil := ""
		if len(row) > 3 {
			desil = strings.TrimSpace(row[3])
		}
		
		status := ""
		if len(row) > 4 {
			status = strings.TrimSpace(strings.ToUpper(row[4]))
		}
		
		if status != "AKTIF" {
			// Maybe skip?
		}

		matched := false
		// Try Exact Match
		for j := range households {
			hh := &households[j]
			if CleanName(hh.HeadName) == name {
				matched = true
				matchedCount++
				hh.IsPBI = "1"
				if desil != "" { hh.WelfareLevel = desil }
				break
			}
		}
		
		if !matched {
			// Try fuzzy match
			for j := range households {
				hh := &households[j]
				hName := CleanName(hh.HeadName)
				if strings.Contains(hName, name) || strings.Contains(name, hName) {
					matched = true
					matchedCount++
					hh.IsPBI = "1"
					if desil != "" { hh.WelfareLevel = desil }
					break
				}
			}
		}
	}
	fmt.Printf("PBI BPJS Data Matched: %d records\n", matchedCount)
	
	return households, nil
}
