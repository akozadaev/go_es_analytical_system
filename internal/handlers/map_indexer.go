package handlers

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type mapCreateIndexRequest struct {
	ESURL       string `json:"esUrl"`
	IndexName   string `json:"indexName"`
	MappingPath string `json:"mappingPath,omitempty"`
	Username    string `json:"username,omitempty"`
	Password    string `json:"password,omitempty"`
}

type mapBulkIndexRequest struct {
	ESURL     string                   `json:"esUrl"`
	IndexName string                   `json:"indexName"`
	Username  string                   `json:"username,omitempty"`
	Password  string                   `json:"password,omitempty"`
	Locations []map[string]interface{} `json:"locations"`
}

type mapSaveXMLRequest struct {
	Filename  string                   `json:"filename"`
	Locations []map[string]interface{} `json:"locations"`
}

func (h *Handlers) MapCreateIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req mapCreateIndexRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 2<<20)).Decode(&req); err != nil {
		writeMapIndexerJSON(w, http.StatusBadRequest, false, "Неверный JSON", nil)
		return
	}

	esURL := normalizeMapESURL(req.ESURL)
	if esURL == "" {
		writeMapIndexerJSON(w, http.StatusBadRequest, false, "Не указан URL Elasticsearch/OpenSearch", nil)
		return
	}

	indexName := strings.TrimSpace(req.IndexName)
	if indexName == "" {
		indexName = "locations"
	}

	checkReq, _ := http.NewRequest(http.MethodHead, esURL+"/"+indexName, nil)
	addMapBasicAuth(checkReq, req.Username, req.Password)
	checkResp, err := mapHTTPClient().Do(checkReq)
	if err != nil {
		writeMapIndexerJSON(w, http.StatusBadGateway, false, "Не удалось проверить индекс", map[string]interface{}{"details": err.Error()})
		return
	}
	defer checkResp.Body.Close()

	if checkResp.StatusCode == http.StatusOK {
		writeMapIndexerJSON(w, http.StatusOK, true, fmt.Sprintf("Индекс %s уже существует", indexName), nil)
		return
	}
	if checkResp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(io.LimitReader(checkResp.Body, 1<<20))
		writeMapIndexerJSON(w, http.StatusBadRequest, false, fmt.Sprintf("Не удалось проверить индекс: %d", checkResp.StatusCode), map[string]interface{}{"details": string(body)})
		return
	}

	mappingData, mappingPath, err := h.readMapMapping(req.MappingPath)
	if err != nil {
		writeMapIndexerJSON(w, http.StatusInternalServerError, false, "Не удалось прочитать mapping", map[string]interface{}{"details": err.Error()})
		return
	}

	createReq, _ := http.NewRequest(http.MethodPut, esURL+"/"+indexName, bytes.NewReader(mappingData))
	createReq.Header.Set("Content-Type", "application/json")
	addMapBasicAuth(createReq, req.Username, req.Password)
	createResp, err := mapHTTPClient().Do(createReq)
	if err != nil {
		writeMapIndexerJSON(w, http.StatusBadGateway, false, "Ошибка создания индекса", map[string]interface{}{"details": err.Error()})
		return
	}
	defer createResp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(createResp.Body, 2<<20))
	if createResp.StatusCode < 200 || createResp.StatusCode >= 300 {
		writeMapIndexerJSON(w, http.StatusBadRequest, false, fmt.Sprintf("Ошибка создания индекса: %d", createResp.StatusCode), map[string]interface{}{"details": string(body)})
		return
	}

	writeMapIndexerJSON(w, http.StatusOK, true, fmt.Sprintf("Индекс %s создан", indexName), map[string]interface{}{
		"mappingPath": mappingPath,
		"details":     string(body),
	})
}

func (h *Handlers) MapBulkIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req mapBulkIndexRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 8<<20)).Decode(&req); err != nil {
		writeMapIndexerJSON(w, http.StatusBadRequest, false, "Неверный JSON", nil)
		return
	}

	esURL := normalizeMapESURL(req.ESURL)
	if esURL == "" {
		writeMapIndexerJSON(w, http.StatusBadRequest, false, "Не указан URL Elasticsearch/OpenSearch", nil)
		return
	}
	if len(req.Locations) == 0 {
		writeMapIndexerJSON(w, http.StatusBadRequest, false, "Список locations пуст", nil)
		return
	}

	indexName := strings.TrimSpace(req.IndexName)
	if indexName == "" {
		indexName = "locations"
	}

	var payload strings.Builder
	for _, loc := range req.Locations {
		id, _ := loc["id"].(string)
		if strings.TrimSpace(id) == "" {
			id = "loc_" + strconv.FormatInt(time.Now().UnixNano(), 10)
			loc["id"] = id
		}
		meta := map[string]map[string]string{
			"index": {"_index": indexName, "_id": id},
		}
		metaJSON, _ := json.Marshal(meta)
		docJSON, _ := json.Marshal(loc)
		payload.Write(metaJSON)
		payload.WriteByte('\n')
		payload.Write(docJSON)
		payload.WriteByte('\n')
	}

	bulkReq, _ := http.NewRequest(http.MethodPost, esURL+"/_bulk?refresh=true", strings.NewReader(payload.String()))
	bulkReq.Header.Set("Content-Type", "application/x-ndjson")
	addMapBasicAuth(bulkReq, req.Username, req.Password)

	bulkResp, err := mapHTTPClient().Do(bulkReq)
	if err != nil {
		writeMapIndexerJSON(w, http.StatusBadGateway, false, "Ошибка bulk индексации", map[string]interface{}{"details": err.Error()})
		return
	}
	defer bulkResp.Body.Close()

	var result map[string]interface{}
	_ = json.NewDecoder(io.LimitReader(bulkResp.Body, 4<<20)).Decode(&result)

	if bulkResp.StatusCode < 200 || bulkResp.StatusCode >= 300 {
		writeMapIndexerJSON(w, http.StatusBadRequest, false, fmt.Sprintf("Ошибка bulk индексации: %d", bulkResp.StatusCode), map[string]interface{}{"details": result})
		return
	}

	hasErrors, _ := result["errors"].(bool)
	if hasErrors {
		writeMapIndexerJSON(w, http.StatusOK, false, "Частичная индексация: есть ошибки в items", map[string]interface{}{"details": result})
		return
	}
	writeMapIndexerJSON(w, http.StatusOK, true, fmt.Sprintf("Успешно проиндексировано %d локаций", len(req.Locations)), nil)
}

func (h *Handlers) MapSaveXML(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req mapSaveXMLRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 8<<20)).Decode(&req); err != nil {
		writeMapIndexerJSON(w, http.StatusBadRequest, false, "Неверный JSON", nil)
		return
	}
	if len(req.Locations) == 0 {
		writeMapIndexerJSON(w, http.StatusBadRequest, false, "Список locations пуст", nil)
		return
	}

	fileName := sanitizeXMLFileName(req.Filename)
	if !strings.HasSuffix(strings.ToLower(fileName), ".xml") {
		fileName += ".xml"
	}

	exportDir, err := resolveMapExportDir()
	if err != nil {
		writeMapIndexerJSON(w, http.StatusInternalServerError, false, "Не удалось определить каталог exports", map[string]interface{}{"details": err.Error()})
		return
	}
	if err := os.MkdirAll(exportDir, 0o755); err != nil {
		writeMapIndexerJSON(w, http.StatusInternalServerError, false, "Не удалось создать каталог exports", map[string]interface{}{"details": err.Error()})
		return
	}

	xmlBody, err := buildLocationsXML(req.Locations)
	if err != nil {
		writeMapIndexerJSON(w, http.StatusInternalServerError, false, "Не удалось сформировать XML", map[string]interface{}{"details": err.Error()})
		return
	}

	outPath := filepath.Join(exportDir, fileName)
	if err := os.WriteFile(outPath, []byte(xmlBody), 0o644); err != nil {
		writeMapIndexerJSON(w, http.StatusInternalServerError, false, "Не удалось сохранить XML", map[string]interface{}{"details": err.Error()})
		return
	}

	writeMapIndexerJSON(w, http.StatusOK, true, "XML сохранен: "+outPath, map[string]interface{}{"filePath": outPath})
}

func writeMapIndexerJSON(w http.ResponseWriter, status int, ok bool, message string, extra map[string]interface{}) {
	resp := map[string]interface{}{
		"ok":      ok,
		"message": message,
	}
	for k, v := range extra {
		resp[k] = v
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(resp)
}

func normalizeMapESURL(v string) string {
	s := strings.TrimSpace(v)
	s = strings.TrimRight(s, "/")
	u, err := url.Parse(s)
	if err != nil || u.Host == "" {
		return s
	}
	host := strings.ToLower(u.Hostname())
	if host == "localhost" || host == "127.0.0.1" {
		if p := u.Port(); p != "" {
			u.Host = "host.docker.internal:" + p
		} else {
			u.Host = "host.docker.internal"
		}
		return strings.TrimRight(u.String(), "/")
	}
	return s
}

func addMapBasicAuth(req *http.Request, username, password string) {
	if req == nil {
		return
	}
	u := strings.TrimSpace(username)
	p := strings.TrimSpace(password)
	if u == "" || p == "" {
		return
	}
	req.SetBasicAuth(u, p)
}

func mapHTTPClient() *http.Client {
	return &http.Client{Timeout: 30 * time.Second}
}

func (h *Handlers) readMapMapping(requestPath string) ([]byte, string, error) {
	candidates := make([]string, 0, 6)
	if strings.TrimSpace(requestPath) != "" {
		candidates = append(candidates, strings.TrimSpace(requestPath))
	}
	if h != nil && h.cfg != nil && strings.TrimSpace(h.cfg.ElasticsearchMappingPath) != "" {
		candidates = append(candidates, strings.TrimSpace(h.cfg.ElasticsearchMappingPath))
	}
	candidates = append(candidates, "migrations/elasticsearch_mapping.json", "migrations/opensearch_mapping.json")

	for _, p := range candidates {
		for _, c := range []string{p, filepath.Join("..", p), filepath.Join("/root", p)} {
			data, err := os.ReadFile(c)
			if err == nil {
				return data, c, nil
			}
		}
	}
	return nil, "", fmt.Errorf("mapping file not found")
}

func sanitizeXMLFileName(name string) string {
	s := strings.TrimSpace(name)
	if s == "" {
		return "locations.xml"
	}
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	out := strings.Trim(b.String(), "._")
	if out == "" {
		return "locations.xml"
	}
	return out
}

func resolveMapExportDir() (string, error) {
	candidates := []string{
		"js_API_Ya_map/exports",
		filepath.Join("..", "js_API_Ya_map", "exports"),
		filepath.Join("/root", "js_API_Ya_map", "exports"),
	}
	for _, c := range candidates {
		if fi, err := os.Stat(c); err == nil && fi.IsDir() {
			return c, nil
		}
	}
	return candidates[len(candidates)-1], nil
}

func buildLocationsXML(locations []map[string]interface{}) (string, error) {
	var b strings.Builder
	b.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<locations>\n")
	for _, loc := range locations {
		b.WriteString("  <location>\n")
		writeXMLTag(&b, 2, "id", asString(loc["id"]))
		writeXMLTag(&b, 2, "name", asString(loc["name"]))
		writeXMLTag(&b, 2, "address", asString(loc["address"]))
		writeXMLTag(&b, 2, "region", asString(loc["region"]))
		writeXMLTag(&b, 2, "city", asString(loc["city"]))
		writeXMLTag(&b, 2, "description", asString(loc["description"]))
		b.WriteString("  </location>\n")
	}
	b.WriteString("</locations>\n")
	return b.String(), nil
}

func writeXMLTag(b *strings.Builder, indent int, tag, value string) {
	pad := strings.Repeat("  ", indent)
	var escaped bytes.Buffer
	_ = xml.EscapeText(&escaped, []byte(value))
	b.WriteString(pad + "<" + tag + ">" + escaped.String() + "</" + tag + ">\n")
}

func asString(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return fmt.Sprintf("%v", t)
	case bool:
		if t {
			return "true"
		}
		return "false"
	default:
		return ""
	}
}
