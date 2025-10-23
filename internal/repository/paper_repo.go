package repository

import (
	"context"
	"database/sql"
	"github.com/lib/pq"
	"hpc-site/internal/models"
	"hpc-site/pkg"
	"log"
	"regexp"
	"strings"
	"time"
)

// 获取所有论文
func GetAllPapers(ctx context.Context) ([]models.Paper, error) {
	rows, err := pkg.DB.QueryContext(ctx,
		`SELECT id, title, authors, abstract, url, software_names, created_at 
		 FROM paper ORDER BY id`)
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

// 第一种情况paper不存在 insert
func InsertNewPaper(paper models.Paper) error {
	log.Printf("Inserting new paper %s with software", paper.ID)
	_, err := pkg.DB.Exec(`INSERT INTO paper(id, title, authors, abstract, url, pdf, software_names, published_time,created_at)
            VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		paper.ID,
		paper.Title,
		pq.Array(paper.Authors),
		paper.Abstract,
		paper.URL,
		paper.Pdf,
		pq.Array(paper.SoftwareNames), // 插入软件名数组
		paper.PublishedTime,
		time.Now(),
	)
	if err != nil {
		return err
	}
	return nil
}

// paper存在但是software不存在
func UpdatePaperSoftware(paperID string, updatedSoftwareNames []string) error {
	sql := `UPDATE paper SET software_names = $1 WHERE id = $2`
	_, err := pkg.DB.Exec(sql, pq.Array(updatedSoftwareNames), paperID)
	if err != nil {
		return err
	}
	return nil
}

// 第三种情况
func InsertOrUpdatePaper(paper models.Paper) error {
	var soft []string
	err := pkg.DB.QueryRow(`SELECT software_names FROM paper WHERE id = $1`, paper.ID).Scan(pq.Array(&soft))
	if err == sql.ErrNoRows {
		//not existing insert
		return InsertNewPaper(paper)
	}
	if err != nil {
		return err
	}

	//existing
	existing := []string(soft)
	merged := MergeUnique(existing, paper.SoftwareNames)
	if len(merged) == len(existing) {
		log.Printf("Paper %s already contains software(s): %v — skip", paper.ID, paper.SoftwareNames)
		return nil
	}

	// 否则更新合并后的列表
	return UpdatePaperSoftware(paper.ID, merged)
}

func parseSoftwareNames(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return []string{}
	}
	// 支持 ; 或 , 或 空格分割的情况
	parts := regexp.MustCompile(`[;,]`).Split(raw, -1)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		s := strings.TrimSpace(p)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func MergeUnique(a, b []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(a)+len(b))

	// 保留 a 的原始顺序/大小写
	for _, s := range a {
		if s == "" {
			continue
		}
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	// 再把 b 的补充加入
	for _, s := range b {
		if s == "" {
			continue
		}
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

func CheckPaperExists(paperID string) (bool, []string, error) {
	var softwares pq.StringArray
	err := pkg.DB.QueryRow(`SELECT software_names FROM paper WHERE id = $1`, paperID).
		Scan(pq.Array(&softwares))

	if err == sql.ErrNoRows {
		return false, nil, nil
	}

	if err != nil {
		//尝试把字符串手动解析成 []string
		var raw string
		rawErr := pkg.DB.QueryRow(`SELECT software_names::text FROM paper WHERE id = $1`, paperID).
			Scan(&raw)
		if rawErr == nil && raw != "" {
			clean := strings.Trim(raw, "{}\" ")
			if clean == "" {
				return true, []string{}, nil
			}
			return true, strings.Split(clean, ","), nil
		}
		log.Printf("检查论文 %s 是否存在时出错: %v", paperID, err)
		return false, nil, err
	}

	return true, []string(softwares), nil
}
