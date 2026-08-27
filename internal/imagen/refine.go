package imagen

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

// refineTimeout：扩写提示词是一次很短的对话补全，几秒就该回来。
// 给到 90 秒是留给慢网关，再长就该报错让用户知道，而不是干等。
const refineTimeout = 90 * time.Second

// 默认用的文本模型。和图像模型一样不写死可用清单——这里只是"没填时用哪个"。
const DefaultRefineModel = "gpt-5-mini"

// RefineRequest 是一次提示词扩写。
type RefineRequest struct {
	Prompt string
	Model  string
	// Purpose 决定用哪套指令：texture 要平铺与平光，image 不该有这些约束。
	Purpose string
}

// Refined 是扩写结果。
type Refined struct {
	Prompt  string `json:"prompt"`
	Model   string `json:"model"`
	Usage   Usage  `json:"usage"`
	Elapsed int64  `json:"elapsed_ms"`
}

// 两套系统提示。分开是必要的：
//
// 材质的提示词必须服从"正交俯视、平光无影、可平铺"这些硬约束——那是 PBR 分解
// 能不能work的前提，扩写时把这些丢了，出来的图再好看也没法用。
// 而普通出图恰恰相反，硬塞平光约束等于把画面限死。
const (
	sysTexture = `你是游戏材质贴图的提示词编辑。把用户给的想法扩写成适合图像模型的英文提示词，用于生成**可平铺的材质贴图**。

必须保持的约束：正交俯视视角、均匀平光、没有投影和高光、没有暗角、边缘可无缝拼接、画面充满同一种材质。
补充材质本身的细节：表面构成、磨损与老化、颗粒与纹理尺度、颜色倾向、湿润或干燥。
不要加入：人物、场景、透视、景深、文字、水印、签名、边框。

只输出提示词本身，不要解释，不要引号，不要分点。`

	sysImage = `你是图像生成的提示词编辑。把用户给的想法扩写成适合图像模型的提示词。

补充有助于成像的细节：主体、构图、光线、氛围、材质与色彩、镜头感。
保留用户明确写出的要求，不要替他改主意；用户没提的风格不要擅自加。
不要加入文字、水印、签名。

用户用中文写就输出中文，用英文写就输出英文。只输出提示词本身，不要解释，不要引号。`
)

// Refiner 是能扩写提示词的来源。
//
// 与 Provider 分开成一个接口：出图和扩写用的是同一个网关、同一把令牌，
// 但未必是同一个模型，也未必每个来源都支持对话补全。
type Refiner interface {
	Refine(ctx context.Context, req RefineRequest) (*Refined, error)
}

// refineTokenID 是文本模型那套凭据的 id，见 model.Providers 里的说明。
const refineTokenID = "openai-text"

// textBase 是扩写要打的地址：单独配了就用它，没配就沿用出图那套。
func (o *OpenAI) textBase() string {
	if v := strings.TrimRight(strings.TrimSpace(o.tokens.Endpoint(refineTokenID)), "/"); v != "" {
		return v
	}
	return o.base()
}

// textAuth 同理：单独配了令牌就用它，没配就沿用出图的令牌。
func (o *OpenAI) textAuth(r *http.Request) error {
	tok := o.tokens.Token(refineTokenID)
	if tok == "" {
		tok = o.tokens.Token("openai")
	}
	if tok == "" {
		return ErrNoToken
	}
	r.Header.Set("Authorization", "Bearer "+tok)
	return nil
}

// Refine 调对话补全把提示词扩写开。
func (o *OpenAI) Refine(ctx context.Context, req RefineRequest) (*Refined, error) {
	if o.tokens.Token(refineTokenID) == "" && !o.Configured() {
		return nil, ErrNoToken
	}
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return nil, fmt.Errorf("先写点什么再让它扩写")
	}
	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = DefaultRefineModel
	}
	system := sysImage
	if req.Purpose == "texture" {
		system = sysTexture
	}

	body, err := json.Marshal(map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": prompt},
		},
	})
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, refineTimeout)
	defer cancel()
	r, err := http.NewRequestWithContext(ctx, http.MethodPost,
		o.textBase()+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	r.Header.Set("Content-Type", "application/json")
	if err := o.textAuth(r); err != nil {
		return nil, err
	}

	start := time.Now()
	resp, err := o.http.Do(r)
	if err != nil {
		return nil, describeTransportError(o.textBase(), err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, apiError(resp.StatusCode, raw)
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}
	if err := decodeJSON(resp, raw, "对话接口", &parsed); err != nil {
		return nil, err
	}
	if len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("模型没有返回内容")
	}
	out := strings.TrimSpace(parsed.Choices[0].Message.Content)
	// 模型偶尔会把结果整个套引号，去掉一层。
	out = strings.Trim(out, "\"“”")
	if out == "" {
		return nil, fmt.Errorf("模型返回了空内容")
	}
	return &Refined{
		Prompt: out, Model: model,
		Usage:   Usage{InputTokens: parsed.Usage.PromptTokens, OutputTokens: parsed.Usage.CompletionTokens},
		Elapsed: time.Since(start).Milliseconds(),
	}, nil
}

// textModelCacheTTL 与图像那边同理：清单变得很慢，但也不能一直不刷。
const textModelCacheTTL = 10 * time.Minute

// notTextModel 认出**肯定不能做对话补全**的模型。
//
// 这里用排除法，与图像模型那边的前缀白名单正好相反，因为两者错认的代价不对称：
//
//   - 图像：白名单漏了一个，用户少一个选项；错认一个，点了生成才报错。宁可漏。
//   - 文本：对话模型的命名太散（gpt-*、o*、chatgpt-*，还有网关自己接的
//     deepseek/qwen/claude/glm…），白名单必然漏掉一大片，而这里漏掉的后果是
//     "我的模型在下拉框里找不到"。所以反过来：只把明确不是对话的剔掉。
//
// 剔错了也不致命——下拉框允许直接手输，见设置页那一栏。
var notTextModel = []string{
	"embedding", "embed-",
	"tts", "whisper", "audio", "realtime", "transcribe", "speech",
	"image", "dall-e", "vision-preview",
	"moderation", "omni-moderation", "guard",
	"rerank", "search-", "codex-mini-latest",
	"babbage", "davinci", "ada-", "curie",
}

func isTextModel(id string) bool {
	l := strings.ToLower(id)
	for _, bad := range notTextModel {
		if strings.Contains(l, bad) {
			return false
		}
	}
	return true
}

// TextModels 列出扩写能用的文本模型。
//
// 走 textBase / textAuth 而不是出图那套：这两者可以配在不同的网关上，
// 拿出图网关的清单去填扩写的下拉框，填出来的东西一个都用不了。
func (o *OpenAI) TextModels(ctx context.Context) ([]string, error) {
	base := o.textBase()

	o.textMu.Lock()
	if o.textCacheKey == base && time.Since(o.textCachedAt) < textModelCacheTTL &&
		len(o.textCache) > 0 {
		out := append([]string{}, o.textCache...)
		o.textMu.Unlock()
		return out, nil
	}
	o.textMu.Unlock()

	ids, err := o.listModels(ctx, base, o.textAuth)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if isTextModel(id) {
			out = append(out, id)
		}
	}
	sort.Strings(out)

	o.textMu.Lock()
	o.textCache, o.textCacheKey, o.textCachedAt = append([]string{}, out...), base, time.Now()
	o.textMu.Unlock()
	return out, nil
}
