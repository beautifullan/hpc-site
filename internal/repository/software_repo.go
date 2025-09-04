package repository

import (
	"context"
	"fmt"
	"github.com/lib/pq"
	"hpc-site/internal/models"
	"hpc-site/pkg"
)

// 软件查询（支持过滤）
func QuerySoftware(ctx context.Context, name, category, tag, search string) ([]models.Software, error) {
	query := `
		SELECT id, name, abstract, homepage, github, categories, tags, created_at
		FROM software
		WHERE 1=1
	`
	var args []interface{}
	argID := 1

	if name != "" {
		query += fmt.Sprintf(" AND LOWER(name) = LOWER($%d)", argID)
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

	rows, err := pkg.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var softwares []models.Software
	for rows.Next() {
		var s models.Software
		err := rows.Scan(&s.ID, &s.Name, &s.Abstract, &s.Homepage, &s.Github,
			pq.Array(&s.Categories), pq.Array(&s.Tags), &s.CreatedAt)
		if err != nil {
			return nil, err
		}
		softwares = append(softwares, s)
	}

	return softwares, rows.Err()
}

// 根据 ID 获取软件
func GetSoftwareByID(ctx context.Context, id int) (*models.Software, error) {
	query := `
		SELECT id, name, abstract, homepage, github, categories, tags, created_at
		FROM software
		WHERE id = $1
	`

	row := pkg.DB.QueryRowContext(ctx, query, id)

	var s models.Software
	err := row.Scan(&s.ID, &s.Name, &s.Abstract, &s.Homepage, &s.Github,
		pq.Array(&s.Categories), pq.Array(&s.Tags), &s.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// 查询某个软件相关的论文
func GetPapersBySoftwareID(ctx context.Context, id int) ([]models.Paper, error) {
	query := `
    SELECT p.id, p.title, p.authors, p.abstract, p.url, p.software_names, p.created_at
    FROM paper p
    JOIN software s 
      ON EXISTS (
          SELECT 1
          FROM unnest(p.software_names) sn
          WHERE LOWER(sn) = LOWER(s.name)
      )
    WHERE s.id = $1
`

	rows, err := pkg.DB.QueryContext(ctx, query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var papers []models.Paper
	for rows.Next() {
		var p models.Paper
		err := rows.Scan(&p.ID, &p.Title, pq.Array(&p.Authors), &p.Abstract, &p.URL, pq.Array(&p.SoftwareNames), &p.CreatedAt)
		if err != nil {
			return nil, err
		}
		papers = append(papers, p)
	}
	return papers, rows.Err()
}
