package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"

	polyclip "github.com/akavel/polyclip-go"
)

type geoJSONFeatureCollection struct {
	Type     string           `json:"type"`
	Name     string           `json:"name,omitempty"`
	CRS      map[string]any   `json:"crs,omitempty"`
	Features []geoJSONFeature `json:"features"`
}

type geoJSONFeature struct {
	Type       string          `json:"type"`
	Properties map[string]any  `json:"properties,omitempty"`
	Geometry   geoJSONGeometry `json:"geometry"`
}

type geoJSONGeometry struct {
	Type        string          `json:"type"`
	Coordinates json.RawMessage `json:"coordinates"`
}

func ClipGeoJSONToPondokrejoBoundary(inputPath, boundaryPath, outputPath string) error {
	if outputPath == "" {
		ext := filepath.Ext(inputPath)
		base := inputPath[:len(inputPath)-len(ext)]
		outputPath = base + "-pondokrejo" + ext
	}

	// Use BBox for clipping instead of complex polygon intersection
	boundaryPoly, err := loadBoundaryAsBBox(boundaryPath)
	if err != nil {
		return err
	}
	boundaryBBox := getPolygonBBox(boundaryPoly)

	inBytes, err := os.ReadFile(inputPath)
	if err != nil {
		return err
	}

	var fc geoJSONFeatureCollection
	if err := json.Unmarshal(inBytes, &fc); err != nil {
		return err
	}
	if fc.Type == "" {
		fc.Type = "FeatureCollection"
	}

	out := geoJSONFeatureCollection{
		Type: fc.Type,
		Name: fc.Name,
		CRS:  fc.CRS,
	}

	for _, f := range fc.Features {
		if f.Geometry.Type == "" || len(f.Geometry.Coordinates) == 0 {
			continue
		}

		switch f.Geometry.Type {
		case "Polygon":
			var coords [][][]float64
			if err := json.Unmarshal(f.Geometry.Coordinates, &coords); err != nil {
				continue
			}
			intersections := clipPolygonCoords(coords, boundaryPoly)
			for _, poly := range intersections {
				g := geoJSONGeometry{Type: "Polygon"}
				raw, err := json.Marshal(poly)
				if err != nil {
					continue
				}
				g.Coordinates = raw
				out.Features = append(out.Features, geoJSONFeature{
					Type:       "Feature",
					Properties: f.Properties,
					Geometry:   g,
				})
			}
		case "MultiPolygon":
			var coords [][][][]float64
			if err := json.Unmarshal(f.Geometry.Coordinates, &coords); err != nil {
				continue
			}
			for _, polyCoords := range coords {
				intersections := clipPolygonCoords(polyCoords, boundaryPoly)
				for _, poly := range intersections {
					g := geoJSONGeometry{Type: "Polygon"}
					raw, err := json.Marshal(poly)
					if err != nil {
						continue
					}
					g.Coordinates = raw
					out.Features = append(out.Features, geoJSONFeature{
						Type:       "Feature",
						Properties: f.Properties,
						Geometry:   g,
					})
				}
			}
		case "LineString":
			var coords [][]float64
			if err := json.Unmarshal(f.Geometry.Coordinates, &coords); err != nil {
				continue
			}
			clippedLines := clipLineStringCoords(coords, boundaryBBox)
			if len(clippedLines) > 0 {
				// If resulting in multiple disjoint lines, use MultiLineString?
				// Or just keep them as separate Features or one MultiLineString Feature.
				// To preserve properties, MultiLineString is better.
				g := geoJSONGeometry{Type: "MultiLineString"}
				raw, err := json.Marshal(clippedLines)
				if err != nil {
					continue
				}
				g.Coordinates = raw
				out.Features = append(out.Features, geoJSONFeature{
					Type:       "Feature",
					Properties: f.Properties,
					Geometry:   g,
				})
			}
		case "MultiLineString":
			var coords [][][]float64
			if err := json.Unmarshal(f.Geometry.Coordinates, &coords); err != nil {
				continue
			}
			var allClippedLines [][][]float64
			for _, lineCoords := range coords {
				clipped := clipLineStringCoords(lineCoords, boundaryBBox)
				allClippedLines = append(allClippedLines, clipped...)
			}
			if len(allClippedLines) > 0 {
				g := geoJSONGeometry{Type: "MultiLineString"}
				raw, err := json.Marshal(allClippedLines)
				if err != nil {
					continue
				}
				g.Coordinates = raw
				out.Features = append(out.Features, geoJSONFeature{
					Type:       "Feature",
					Properties: f.Properties,
					Geometry:   g,
				})
			}
		case "Point":
			var coords []float64
			if err := json.Unmarshal(f.Geometry.Coordinates, &coords); err != nil {
				continue
			}
			if len(coords) < 2 {
				continue
			}
			if isPointInBBox(coords, boundaryBBox) {
				out.Features = append(out.Features, f)
			}
		case "MultiPoint":
			var coords [][]float64
			if err := json.Unmarshal(f.Geometry.Coordinates, &coords); err != nil {
				continue
			}
			var validCoords [][]float64
			for _, p := range coords {
				if len(p) >= 2 && isPointInBBox(p, boundaryBBox) {
					validCoords = append(validCoords, p)
				}
			}
			if len(validCoords) > 0 {
				g := geoJSONGeometry{Type: "MultiPoint"}
				raw, err := json.Marshal(validCoords)
				if err != nil {
					continue
				}
				g.Coordinates = raw
				out.Features = append(out.Features, geoJSONFeature{
					Type:       "Feature",
					Properties: f.Properties,
					Geometry:   g,
				})
			}
		default:
			// Pass through other types or ignore?
			// For now, ignore points/etc as we are focusing on clipping areas/lines
			continue
		}
	}

	outBytes, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(outputPath, outBytes, 0644)
}

func loadBoundaryUnion(boundaryPath string) (polyclip.Polygon, error) {
	bBytes, err := os.ReadFile(boundaryPath)
	if err != nil {
		return nil, err
	}

	var fc geoJSONFeatureCollection
	if err := json.Unmarshal(bBytes, &fc); err != nil {
		return nil, err
	}
	if fc.Type != "FeatureCollection" || len(fc.Features) == 0 {
		return nil, fmt.Errorf("boundary must be a FeatureCollection with at least one feature")
	}

	var union polyclip.Polygon
	unionInitialized := false

	for _, feat := range fc.Features {
		switch feat.Geometry.Type {
		case "Polygon":
			var coords [][][]float64
			if err := json.Unmarshal(feat.Geometry.Coordinates, &coords); err != nil {
				continue
			}
			p := polygonFromGeoJSONRings(coords)
			if len(p) == 0 {
				continue
			}
			if !unionInitialized {
				union = p
				unionInitialized = true
			} else {
				union = union.Construct(polyclip.UNION, p)
			}
		case "MultiPolygon":
			var coords [][][][]float64
			if err := json.Unmarshal(feat.Geometry.Coordinates, &coords); err != nil {
				continue
			}
			for _, polyCoords := range coords {
				p := polygonFromGeoJSONRings(polyCoords)
				if len(p) == 0 {
					continue
				}
				if !unionInitialized {
					union = p
					unionInitialized = true
				} else {
					union = union.Construct(polyclip.UNION, p)
				}
			}
		}
	}

	if !unionInitialized || len(union) == 0 {
		return nil, fmt.Errorf("failed to parse any valid boundary polygon from %s", boundaryPath)
	}
	return union, nil
}

func loadBoundaryAsBBox(boundaryPath string) (polyclip.Polygon, error) {
	if strings.EqualFold(boundaryPath, "DEFAULT_VIEW") || strings.TrimSpace(boundaryPath) == "" {
		bbox := defaultViewBBox()
		rect := []polyclip.Point{
			{X: bbox.MinX, Y: bbox.MinY},
			{X: bbox.MaxX, Y: bbox.MinY},
			{X: bbox.MaxX, Y: bbox.MaxY},
			{X: bbox.MinX, Y: bbox.MaxY},
		}
		return polyclip.Polygon{polyclip.Contour(rect)}, nil
	}

	// First load the actual boundary
	actualPoly, err := loadBoundaryUnion(boundaryPath)
	if err != nil {
		return nil, err
	}

	// Calculate BBox
	bbox := getPolygonBBox(actualPoly)

	// Create a rectangular polygon from BBox
	// Order: MinX,MinY -> MaxX,MinY -> MaxX,MaxY -> MinX,MaxY
	rect := []polyclip.Point{
		{X: bbox.MinX, Y: bbox.MinY},
		{X: bbox.MaxX, Y: bbox.MinY},
		{X: bbox.MaxX, Y: bbox.MaxY},
		{X: bbox.MinX, Y: bbox.MaxY},
	}
	// Close the loop (polyclip contour doesn't strictly need duplicate end point if handled as ring, but let's see)
	// polyclip.Contour is just a slice of points.

	// Construct the polygon (single contour)
	poly := polyclip.Polygon{polyclip.Contour(rect)}
	return poly, nil
}

func clipPolygonCoords(coords [][][]float64, boundaryUnion polyclip.Polygon) [][][][]float64 {
	subject := polygonFromGeoJSONRings(coords)
	if len(subject) == 0 {
		return nil
	}

	clipped := subject.Construct(polyclip.INTERSECTION, boundaryUnion)
	return geoJSONPolygonsFromPolyclip(clipped)
}

func polygonFromGeoJSONRings(rings [][][]float64) polyclip.Polygon {
	if len(rings) == 0 {
		return nil
	}

	var p polyclip.Polygon
	for _, ring := range rings {
		contour := contourFromGeoJSONRing(ring)
		if len(contour) < 3 {
			continue
		}
		p = append(p, contour)
	}
	return p
}

func contourFromGeoJSONRing(ring [][]float64) polyclip.Contour {
	if len(ring) == 0 {
		return nil
	}

	points := make([]polyclip.Point, 0, len(ring))
	for _, c := range ring {
		if len(c) < 2 {
			continue
		}
		points = append(points, polyclip.Point{X: c[0], Y: c[1]})
	}

	if len(points) >= 2 {
		first := points[0]
		last := points[len(points)-1]
		if first.X == last.X && first.Y == last.Y {
			points = points[:len(points)-1]
		}
	}
	return polyclip.Contour(points)
}

func geoJSONPolygonsFromPolyclip(p polyclip.Polygon) [][][][]float64 {
	if len(p) == 0 {
		return nil
	}

	out := make([][][][]float64, 0, len(p))
	for _, contour := range p {
		if len(contour) < 3 {
			continue
		}
		ring := make([][]float64, 0, len(contour)+1)
		for _, pt := range contour {
			ring = append(ring, []float64{pt.X, pt.Y})
		}
		ring = append(ring, []float64{contour[0].X, contour[0].Y})

		out = append(out, [][][]float64{ring})
	}
	return out
}

// --- Line Clipping Logic ---

func clipLineStringCoords(lineCoords [][]float64, bbox BBox) [][][]float64 {
	if len(lineCoords) < 2 {
		return nil
	}

	dim := 2
	if len(lineCoords[0]) >= 3 {
		dim = 3
	}

	var out [][][]float64
	var current [][]float64

	for i := 0; i < len(lineCoords)-1; i++ {
		a := lineCoords[i]
		b := lineCoords[i+1]
		if len(a) < 2 || len(b) < 2 {
			continue
		}

		p1 := polyclip.Point{X: a[0], Y: a[1]}
		p2 := polyclip.Point{X: b[0], Y: b[1]}
		if !bbox.Overlaps(getSegmentBBox(p1, p2)) {
			current = nil
			continue
		}

		t0, t1, ok := liangBarsky(p1, p2, bbox)
		if !ok {
			current = nil
			continue
		}

		qa := pointAtT(a, b, t0, dim)
		qb := pointAtT(a, b, t1, dim)

		if len(current) == 0 {
			current = append(current, qa, qb)
		} else {
			last := current[len(current)-1]
			if len(last) >= 2 && almostEqual(last[0], qa[0]) && almostEqual(last[1], qa[1]) {
				current = append(current, qb)
			} else {
				out = append(out, current)
				current = [][]float64{qa, qb}
			}
		}
	}

	if len(current) >= 2 {
		out = append(out, current)
	}

	return out
}

type BBox struct {
	MinX, MinY, MaxX, MaxY float64
}

func getPolygonBBox(poly polyclip.Polygon) BBox {
	inf := 1e308
	bbox := BBox{inf, inf, -inf, -inf}
	for _, c := range poly {
		for _, p := range c {
			if p.X < bbox.MinX {
				bbox.MinX = p.X
			}
			if p.X > bbox.MaxX {
				bbox.MaxX = p.X
			}
			if p.Y < bbox.MinY {
				bbox.MinY = p.Y
			}
			if p.Y > bbox.MaxY {
				bbox.MaxY = p.Y
			}
		}
	}
	return bbox
}

func (b BBox) Overlaps(other BBox) bool {
	return b.MinX <= other.MaxX && b.MaxX >= other.MinX &&
		b.MinY <= other.MaxY && b.MaxY >= other.MinY
}

func getSegmentBBox(p1, p2 polyclip.Point) BBox {
	return BBox{
		MinX: min(p1.X, p2.X), MinY: min(p1.Y, p2.Y),
		MaxX: max(p1.X, p2.X), MaxY: max(p1.Y, p2.Y),
	}
}

func isPointInBBox(p []float64, bbox BBox) bool {
	return p[0] >= bbox.MinX && p[0] <= bbox.MaxX &&
		p[1] >= bbox.MinY && p[1] <= bbox.MaxY
}

func liangBarsky(a, b polyclip.Point, bbox BBox) (float64, float64, bool) {
	dx := b.X - a.X
	dy := b.Y - a.Y

	t0 := 0.0
	t1 := 1.0

	if !lbClip(-dx, a.X-bbox.MinX, &t0, &t1) {
		return 0, 0, false
	}
	if !lbClip(dx, bbox.MaxX-a.X, &t0, &t1) {
		return 0, 0, false
	}
	if !lbClip(-dy, a.Y-bbox.MinY, &t0, &t1) {
		return 0, 0, false
	}
	if !lbClip(dy, bbox.MaxY-a.Y, &t0, &t1) {
		return 0, 0, false
	}
	return t0, t1, t0 <= t1
}

func lbClip(p, q float64, t0, t1 *float64) bool {
	if p == 0 {
		return q >= 0
	}
	r := q / p
	if p < 0 {
		if r > *t1 {
			return false
		}
		if r > *t0 {
			*t0 = r
		}
		return true
	}
	if r < *t0 {
		return false
	}
	if r < *t1 {
		*t1 = r
	}
	return true
}

func pointAtT(a, b []float64, t float64, dim int) []float64 {
	if dim == 3 && len(a) >= 3 && len(b) >= 3 {
		return []float64{
			a[0] + t*(b[0]-a[0]),
			a[1] + t*(b[1]-a[1]),
			a[2] + t*(b[2]-a[2]),
		}
	}
	return []float64{
		a[0] + t*(b[0]-a[0]),
		a[1] + t*(b[1]-a[1]),
	}
}

func defaultViewBBox() BBox {
	centerLat := -7.65
	centerLon := 110.31
	zoom := 13
	pixelWidth := 760.0
	pixelHeight := 704.0

	worldSize := 256.0 * float64(uint(1)<<uint(zoom))

	x := (centerLon + 180.0) / 360.0 * worldSize
	latRad := centerLat * math.Pi / 180.0
	y := (1.0 - math.Log(math.Tan(latRad)+1.0/math.Cos(latRad))/math.Pi) / 2.0 * worldSize

	minX := x - pixelWidth/2.0
	maxX := x + pixelWidth/2.0
	minY := y - pixelHeight/2.0
	maxY := y + pixelHeight/2.0

	minLon := minX/worldSize*360.0 - 180.0
	maxLon := maxX/worldSize*360.0 - 180.0

	minLat := 180.0 / math.Pi * math.Atan(math.Sinh(math.Pi*(1.0-2.0*maxY/worldSize)))
	maxLat := 180.0 / math.Pi * math.Atan(math.Sinh(math.Pi*(1.0-2.0*minY/worldSize)))

	return BBox{MinX: minLon, MinY: minLat, MaxX: maxLon, MaxY: maxLat}
}

func almostEqual(a, b float64) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d < 1e-12
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
