package cmd

import (
  "bytes"
  "io/ioutil"
  "log"
  "os"
  "path"

  "github.com/dealtap/graphql-gen-go/generator"
  "github.com/spf13/cobra"
)

var (
  pkgName string
  outDir  string
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
  Use:   "graphql-gen-go",
  Short: "",
  Long:  ``,
  Run: func(cmd *cobra.Command, args []string) {
    fileData := &bytes.Buffer{}

    files := args

    for _, file := range files {
      f, err := ioutil.ReadFile(file)
      if err != nil {
        log.Fatal(err)
      }
      fileData.WriteString("\n")
      fileData.Write(f)
    }

    // generate resolver output
    resGen := generator.New()
    err := resGen.Parse(fileData.Bytes())
    check(err)
    resOut := resGen.SetPkgName(pkgName).GenSchemaResolversFile()

    // generate server output
    srvGen := generator.New()
    srvOut := srvGen.SetPkgName(pkgName).GenServerFile()

    targetDir := outDir
    if pkgName != "main" {
      targetDir = path.Join(outDir, "/", pkgName)
    }

    // create directory if it does not exist
    if _, err = os.Stat(targetDir); os.IsNotExist(err) {
      os.Mkdir(targetDir, os.ModePerm)
    }

    // create resolver file
    resFile := pkgName + ".gql.go"
    createFile(targetDir, resFile, resOut)

    // create server file
    srvFile := "server.gql.go"
    createFile(targetDir, srvFile, srvOut)
  },
}

func createFile(dir, fileName string, out []byte) {
  outFile := path.Join(dir, fileName)
  // open the file and write to it
  f, err := os.Create(outFile)
  check(err)
  defer f.Close()
  _, err = f.Write(out)
  check(err)
  f.Sync()
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
  if err := RootCmd.Execute(); err != nil {
    log.Fatal(err)
    os.Exit(-1)
  }
}

func init() {
  RootCmd.PersistentFlags().StringVar(&pkgName, "pkg", "main", "generated golang package name")
  RootCmd.PersistentFlags().StringVar(&outDir, "out_dir", "./", "output directory (default is current directory)")
}

func check(err error) {
  if err != nil {
    log.Fatal(err)
  }
}
