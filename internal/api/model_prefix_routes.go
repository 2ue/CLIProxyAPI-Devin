package api

import (
	"bytes"
	"net/http"
	"strings"
	"sync"

	"github.com/MoeForge/moego/ojson"
	"github.com/gin-gonic/gin"
	claudemodels "github.com/router-for-me/CLIProxyAPI/v8/internal/client/claude/models"
	codexmodels "github.com/router-for-me/CLIProxyAPI/v8/internal/client/codex/models"
	"github.com/router-for-me/CLIProxyAPI/v8/internal/thinking"
	"github.com/router-for-me/CLIProxyAPI/v8/internal/util"
	"github.com/router-for-me/CLIProxyAPI/v8/sdk/api/handlers"
	"github.com/router-for-me/CLIProxyAPI/v8/sdk/api/handlers/claude"
	"github.com/router-for-me/CLIProxyAPI/v8/sdk/api/handlers/gemini"
	"github.com/router-for-me/CLIProxyAPI/v8/sdk/api/handlers/openai"
	log "github.com/sirupsen/logrus"
)

func (s *Server) setupModelPrefixRoutes(o *openai.OpenAIAPIHandler, c *claude.ClaudeCodeAPIHandler, g *gemini.GeminiAPIHandler, r *openai.OpenAIResponsesAPIHandler) {
	group := s.engine.Group("/llm/:prefix", AuthMiddleware(s.accessManager), s.modelPrefixMiddleware())
	group.GET("/v1/models", modelPrefixModels)
	group.GET("/v1/models/*model", modelPrefixModels)
	group.POST("/v1/chat/completions", o.ChatCompletions)
	group.POST("/v1/completions", o.Completions)
	group.POST("/v1/messages", c.ClaudeMessages)
	group.POST("/v1/messages/count_tokens", c.ClaudeCountTokens)
	group.POST("/v1/responses", r.Responses)
	group.POST("/v1/responses/compact", r.Compact)
	group.POST("/v1/images/generations", o.ImagesGenerations)
	group.POST("/v1/images/edits", o.ImagesEdits)
	group.POST("/v1/audio/speech", o.AudioSpeech)
	group.GET("/v1beta/models", modelPrefixGeminiModels)
	group.GET("/v1beta/models/*action", modelPrefixGeminiModels)
	group.POST("/v1beta/models/*action", g.GeminiHandler)
	group.POST("/v1beta/interactions", g.Interactions)
}

func (s *Server) modelPrefixMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		prefix := c.Param("prefix")
		entries := buildModelPrefixCatalog(prefix, s.getConfig())
		routes := make(map[string]string, len(entries))
		public := make(map[string]string, len(entries)*2)
		for _, e := range entries {
			routes[e.public] = e.route
			public[e.route] = e.public
			public[strings.TrimPrefix(e.route, prefix+"/")] = e.public
		}
		c.Set(handlers.ModelPrefixContextKey, prefix)
		c.Set(handlers.ModelPrefixRoutesContextKey, routes)
		c.Set(modelPrefixCatalogContextKey, entries)
		writer := &modelPrefixResponseWriter{ResponseWriter: c.Writer, prefix: prefix, public: public}
		c.Writer = writer
		c.Next()
		writer.finish()
	}
}

const modelPrefixCatalogContextKey = "cliproxy.model_prefix_catalog"

func modelPrefixCatalog(c *gin.Context) []modelPrefixEntry {
	value, _ := c.Get(modelPrefixCatalogContextKey)
	entries, _ := value.([]modelPrefixEntry)
	return entries
}

func modelPrefixModels(c *gin.Context) {
	entries := modelPrefixCatalog(c)
	rows := make([]map[string]any, 0, len(entries))
	anthropic := isAnthropicModelsRequest(c)
	_, codex := c.Request.URL.Query()["client_version"]
	if codex {
		anthropic = false
	}
	protocol := "openai"
	if anthropic {
		protocol = "claude"
	}
	available := modelPrefixProtocolModels(protocol)
	detail := strings.TrimPrefix(c.Param("model"), "/")
	for _, entry := range entries {
		row := clonePrefixModel(available[entry.route])
		row["id"] = entry.public
		if codex {
			row["id"] = entry.route
		}
		if detail != "" {
			if detail == entry.public {
				c.JSON(http.StatusOK, row)
				return
			}
			continue
		}
		rows = append(rows, row)
	}
	if detail != "" {
		modelPrefixNotFound(c)
		return
	}
	if codex {
		body := codexmodels.BuildResponseForClient(rows, util.GetProviderName, false, c.Query("client_version"))
		public := map[string]string{}
		for _, e := range entries {
			public[e.route] = e.public
		}
		if entries, ok := body["models"].([]map[string]any); ok {
			for _, entry := range entries {
				if slug, ok := entry["slug"].(string); ok && public[slug] != "" {
					entry["slug"] = public[slug]
				}
			}
		}
		c.JSON(http.StatusOK, body)
		return
	}
	if anthropic {
		c.JSON(http.StatusOK, claudemodels.BuildResponse(rows, true))
		return
	}
	c.JSON(http.StatusOK, gin.H{"object": "list", "data": rows})
}

func modelPrefixGeminiModels(c *gin.Context) {
	rows := make([]map[string]any, 0)
	available := modelPrefixProtocolModels("gemini")
	detail := strings.TrimPrefix(c.Param("action"), "/")
	for _, entry := range modelPrefixCatalog(c) {
		row := clonePrefixModel(available[entry.route])
		row["name"] = "models/" + entry.public
		if detail != "" {
			if detail == entry.public {
				c.JSON(http.StatusOK, row)
				return
			}
			continue
		}
		rows = append(rows, row)
	}
	if detail != "" {
		modelPrefixNotFound(c)
		return
	}
	c.JSON(http.StatusOK, gin.H{"models": rows})
}

func modelPrefixNotFound(c *gin.Context) {
	c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "Model not found in prefix catalog", "type": "invalid_request_error", "code": "model_not_found"}})
}

// Only protocol metadata is rewritten. Text, tool arguments and signatures are opaque.
func prefixPublicResponseModel(raw []byte, prefix string, public map[string]string) []byte {
	if !ojson.ValidBytes(raw) {
		return raw
	}
	doc := ojson.NewDoc(raw)
	for _, path := range []string{"model", "modelVersion", "message.model", "response.model", "interaction.model"} {
		value := doc.Get(path).String()
		mapped := public[value]
		if mapped == "" {
			parsed := thinking.ParseSuffix(value)
			if parsed.HasSuffix && public[parsed.ModelName] != "" {
				mapped = public[parsed.ModelName] + "(" + parsed.RawSuffix + ")"
			}
		}
		if mapped != "" {
			_ = doc.Set(path, mapped)
		} else if strings.HasPrefix(value, prefix+"/") {
			_ = doc.Set(path, strings.TrimPrefix(value, prefix+"/"))
		}
	}
	return doc.Bytes()
}

type modelPrefixResponseWriter struct {
	gin.ResponseWriter
	prefix  string
	public  map[string]string
	pending bytes.Buffer
	mu      sync.Mutex
}

func (w *modelPrefixResponseWriter) WriteHeader(code int) {
	w.Header().Del("Content-Length")
	w.ResponseWriter.WriteHeader(code)
}

func (w *modelPrefixResponseWriter) Write(b []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.Header().Del("Content-Length")
	n := len(b)
	contentType := w.Header().Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/event-stream") {
		if contentType != "" && !strings.Contains(contentType, "json") {
			return w.ResponseWriter.Write(b)
		}
		_, _ = w.pending.Write(b)
		return n, nil
	}
	_, _ = w.pending.Write(b)
	for {
		data := w.pending.Bytes()
		end := bytes.IndexByte(data, '\n')
		if end < 0 {
			break
		}
		line := append([]byte(nil), data[:end+1]...)
		w.pending.Next(end + 1)
		if err := w.writeSSELine(line); err != nil {
			return 0, err
		}
	}
	return n, nil
}

func (w *modelPrefixResponseWriter) WriteString(s string) (int, error) { return w.Write([]byte(s)) }

func (w *modelPrefixResponseWriter) writeSSELine(line []byte) error {
	if bytes.HasPrefix(line, []byte("data:")) {
		payload := bytes.TrimSpace(line[5:])
		if ojson.ValidBytes(payload) {
			suffix := "\n"
			if bytes.HasSuffix(line, []byte("\r\n")) {
				suffix = "\r\n"
			} else if !bytes.HasSuffix(line, []byte("\n")) {
				suffix = ""
			}
			line = append(append([]byte("data: "), prefixPublicResponseModel(payload, w.prefix, w.public)...), suffix...)
		}
	}
	_, err := w.ResponseWriter.Write(line)
	return err
}

func (w *modelPrefixResponseWriter) Flush() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if strings.HasPrefix(w.Header().Get("Content-Type"), "text/event-stream") {
		w.ResponseWriter.Flush()
	}
}

func (w *modelPrefixResponseWriter) finish() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.pending.Len() == 0 {
		return
	}
	var err error
	if strings.HasPrefix(w.Header().Get("Content-Type"), "text/event-stream") {
		err = w.writeSSELine(w.pending.Bytes())
	} else {
		_, err = w.ResponseWriter.Write(prefixPublicResponseModel(w.pending.Bytes(), w.prefix, w.public))
	}
	w.pending.Reset()
	if err != nil {
		log.WithError(err).Debug("Failed to finish model prefix response")
	}
}
