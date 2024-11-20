package main

import (
  "fmt"
  "net/http"
  "html/template"
)

func (cfg *ApiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
  return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
    cfg.FileServerHits.Add(1)
    fmt.Println("incremented FileServerHits now:", cfg.FileServerHits.Load())
    next.ServeHTTP(w, req)
  })
}

func (cfg *ApiConfig) serveAdminMetrics(w http.ResponseWriter, req *http.Request) {
  const tpl = `<html>

<body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited {{.FileServerHits.Load}} times!</p>
</body>

</html>`
  t, err := template.New("webpage").Parse(tpl)
  if err != nil {
    fmt.Println(err)
    return
  }
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
  err = t.Execute(w, cfg)
  if err != nil {
    fmt.Println(err)
  }
}

