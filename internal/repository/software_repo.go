package repository

import (
	"context"
	"fmt"
	"hpc-site/internal/models"
	"hpc-site/pkg"
)

func QuerySoftware(ctx context.Context, name, category, tag, search string) ([]models.Software, error) {
	query := `
		SELECT id, name, abstract, homepage, github, categories, tags, created_at
		FROM software
		WHERE 1=1
		`
	var args []interface{}
	argID := 1

	if name != "" {
		query += fmt.Sprintf(" AND LOWER(name) = $%d", argID)
		args = append(args, name)
		argID++
	}
	if category != "" {
		query += fmt.Sprintf(" AND EXISTS (SELECT 1 FROM unnest(categories) c WHERE LOWER(c) = LOWER($%d))", argID)
		args = append(args, category)
		argID++
	}
	if tag != "" {
		query += fmt.Sprintf(" AND EXISTS (SELECT 1 FROM unnest(tags) t WHERE LOWER(t) = LOWER($%d))", argID)
		args = append(args, tag)
		argID++
	}
	if search != "" {
		query += fmt.Sprintf(" AND (name ILIKE $%d OR abstract ILIKE $%d)", argID, argID)
		args = append(args, "%"+search+"%")
		argID++
	}

	query += " ORDER BY id"

	rows, err := pkg.DB.Query(ctx, query, args...)
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
