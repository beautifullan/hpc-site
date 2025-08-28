package repository

import (
	"context"
	"hpc-site/internal/models"
	"hpc-site/pkg"
)

func GetAllSoftware(ctx context.Context) ([]models.Software, error) {
	rows, err := pkg.DB.Query(ctx,
		"SELECT id, name, abstract, homepage, github, categories, tags, created_at FROM software ORDER BY id",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var softwares []models.Software
	for rows.Next() {
		var s models.Software
		err := rows.Scan(&s.ID, &s.Name, &s.Abstract, &s.Homepage, &s.Github, &s.Categories, &s.Tags, &s.CreatedAt)
		if err != nil {
			return nil, err
		}
		softwares = append(softwares, s)
	}

	return softwares, rows.Err()
}
