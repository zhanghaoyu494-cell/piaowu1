package ai

// 包含NLP自然语言处理、文本分类、敏感信息脱敏等功能
// 主要用于消息自动分类和智能分析
import (
	"encoding/json"
	"math"
	"regexp"
	"sort"
	"strings"
	"sync"
	"unicode"
)

// ============ 中文分词器 ============

// Tokenizer 中文分词器
// 基于词典的正向最大匹配算法
type Tokenizer struct {
	dict      map[string]int  // 词典：词 -> 词频
	maxLen    int             // 最大词长
	stopWords map[string]bool // 停用词表
	mu        sync.RWMutex
}

// NewTokenizer 创建分词器
func NewTokenizer() *Tokenizer {
	t := &Tokenizer{
		dict:      make(map[string]int),
		maxLen:    5,
		stopWords: make(map[string]bool),
	}
	t.initDict()
	t.initStopWords()
	return t
}

// initDict 初始化客服领域词典
func (t *Tokenizer) initDict() {
	words := []string{
		// 问题类型
		"退款", "退货", "发货", "物流", "快递", "配送", "订单", "商品", "产品",
		"投诉", "建议", "反馈", "咨询", "帮助", "问题", "售后", "维修", "换货",
		"价格", "优惠", "活动", "促销", "折扣", "优惠券", "红包", "满减",
		"账号", "密码", "登录", "注册", "会员", "积分", "充值", "提现",
		"支付", "付款", "转账", "余额", "银行卡", "微信", "支付宝",
		// 情感词
		"满意", "不满", "生气", "着急", "焦虑", "感谢", "抱歉", "失望",
		"好评", "差评", "举报", "催促", "急", "紧急", "尽快",
		// 业务词
		"发票", "收据", "保修", "质保", "售后服务", "客服", "人工",
		"取消订单", "修改地址", "修改订单", "申请退款", "退款进度",
		"物流查询", "快递查询", "配送时间", "送货上门", "自提",
		"质量问题", "尺寸问题", "颜色问题", "破损", "缺货", "延迟",
	}
	for _, w := range words {
		t.dict[w] = 100
		if len([]rune(w)) > t.maxLen {
			t.maxLen = len([]rune(w))
		}
	}
}

// initStopWords 初始化停用词
func (t *Tokenizer) initStopWords() {
	stops := []string{
		"的", "了", "是", "在", "我", "有", "和", "就", "不", "人", "都", "一",
		"这", "上", "也", "很", "到", "说", "要", "去", "你", "会", "着", "没有",
		"看", "好", "自己", "个", "所以", "什么", "但是", "因为", "如果", "那",
		"吗", "呢", "吧", "啊", "哦", "嗯", "哈", "呀", "啦", "么", "呐",
	}
	for _, w := range stops {
		t.stopWords[w] = true
	}
}

// AddWord 添加自定义词到词典
func (t *Tokenizer) AddWord(word string, freq int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.dict[word] = freq
	if len([]rune(word)) > t.maxLen {
		t.maxLen = len([]rune(word))
	}
}

// Tokenize 分词（正向最大匹配）
func (t *Tokenizer) Tokenize(text string) []string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	text = strings.ToLower(text)
	runes := []rune(text)
	var tokens []string
	i := 0
	for i < len(runes) {
		matched := false
		for l := t.maxLen; l > 0; l-- {
			if i+l > len(runes) {
				continue
			}
			word := string(runes[i : i+l])
			if _, ok := t.dict[word]; ok {
				if !t.stopWords[word] {
					tokens = append(tokens, word)
				}
				i += l
				matched = true
				break
			}
		}
		if !matched {
			ch := runes[i]
			if unicode.Is(unicode.Han, ch) {
				w := string(ch)
				if !t.stopWords[w] {
					tokens = append(tokens, w)
				}
			}
			i++
		}
	}
	return tokens
}

// ============ TF-IDF 计算器 ============

// TFIDF TF-IDF计算器
type TFIDF struct {
	docFreq   map[string]int // 文档频率
	totalDocs int            // 总文档数
	tokenizer *Tokenizer
	mu        sync.RWMutex
}

// NewTFIDF 创建TF-IDF计算器
func NewTFIDF() *TFIDF {
	return &TFIDF{
		docFreq:   make(map[string]int),
		tokenizer: NewTokenizer(),
	}
}

// AddDocument 添加文档用于IDF计算
func (t *TFIDF) AddDocument(text string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	tokens := t.tokenizer.Tokenize(text)
	seen := make(map[string]bool)
	for _, token := range tokens {
		if !seen[token] {
			t.docFreq[token]++
			seen[token] = true
		}
	}
	t.totalDocs++
}

// ComputeVector 计算文本的TF-IDF向量
func (t *TFIDF) ComputeVector(text string) map[string]float64 {
	t.mu.RLock()
	defer t.mu.RUnlock()

	tokens := t.tokenizer.Tokenize(text)
	if len(tokens) == 0 {
		return make(map[string]float64)
	}

	tf := make(map[string]int)
	for _, token := range tokens {
		tf[token]++
	}

	vector := make(map[string]float64)
	for token, count := range tf {
		tfVal := float64(count) / float64(len(tokens))
		df := t.docFreq[token]
		if df == 0 {
			df = 1
		}
		idfVal := math.Log(float64(t.totalDocs+1) / float64(df+1))
		vector[token] = tfVal * idfVal
	}
	return vector
}

// CosineSimilarity 计算两个向量的余弦相似度
func CosineSimilarity(v1, v2 map[string]float64) float64 {
	var dotProduct, norm1, norm2 float64
	for k, val1 := range v1 {
		if val2, ok := v2[k]; ok {
			dotProduct += val1 * val2
		}
		norm1 += val1 * val1
	}
	for _, val2 := range v2 {
		norm2 += val2 * val2
	}
	if norm1 == 0 || norm2 == 0 {
		return 0
	}
	return dotProduct / (math.Sqrt(norm1) * math.Sqrt(norm2))
}

// ============ NLP分类器 ============

// CategoryProfile 分类特征模型
type CategoryProfile struct {
	CategoryID   int64
	CategoryName string
	Keywords     []string           // 关键词列表
	TFIDFVector  map[string]float64 // TF-IDF特征向量
}

// Classifier NLP文本分类器
// 采用多策略融合分类：关键词匹配 + TF-IDF相似度 + 语义规则
type Classifier struct {
	profiles  []*CategoryProfile
	tfidf     *TFIDF
	tokenizer *Tokenizer
	mu        sync.RWMutex
}

// NewClassifier 创建分类器
func NewClassifier() *Classifier {
	return &Classifier{
		tfidf:     NewTFIDF(),
		tokenizer: NewTokenizer(),
	}
}

// AddCategory 添加分类及其关键词
func (c *Classifier) AddCategory(id int64, name string, keywords []string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	profile := &CategoryProfile{
		CategoryID:   id,
		CategoryName: name,
		Keywords:     keywords,
		TFIDFVector:  make(map[string]float64),
	}

	// 使用关键词构建特征向量
	for _, kw := range keywords {
		c.tfidf.AddDocument(kw)
	}
	keywordText := strings.Join(keywords, " ")
	profile.TFIDFVector = c.tfidf.ComputeVector(keywordText)

	c.profiles = append(c.profiles, profile)
}

// ClassifyResult 分类结果
type ClassifyResult struct {
	CategoryID      int64
	CategoryName    string
	Confidence      float64
	MatchedKeywords []string
	NeedManual      bool
}

// Classify 对文本进行分类
// 采用多策略融合：关键词匹配(40%) + TF-IDF相似度(40%) + 语义规则(20%)
func (c *Classifier) Classify(text string) *ClassifyResult {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.profiles) == 0 {
		return &ClassifyResult{
			CategoryID:   0,
			CategoryName: "未分类",
			Confidence:   0,
			NeedManual:   true,
		}
	}

	textLower := strings.ToLower(text)
	tokens := c.tokenizer.Tokenize(text)
	textVector := c.tfidf.ComputeVector(text)

	type categoryScore struct {
		profile         *CategoryProfile
		keywordScore    float64
		tfidfScore      float64
		semanticScore   float64
		totalScore      float64
		matchedKeywords []string
	}

	var scores []categoryScore

	for _, profile := range c.profiles {
		var matched []string
		keywordHits := 0

		// 1. 关键词匹配评分
		for _, kw := range profile.Keywords {
			kwLower := strings.ToLower(kw)
			if strings.Contains(textLower, kwLower) {
				keywordHits++
				matched = append(matched, kw)
			}
		}
		kwScore := float64(0)
		if len(profile.Keywords) > 0 {
			kwScore = float64(keywordHits) / float64(len(profile.Keywords))
		}

		// 2. TF-IDF余弦相似度评分
		tfidfScore := CosineSimilarity(textVector, profile.TFIDFVector)

		// 3. 语义规则评分（基于分词结果的覆盖率）
		semanticHits := 0
		for _, token := range tokens {
			for _, kw := range profile.Keywords {
				if strings.Contains(kw, token) || strings.Contains(token, kw) {
					semanticHits++
					break
				}
			}
		}
		semanticScore := float64(0)
		if len(tokens) > 0 {
			semanticScore = float64(semanticHits) / float64(len(tokens))
		}

		// 融合评分：关键词40% + TF-IDF40% + 语义20%
		totalScore := kwScore*0.4 + tfidfScore*0.4 + semanticScore*0.2

		scores = append(scores, categoryScore{
			profile:         profile,
			keywordScore:    kwScore,
			tfidfScore:      tfidfScore,
			semanticScore:   semanticScore,
			totalScore:      totalScore,
			matchedKeywords: matched,
		})
	}

	// 按总分排序
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].totalScore > scores[j].totalScore
	})

	best := scores[0]
	confidence := best.totalScore

	// 置信度阈值判断
	needManual := true
	if confidence >= 0.5 {
		needManual = false
	} else if confidence >= 0.3 && len(best.matchedKeywords) >= 2 {
		needManual = false
	}

	return &ClassifyResult{
		CategoryID:      best.profile.CategoryID,
		CategoryName:    best.profile.CategoryName,
		Confidence:      confidence,
		MatchedKeywords: best.matchedKeywords,
		NeedManual:      needManual,
	}
}

// ============ 敏感信息脱敏 ============

// Desensitizer 敏感信息脱敏器
type Desensitizer struct {
	patterns map[string]*regexp.Regexp
}

// NewDesensitizer 创建脱敏器
func NewDesensitizer() *Desensitizer {
	d := &Desensitizer{
		patterns: make(map[string]*regexp.Regexp),
	}
	d.initPatterns()
	return d
}

// initPatterns 初始化脱敏规则
func (d *Desensitizer) initPatterns() {
	d.patterns["phone"] = regexp.MustCompile(`1[3-9]\d{9}`)
	d.patterns["idcard"] = regexp.MustCompile(`[1-9]\d{5}(18|19|20)\d{2}(0[1-9]|1[0-2])(0[1-9]|[12]\d|3[01])\d{3}[\dXx]`)
	d.patterns["bankcard"] = regexp.MustCompile(`\d{16,19}`)
	d.patterns["email"] = regexp.MustCompile(`[\w.-]+@[\w.-]+\.\w+`)
}

// Desensitize 对文本进行脱敏处理
func (d *Desensitizer) Desensitize(text string) string {
	result := text

	// 手机号脱敏: 138****1234
	result = d.patterns["phone"].ReplaceAllStringFunc(result, func(s string) string {
		if len(s) == 11 {
			return s[:3] + "****" + s[7:]
		}
		return s
	})

	// 身份证脱敏: 110***********1234
	result = d.patterns["idcard"].ReplaceAllStringFunc(result, func(s string) string {
		if len(s) >= 15 {
			return s[:3] + strings.Repeat("*", len(s)-7) + s[len(s)-4:]
		}
		return s
	})

	// 银行卡脱敏: 6222****1234
	result = d.patterns["bankcard"].ReplaceAllStringFunc(result, func(s string) string {
		if len(s) >= 16 {
			return s[:4] + strings.Repeat("*", len(s)-8) + s[len(s)-4:]
		}
		return s
	})

	// 邮箱脱敏: t***@example.com
	result = d.patterns["email"].ReplaceAllStringFunc(result, func(s string) string {
		parts := strings.Split(s, "@")
		if len(parts) == 2 && len(parts[0]) > 1 {
			return parts[0][:1] + "***@" + parts[1]
		}
		return s
	})

	return result
}

// DetectSensitiveInfo 检测敏感信息
func (d *Desensitizer) DetectSensitiveInfo(text string) []string {
	var found []string
	for name, pattern := range d.patterns {
		if pattern.MatchString(text) {
			found = append(found, name)
		}
	}
	return found
}

// ============ 辅助函数 ============

// ParseKeywordsJSON 解析关键词JSON字符串
func ParseKeywordsJSON(keywordsJSON string) []string {
	if keywordsJSON == "" {
		return nil
	}
	var keywords []string
	if err := json.Unmarshal([]byte(keywordsJSON), &keywords); err != nil {
		return nil
	}
	return keywords
}
