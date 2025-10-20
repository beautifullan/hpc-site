package handler

import (
	"database/sql"
	"encoding/xml"
	"fmt"
	"github.com/lib/pq"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

type PaperResponse struct {
	Success bool  `json:"success"`
	Data    Paper `json:"data"`
}

type Paper struct {
	ID            string `json:"ID"`
	Title         string `json:"Title"`
	Authors       string `json:"Authors"`
	Abstract      string `json:"Abstract"`
	URL           string `json:"URL"`
	PDF           string `json:"Pdf"`
	SoftwareName  string `json:"SoftwareName"`
	PublishedTime string `json:"PublishTime"`
}

// RSS represents the root of the RSS feed
type RSS struct {
	XMLName xml.Name `xml:"rss"`
	Channel Channel  `xml:"channel"`
}

// Channel represents the channel element in the RSS feed
type Channel struct {
	Title          string   `xml:"title"`
	Link           string   `xml:"link"`
	Description    string   `xml:"description"`
	AtomLink       AtomLink `xml:"atom:link"`
	Docs           string   `xml:"docs"`
	Language       string   `xml:"language"`
	LastBuildDate  string   `xml:"lastBuildDate"`
	ManagingEditor string   `xml:"managingEditor"`
	PubDate        string   `xml:"pubDate"`
	SkipDays       SkipDays `xml:"skipDays"`
	Items          []Item   `xml:"item"`
}

// AtomLink represents the atom link element
type AtomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
	Type string `xml:"type,attr"`
}

// SkipDays represents the skipDays element
type SkipDays struct {
	Day []string `xml:"day"`
}

// Item represents each item in the channel
type Item struct {
	Title        string `xml:"title"`
	Link         string `xml:"link"`
	Description  string `xml:"description"`
	GUID         string `xml:"guid"`
	Category     string `xml:"category"`
	PubDate      string `xml:"pubDate"`
	AnnounceType string `xml:"arxiv:announce_type"`
	Rights       string `xml:"dc:rights"`
	Creator      string `xml:"dc:creator"`
}

func encodeParams(params map[string]string) string {
	values := make(url.Values)
	for key, value := range params {
		values.Add(key, value)
	}
	return values.Encode()
}

const CrawlPattern = "https://arxiv.org/list/%s/%s"
const SearchPattern = "https://arxiv.org/search/?query=%s&searchtype=all&abstracts=hide&order=-announced_date_first&size=50&start=%d"
const SearchPageSize = 50 // arXiv 搜索页面默认每页显示 50 条
const linkReg = `<dt>(.|\s)*?<\/dt>`
const pageReg = `<div\s*class='paging'>(.|\s)*?<\/div>`

func IsWithDrawn(source string) bool {
	return strings.Contains(source, "This paper has been withdrawn by")
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

func MatchAbstract(source string) string {
	regex := `(?s)<meta property="og:description" content="(.*?)"\/>`
	return MatchContent(source, regex)
}

func MatchTitle(source string) string {
	regex := `(?s)<meta property="og:title" content="(.*?)"\s*\/>`
	return MatchContent(source, regex)
}

func MatchPdf(source string) string {
	regex := `<a\s*href="(.*)?"\s*aria-describedby="download-button-info" accesskey="f" class="abs-button download-pdf">View PDF<\/a>`
	fmt.Println(MatchContent(source, regex))
	return fmt.Sprintf("https://arxiv.org%s", MatchContent(source, regex))
}

func MatchAuthors(source string) string {
	regex := `<div class="authors"><span class="descriptor">Authors:<\/span>(.*?)<\/div>`
	authorListContent := MatchContent(source, regex)
	authorNameList := strings.Split(authorListContent, ",")
	authors := make([]string, 0)
	namePattern := `>(.*?)<`
	for _, authorItem := range authorNameList {
		author := MatchContent(authorItem, namePattern)
		if author != "" {
			authors = append(authors, author)
		}
	}
	return strings.Join(authors, "; ")
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

// 详情页的
func FormatPageUrl(id string, isWithDrawn bool, version int) string {
	if isWithDrawn {
		return fmt.Sprintf("https://arxiv.org/abs/%sv%d", id, version)
	} else {
		return fmt.Sprintf("https://arxiv.org/abs/%s", id)
	}
}

func FetchArxivSearchHtml(softwareName string, start int) (string, error) {
	url := fmt.Sprintf(SearchPattern, softwareName, start)
	for i := 0; i < 3; i++ { // 最多重试三次
		resp, err := http.Get(url)
		if err != nil {
			log.Printf("⚠️ 第 %d 次请求失败: %v", i+1, err)
			time.Sleep(2 * time.Second)
			continue
		}
		defer resp.Body.Close()

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

// 数据写入

// EnsureSoftwareExists 确保软件名称存在于 software 表中。
func EnsureSoftwareExists(db *sql.DB, name string) {
	// 使用 PostgreSQL 的 UPSERT 避免重复插入
	sql := `INSERT INTO software (name) VALUES ($1) ON CONFLICT (name) DO NOTHING`
	_, err := db.Exec(sql, name)
	if err != nil {
		log.Printf("Error ensuring software '%s' exists: %v", name, err)
	}
}

// 检索现有论文的软件列表
func GetPaperSoftwareNames(db *sql.DB, paperID string) ([]string, error) {
	var softwareNames []string // 使用 lib/pq 的类型处理 PostgreSQL 数组
	// 确保查询的是 paper 表中 id 对应的 software_names 字段
	sql := `SELECT software_names FROM paper WHERE id = $1`

	row := db.QueryRow(sql, paperID)
	err := row.Scan(&softwareNames)

	if err != nil {
		return nil, err // 如果未找到，返回 sql.ErrNoRows
	}
	return softwareNames, nil
}

// 第一种情况paper不存在 insert
func InsertNewPaper(db *sql.DB, paper Paper) {
	log.Printf("Inserting new paper %s with software", paper.ID)
	sql := `INSERT INTO paper(id, title, authors, abstract, url, pdf, software_names, published_time)
            VALUES($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := db.Exec(
		sql,
		paper.ID,
		paper.Title,
		pq.Array(paper.Authors),
		paper.Abstract,
		paper.URL,
		paper.PDF,
		pq.Array(paper.SoftwareName), // 插入软件名数组
		paper.PublishedTime,
	)
	if err != nil {
		log.Fatalf("Error inserting paper %s: %v", paper.ID, err)
	} else {
		log.Printf("Successfully inserted new paper: %s", paper.ID)
	}
}

// paper存在但是software不存在
func UpdatePaperSoftware(db *sql.DB, paperID string, updatedSoftwareNames []string) {
	log.Printf("Updating software names for existing paper %s. New list: %v", paperID, updatedSoftwareNames)
	//Update software names need to merge existing software names and new software
	//Or we can append the new software in sql
	sql := `UPDATE paper SET software_names = $1 WHERE id = $2`

	_, err := db.Exec(sql, pq.Array(updatedSoftwareNames), paperID)
	if err != nil {
		log.Fatalf("Error updating paper %s: %v", paperID, err)
	} else {
		log.Printf("Successfully updated software names for paper: %s", paperID)
	}
}

// 这是论文详情页的source代码
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

func getPaperFromMetaData(extractedId string) Paper {
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
		return Paper{
			ID:            extractedId,
			Title:         title,
			Authors:       authors,
			Abstract:      abstract,
			URL:           url,
			PDF:           pdf,
			PublishedTime: publishedTime,
		}
	} else {
		url := fmt.Sprintf("https://arxiv.org/abs/%s", extractedId)
		log.Println("source code get,start to match content")
		pdf := MatchPdf(sourceCode)
		publishedTime := MatchSubmissionDate(sourceCode, false, 0)
		return Paper{
			ID:            extractedId,
			Title:         title,
			Authors:       authors,
			Abstract:      abstract,
			URL:           url,
			PDF:           pdf,
			PublishedTime: publishedTime,
		}
	}

}

func parseGuid(seg string) string {
	hrefRe := regexp.MustCompile(`<a.*?title="Abstract".*?>((\s|.)*?)<\/a>`)
	matches := hrefRe.FindAllStringSubmatch(seg, -1)
	// 输出匹配结果和组
	for _, match := range matches {
		if len(match) > 1 {
			return strings.TrimSpace(match[1])
		}
	}
	return ""
}

func parsePage(tt string) int {
	re := regexp.MustCompile(`(\d+)`)
	matches := re.FindAllStringSubmatch(tt, -1)
	number, _ := strconv.Atoi((matches[len(matches)-1][0]))
	return number
}

func parseTotalCount(seg string) int {
	pages := make([]string, 0)
	for _, page := range strings.Split(seg, "<a href") {
		reg := regexp.MustCompile(`>.*?<`)
		p := reg.FindString(page)
		p = strings.TrimLeft(p, ">")
		p = strings.TrimRight(p, "<")
		if p != "" {
			pages = append(pages, p)
		}
	}
	if len(pages) > 0 {
		lastPage := pages[len(pages)-1]
		return parsePage(lastPage)
	} else {
		return -1
	}
}

func traverse(doc string) ([]string, int) {
	linkRe := regexp.MustCompile(linkReg)
	pageRe := regexp.MustCompile(pageReg)

	linkMatches := linkRe.FindAllString(doc, -1)
	pageMatch := pageRe.FindString(doc)
	total := parseTotalCount(pageMatch)
	// 输出匹配结果
	ids := make([]string, 0)
	for _, match := range linkMatches {
		link := parseGuid(match)
		ids = append(ids, link)
	}
	return ids, total
}

func CrawlArchivePage(subject string, month string, offset int, limit int) ([]string, int) {
	url := fmt.Sprintf(CrawlPattern, subject, month)
	params := map[string]string{
		"skip": strconv.Itoa(offset),
		"show": strconv.Itoa(limit),
	}
	result := make([]string, 0)
	req, _ := http.NewRequest("GET", url+"?"+encodeParams(params), nil)

	req.Header.Set("User-Agent", "bingbot")
	client := &http.Client{}
	resp, err := client.Do(req)
	time.Sleep(1000 * time.Millisecond)
	if err != nil {
		log.Println(err)
	}
	// Read the response body
	doc, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error reading response body:", err)
		return result, 0
	} else {
		return traverse(string(doc))
	}
}

const dsn = "postgres://readwise:my_readwise_secret@tcp(%s:5432)/readwise"

func GetEnvVariable(key string, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	} else {
		return value
	}
}

func DBConnectionString() string {
	return fmt.Sprintf(dsn, GetEnvVariable("DB_HOST", "127.0.0.1"))
}

func CheckPaperExist(id string) bool {
	db, err := sql.Open("postgres", DBConnectionString())
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Check if the connection is valid
	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}
	rows, err := db.Query("SELECT id FROM paper where id = $1", id)
	if err != nil {
		log.Fatalf("Error executing query: %v", err)
	}
	defer rows.Close()
	return rows.Next()
}
func InsertPaper(paper Paper) {
	log.Println("save paper ", paper.ID, " to database")
	db, err := sql.Open("postgres", DBConnectionString())
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Check if the connection is valid
	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	now := time.Now()
	timestamp := strconv.FormatInt(now.Unix(), 10)
	stmt, err := db.Prepare("INSERT INTO paper(id,title,abstract_url,pdf,authors,published,source,fetched_at) VALUES($1, $2, $3, $4, $5, $6, $7, $8)")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	res, err := stmt.Exec(paper.ID,
		paper.Title,
		paper.URL,
		paper.PDF,
		paper.Authors,

		"arxiv",
		timestamp)
	if err != nil {
		log.Fatal(err)
	}

	// Get the last inserted ID
	_, err = res.LastInsertId()
	if err != nil {
		log.Fatal(err)
	}

	tt, err := db.Prepare("UPDATE paper SET abstract = $1 WHERE id = $2")

	if err != nil {
		log.Fatal(err)
	}
	defer tt.Close()
	log.Println("try to update paper abstract ", paper.Abstract)
	res, err = tt.Exec(paper.Abstract, paper.ID)
	if err != nil {
		log.Fatal(err)
	}

	// Get the last inserted ID
	_, err = res.LastInsertId()
	if err != nil {
		log.Fatal(err)
	}
}

func CrawlArxivDailyFeed(subject string) {
	guuids, _ := CrawlArchivePage(subject, "new", 0, 1000)
	log.Println(" Total count from page parse: ", len(guuids))
	for _, guid := range guuids {
		id := strings.ReplaceAll(guid, "arXiv:", "")
		log.Println(id)
		if !CheckPaperExist(id) {
			paper := getPaperFromMetaData(id)
			InsertPaper(paper)
		}
	}
}

func CrawlArchiveMonthly(subject string, month string) {
	log.Println("start to fetch papers for ", subject, " on ", month)
	monthResult := make([]string, 0)
	total := 50
	var page []string
	for len(monthResult) < total {
		page, total = CrawlArchivePage(subject, month, len(monthResult), 50)
		//page, _ = CrawlArchivePage(subject, month, len(monthResult), 50)
		for _, guid := range page {
			id := strings.ReplaceAll(guid, "arXiv:", "")
			if !CheckPaperExist(id) {
				paper := getPaperFromMetaData(id)
				InsertPaper(paper)
			}
		}
		monthResult = append(monthResult, page...)
		log.Println("fetched ", len(monthResult), " for  ", month, " => ", subject)
	}
}

func dateRange() []string {
	start := time.Now()
	end := time.Date(2007, time.January, 1, 0, 0, 0, 0, time.UTC)

	// Create a slice to hold the formatted date strings
	var dateStrings []string

	// Iterate from start to end
	for current := start; !current.Before(end); current = current.AddDate(0, -1, 0) {
		dateStrings = append(dateStrings, current.Format("2006-01"))
	}
	return dateStrings
}

func CrawlArchive(subject string) {
	monthArray := dateRange()
	for _, month := range monthArray {
		CrawlArchiveMonthly(subject, month)
	}
}

func CrawlArxivMonthlyFeed(subject string) {
	start := time.Now()
	month := start.Format("2006-01")
	CrawlArchiveMonthly(subject, month)
}

func main() {
	args := os.Args[1:]

	if len(args) == 2 {
		crawlType := args[0]
		category := args[1]
		switch crawlType {
		case "daily":
			log.Println("start to fetch daily papers")
			CrawlArxivDailyFeed(category)
		case "monthly":
			log.Println("start to fetch monthly papers")
			CrawlArxivMonthlyFeed(category)
		case "archive":
			log.Println("start to fetch archive ...")
			CrawlArchive(category)
		case "single":

			if !CheckPaperExist(category) {
				log.Println("start to fetch single paper ...")
				paper := getPaperFromMetaData(category)
				if paper.ID != "" {
					InsertPaper(paper)
				}
			} else {
				log.Println("skip ", category)
			}
		}
	}
}
