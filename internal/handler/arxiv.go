package handler

import (
	"bytes"
	"database/sql"
	"encoding/xml"
	"fmt"
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

	"github.com/ledongthuc/pdf"

)

type PaperResponse struct {
	Success bool  `json:"success"`
	Data    Paper `json:"data"`
}

type Paper struct {
	ID           string `json:"ID"`
	Title        string `json:"Title"`
	Authors      string `json:"Authors"`
	Abstract     string `json:"Abstract"`
	URL          string `json:"URL"`
	PDF          string `json:"Pdf"`
	SoftwareName string `json:"SoftwareName"`
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

func readPdf(path string) (string, error) {
	f, r, err := pdf.Open(path)
	if err != nil {
		return "", fmt.Errorf("error opening PDF: %w", err)
	}
	defer f.Close()

	var buf bytes.Buffer
	b, err := r.GetPlainText()
	if err != nil {
		return "", fmt.Errorf("error getting plain text from PDF: %w", err)
	}
	buf.ReadFrom(b)
	content := buf.String()
	return content, nil
}

func searchPDFMultipleText(filePath string, search ...string) []string {
	result := make([]string, 0)
	fullContext, err := readPdf(filePath)

	if err != nil {
		log.Printf("Failed to read PDF file %s: %v", filePath, err)
		return result
	}

	lowerText := strings.ToLower(fullContext)
	for _, text := range search {
		lowerSearchText := strings.ToLower(text)
		if strings.Contains(lowerText, lowerSearchText) {
			result = append(result, text)
		}
	}
	return result
}
func downloadPDFToTemp(pdfURL, paperID string) (string, error) {
	resp, err := http.Get(pdfURL)
	if err != nil {
		return "", fmt.Errorf("error downloading PDF: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error: received non-200 status code %d", resp.StatusCode)
	}

	// 使用 os.CreateTemp 创建一个临时文件
	tempFile, err := os.CreateTemp("", fmt.Sprintf("arxiv-%s-*.pdf", paperID))
	if err != nil {
		return "", fmt.Errorf("error creating temporary file: %w", err)
	}
	defer tempFile.Close() // 确保在函数退出时关闭文件句柄

	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		os.Remove(tempFile.Name()) // 如果写入失败，删除文件
		return "", fmt.Errorf("error saving PDF: %w", err)
	}
	log.Printf("Successfully downloaded PDF to temporary file %s", tempFile.Name())
	return tempFile.Name(), nil
}
func encodeParams(params map[string]string) string {
	values := make(url.Values)
	for key, value := range params {
		values.Add(key, value)
	}
	return values.Encode()
}

const CrawlPattern = "https://arxiv.org/list/%s/%s"

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

func FormatPageUrl(id string, isWithDrawn bool, version int) string {
	if isWithDrawn {
		return fmt.Sprintf("https://arxiv.org/abs/%sv%d", id, version)
	} else {
		return fmt.Sprintf("https://arxiv.org/abs/%s", id)
	}
}

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
			ID:          extractedId,
			Title:       title,
			Authors:     authors,
			Abstract:    abstract,
			URL:         url,
			SoftwareName : ,
		}
	} else {
		url := fmt.Sprintf("https://arxiv.org/abs/%s", extractedId)
		log.Println("source code get,start to match content")
		pdf := MatchPdf(sourceCode)
		publishedTime := MatchSubmissionDate(sourceCode, false, 0)
		return Paper{
			ID:          extractedId,
			Title:       title,
			Authors:     authors,
			Abstract:    abstract,
			AbstractURL: url,
			PDF:         pdf,
			Published:   publishedTime,
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
