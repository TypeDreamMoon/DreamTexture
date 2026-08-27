package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/mengye/dreamtexture/internal/workflow"
)

// hasGraph 挡住"对没有节点图的模板做图上的事"。
//
// 直出的工作流整条链路不经过 ComfyUI，模板里根本没有图。不在入口挡一道的话，
// 报出来的是 "unexpected end of JSON input"——那是在说 JSON 坏了，
// 而真相是这个工作流本来就不该走到这儿。
func hasGraph(tpl *workflow.Template) error {
	if tpl.Meta.Direct() {
		return fmt.Errorf("%s 是纯云端直出，本机没有节点图可编辑或下载", tpl.Meta.ID)
	}
	if len(tpl.RawTemplate()) == 0 {
		return fmt.Errorf("%s 没有可用的节点图", tpl.Meta.ID)
	}
	return nil
}

// nodeSpecs 取回并解析 ComfyUI 的节点定义，供图格式转换使用。
func (s *Server) nodeSpecs(r *http.Request) (map[string]workflow.NodeSpec, error) {
	raw, err := s.Sup.Client().ObjectInfo(r.Context())
	if err != nil {
		return nil, fmt.Errorf("读取 ComfyUI 节点列表失败: %w", err)
	}
	specs := make(map[string]workflow.NodeSpec, len(raw))
	for name, b := range raw {
		var spec workflow.NodeSpec
		if json.Unmarshal(b, &spec) == nil {
			specs[name] = spec
		}
	}
	return specs, nil
}

// openInComfy 把模板转成编辑器格式并写进 ComfyUI 的用户工作流目录。
//
// 必须转换：API 格式只有节点与取值，没有槽位、连线和坐标，直接给编辑器会得到
// 一张空图。转换后用户在 ComfyUI 左侧的工作流列表里就能找到它、可视化地改。
func (s *Server) openInComfy(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	tpl, ok := s.Reg.Get(id)
	if !ok {
		writeErr(w, http.StatusNotFound, "工作流不存在")
		return
	}
	if err := hasGraph(tpl); err != nil {
		writeErr(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	specs, err := s.nodeSpecs(r)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	name := "dreamtexture-" + id
	graph, err := tpl.GraphForEditor(specs, name)
	if err != nil {
		writeErr(w, http.StatusUnprocessableEntity, "转换为编辑器格式失败: "+err.Error())
		return
	}
	body, err := json.MarshalIndent(graph, "", "  ")
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	file := name + ".json"
	if err := s.Sup.Client().SaveUserWorkflow(r.Context(), file, body); err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"file":      file,
		"comfy_url": s.Sup.Health().BaseURL,
		"hint":      "已放进 ComfyUI 的工作流列表，在左侧「工作流」里打开 " + file,
	})
}

// templateJSON 下载模板的原始 API-format JSON。
func (s *Server) templateJSON(w http.ResponseWriter, r *http.Request) {
	tpl, ok := s.Reg.Get(chi.URLParam(r, "id"))
	if !ok {
		writeErr(w, http.StatusNotFound, "工作流不存在")
		return
	}
	// 不挡的话直出模板会下载到一个 0 字节的 json——比报错更难查。
	if err := hasGraph(tpl); err != nil {
		writeErr(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition",
		fmt.Sprintf("attachment; filename=%q", tpl.Meta.ID+".json"))
	_, _ = w.Write(tpl.RawTemplate())
}

var safeID = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)

// importWorkflow 接收一份 API-format 工作流，存成模板。
//
// 只接受 API 格式（ComfyUI 的「导出(API)」）：UI 格式里没有我们需要的
// class_type/inputs 结构。参数声明可以一起传，没传就生成一份最小可用的。
func (s *Server) importWorkflow(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ID       string          `json:"id"`
		Name     string          `json:"name"`
		Style    string          `json:"style"`
		Graph    json.RawMessage `json:"graph"`
		Params   json.RawMessage `json:"params"`
		Override bool            `json:"override"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "请求体不是合法 JSON: "+err.Error())
		return
	}
	body.ID = strings.TrimSpace(strings.ToLower(body.ID))
	if !safeID.MatchString(body.ID) {
		writeErr(w, http.StatusBadRequest,
			"id 只能用小写字母、数字和 . _ - ，且以字母或数字开头")
		return
	}
	if len(body.Graph) == 0 {
		writeErr(w, http.StatusBadRequest, "缺少 graph 字段")
		return
	}

	// 先按 API 格式解析一遍，格式不对就别写进磁盘。
	var probe map[string]struct {
		ClassType string `json:"class_type"`
	}
	if err := json.Unmarshal(body.Graph, &probe); err != nil || len(probe) == 0 {
		writeErr(w, http.StatusBadRequest,
			"这不像 ComfyUI 的 API 格式工作流。请在 ComfyUI 里用「工作流 → 导出(API)」导出，而不是普通的保存/导出")
		return
	}
	for id, n := range probe {
		if n.ClassType == "" {
			writeErr(w, http.StatusBadRequest,
				fmt.Sprintf("节点 %s 缺少 class_type，这通常说明导出的是 UI 格式而不是 API 格式", id))
			return
		}
	}

	dir := s.Reg.Dir()
	tplPath := filepath.Join(dir, body.ID+".json")
	parPath := filepath.Join(dir, body.ID+".params.json")
	if !body.Override {
		if _, err := os.Stat(parPath); err == nil {
			writeErr(w, http.StatusConflict, "同名工作流已存在，如需覆盖请勾选覆盖")
			return
		}
	}

	params := body.Params
	if len(params) == 0 {
		gen, err := generateParams(body, probe)
		if err != nil {
			writeErr(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		params = gen
	}

	pretty, err := indentJSON(body.Graph)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := os.WriteFile(tplPath, pretty, 0o644); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := os.WriteFile(parPath, params, 0o644); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 立刻重载校验：声明和图对不上就把刚写的文件撤掉，别留下一个坏模板。
	if err := s.Reg.Load(); err != nil {
		os.Remove(tplPath)
		os.Remove(parPath)
		_ = s.Reg.Load()
		writeErr(w, http.StatusUnprocessableEntity, "工作流校验未通过，已撤销导入: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": body.ID, "template": tplPath, "params": parPath})
}

// generateParams 为导入的工作流生成一份最小可用的参数声明。
//
// 只认我们自己的命名约定（dt.positive / dt.sampler 等）。认不出来也不算失败——
// 生成一份没有可调参数的声明，让用户先跑起来，再自己去 params.json 里补。
func generateParams(body struct {
	ID       string          `json:"id"`
	Name     string          `json:"name"`
	Style    string          `json:"style"`
	Graph    json.RawMessage `json:"graph"`
	Params   json.RawMessage `json:"params"`
	Override bool            `json:"override"`
}, probe map[string]struct {
	ClassType string `json:"class_type"`
}) ([]byte, error) {
	var full map[string]workflow.Node
	if err := json.Unmarshal(body.Graph, &full); err != nil {
		return nil, err
	}
	titles := map[string]bool{}
	outputs := map[string]any{}
	for _, n := range full {
		t := n.Meta.Title
		if t == "" {
			continue
		}
		titles[t] = true
		if strings.HasPrefix(t, "dt.out.") && n.ClassType == "SaveImage" {
			ch := strings.TrimPrefix(t, "dt.out.")
			cs := "linear"
			if ch == "basecolor" || ch == "source" {
				cs = "srgb"
			}
			outputs[ch] = map[string]any{"node": t, "colorspace": cs}
		}
	}
	if len(outputs) == 0 {
		return nil, fmt.Errorf("工作流里没有找到 dt.out.* 命名的 SaveImage 节点。" +
			"请给每个输出节点起 dt.out.basecolor 这样的标题，DreamTexture 靠它把产物对上通道")
	}

	params := []any{}
	if titles["dt.positive"] {
		params = append(params, map[string]any{
			"key": "prompt", "label": "提示词", "type": "string", "multiline": true,
			"target": "dt.positive.text", "default": "",
		})
	}
	if titles["dt.negative"] {
		params = append(params, map[string]any{
			"key": "negative", "label": "负面词", "type": "string", "multiline": true,
			"target": "dt.negative.text", "default": "",
		})
	}
	if titles["dt.sampler"] {
		params = append(params, map[string]any{
			"key": "seed", "label": "种子", "type": "int",
			"target": "dt.sampler.seed", "default": -1,
		})
	}

	name := body.Name
	if name == "" {
		name = body.ID
	}
	style := body.Style
	if style == "" {
		style = "custom"
	}
	meta := map[string]any{
		"id": body.ID, "version": 1, "name": name, "style": style,
		"description": "从 ComfyUI 导入的工作流",
		"template":    body.ID + ".json",
		"resolution":  1024, "tileable": true,
		"model_requirements": []any{},
		"node_packs":         []any{},
		"params":             params,
		"advanced":           []any{},
		"outputs":            outputs,
	}
	return json.MarshalIndent(meta, "", "  ")
}

// reloadWorkflows 重新扫描工作流目录，改完文件不用重启后端。
func (s *Server) reloadWorkflows(w http.ResponseWriter, _ *http.Request) {
	if err := s.Reg.Load(); err != nil {
		writeErr(w, http.StatusUnprocessableEntity, "重新加载失败: "+err.Error())
		return
	}
	list := s.Reg.List()
	ids := make([]string, 0, len(list))
	for _, t := range list {
		ids = append(ids, t.Meta.ID)
	}
	writeJSON(w, http.StatusOK, map[string]any{"workflows": ids})
}

func indentJSON(raw json.RawMessage) ([]byte, error) {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, err
	}
	return json.MarshalIndent(v, "", "  ")
}
