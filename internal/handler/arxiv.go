package handler

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"hpc-site/internal/models"
	"hpc-site/internal/repository"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

const SearchPattern = "https://arxiv.org/search/?query=%s&searchtype=all&abstracts=hide&order=-announced_date_first&size=50&start=%d"
const SearchPageSize = 50 // arXiv 搜索页面默认每页显示 50 条

func FetchArxivSearchHtml(softwareName string, start int) (string, error) {
	url := fmt.Sprintf(SearchPattern, softwareName, start)
	for i := 0; i < 3; i++ { // 最多重试三次
		resp, err := http.Get(url)
		defer resp.Body.Close() //::TODO 在循环中调用 defer 有可能导致资源泄漏
		if err != nil {
			log.Printf("⚠️ 第 %d 次请求失败: %v", i+1, err)
			time.Sleep(2 * time.Second)
			continue
		}

		if resp.StatusCode == http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return string(body), nil
		}
		log.Printf("⚠️ 返回状态码 %d，重试中...", resp.StatusCode)
		time.Sleep(2 * time.Second)
	}
	return "", fmt.Errorf("连续请求失败: %s", url)
}

func GetArxivIDsFromSearchHtml(html string) []string {
	// 更宽松的匹配
	liRe := regexp.MustCompile(`(?s)<li[^>]*class="[^"]*arxiv-result[^"]*"[^>]*>(.*?)</li>`)
	blocks := liRe.FindAllStringSubmatch(html, -1)

	idRe := regexp.MustCompile(`https://arxiv\.org/abs/(\d{4}\.\d{4,5})`)
	seen := make(map[string]bool)
	ids := make([]string, 0)

	for _, block := range blocks {
		section := block[1]
		m := idRe.FindStringSubmatch(section)
		if len(m) > 1 {
			id := m[1]
			if !seen[id] {
				seen[id] = true
				ids = append(ids, id)
			}
		}
	}
	return ids
}

// 提取总的结果数量，用于分页终止处理
func MatchTotalResults(html string) int {
	re := regexp.MustCompile(`of\s+([0-9,]+)\s+results`)
	match := re.FindStringSubmatch(html) //html中第一个符合正则的部分
	if len(match) > 1 {
		// 移除逗号，并尝试转换成整数
		totalStr := strings.ReplaceAll(match[1], ",", "")
		total, err := strconv.Atoi(totalStr)
		if err != nil {
			log.Printf("Error converting total results string '%s' to int: %v", totalStr, err)
			return 0
		}
		return total
	}
	return 0
}

func CrawlArxivAll(softwareName string) []string {
	start := 0
	page := 1
	allIDs := make(map[string]bool)
	total := 0

	for {
		log.Printf("第 %d 页 start=%d", page, start)
		html, err := FetchArxivSearchHtml(softwareName, start)
		if err != nil {
			log.Printf("获取失败: %v", err)
			break
		}

		if total == 0 {
			total = MatchTotalResults(html)
			log.Printf("📦 总共 %d 条结果", total)
			if total == 0 {
				break
			}
		}

		ids := GetArxivIDsFromSearchHtml(html)
		log.Printf("第 %d 页解析出 %d 条", page, len(ids))
		for _, id := range ids {
			allIDs[id] = true
		}

		// 检查是否已到末页
		if len(ids) == 0 || start+SearchPageSize >= total {
			log.Printf("抓取结束: 共 %d 唯一论文", len(allIDs))
			break
		}

		start += SearchPageSize
		page++
		time.Sleep(2 * time.Second)
	}

	// 转成 slice
	result := make([]string, 0, len(allIDs))
	for id := range allIDs {
		result = append(result, id)
	}
	return result
}

// loop to get all papers by paper-id
func GetArxivPageSource(id string, isWithDrawn bool, version int) string {
	url := FormatPageUrl(id, isWithDrawn, version)
	fmt.Println(url)
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Error fetching data: %v", err)
		return ""
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		log.Printf("Error: received status code %d", resp.StatusCode)
		return ""
	}

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response body: %v", err)
		return ""
	}
	return string(body)
}

// 详情页的
func FormatPageUrl(id string, isWithDrawn bool, version int) string {
	if isWithDrawn {
		return fmt.Sprintf("https://arxiv.org/abs/%sv%d", id, version)
	} else {
		return fmt.Sprintf("https://arxiv.org/abs/%s", id)
	}
}

func GetPaperFromMetaData(extractedId string, software string) models.Paper {
	sourceCode := GetArxivPageSource(extractedId, false, 0)
	isLatestVersionWithDrawn := IsWithDrawn(sourceCode)
	title := MatchTitle(sourceCode)
	authors := MatchAuthors(sourceCode)
	abstract := MatchAbstract(sourceCode)
	if isLatestVersionWithDrawn {
		version := FindLastValidVersion(sourceCode)
		url := FormatPageUrl(extractedId, true, version)
		code := GetArxivPageSource(extractedId, true, version)
		log.Println("try to find latest valid version")
		pdf := MatchPdf(code)
		publishedTime := MatchSubmissionDate(code, isLatestVersionWithDrawn, version)
		return models.Paper{
			ID:            extractedId,
			Title:         title,
			Authors:       authors,
			Abstract:      abstract,
			URL:           url,
			Pdf:           pdf,
			PublishedTime: publishedTime,
			SoftwareNames: []string{software},
		}
	} else {
		url := fmt.Sprintf("https://arxiv.org/abs/%s", extractedId)
		log.Println("source code get,start to match content")
		pdf := MatchPdf(sourceCode)
		publishedTime := MatchSubmissionDate(sourceCode, false, 0)
		return models.Paper{
			ID:            extractedId,
			Title:         title,
			Authors:       authors,
			Abstract:      abstract,
			URL:           url,
			Pdf:           pdf,
			PublishedTime: publishedTime,
			SoftwareNames: []string{software}, // ✅ 一样要加
		}
	}
}

func IsWithDrawn(source string) bool {
	return strings.Contains(source, "This paper has been withdrawn by")
}

func MatchTitle(source string) string {
	regex := `(?s)<meta property="og:title" content="(.*?)"\s*\/>`
	return MatchContent(source, regex)
}

func MatchAuthors(source string) []string {
	regex := `<div class="authors"><span class="descriptor">Authors:<\/span>(.*?)<\/div>`
	authorListContent := MatchContent(source, regex)
	if authorListContent == "" {
		return []string{}
	}
	//拆分每个作者部分
	authorNameList := strings.Split(authorListContent, ",")
	authors := make([]string, 0, len(authorNameList))
	namePattern := `>(.*?)<`
	for _, authorItem := range authorNameList {
		author := MatchContent(authorItem, namePattern)
		if author != "" {
			authors = append(authors, author)
		}
	}
	return authors
}

func MatchAbstract(source string) string {
	regex := `(?s)<meta property="og:description" content="(.*?)"\/>`
	return MatchContent(source, regex)
}

func FindLastValidVersion(source string) int {
	regex := `(?s)<h2>Submission history<\/h2>(.*?)<\/div>`
	submissionSection := MatchContent(source, regex)
	submissionList := strings.Split(submissionSection, "</strong>")
	result := 0
	for index, line := range submissionList {
		if strings.Contains(line, "<a href") {
			result = index + 1
		}
	}
	return result
}

func MatchPdf(source string) string {
	regex := `<a\s*href="(.*)?"\s*aria-describedby="download-button-info" accesskey="f" class="abs-button download-pdf">View PDF<\/a>`
	fmt.Println(MatchContent(source, regex))
	return fmt.Sprintf("https://arxiv.org%s", MatchContent(source, regex))
}

func MatchSubmissionDate(source string, isWithDrawn bool, version int) string {
	regex := `(?s)<h2>Submission history<\/h2>(.*?)<\/div>`
	submissionSection := MatchContent(source, regex)
	submissionList := strings.Split(submissionSection, "</strong>")
	submissionArray := make([]string, 0)
	for _, item := range submissionList {
		if strings.Contains(item, "UTC") {
			submissionArray = append(submissionArray, item)
		}
	}
	if !isWithDrawn {
		matchedTime := strings.TrimSpace(MatchContent(submissionArray[len(submissionArray)-1], `(.*?UTC)`))
		return matchedTime
	} else {
		matchedTime := strings.TrimSpace(MatchContent(submissionArray[version-1], `(.*?UTC)`))
		return matchedTime
	}
}

func MatchContent(source string, regex string) string {
	re := regexp.MustCompile(regex)

	matches := re.FindStringSubmatch(source)
	// 检查是否有匹配项
	if len(matches) > 1 {
		// matches[0] 是完整的匹配，matches[1] 是第一个捕获组
		return matches[1]
	} else {
		return ""
	}
}

// 数据写入

// splitAuthors 把 "Alice; Bob" -> []string{"Alice","Bob"}
func splitAuthors(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return []string{}
	}
	parts := strings.Split(raw, ";")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		s := strings.TrimSpace(p)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func ProcessSoftwarePapers(softwareName string) {
	paperIds := CrawlArxivAll(softwareName)
	//只有paper不存在的情况才需要去抓
	log.Printf("开始抓取与软件 [%s] 相关的论文", softwareName)
	for i, paperId := range paperIds {
		log.Printf("[%d/%d] 检查论文 %s 是否存在", i+1, len(paperIds), paperId)
		//判断是否存在
		exists, existingSoftwares, err := repository.CheckPaperExists(paperId)
		if err != nil {
			log.Printf("检查论文 %s 是否存在时出错: %v", paperId, err)
			continue
		}
		//存在则只更新software
		if exists {
			merged := repository.MergeUnique(existingSoftwares, []string{softwareName})
			if len(merged) != len(existingSoftwares) {
				log.Printf("论文 %s 已存在，合并软件 [%s]", paperId, softwareName)
				err := repository.UpdatePaperSoftware(paperId, merged)
				if err != nil {
					log.Printf("更新论文 %s 软件失败: %v", paperId, err)
				} else {
					log.Printf("已为论文 %s 添加新软件 [%s]", paperId, softwareName)
				}
			} else {
				log.Printf("论文与软件已经存在，无需更新")
				continue
			}
		} else {
			//paper不存在就去抓详情页
			paper := GetPaperFromMetaData(paperId, softwareName)
			if paper.ID == "" || paper.Title == "" {
				log.Printf("论文 %s 抓取失败，跳过", paperId)
				continue
			}
			err = repository.InsertNewPaper(paper)
			if err != nil {
				if err != nil {
					log.Printf("插入论文 %s 失败: %v", paperId, err)
				} else {
					log.Printf("✅ 成功插入论文 %s", paperId)
				}
			}
		}
	}
	log.Printf("[%s] 抓取完成", softwareName)

}

func TestLammps(c *gin.Context) {
	ProcessSoftwarePapers("Lammps")
	c.JSON(http.StatusOK, gin.H{
		"message": "test",
	})
}
