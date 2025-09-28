package repository

import (
	"context"
	"encoding/json"

	"hpc-site/internal/models"
	"hpc-site/pkg"
)

// 获取所有 Benchmark
func GetAllBenchmarks(ctx context.Context) ([]models.Benchmark, error) {
	query := `
		SELECT
			id,
			software_id,
			name,
			dataset,
			COALESCE(hardware, '{}'::jsonb),
			COALESCE(metrics, '{}'::jsonb),
			version,
			created_at
		FROM benchmark
		ORDER BY id DESC
	`

	rows, err := pkg.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var benchmarks []models.Benchmark
	for rows.Next() {
		var b models.Benchmark
		var hw, mt []byte

		if err := rows.Scan(
			&b.ID,
			&b.SoftwareID,
			&b.Name,
			&b.Dataset,
			&hw,
			&mt,
			&b.Version,
			&b.CreatedAt,
		); err != nil {
			return nil, err
		}

		json.Unmarshal(hw, &b.Hardware)
		json.Unmarshal(mt, &b.Metrics)

		benchmarks = append(benchmarks, b)
	}

	return benchmarks, rows.Err()
}

// 按 software_id 获取指定软件的 Benchmark
func GetBenchmarksBySoftwareID(ctx context.Context, softwareID int) ([]models.Benchmark, error) {
	query := `
		SELECT
			id,
			software_id,
			name,
			dataset,
			COALESCE(hardware, '{}'::jsonb),
			COALESCE(metrics, '{}'::jsonb),
			version,
			created_at
		FROM benchmark
		WHERE software_id = $1
		ORDER BY id
	`

	rows, err := pkg.DB.QueryContext(ctx, query, softwareID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var benchmarks []models.Benchmark
	for rows.Next() {
		var b models.Benchmark
		var hw, mt []byte

		if err := rows.Scan(
			&b.ID,
			&b.SoftwareID,
			&b.Name,
			&b.Dataset,
			&hw,
			&mt,
			&b.Version,
			&b.CreatedAt,
		); err != nil {
			return nil, err
		}

		json.Unmarshal(hw, &b.Hardware)
		json.Unmarshal(mt, &b.Metrics)

		benchmarks = append(benchmarks, b)
	}

	return benchmarks, rows.Err()
}
