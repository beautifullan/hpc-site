package handler

import (
	"fmt"
	"log"
	"reflect"
	"testing"
	"time"
)

func TestFetchArxivSearchHtml(t *testing.T) {
	html, err := FetchArxivSearchHtml("lammps", 0)
	if err != nil {
		log.Fatal("Fetch failed:", err)
	}
	fmt.Println("HTML fetched successfully, length =", len(html))
	fmt.Println(html[:800]) // 打印前 800 个字符看看是否是 arXiv 页面
}

// TestGetArxivIDsFromSearchHtml 单元测试 GetArxivIDsFromSearchHtml 函数
func TestGetArxivIDsFromSearchHtml(t *testing.T) {
	tests := []struct {
		name    string
		html    string
		wantIDs []string
		wantLen int
	}{
		{
			name: "Success_TwoUniqueIDs",
			html: `
           <li class="arxiv-result">
   <div class="is-marginless">
     <p class="list-title is-inline-block"><a href="https://arxiv.org/abs/2508.15522">arXiv:2508.15522</a>
       <span>&nbsp;[<a href="https://arxiv.org/pdf/2508.15522">pdf</a>, <a href="https://arxiv.org/ps/2508.15522">ps</a>, <a href="https://arxiv.org/format/2508.15522">other</a>]&nbsp;</span>
     </p>
     <div class="tags is-inline-block">
       <span class="tag is-small is-link tooltip is-tooltip-top" data-tooltip="Chemical Physics">physics.chem-ph</span>
       </div>

   </div>

   <p class="title is-5 mathjax">

       GridFF: Efficient Simulation of Organic Molecules on Rigid Substrates

   </p>
   <p class="authors">
     <span class="has-text-black-bis has-text-weight-semibold">Authors:</span>

     <a href="/search/?searchtype=author&amp;query=Mal%2C+I">Indranil Mal</a>,

     <a href="/search/?searchtype=author&amp;query=Ko%C4%8D%C3%AD%2C+M">Milan Kočí</a>,

     <a href="/search/?searchtype=author&amp;query=Nicolini%2C+P">Paolo Nicolini</a>,

     <a href="/search/?searchtype=author&amp;query=Hapala%2C+P">Prokop Hapala</a>

   </p>


   <p class="is-size-7"><span class="has-text-black-bis has-text-weight-semibold">Submitted</span> 21 August, 2025;
     <span class="has-text-black-bis has-text-weight-semibold">originally announced</span> August 2025.

   </p>





 </li>
           `,
			wantIDs: []string{"2508.15522"},
			wantLen: 1,
		},
		{
			name: "Success_HandleDuplicates",
			html: `
           <li class="arxiv-result"><p><a href="https://arxiv.org/abs/2501.11111">ID1</a></p></li>
           <li class="arxiv-result"><p><a href="https://arxiv.org/abs/2501.11111">ID1</a></p></li>
           <li class="arxiv-result"><p><a href="https://arxiv.org/abs/2502.22222">ID2</a></p></li>
           `,
			wantIDs: []string{"2501.11111", "2502.22222"},
			wantLen: 2,
		},
		{
			name:    "EmptyHtml",
			html:    "",
			wantIDs: []string{},
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotIDs := GetArxivIDsFromSearchHtml(tt.html)

			if len(gotIDs) != tt.wantLen {
				t.Errorf("GetArxivIDsFromSearchHtml() returned wrong length. Got %d, want %d. IDs: %v", len(gotIDs), tt.wantLen, gotIDs)
			}

			// 验证提取出的内容是否一致
			if !reflect.DeepEqual(gotIDs, tt.wantIDs) {
				t.Errorf("GetArxivIDsFromSearchHtml() got %v, want %v", gotIDs, tt.wantIDs)
			}
		})
	}
}

// 测试分页

func TestGetArxivBySoftwarePagination_All(t *testing.T) {
	software := "lammps"
	log.Printf("🚀 开始测试分页抓取：%s", software)

	html, err := FetchArxivSearchHtml(software, 0)
	if err != nil {
		t.Fatalf("抓取第一页失败：%v", err)
	}

	total := MatchTotalResults(html)
	if total == 0 {
		t.Fatalf("未找到任何论文，总数为 0")
	}
	log.Printf("📦 共 %d 篇论文", total)

	page := 1
	start := 0
	allIDs := make(map[string]bool)
	emptyCount := 0 // 连续空页计数

	for start < total && emptyCount < 3 {
		log.Printf("🧭 抓取第 %d 页 (start=%d)...", page, start)
		html, err := FetchArxivSearchHtml(software, start)
		if err != nil {
			t.Fatalf("第 %d 页抓取失败: %v", page, err)
			emptyCount++
			time.Sleep(5 * time.Second)
			continue
		}

		ids := GetArxivIDsFromSearchHtml(html)
		if len(ids) == 0 {
			log.Printf("⚠️ 第 %d 页为空页，重试一次", page)
			time.Sleep(5 * time.Second)
			emptyCount++
			continue
		}

		emptyCount = 0 // 重置空页计数
		for _, id := range ids {
			allIDs[id] = true
		}

		fmt.Printf("✅ 第 %d 页解析到 %d 篇论文，示例: %v\n", page, len(ids), ids[:min(len(ids), 3)])
		start += SearchPageSize
		page++

		time.Sleep(2 * time.Second)
	}

	log.Printf("🎯 共抓取到 %d 篇唯一论文", len(allIDs))
	log.Println("✅ 全量分页抓取逻辑测试完成")
}
