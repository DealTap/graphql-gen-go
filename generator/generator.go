package generator

import (
  "bytes"
  "fmt"
  "log"
  "os"
  "strings"
  "unicode"

  "github.com/neelance/graphql-go"
  "github.com/neelance/graphql-go/introspection"
)

const (
  gqlSCALAR       = "SCALAR"
  gqlINTERFACE    = "INTERFACE"
  gqlINPUT_OBJECT = "INPUT_OBJECT"
  gqlUNION        = "UNION"
  gqlLIST         = "LIST"
  gqlOBJECT       = "OBJECT"
  gqlENUM         = "ENUM"
  gqlDIRECTIVE    = "DIRECTIVE"
  gqlID           = "ID"
  gqlQuery        = "Query"
  gqlMutation     = "Mutation"
)

var KnownGQLTypes = map[string]bool{
  "__Directive":         true,
  "__DirectiveLocation": true,
  "__EnumValue":         true,
  "__Field":             true,
  "__InputValue":        true,
  "__Schema":            true,
  "__Type":              true,
  "__TypeKind":          true,
  "LIST":                true,
  "String":              true,
  "Float":               true,
  "ID":                  true,
  "Int":                 true,
  "Boolean":             true,
  "Time":                true,
}

var KnownGoTypes = map[string]bool{
  "string":  true,
  "bool":    true,
  "float32": true,
  "float64": true,
  "int":     true,
  "int32":   true,
  "int64":   true,
}

type TypeDef struct {
  Name        string
  Description string
  Fields      map[string]*FieldDef
  GQLType     string
  gqlType     *introspection.Type
}

func NewType(t *introspection.Type) *TypeDef {

  tp := &TypeDef{
    Name:        pts(t.Name()),
    Description: pts(t.Description()),
    Fields:      map[string]*FieldDef{},
    gqlType:     t,
  }

  /**
    * union & input object types do not have fields
    * so we ignore it to avoid nil pointer dereference error
    * for input object type we create fields from InputFields instead
   */
  if t.Kind() != gqlUNION && t.Kind() != gqlINPUT_OBJECT {
    for _, fld := range *t.Fields(nil) {
      f := NewField(fld)
      f.Parent = tp.Name
      tp.Fields[f.Name] = f
    }
  } else if t.Kind() == gqlINPUT_OBJECT {
    for _, input := range *t.InputFields() {
      f := newField(input.Name(), input.Description(), input.Type())
      f.Parse()
      tp.Fields[f.Name] = f
    }
  }

  return tp
}

type FieldDef struct {
  Name        string
  Parent      string
  Description string
  Type        *Typ
  gqlField    *introspection.Field
  Args        []*FieldDef
}

func fieldName(name string) string {
  name = upperFirst(name)
  if name == "Id" {
    name = "ID"
  }
  return name
}

func newField(name string, desc *string, typ *introspection.Type) *FieldDef {
  return &FieldDef{
    Name:        fieldName(name),
    Description: pts(desc),
    Type: &Typ{
      IsNullable: true,
      gqlType:    typ,
    },
  }
}

func NewField(t *introspection.Field) *FieldDef {

  fld := newField(t.Name(), t.Description(), t.Type())
  fld.Parse()

  // parse arguments (i.e., interface function)
  for _, arg := range t.Args() {
    argFld := newField(arg.Name(), arg.Description(), arg.Type())
    argFld.Parse()
    fld.Args = append(fld.Args, argFld)
  }

  return fld
}

func (f *FieldDef) Parse() {

  tp := f.Type.gqlType
  td := f.Type

FindGoType:
  td.gqlType = tp
  if tp.Kind() == "NON_NULL" {
    td.IsNullable = false
    tp = tp.OfType()
  }

  if tp.Kind() == "LIST" {
    td.GoType = "[]"
    td.GQLType = "[]"

    td.Type = &Typ{
      IsNullable: true,
    }

    tp = tp.OfType()
    td = td.Type
    goto FindGoType
  }

  switch *tp.Name() {
  case "String":
    td.GoType = "string"
    td.GQLType = "string"
  case "Int":
    td.GoType = "int32"
    td.GQLType = "int32"
  case "Float":
    td.GoType = "float32"
    td.GQLType = "float32"
  case "ID":
    // TODO - shouldn't we use graphql.ID type for `ID` fields
    // because it may not work for query and mutation calls?
    td.GoType = "string"
    td.GQLType = "graphql.ID"
  case "Boolean":
    td.GoType = "bool"
    td.GQLType = "bool"
  case "Time":
    td.GoType = "time.Time"
    td.GQLType = "graphql.Time"
  default:
    if tp.Kind() == gqlENUM {
      td.GoType = "string"
      td.GQLType = "string"
    } else {
      td.GoType = pts(tp.Name())
      td.GQLType = pts(tp.Name()) + "Resolver"
    }
  }
  return
}

type Typ struct {
  GoType     string
  GQLType    string
  IsNullable bool
  Type       *Typ
  Values     []string
  gqlType    *introspection.Type
}

func (t Typ) genType(mode string) string {

  var r string

  if mode == "struct" || mode == "argStruct" {
    r = t.GoType
  } else {
    r = t.GQLType
  }

  if mode == "struct" {
    ok := KnownGoTypes[t.GoType]
    if t.IsNullable && t.GQLType != "[]" && !ok {
      r = "*" + r
    }
  } else {
    if t.IsNullable {
      r = "*" + r
    }
  }

  if t.Type == nil {
    return r
  }

  r += t.Type.genType(mode)
  return r
}

// FIXME could be refactored and become part of GenStruct()
func (t *TypeDef) GenInterface() string {
  r := "type " + t.Name + " interface {\n"
  for _, f := range t.Fields {
    r += "  " + f.Name + "("
    if len(f.Args) > 0 {
      r += f.Name + "Request"
    }
    r += ") " + f.Type.genType("interface") + "\n"
  }
  r += "}"
  return r
}

// FIXME could be refactored and become part of GenResStruct()
func (t *TypeDef) GenInterfaceResStruct(typePkgName string) string {
  if typePkgName != "" {
    typePkgName = typePkgName + "."
  }
  r := "type " + t.Name + "Resolver struct {\n"
  r += "  r *" + t.Name + "\n"
  r += "}"
  return r
}

func (t *TypeDef) GenStruct(typ string) string {
  r := "type " + t.Name + " struct {\n"
  for _, fld := range t.Fields {
    r += "  " + fld.Name + " " + fld.Type.genType(typ) + "\n"
  }
  r += "}"
  return r
}

func (t *TypeDef) GenResStruct(typePkgName string) string {
  if typePkgName != "" {
    typePkgName = typePkgName + "."
  }
  r := "type " + t.Name + "Resolver struct {\n"
  r += "  R *" + t.Name + "\n"
  r += "}"
  return r
}

// FIXME could be refactored and become part of GenResStruct()
func (f *FieldDef) GenFuncArgs() string {
  r := "type " + f.Name + "Request struct {\n"
  for _, arg := range f.Args {
    r += "  " + arg.Name + " " + arg.Type.genType("argStruct") + "\n"
  }
  r += "}"
  return r
}

// Resolvers generates the resolver function for the given FieldDef
func (f *FieldDef) GenResolver() string {
  res := f.Type.genType("resolver")
  returnType := res
  r := "func (r " + f.Parent + "Resolver) " + f.Name + "("
  r += ") " + res + " {\n"
  itm := ""

  if f.Type.GQLType == "[]" {
    if f.Type.IsNullable {
      returnType = strings.Replace(returnType, "*", "", 1)
    }
    if _, ok := KnownGoTypes[f.Type.Type.GoType]; !ok {
      ref := ""
      dref := ""
      if f.Type.Type.IsNullable {
        itm = "&"
        dref = "*"
      } else {
        ref = "&"
      }

      if f.Type.Type.GQLType == "graphql.ID" {
        itm = f.Type.Type.GQLType + "(itm)"
      } else if f.Type.Type.GQLType == "graphql.Time" {
        itm = itm + f.Type.Type.GQLType + "{Time: " + dref + "itm}"
      } else {
        itm = itm + f.Type.Type.GQLType + "{" + ref + "itm}"
      }
    } else {
      if f.Type.Type.IsNullable {
        itm = "&itm"
      }
    }

    if itm == "" {
      itm = "itm"
    }
    r += "  items := " + returnType + "{}\n"
    r += "  for _, itm := range r.R." + f.Name + " {\n"
    r += "    items = append(items, " + itm + ")\n"
    r += "  }\n"
    r += "  return "
    if f.Type.IsNullable {
      r += "&"
    }
    r += "items\n"
    r += "}"
    return r
  }

  if f.Type.GQLType != "graphql.ID" {
    r += "  return "
    if f.Type.IsNullable {
      r += "&"
    }
  }

  if _, ok := KnownGoTypes[f.Type.GQLType]; !ok {
    if f.Type.GQLType == "graphql.ID" {
      r += "  id := graphql.ID(r.R." + f.Name + ")\n"
      r += "  return "
      if f.Type.IsNullable {
        r += "&"
      }
      r += "id"
    } else if f.Type.GQLType == "graphql.Time" {
      dref := ""
      if f.Type.IsNullable {
        dref = "*"
      }
      r += f.Type.GQLType + "{Time: " + dref + "r.R." + f.Name + "}"
    } else {
      ref := ""
      if !f.Type.IsNullable {
        ref = "&"
      }
      r += f.Type.GQLType + "{" + ref + "r.R." + f.Name + "}"
    }
  } else {
    r += "r.R." + f.Name
  }

  r += "\n}"
  return r
}

func (t *TypeDef) GenUnionResStruct() string {
  r := "type " + t.Name + "Resolver struct {\n"
  r += "  Result interface{}\n"
  r += "}"
  return r
}

func (t *TypeDef) GenUnionResolver(parentName string) string {
  r := "func (r *" + t.Name + "Resolver) To" + parentName + "() (*" + parentName + "Resolver, bool) {\n"
  r += "  res, ok := r.Result.(*" + parentName + "Resolver)\n"
  r += "  return res, ok\n"
  r += "}"
  return r
}

type Generator struct {
  *bytes.Buffer

  PkgName   string
  rawSchema []byte
  schema    *graphql.Schema

  //Param             map[string]string // Command-line parameters.
  //PackageImportPath string            // Go import path of the package we're generating code for
  //ImportPrefix      string            // String to prefix to imported package file names.
  //ImportMap         map[string]string // Mapping from .proto file name to import path

  //Pkg map[string]string // The names under which we import support packages

  //packageName string // What we're calling ourselves.
  //allFiles         []*FileDescriptor          // All files in the tree
  //allFilesByName   map[string]*FileDescriptor // All files by filename.
  //genFiles         []*FileDescriptor          // Those files we will generate output for.
  //file             *FileDescriptor            // The file we are compiling now.
  //usedPackages map[string]bool // Names of packages used in current file.
  //typeNameToObject map[string]Object          // Key is a fully-qualified name in input syntax.
  //init   []string // Lines to emit in the init function.
  indent string

  writeOutput bool
}

// P prints the arguments to the generated output.  It handles strings and int32s, plus
// handling indirections because they may be *string, etc.
func (g *Generator) P(str ...interface{}) {
  if !g.writeOutput {
    return
  }
  g.WriteString(g.indent)
  for _, v := range str {
    switch s := v.(type) {
    case string:
      g.WriteString(s)
    case *string:
      g.WriteString(*s)
    case bool:
      fmt.Fprintf(g, "%t", s)
    case *bool:
      fmt.Fprintf(g, "%t", *s)
    case int:
      fmt.Fprintf(g, "%d", s)
    case *int32:
      fmt.Fprintf(g, "%d", *s)
    case *int64:
      fmt.Fprintf(g, "%d", *s)
    case float64:
      fmt.Fprintf(g, "%g", s)
    case *float64:
      fmt.Fprintf(g, "%g", *s)
    default:
      g.Fail(fmt.Sprintf("unknown type in printer: %T", v))
    }
  }
  g.WriteByte('\n')
}

// New creates a new generator and allocates the request and response protobufs.
func New() *Generator {
  g := new(Generator)
  g.Buffer = new(bytes.Buffer)
  g.writeOutput = true
  //g.Request = new(plugin.CodeGeneratorRequest)
  //g.Response = new(plugin.CodeGeneratorResponse)
  return g
}

func (g *Generator) Parse(fileData []byte) error {
  g.rawSchema = fileData
  schema, err := graphql.ParseSchema(string(fileData), nil)
  g.schema = schema
  return err
}

func (g *Generator) SetPkgName(name string) *Generator {
  g.PkgName = name
  return g
}

// In Indents the output one tab stop.
func (g *Generator) In() { g.indent += "\t" }

// Out unindents the output one tab stop.
func (g *Generator) Out() {
  if len(g.indent) > 0 {
    g.indent = g.indent[1:]
  }
}

// Error reports a problem, including an error, and exits the program.
func (g *Generator) Error(err error, msgs ...string) {
  s := strings.Join(msgs, " ") + ":" + err.Error()
  log.Print("graphql-gen-go: error:", s)
  os.Exit(1)
}

// Fail reports a problem and exits the program.
func (g *Generator) Fail(msgs ...string) {
  s := strings.Join(msgs, " ")
  log.Print("graphql-gen-go: error:", s)
  os.Exit(1)
}

type SchemaMap struct {
  RootTypes     map[string]string
  ResolverTypes map[string]string
  ResolverFuncs map[string]string
}

func NewSchemaMap() *SchemaMap {
  return &SchemaMap{
    RootTypes:     map[string]string{},
    ResolverTypes: map[string]string{},
    ResolverFuncs: map[string]string{},
  }
}

// Fill the buffer with the generated output for all the files we're supposed to generate.
func (g Generator) GenSchemaResolversFile() []byte {

  g.P("package ", g.PkgName)
  g.P("")
  // FIXME extract imports out of generator or find a better way to generate them
  // Check protoc-gen-go codebase
  g.P("import (")
  g.In()
  // FIXME include only when time field is present
  //g.P(`"time"`)
  g.P(`graphql "github.com/neelance/graphql-go"`)
  g.Out()
  g.P(")")
  g.P("")

  types := []*TypeDef{}
  resType := &TypeDef{
    Name:   "GqlResolver",
    Fields: make(map[string]*FieldDef),
  }
  for _, typ := range g.schema.Inspect().Types() {
    if KnownGQLTypes[*typ.Name()] {
      continue
    }
    switch typ.Kind() {
    case gqlOBJECT:
      gtp := NewType(typ)
      typName := pts(typ.Name())

      // save Query & Mutation definitions to be generated later
      if typName == gqlQuery || typName == gqlMutation {
        for key, value := range gtp.Fields {
          resType.Fields[key] = value
        }
      } else {
        g.P(gtp.GenStruct("struct"))
        g.P("")

        g.P(gtp.GenResStruct(""))
        g.P("")

        for _, f := range gtp.Fields {
          // do not generate a resolver function that has additional arguments
          // as it requires additional logic
          // let the user create it manually
          if len(f.Args) == 0 {
            g.P(f.GenResolver())
            g.P("")
          }
        }
      }

      types = append(types, gtp)
    case gqlSCALAR:
      //TODO: Implement scalar type code generation
      fmt.Printf("%s not implemented yet\n", *typ.Name())
    case gqlINTERFACE:
      gtp := NewType(typ)

      g.P(gtp.GenInterface())
      g.P("")

      g.P(gtp.GenInterfaceResStruct(""))
      g.P("")

      types = append(types, gtp)
    case gqlENUM:
      //TODO: should we generate a pseudo enum or stick with string?
    case gqlUNION:
      gtp := NewType(typ)

      g.P(gtp.GenUnionResStruct())
      g.P("")

      for _, t := range *typ.PossibleTypes() {
        g.P(gtp.GenUnionResolver(pts(t.Name())))
        g.P("")
      }
    case gqlINPUT_OBJECT:
      gtp := NewType(typ)
      g.P(gtp.GenStruct("argStruct"))
      g.P("")
    default:
      fmt.Println("unknown graphql type ", *typ.Name(), ":", typ.Kind())
    }
  }

  // generate additional structs for func arguments
  fncArgs := make(map[string]struct{})
  for _, t := range types {
    for _, f := range t.Fields {
      if len(f.Args) > 0 {
        fnArgName := f.Name + "Request"
        _, exists := fncArgs[fnArgName]
        if exists == false {
          fncArgs[fnArgName] = struct{}{}
          g.P(f.GenFuncArgs())
          g.P("")
        }
      }
    }
  }

  // generate interface for Query & Mutation
  g.P(resType.GenInterface())
  g.P("")

  g.P("var Schema = `")
  g.P(string(g.rawSchema))
  g.P("`")

  return g.Bytes()
}

func (g Generator) GenServerFile() []byte {

  g.P("package ", g.PkgName)
  g.P("")
  g.P("import (")
  g.In()
  g.P(`"encoding/json"`)
  g.P(`"errors"`)
  g.P(`"io/ioutil"`)
  g.P(`"net/http"`)
  g.P(`"strings"`)
  g.P(`"sync"`)
  g.P("")
  g.P(`graphql "github.com/neelance/graphql-go"`)
  g.P(`"github.com/rs/cors"`)
  g.Out()
  g.P(")")
  g.P("")

  // generate code to run graphql server
  g.P(GenServer())

  //out := append(g.Bytes(), srvFile...)
  //return out
  return g.Bytes()
}

func GenServer() string {

  // FIXME should we read this from a file
  s := `const (
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
    Schema: graphql.MustParseSchema(Schema, res),
    Port:   port,
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
  Query     string                 ` + "`" + `json:"query"` + "`" + `
  OpName    string                 ` + "`" + `json:"operationName"` + "`" + `
  Variables map[string]interface{} ` + "`" + `json:"variables"` + "`" + `
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
}`
  return s
}

// Helper functions
func lowerFirst(s string) string {
  return firstCharToCase(s, "lower")
}

func upperFirst(s string) string {
  return firstCharToCase(s, "upper")
}

func firstCharToCase(s string, c string) string {
  done := false
  mapFunc := func(r rune) rune {
    if done || !unicode.IsLetter(r) {
      return r
    }
    done = true
    if c == "upper" {
      return unicode.ToUpper(r)
    }
    return unicode.ToLower(r)
  }
  return strings.Map(mapFunc, s)
}

func pts(s *string) string {
  if s == nil {
    return ""
  }
  return *s
}
