package imagen

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"
	"sync"
	"time"
)

// DefaultOpenAIBase 是官方地址。可以改成兼容网关（例如自建中转），
// 但改了之后计费与可用模型都以那边为准。
const DefaultOpenAIBase = "https://api.openai.com/v1"

// openaiTimeout 是单次生成的上限。
//
// 给到 6 分钟是有依据的：高质量档的图像生成是同步接口，服务端要跑完整个
// 采样才返回，实测中高档常在一分钟以上。给短了会在快出结果时把连接掐断，
// 而钱照扣。
const openaiTimeout = 6 * time.Minute

// price 是每百万 token 的价格，美元。
type price struct{ TextIn, ImageIn, Out float64 }

// pricePerMTok 只用于在界面上估个花费，不是账单。
//
// 查不到的模型不显示估算，而不是拿别的模型的价格顶上——宁可不说，
// 也不要报个错的数字。真实花费一律以响应里的 usage 为准。
var pricePerMTok = map[string]price{
	"gpt-image-2":      {TextIn: 5, ImageIn: 8, Out: 30},
	"gpt-image-1.5":    {TextIn: 5, ImageIn: 8, Out: 32},
	"gpt-image-1":      {TextIn: 5, ImageIn: 10, Out: 40},
	"gpt-image-1-mini": {TextIn: 2, ImageIn: 2.5, Out: 8},
}

// gptImageSizes 是 gpt-image 系列已验证可用的尺寸。
//
// 服务端实际接受任意 WxH，只要满足 §sizeOK 的四条约束；这里列出的是
// 做纹理用得上的那些。正方形止步 2880——2880² = 8,294,400 正好是总像素
// 上限，4096² 要不到。
var gptImageSizes = []string{
	"auto", "1024x1024", "1536x1024", "1024x1536",
	"2048x2048", "2048x1152", "1152x2048", "2880x2880", "3840x2160", "2160x3840",
}

var gptImageQualities = []string{"low", "medium", "high", "auto"}

// knownModels 是我们认得参数表的模型。
//
// 这张表只决定"标签和参数选项"，不决定"有哪些模型可用"——可用模型一律
// 向 /v1/models 现问。写死一份清单迟早会过期：新模型发布时用户明明有权限，
// 工具却不认；下架的老模型又一直挂在那儿（dall-e 系列已于 2026-05-12 下线，
// 就没往这张表里放）。
var knownModels = []Model{
	{
		ID: "gpt-image-2", Label: "GPT Image 2",
		Sizes: gptImageSizes, Qualities: gptImageQualities, Edits: true,
		Note: "当前旗舰。图内文字与符号是它的强项；high 档单张要 2~4 分钟",
	},
	{
		ID: "gpt-image-1.5", Label: "GPT Image 1.5",
		Sizes: gptImageSizes, Qualities: gptImageQualities, Edits: true,
		Note: "上一代，比 2 代快",
	},
	{
		ID: "gpt-image-1", Label: "GPT Image 1",
		Sizes: gptImageSizes, Qualities: gptImageQualities, Edits: true,
	},
	{
		ID: "gpt-image-1-mini", Label: "GPT Image 1 mini",
		Sizes: gptImageSizes, Qualities: gptImageQualities, Edits: true,
		Note: "最便宜的一档，适合先大量试提示词再上大模型",
	},
}

// imageModelPrefixes 用于从 /v1/models 的全量清单里挑出图像模型。
//
// OpenAI 的模型列表把对话、向量、语音、图像混在一起，且没有类型字段可用，
// 只能按 id 前缀认。宁可漏认也不要错认：错认的后果是用户选了个不能出图的
// 模型，点生成才报错。
var imageModelPrefixes = []string{"gpt-image"}

// sizeOK 校验尺寸是否满足 gpt-image 系列的约束。
//
// 在本地先校验，是为了不让用户等一趟往返（high 档要好几分钟）才收到 400。
// 四条约束来自官方文档：边长须为 16 的倍数、长边不超过 3840、
// 宽高比不超过 3:1、总像素落在 [655360, 8294400]。
func sizeOK(s string) error {
	if s == "" || strings.EqualFold(s, "auto") {
		return nil
	}
	w, h, ok := ParseSize(s)
	if !ok {
		return fmt.Errorf("尺寸 %q 不是 宽x高 的形式", s)
	}
	switch {
	case w%16 != 0 || h%16 != 0:
		return fmt.Errorf("尺寸 %dx%d 无效：两边都必须是 16 的倍数", w, h)
	case w > 3840 || h > 3840:
		return fmt.Errorf("尺寸 %dx%d 无效：长边不能超过 3840", w, h)
	case w*3 < h || h*3 < w:
		return fmt.Errorf("尺寸 %dx%d 无效：宽高比不能超过 3:1", w, h)
	case w*h < 655360:
		return fmt.Errorf("尺寸 %dx%d 无效：总像素不足 655,360", w, h)
	case w*h > 8294400:
		return fmt.Errorf("尺寸 %dx%d 无效：总像素超过 8,294,400（正方形最大 2880x2880）", w, h)
	}
	return nil
}

// OpenAI 实现 Provider。
type OpenAI struct {
	// fallback 是配置文件里写的地址，用户没在设置页填时用它；
	// 两个都空则用官方地址。
	fallback string
	tokens   TokenSource
	http     *http.Client

	mu       sync.Mutex
	cache    []Model
	cacheKey string // 缓存对应的 base，换了网关必须重新问
	cachedAt time.Time

	// 文本模型单独缓存：它可能来自另一个网关，和上面那份不是一回事。
	textMu       sync.Mutex
	textCache    []string
	textCacheKey string
	textCachedAt time.Time
}

// NewOpenAI 造一个 OpenAI 来源。fallback 留空用官方地址。
func NewOpenAI(fallback string, tokens TokenSource, tr *http.Transport) *OpenAI {
	return &OpenAI{
		fallback: strings.TrimRight(strings.TrimSpace(fallback), "/"),
		tokens:   tokens,
		http:     &http.Client{Transport: tr, Timeout: openaiTimeout},
	}
}

func (o *OpenAI) ID() string    { return "openai" }
func (o *OpenAI) Label() string { return "OpenAI" }

// base 每次现取：用户在设置页换网关之后不该还要重启后端。
// 优先级是 设置页 > 配置文件 > 官方地址。
func (o *OpenAI) base() string {
	if v := strings.TrimRight(strings.TrimSpace(o.tokens.Endpoint("openai")), "/"); v != "" {
		return v
	}
	if o.fallback != "" {
		return o.fallback
	}
	return DefaultOpenAIBase
}

func (o *OpenAI) Configured() bool { return o.tokens.Token("openai") != "" }

func (o *OpenAI) auth(r *http.Request) error {
	tok := o.tokens.Token("openai")
	if tok == "" {
		return ErrNoToken
	}
	r.Header.Set("Authorization", "Bearer "+tok)
	return nil
}

// modelCacheTTL：模型清单变化很慢，但也不能一直不刷——用户可能刚开通新模型权限。
const modelCacheTTL = 10 * time.Minute

// Models 向服务端要可用模型，失败时退回内置清单。
//
// 退回而不是报错，是因为"列不出模型"不该让整个功能不可用：网络抖一下、
// 或者令牌只有生成权限没有列举权限，都不影响实际出图。
func (o *OpenAI) Models(ctx context.Context) ([]Model, error) {
	if !o.Configured() {
		return nil, ErrNoToken
	}
	base := o.base()
	o.mu.Lock()
	if o.cacheKey == base && time.Since(o.cachedAt) < modelCacheTTL && len(o.cache) > 0 {
		out := append([]Model{}, o.cache...)
		o.mu.Unlock()
		return out, nil
	}
	o.mu.Unlock()

	remote, err := o.listRemote(ctx)
	if err != nil {
		return o.fallbackModels(), nil //nolint:nilerr // 见上方注释：列不出不等于不能用
	}

	byID := map[string]Model{}
	for _, m := range knownModels {
		m.Provider = o.ID()
		m.Known = true
		byID[m.ID] = m
	}
	out := make([]Model, 0, len(remote))
	for _, id := range remote {
		if m, ok := byID[id]; ok {
			out = append(out, m)
			continue
		}
		// 服务端有、我们不认识：仍然给出来，但参数表按同族的通用值猜，
		// 并在界面上标明是"未收录"，出错时用户知道该往哪想。
		out = append(out, Model{
			ID: id, Label: id, Provider: o.ID(), Known: false,
			Sizes:     []string{"1024x1024", "1024x1536", "1536x1024", "auto"},
			Qualities: []string{"low", "medium", "high"},
			Edits:     true,
			Note:      "未收录的模型，参数按 gpt-image 系列推定，可能不完全适用",
		})
	}
	if len(out) == 0 {
		out = o.fallbackModels()
	}
	sortModels(out)

	o.mu.Lock()
	o.cache, o.cacheKey, o.cachedAt = append([]Model{}, out...), base, time.Now()
	o.mu.Unlock()
	return out, nil
}

func (o *OpenAI) fallbackModels() []Model {
	out := make([]Model, 0, len(knownModels))
	for _, m := range knownModels {
		m.Provider = o.ID()
		m.Known = true
		out = append(out, m)
	}
	return out
}

// Ping 探一次可达性与鉴权，给环境自检用。
//
// 值得单独打一枪：这条链路上能出错的地方特别多——没配令牌、代理没起来、
// 组织没做身份验证、余额不足。等用户提交了任务再卡上几分钟才报错太糟了，
// 尤其 high 档本来就要两三分钟，用户分不清是在生成还是已经挂了。
func (o *OpenAI) Ping(ctx context.Context) (detail string, err error) {
	if !o.Configured() {
		return "", ErrNoToken
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	base := o.base()
	start := time.Now()
	ids, err := o.listRemote(ctx)
	if err != nil {
		return "", err
	}
	// 顺带报出实际打的是哪个地址：用了自定义网关时，这一行是唯一能确认
	// "请求真的走了我配的那台机器"的地方。
	where := ""
	if base != DefaultOpenAIBase {
		if u, e := url.Parse(base); e == nil && u.Host != "" {
			where = "，经 " + u.Host
		}
	}
	return fmt.Sprintf("可达，%d 个图像模型可用（%s%s）",
		len(ids), time.Since(start).Round(10*time.Millisecond), where), nil
}

// listRemote 拉 /v1/models 并筛出图像模型的 id。
func (o *OpenAI) listRemote(ctx context.Context) ([]string, error) {
	ids, err := o.listModels(ctx, o.base(), o.auth)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, id := range ids {
		for _, p := range imageModelPrefixes {
			if strings.HasPrefix(id, p) {
				out = append(out, id)
				break
			}
		}
	}
	return out, nil
}

// listModels 拉一份 /models 的全量 id。
//
// base 与鉴权都由调用方给：出图和扩写可以配在两个不同的网关上
// （见 model.Providers 里 openai-text 那一条），拉列表自然也得各拉各的。
func (o *OpenAI) listModels(ctx context.Context, base string,
	auth func(*http.Request) error) ([]string, error) {

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/models", nil)
	if err != nil {
		return nil, err
	}
	if err := auth(req); err != nil {
		return nil, err
	}
	resp, err := o.http.Do(req)
	if err != nil {
		return nil, describeTransportError(base, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, apiError(resp.StatusCode, body)
	}
	var parsed struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := decodeJSON(resp, body, "模型列表接口", &parsed); err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(parsed.Data))
	for _, d := range parsed.Data {
		if d.ID != "" {
			ids = append(ids, d.ID)
		}
	}
	return ids, nil
}

type openaiImageResponse struct {
	Data []struct {
		B64      string `json:"b64_json"`
		URL      string `json:"url"`
		Revised  string `json:"revised_prompt"`
		Refusal  string `json:"refusal"`
		MimeType string `json:"output_format"`
	} `json:"data"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
		TotalTokens  int `json:"total_tokens"`
		InputDetails struct {
			TextTokens  int `json:"text_tokens"`
			ImageTokens int `json:"image_tokens"`
		} `json:"input_tokens_details"`
	} `json:"usage"`
}

// cost 按各档单价算这次调用的花费。
//
// 输入侧要分文字和图片：图生图时图片 token 的单价是文字的 1.6 倍，而
// gpt-image-2 强制按高保真处理输入图，编辑请求的图片 token 会明显偏高。
// 拿总 input_tokens 一把乘会低估。
func (r *openaiImageResponse) cost(model string) float64 {
	p, ok := pricePerMTok[model]
	if !ok {
		return 0
	}
	u := r.Usage
	textIn, imgIn := u.InputDetails.TextTokens, u.InputDetails.ImageTokens
	// 服务端没给明细时退回总量，按文字价算——宁可估低也不要凭空按图片价放大。
	if textIn == 0 && imgIn == 0 {
		textIn = u.InputTokens
	}
	return (float64(textIn)*p.TextIn + float64(imgIn)*p.ImageIn + float64(u.OutputTokens)*p.Out) / 1e6
}

// Generate 出一张底图。有 Reference 走编辑接口，否则走生成接口。
func (o *OpenAI) Generate(ctx context.Context, req Request) (*Result, error) {
	if !o.Configured() {
		return nil, ErrNoToken
	}
	if strings.TrimSpace(req.Prompt) == "" {
		return nil, fmt.Errorf("提示词不能为空")
	}
	if req.Model == "" {
		return nil, fmt.Errorf("没有指定模型")
	}
	// 提示词上限 32000 字符。本地先拦，省一趟往返。
	if n := len([]rune(req.Prompt)); n > 32000 {
		return nil, fmt.Errorf("提示词 %d 字超过上限 32000 字", n)
	}
	if err := sizeOK(req.Size); err != nil {
		return nil, err
	}

	start := time.Now()
	var (
		httpReq *http.Request
		err     error
	)
	if len(req.Reference) > 0 {
		httpReq, err = o.editRequest(ctx, req)
	} else {
		httpReq, err = o.generateRequest(ctx, req)
	}
	if err != nil {
		return nil, err
	}
	if err := o.auth(httpReq); err != nil {
		return nil, err
	}

	resp, err := o.http.Do(httpReq)
	if err != nil {
		return nil, describeTransportError(o.base(), err)
	}
	defer resp.Body.Close()

	// 单张 1024 的 PNG base64 大约 2~4MB，留足余量但仍设上限，
	// 免得服务端异常时把内存吃光。
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, apiError(resp.StatusCode, body)
	}

	var parsed openaiImageResponse
	if err := decodeJSON(resp, body, "图像接口", &parsed); err != nil {
		return nil, err
	}
	if len(parsed.Data) == 0 {
		return nil, fmt.Errorf("服务端没有返回任何图像")
	}
	d := parsed.Data[0]
	if d.Refusal != "" {
		return nil, &Refusal{Reason: d.Refusal}
	}
	if d.B64 == "" {
		if d.URL != "" {
			return nil, fmt.Errorf("服务端返回的是图片链接而不是内容；本工具只接受 b64_json 形式")
		}
		return nil, fmt.Errorf("服务端返回的条目里没有图像内容")
	}
	img, err := base64.StdEncoding.DecodeString(d.B64)
	if err != nil {
		return nil, fmt.Errorf("图像内容不是合法的 base64: %w", err)
	}

	u := Usage{
		InputTokens:  parsed.Usage.InputTokens,
		OutputTokens: parsed.Usage.OutputTokens,
		CostUSD:      parsed.cost(req.Model),
	}
	return &Result{
		Image: img, Revised: d.Revised, Usage: u,
		Model: req.Model, Elapsed: time.Since(start),
	}, nil
}

func (o *OpenAI) generateRequest(ctx context.Context, req Request) (*http.Request, error) {
	body := map[string]any{
		"model":  req.Model,
		"prompt": req.Prompt,
		"n":      1,
	}
	// 只在用户真的选了的时候才带上：不同模型认的取值不一样，
	// 硬塞一个默认值反而会让本来能跑的模型报参数错误。
	putIfSet(body, "size", req.Size)
	putIfSet(body, "quality", req.Quality)
	putIfSet(body, "background", req.Background)
	if strings.HasPrefix(req.Model, "gpt-image") {
		// 纹理素材经常出现"石头裂缝""血迹""锈蚀"这类词，默认审核档容易误伤。
		// 这是官方支持的取值，不是绕过审核——low 档仍然拦真正的违规内容。
		body["moderation"] = "low"
		// PNG 无损：底图要送进 PBR 分解，JPEG 的块效应会被法线放大成一片麻点。
		body["output_format"] = "png"
	}
	// 刻意不传的两个参数：
	//   response_format —— 对 gpt-image 系列无效，它固定返回 base64
	//   input_fidelity  —— gpt-image-2 明确不允许改（它一律按高保真处理输入图），
	//                      传了会被拒；老模型上这个参数也不影响我们的用法
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, o.base()+"/images/generations", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	r.Header.Set("Content-Type", "application/json")
	return r, nil
}

func (o *OpenAI) editRequest(ctx context.Context, req Request) (*http.Request, error) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	fields := map[string]string{"model": req.Model, "prompt": req.Prompt, "n": "1"}
	putIfSetStr(fields, "size", req.Size)
	putIfSetStr(fields, "quality", req.Quality)
	putIfSetStr(fields, "background", req.Background)
	if strings.HasPrefix(req.Model, "gpt-image") {
		fields["moderation"] = "low"
	}
	for k, v := range fields {
		if err := mw.WriteField(k, v); err != nil {
			return nil, err
		}
	}

	name := req.ReferenceName
	if name == "" {
		name = "reference.png"
	}
	if err := writeFilePart(mw, "image", name, req.Reference); err != nil {
		return nil, err
	}
	if len(req.Mask) > 0 {
		if err := writeFilePart(mw, "mask", "mask.png", req.Mask); err != nil {
			return nil, err
		}
	}
	if err := mw.Close(); err != nil {
		return nil, err
	}

	r, err := http.NewRequestWithContext(ctx, http.MethodPost, o.base()+"/images/edits", &buf)
	if err != nil {
		return nil, err
	}
	r.Header.Set("Content-Type", mw.FormDataContentType())
	return r, nil
}

// writeFilePart 写一个带 Content-Type 的文件分片。
//
// 不用 CreateFormFile：它固定写 application/octet-stream，而图像编辑接口
// 会按 Content-Type 校验，收到 octet-stream 时报的是"不支持的文件类型"，
// 跟真实原因差得很远。
func writeFilePart(mw *multipart.Writer, field, filename string, data []byte) error {
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition",
		fmt.Sprintf(`form-data; name="%s"; filename="%s"`, escapeQuotes(field), escapeQuotes(filename)))
	h.Set("Content-Type", sniffImageMIME(data))
	w, err := mw.CreatePart(h)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func escapeQuotes(s string) string {
	return strings.NewReplacer("\\", "\\\\", `"`, "\\\"").Replace(s)
}

func sniffImageMIME(data []byte) string {
	if t := http.DetectContentType(data); strings.HasPrefix(t, "image/") {
		return t
	}
	return "image/png"
}

func putIfSet(m map[string]any, k, v string) {
	if s := strings.TrimSpace(v); s != "" && s != "auto" {
		m[k] = s
	}
}

// 同名的字符串版本，multipart 的字段值只能是 string。
func putIfSetStr(m map[string]string, k, v string) {
	if s := strings.TrimSpace(v); s != "" && s != "auto" {
		m[k] = s
	}
}

// decodeJSON 解析响应体，解析不了时把实际收到的东西说清楚。
//
// 光报 "invalid character '<'" 等于什么都没说。这条路径上最常见的情况是
// 自建网关在某个路径上返回了一张 HTML 页面（登录页、Cloudflare 盾、
// 或者干脆是它自己的 404），而且往往还带着 200 状态码——不把响应内容摆出来，
// 用户根本无从判断是地址填错了还是网关不支持这个接口。
func decodeJSON(resp *http.Response, body []byte, what string, out any) error {
	if json.Unmarshal(body, out) == nil {
		return nil
	}
	ct := resp.Header.Get("Content-Type")
	if i := strings.IndexByte(ct, ';'); i > 0 {
		ct = ct[:i]
	}
	snippet := strings.Join(strings.Fields(string(body)), " ")
	if len(snippet) > 200 {
		snippet = snippet[:200] + "…"
	}
	if snippet == "" {
		snippet = "（空响应）"
	}

	msg := fmt.Sprintf("%s 返回的不是 JSON（HTTP %d", what, resp.StatusCode)
	if ct != "" {
		msg += ", " + ct
	}
	msg += "）：" + snippet
	if strings.Contains(ct, "html") || strings.HasPrefix(snippet, "<") {
		msg += "\n收到的是一个网页而不是接口响应。多半是接口地址填得不对，" +
			"或者这个网关不提供该接口——去设置页确认一下地址"
	}
	return errors.New(msg)
}

// apiError 把服务端的错误体翻译成能照着做的中文提示。
//
// 原样透传英文 message 是不够的：insufficient_quota 之类的措辞对着账单页面
// 才说得清，而 403 在这里几乎总是"组织未验证"，光看 message 想不到该去哪。
func apiError(status int, body []byte) error {
	var e struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    string `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(body, &e)
	msg := strings.TrimSpace(e.Error.Message)
	if msg == "" {
		msg = strings.TrimSpace(string(body))
		if len(msg) > 300 {
			msg = msg[:300] + "…"
		}
	}
	if msg == "" {
		msg = "（服务端没有给出说明）"
	}

	switch {
	case e.Error.Code == "content_policy_violation" || e.Error.Type == "image_generation_user_error":
		return &Refusal{Reason: msg}
	case status == http.StatusUnauthorized:
		return fmt.Errorf("令牌无效或已失效（401）：%s\n去设置页重新填一次 OpenAI 令牌", msg)
	case status == http.StatusForbidden:
		return fmt.Errorf("没有权限使用该模型（403）：%s\n"+
			"图像模型通常要求组织完成身份验证，在 platform.openai.com/settings/organization/general 里办", msg)
	case status == http.StatusNotFound:
		return fmt.Errorf("模型不存在或当前令牌不可用（404）：%s", msg)
	case status == http.StatusTooManyRequests:
		if e.Error.Code == "insufficient_quota" {
			return fmt.Errorf("账户余额不足：%s\n去 platform.openai.com/settings/organization/billing 充值", msg)
		}
		return fmt.Errorf("触发频率限制（429）：%s\n稍后再试，或把变体数调小", msg)
	case status >= 500:
		return fmt.Errorf("服务端故障（%d）：%s", status, msg)
	}
	return fmt.Errorf("请求失败（%d）：%s", status, msg)
}

// describeTransportError 给网络层错误补上排查方向。
//
// 这一步在国内网络下值回票价：超时的真实原因几乎总是没走代理，
// 但原始错误信息里只有 "context deadline exceeded"。
//
// 报的是实际请求的主机而不是写死 "OpenAI"：用户可能指着自建网关，
// 那时候说"连接 OpenAI 超时"会把人往错的方向带。
func describeTransportError(base string, err error) error {
	host := base
	if u, e := url.Parse(base); e == nil && u.Host != "" {
		host = u.Host
	}
	s := err.Error()
	switch {
	case strings.Contains(s, "context deadline exceeded"),
		strings.Contains(s, "timeout"),
		strings.Contains(s, "i/o timeout"):
		return fmt.Errorf("连接 %s 超时：%w\n"+
			"若本机需要代理才能访问，请确认后端进程能读到 HTTPS_PROXY，"+
			"或在配置里写死 imagen.proxy", host, err)
	case strings.Contains(s, "no such host"):
		return fmt.Errorf("%s 域名解析失败：%w\n检查网络、代理，或接口地址是否写错", host, err)
	case strings.Contains(s, "connection refused"):
		return fmt.Errorf("%s 拒绝连接：%w\n若配了代理，确认代理进程正在运行", host, err)
	case strings.Contains(s, "certificate"), strings.Contains(s, "x509"):
		return fmt.Errorf("%s 的 TLS 证书校验失败：%w\n自建网关请用受信任的证书", host, err)
	}
	return fmt.Errorf("请求 %s 失败: %w", host, err)
}
