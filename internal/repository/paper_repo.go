package repository

import (
	"context"
	"hpc-site/internal/models"
	"hpc-site/pkg"
)

// 获取所有论文
func GetAllPapers(ctx context.Context) ([]models.Paper, error) {
	rows, err := pkg.DB.Query(ctx,
		`SELECT id, title, authors, abstract, url, software_names, created_at 
		 FROM paper ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var papers []models.Paper
	for rows.Next() {
		var p models.Paper
		err := rows.Scan(&p.ID, &p.Title, &p.Authors, &p.Abstract, &p.URL, &p.SoftwareNames, &p.CreatedAt)
		if err != nil {
			return nil, err
		}
		papers = append(papers, p)
	}

	return papers, rows.Err()
}
