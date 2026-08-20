package api

import (
	"encoding/json"
	"net/http"
	"strings"
)

// wantsJSON checks if the client wants JSON output.
func wantsJSON(r *http.Request) bool {
	if r.URL.Query().Get("format") == "json" {
		return true
	}
	accept := r.Header.Get("Accept")
	return strings.Contains(accept, "application/json")
}

// writeRecord writes a single record as either plain text or JSON.
func writeRecord(w http.ResponseWriter, r *http.Request, fields map[string]interface{}) {
	if wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(fields)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	var parts []string
	for k, v := range fields {
		parts = append(parts, k+"="+fmtVal(v))
	}
	w.Write([]byte(strings.Join(parts, " ")))
}

// writeRecords writes multiple records as either plain text or JSON.
func writeRecords(w http.ResponseWriter, r *http.Request, records []map[string]interface{}) {
	if wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(records)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	for _, rec := range records {
		var parts []string
		for k, v := range rec {
			parts = append(parts, k+"="+fmtVal(v))
		}
		w.Write([]byte(strings.Join(parts, " ") + "\n"))
	}
}

// writeError writes an error response with a hint.
func writeError(w http.ResponseWriter, r *http.Request, status int, msg, hint string) {
	w.WriteHeader(status)
	if wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"error": msg,
			"hint":  hint,
		})
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("error: " + msg + " | hint: " + hint))
}

func fmtVal(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case int:
		return intToStr(val)
	case int64:
		return int64ToStr(val)
	case float64:
		return floatToStr(val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}

func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		b = append([]byte{'-'}, b...)
	}
	return string(b)
}

func int64ToStr(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		b = append([]byte{'-'}, b...)
	}
	return string(b)
}

func floatToStr(f float64) string {
	b, _ := json.Marshal(f)
	return string(b)
}
