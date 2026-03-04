package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

type SaveLayerRequest struct {
	Filename string      `json:"filename"`
	Data     interface{} `json:"data"` // GeoJSON Object
}

func main() {
	// Parse CLI flags
	genStatic := flag.Bool("gen", false, "Generate static JSON files for deployment")
	clipGeoJSON := flag.String("clip-geojson", "", "Clip input GeoJSON to Pondokrejo boundary and write output")
	clipOut := flag.String("clip-out", "", "Output GeoJSON file path for -clip-geojson")
	clipBoundary := flag.String("clip-boundary", "DEFAULT_VIEW", "Boundary GeoJSON file for -clip-geojson")
	flag.Parse()

	if *clipGeoJSON != "" {
		if err := ClipGeoJSONToPondokrejoBoundary(*clipGeoJSON, *clipBoundary, *clipOut); err != nil {
			log.Fatalf("Error clipping geojson: %v", err)
		}
		fmt.Println("Clipped GeoJSON written successfully")
		return
	}

	// Parse Excel Data
	households, err := ParseExcel("../data/penduduk_04_03_2026.xlsx")
	if err != nil {
		log.Printf("Warning: gagal memparsing Excel: %v", err)
		households = []Household{}
	}

	// Sync PKH Data
	households, err = SyncPKHData(households, "../data/pkh-sijenggung.xlsx")
	if err != nil {
		log.Printf("Warning: gagal sinkronisasi PKH: %v", err)
	}

	// Sync BPNT Data
	households, err = SyncBPNTData(households, "../data/bpnt-sijenggung.xlsx")
	if err != nil {
		log.Printf("Warning: gagal sinkronisasi BPNT: %v", err)
	}

	// Sync PBI BPJS Data
	households, err = SyncPBIData(households, "../data/pbi-bpjs-sijenggung.xlsx")
	if err != nil {
		log.Printf("Warning: gagal sinkronisasi PBI BPJS: %v", err)
	}

	// Sync Land Data (Tanah)
	households, err = SyncTanahData(households, "../data/tanah-sijenggung.xlsx")
	if err != nil {
		log.Printf("Warning: gagal sinkronisasi Tanah: %v", err)
	}

	// Generate Static Files Mode
	if *genStatic {
		fmt.Println("Generating static files...")

		// 1. residents.json
		jsonData, err := json.MarshalIndent(households, "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		if err := os.WriteFile("residents.json", jsonData, 0644); err != nil {
			log.Fatal(err)
		}
		fmt.Println("Created residents.json")

		// 2. boundary.geojson (Copy)
		boundaryData, err := os.ReadFile("sijenggung.geojson")
		if err != nil {
			log.Printf("Warning: Could not read sijenggung.geojson: %v", err)
		} else {
			if err := os.WriteFile("boundary.geojson", boundaryData, 0644); err != nil {
				log.Fatal(err)
			}
			fmt.Println("Created boundary.geojson")
		}

		fmt.Println("Static build complete. You can now upload 'index.html', 'residents.json', and 'boundary.geojson' to a static host.")
		return
	}

	// Server Mode
	r := gin.Default()

	// Load HTML templates
	r.LoadHTMLGlob("templates/*")

	// Routes
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	r.GET("/login", func(c *gin.Context) {
		c.HTML(http.StatusOK, "login.html", nil)
	})

	r.GET("/editor", func(c *gin.Context) {
		c.HTML(http.StatusOK, "editor.html", nil)
	})

	// Serve Verifikator Page
	r.GET("/verifikator", func(c *gin.Context) {
		c.HTML(http.StatusOK, "verifikator.html", nil)
	})
	
	// Export Verification Excel
	r.GET("/api/export-verifikasi", func(c *gin.Context) {
		f := excelize.NewFile()
		sheet := "Verifikasi"
		f.SetSheetName("Sheet1", sheet)

		// Headers
		headers := []string{
			"NO", "NO KK", "NAMA KEPALA", "ALAMAT", "DUSUN", "RT/RW", 
			"DESIL", "BPNT", "PKH", "PBI", "JML TANAH", "LUAS TANAH (m2)", "STATUS TANAH", "CATATAN",
		}
		for i, h := range headers {
			cell, _ := excelize.CoordinatesToCellName(i+1, 1)
			f.SetCellValue(sheet, cell, h)
		}

		// Style
		style, _ := f.NewStyle(&excelize.Style{
			Font: &excelize.Font{Bold: true},
			Fill: excelize.Fill{Type: "pattern", Color: []string{"#E0E0E0"}, Pattern: 1},
		})
		f.SetCellStyle(sheet, "A1", "N1", style)

		// Data
		for i, h := range households {
			row := i + 2
			f.SetCellValue(sheet, fmt.Sprintf("A%d", row), i+1)
			f.SetCellValue(sheet, fmt.Sprintf("B%d", row), h.NoKK)
			f.SetCellValue(sheet, fmt.Sprintf("C%d", row), h.HeadName)
			f.SetCellValue(sheet, fmt.Sprintf("D%d", row), h.Address)
			f.SetCellValue(sheet, fmt.Sprintf("E%d", row), h.Dusun)
			f.SetCellValue(sheet, fmt.Sprintf("F%d", row), "") // h.RtRw not in struct
			f.SetCellValue(sheet, fmt.Sprintf("G%d", row), h.WelfareLevel)
			
			f.SetCellValue(sheet, fmt.Sprintf("H%d", row), h.IsBPNT)
			f.SetCellValue(sheet, fmt.Sprintf("I%d", row), h.IsPKH)
			f.SetCellValue(sheet, fmt.Sprintf("J%d", row), h.IsPBI)
			
			f.SetCellValue(sheet, fmt.Sprintf("K%d", row), h.LandCount)
			f.SetCellValue(sheet, fmt.Sprintf("L%d", row), h.LandTotal)
			
			statusTanah := ""
			if h.LandCount > 0 {
				statusTanah = "Memiliki Aset"
			}
			f.SetCellValue(sheet, fmt.Sprintf("M%d", row), statusTanah)
			
			f.SetCellValue(sheet, fmt.Sprintf("N%d", row), h.Keterangan)
		}

		// Set Response Headers
		c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		c.Header("Content-Disposition", "attachment; filename=verifikasi_data_sijenggung.xlsx")
		
		if err := f.Write(c.Writer); err != nil {
			c.JSON(500, gin.H{"error": "Failed to generate excel"})
		}
	})

	// --- EDITOR API ---

	// List Layers
	r.GET("/api/layers", func(c *gin.Context) {
		files, err := os.ReadDir("./layers")
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		var filenames []string
		for _, f := range files {
			if !f.IsDir() && filepath.Ext(f.Name()) == ".geojson" {
				filenames = append(filenames, f.Name())
			}
		}
		c.JSON(200, filenames)
	})

	// Save Layer (GeoJSON)
	r.POST("/api/save-layer", func(c *gin.Context) {
		var req SaveLayerRequest
		if err := c.BindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request"})
			return
		}
		
		// Basic security check on filename
		if req.Filename == "" || strings.Contains(req.Filename, "..") || strings.Contains(req.Filename, "/") {
			c.JSON(400, gin.H{"error": "Invalid filename"})
			return
		}
		
		// Ensure extension
		if !strings.HasSuffix(req.Filename, ".geojson") {
			req.Filename += ".geojson"
		}

		path := filepath.Join("layers", req.Filename)
		
		data, err := json.MarshalIndent(req.Data, "", "  ")
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to marshal data"})
			return
		}

		if err := os.WriteFile(path, data, 0644); err != nil {
			c.JSON(500, gin.H{"error": "Failed to write file"})
			return
		}

		c.JSON(200, gin.H{"status": "ok"})
	})

	// Save Households (Update In-Memory and Residents.json)
	r.POST("/api/save-households", func(c *gin.Context) {
		var updatedHouseholds []Household
		if err := c.BindJSON(&updatedHouseholds); err != nil {
			log.Printf("Error binding JSON: %v", err)
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		fmt.Printf("Received %d households to update.\n", len(updatedHouseholds))

		// Update In-Memory
		count := 0
		for _, u := range updatedHouseholds {
			for i := range households {
				if households[i].NoKK == u.NoKK {
					// Only update fields if they are not empty/zero, or assume overwrite?
					// Editor sends full object with new coords.
					
					// Update coordinates
					households[i].Latitude = u.Latitude
					households[i].Longitude = u.Longitude
					
					// Update props if provided
					if u.HeadName != "" { households[i].HeadName = u.HeadName }
					if u.Address != "" { households[i].Address = u.Address }
					if u.Dusun != "" { households[i].Dusun = u.Dusun }
					if u.WelfareLevel != "" { households[i].WelfareLevel = u.WelfareLevel }
					if u.Keterangan != "" { households[i].Keterangan = u.Keterangan }

					// Update Land if provided
					if u.LandList != nil {
						households[i].LandList = u.LandList
						households[i].LandCount = u.LandCount
						households[i].LandTotal = u.LandTotal
					}
					
					count++
					break
				}
			}
		}
		
		fmt.Printf("Updated %d records in memory.\n", count)

		// Write to residents.json
		jsonData, err := json.MarshalIndent(households, "", "  ")
		if err != nil {
			log.Printf("Error marshaling households: %v", err)
			c.JSON(500, gin.H{"error": "Failed to marshal data"})
			return
		}
		
		if err := os.WriteFile("residents.json", jsonData, 0644); err != nil {
			log.Printf("Error writing residents.json: %v", err)
			c.JSON(500, gin.H{"error": "Failed to write file"})
			return
		}

		c.JSON(200, gin.H{"status": "ok", "updated": count})
	})

	// Serve residents.json dynamically (matching static filename)
	r.GET("/residents.json", func(c *gin.Context) {
		c.JSON(http.StatusOK, households)
	})

	// Serve boundary.geojson (matching static filename)
	r.StaticFile("/boundary.geojson", "sijenggung.geojson")

	// Serve Sijenggung Layers
	r.Static("/layers", "./layers")

	// Serve Images
	r.StaticFile("/logo.png", "./img/logo-banjarnegara.png")
	r.StaticFile("/veda-logo.png", "./veda-logo.png")
	r.StaticFile("/clasnet-logo.png", "./clasnet-logo.png")

	// Start server
	log.Println("Server starting on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
