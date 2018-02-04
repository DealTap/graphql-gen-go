package api

import (
  "encoding/json"
  "errors"
  "io/ioutil"
  "net/http"
  "strings"
  "sync"

  "github.com/neelance/graphql-go"
  "github.com/rs/cors"
)

const (
  ContentTypeJSON    = "application/json"
  ContentTypeGraphQL = "application/graphql"
  Post               = "POST"
  Get                = "GET"
)

type GqlServer struct {
  Schema      *graphql.Schema
  Port        string
  Request     gqlRequest
  CorsOptions *cors.Options
  // TODO add facebook dataloader
}

func NewGqlServer(res GqlResolver, port string, corsOptions *cors.Options) *GqlServer {
  return &GqlServer{
    Schema:      graphql.MustParseSchema(Schema, res),
    Port:        port,
    CorsOptions: corsOptions,
  }
}

func (g *GqlServer) Serve() error {

  // TODO should we validate required fields of GqlServer
  addr := ":" + g.Port
  srv := &httpServer{g}
  mux := http.NewServeMux()

  // configure pre-flight/cors request handler
  var c *cors.Cors
  if g.CorsOptions == nil {
    c = cors.AllowAll()
  } else {
    c = cors.New(*g.CorsOptions)
  }

  mux.Handle("/graphql", srv)
  handler := c.Handler(mux)
  return http.ListenAndServe(addr, handler)
}

type httpServer struct {
  *GqlServer
}

func (h *httpServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {

  if isContentSupported(r.Header.Get("Content-Type")) == false {
    http.Error(w, "GraphQL only supports json and graphql content type.", http.StatusBadRequest)
    return
  }

  req, htpErr := parse(r)
  if htpErr != nil {
    http.Error(w, htpErr.message, htpErr.status)
    return
  }

  numReqs := len(req.requests)
  responses := make([]*graphql.Response, numReqs)

  // Use the WaitGroup to wait for all executions to finish
  // TODO handle facebook data loader
  var wg sync.WaitGroup
  wg.Add(numReqs)

  // iterate over all parsed requests and use go routine to process them in parallel
  for i, q := range req.requests {
    go func(i int, q gqlRequest) {
      h.Request = q
      res := h.Schema.Exec(r.Context(), q.Query, q.OpName, q.Variables)
      // FIXME expand returned errors to handle a resolver returning more than one error
      responses[i] = res
      wg.Done()
    }(i, q)
  }

  wg.Wait()

  // TODO should we log errors?

  var err error
  var resp []byte
  /**
    * at this point there should be at least one response.
    * in case of batch, we send a json array object otherwise a single json object
   */
  if req.batch {
    resp, err = json.Marshal(responses)
  } else {
    resp, err = json.Marshal(responses[0])
  }
  if err != nil {
    http.Error(w, "Server error", http.StatusInternalServerError)
    return
  }

  w.Header().Set("Content-Type", ContentTypeJSON)
  w.WriteHeader(http.StatusOK)
  w.Write(resp)
}

func isContentSupported(contentType string) bool {
  return strings.HasPrefix(contentType, ContentTypeJSON) || strings.HasPrefix(contentType, ContentTypeGraphQL)
}

type request struct {
  requests []gqlRequest
  batch    bool
}

type gqlRequest struct {
  Query     string                 `json:"query"`
  OpName    string                 `json:"operationName"`
  Variables map[string]interface{} `json:"variables"`
}

type httpError struct {
  status  int
  message string
  error
}

func parse(r *http.Request) (*request, *httpError) {

  if r.Method == Get {
    return parseGet(r)
  } else if r.Method == Post {
    return parsePost(r)
  }

  return nil, &httpError{
    status:  http.StatusMethodNotAllowed,
    message: "GraphQL only supports POST and GET requests.",
    error:   errors.New(r.Method + " is not allowed"),
  }
}

func parseGet(r *http.Request) (*request, *httpError) {

  v := r.URL.Query()
  var (
    queries   = v["query"]
    opNames   = v["operationName"]
    variables = v["variables"]
    qLen      = len(queries)
    nLen      = len(opNames)
    vLen      = len(variables)
  )

  if qLen == 0 {
    return nil, &httpError{
      status:  http.StatusBadRequest,
      message: "Missing request parameters",
      error:   errors.New("missing request parameters"),
    }
  }

  requests := make([]gqlRequest, 0, qLen)

  // This loop assumes there will be a corresponding element at each index
  // for query, operation name, and variable fields.
  // TODO maybe we should do some validation?
  for i, q := range queries {
    var opName string

    if i < nLen {
      opName = opNames[i]
    }

    var m = map[string]interface{}{}
    if i < vLen {
      variable := variables[i]
      if err := json.Unmarshal([]byte(variable), &m); err != nil {
        return nil, &httpError{
          status:  http.StatusBadRequest,
          message: "Unable to read variables.",
          error:   err,
        }
      }
    }

    requests = append(requests, gqlRequest{q, opName, m})
  }

  return &request{requests: requests, batch: qLen > 1}, nil
}

func parsePost(r *http.Request) (*request, *httpError) {

  readBodyErr := &httpError{
    status:  http.StatusBadRequest,
    message: "Unable to read body.",
  }

  // read and close the body
  body, err := ioutil.ReadAll(r.Body)
  if err != nil {
    readBodyErr.error = err
    return nil, readBodyErr
  }
  r.Body.Close()

  if len(body) == 0 {
    return nil, &httpError{
      status:  http.StatusBadRequest,
      message: "Missing request body.",
      error:   errors.New("missing request body"),
    }
  }

  var requests []gqlRequest

  // Graphql content type request will send only one query
  if strings.HasPrefix(r.Header.Get("Content-Type"), ContentTypeGraphQL) {
    req := gqlRequest{}
    req.Query = string(body)
    requests = append(requests, req)
  } else {
    // Inspect the first character to inform how the body is parsed.
    switch body[0] {
    case '{':
      req := gqlRequest{}
      if err := json.Unmarshal(body, &req); err != nil {
        readBodyErr.error = err
        return nil, readBodyErr
      }
      requests = append(requests, req)
    case '[':
      if err := json.Unmarshal(body, &requests); err != nil {
        readBodyErr.error = err
        return nil, readBodyErr
      }
    }
  }

  return &request{requests: requests, batch: len(requests) > 1}, nil
}
