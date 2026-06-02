// Package exporter contains geocoder export formats for external import pipelines.
package exporter

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"

	geodomain "github.com/kishert-lab/taxi-platform/internal/geocoder/domain"
)

func WritePeliasCSV(writer io.Writer, points []geodomain.LocalGeoPoint) error {
	csvWriter := csv.NewWriter(writer)
	header := []string{
		"id",
		"source",
		"layer",
		"name",
		"address",
		"city_id",
		"lat",
		"lon",
		"confidence",
	}
	if err := csvWriter.Write(header); err != nil {
		return fmt.Errorf("write pelias csv header: %w", err)
	}
	for _, point := range points {
		row := []string{
			point.ID.String(),
			"taxi_platform",
			"address",
			point.Name,
			point.Address,
			point.CityID.String(),
			strconv.FormatFloat(point.Coordinates.Latitude, 'f', 6, 64),
			strconv.FormatFloat(point.Coordinates.Longitude, 'f', 6, 64),
			strconv.FormatFloat(point.Confidence, 'f', 4, 64),
		}
		if err := csvWriter.Write(row); err != nil {
			return fmt.Errorf("write pelias csv row: %w", err)
		}
	}
	csvWriter.Flush()
	if err := csvWriter.Error(); err != nil {
		return fmt.Errorf("flush pelias csv: %w", err)
	}
	return nil
}
