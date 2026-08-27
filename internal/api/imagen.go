package api

import (
	"errors"
	"net/http"
	"sort"

	"github.com/mengye/dreamtexture/internal/imagen"
)

// imagenProviders 列出外部底图来源及各自的可用模型。
//
// 模型是现问服务端的，不是写死的清单：新模型发布时用户明明有权限、工具却
// 不认得，是很没道理的事。没配令牌就只报告"未配置"，不当成错误——界面要
// 能把这一档正常显示出来并指引去设置。
func (s *Server) imagenProviders(w http.ResponseWriter, r *http.Request) {
	type provOut struct {
		ID         string         `json:"id"`
		Label      string         `json:"label"`
		Configured bool           `json:"configured"`
		Models     []imagen.Model `json:"models"`
		Error      string         `json:"error,omitempty"`
	}
	out := []provOut{}
	if s.Imagen != nil {
		for _, p := range s.Imagen.All() {
			o := provOut{ID: p.ID(), Label: p.Label(), Configured: p.Configured(),
				Models: []imagen.Model{}}
			if o.Configured {
				ms, err := p.Models(r.Context())
				switch {
				case errors.Is(err, imagen.ErrNoToken):
					o.Configured = false
				case err != nil:
					o.Error = err.Error()
				default:
					o.Models = ms
				}
			}
			out = append(out, o)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"providers": out})
}

// checkImagen 探一次外部底图来源的可达性。
//
// 第二个返回值为 false 表示"这台机器上没有工作流用到云端底图"，自检里就
// 不显示这一条——没用这个功能的人不该看到一条与自己无关的警告。
func (s *Server) checkImagen(r *http.Request) (Check, bool) {
	c := Check{Key: "imagen", Label: "云端底图"}
	if s.Imagen == nil {
		return c, false
	}

	need := map[string]bool{}
	for _, t := range s.Reg.List() {
		if src := t.Meta.Source; src != nil {
			need[src.Provider] = true
		}
	}
	if len(need) == 0 {
		return c, false
	}
	ids := make([]string, 0, len(need))
	for id := range need {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	var live, notSet, failed []string
	for _, id := range ids {
		p, ok := s.Imagen.Get(id)
		if !ok {
			failed = append(failed, id+"：来源未注册")
			continue
		}
		if !p.Configured() {
			notSet = append(notSet, p.Label())
			continue
		}
		detail, err := p.Ping(r.Context())
		if err != nil {
			failed = append(failed, p.Label()+"："+firstLine(err.Error()))
			continue
		}
		live = append(live, p.Label()+" "+detail)
	}

	switch {
	case len(failed) > 0:
		// 只影响云端那几条管线，本地管线照跑，所以是 warn 不是 fail。
		c.Status = "warn"
		c.Items = failed
		c.Detail = "云端底图当前不可用，本地风格预设不受影响。常见原因：" +
			"本机需要代理而后端读不到代理设置（可在配置里写 imagen.proxy）、" +
			"组织未完成身份验证、账户余额不足"
	case len(live) == 0:
		c.Status = "warn"
		c.Items = notSet
		c.Detail = "还没填访问令牌。在「模型 → 设置」里填上之后，云端底图的风格预设才能用"
		c.Fix = "open-models"
	default:
		c.Status = "ok"
		c.Detail = join(live, "；")
		if len(notSet) > 0 {
			c.Items = notSet
		}
	}
	return c, true
}

func firstLine(s string) string {
	for i, ch := range s {
		if ch == '\n' {
			return s[:i] + " …"
		}
	}
	return s
}

func join(parts []string, sep string) string {
	out := ""
	for _, p := range parts {
		if p == "" {
			continue
		}
		if out != "" {
			out += sep
		}
		out += p
	}
	return out
}
